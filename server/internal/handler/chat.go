package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/madeindra/mock-conversation/server/internal/config"
	"github.com/madeindra/mock-conversation/server/internal/data"
	"github.com/madeindra/mock-conversation/server/internal/middleware"
	"github.com/madeindra/mock-conversation/server/internal/model"
	"github.com/madeindra/mock-conversation/server/internal/openai"
	"github.com/madeindra/mock-conversation/server/internal/util"
)

func (h *handler) Status(w http.ResponseWriter, _ *http.Request) {
	isKeyValid, err := h.ai.IsKeyValid()
	if err != nil {
		log.Printf("failed to check key validity: %v", err)
		util.SendResponse(w, nil, "failed to check key validity", http.StatusInternalServerError)

		return
	}

	status, err := h.ai.Status()
	if err != nil {
		log.Printf("failed to check API availability: %v", err)
		util.SendResponse(w, nil, "failed to check API availability", http.StatusInternalServerError)

		return
	}

	var apiState *bool

	switch status {
	case openai.STATUS_OPERATIONAL:
		apiState = util.Pointer(true)
	case openai.STATUS_DEGRADED_PERFORMANCE, openai.STATUS_PARTIAL_OUTAGE, openai.STATUS_MAJOR_OUTAGE:
		apiState = util.Pointer(false)
	case openai.STATUS_UNKNOWN:
		apiState = nil
	}

	apiStatus := util.Pointer(string(status))

	response := model.StatusResponse{
		Server:    true,
		Key:       isKeyValid,
		API:       apiState,
		ApiStatus: apiStatus,
	}

	util.SendResponse(w, response, "success", http.StatusOK)
}

func (h *handler) StartChat(w http.ResponseWriter, req *http.Request) {
	var startChatRequest model.StartChatRequest
	if err := json.NewDecoder(req.Body).Decode(&startChatRequest); err != nil {
		log.Printf("failed to read start chat request body: %v", err)
		util.SendResponse(w, nil, "failed to read request", http.StatusBadRequest)

		return
	}

	chatLanguage := h.ai.GetDefaultTranscriptLanguage()
	if startChatRequest.Language != "" {
		chatLanguage = config.GetLanguage(startChatRequest.Language)
	}

	// Resolve subtitle language name for translation
	subtitleLanguage := ""
	if startChatRequest.SubtitleLanguage != "" {
		subtitleLanguage = config.GetLanguageName(startChatRequest.SubtitleLanguage)
	}

	systemPrompt, initialText, err := util.GetChatAssets(h.ai, startChatRequest.Role, startChatRequest.Topic, config.GetLanguageName(startChatRequest.Language))
	if err != nil {
		log.Printf("failed to get system prompt or initial text: %v", err)
		util.SendResponse(w, nil, "failed to prepare chat", http.StatusInternalServerError)

		return
	}

	// Pick a random ElevenLabs voice for this conversation
	var voice string
	if h.el != nil {
		voice = h.el.RandomVoice()
	}

	initialAudio, err := util.GenerateSpeech(h.el, initialText, voice)
	if err != nil {
		log.Printf("failed to generate speech: %v", err)
		util.SendResponse(w, nil, "failed to generate speech", http.StatusInternalServerError)

		return
	}

	// Generate subtitle for initial text if subtitle language is enabled
	var initialSubtitle string
	if subtitleLanguage != "" {
		initialSubtitle, err = util.GenerateSubtitle(h.ai, initialText, subtitleLanguage)
		if err != nil {
			log.Printf("failed to generate subtitle: %v", err)
			// Non-fatal: continue without subtitle
		}
	}

	plainSecret := util.GenerateRandom()
	hashed, err := util.CreateHash(plainSecret)
	if err != nil {
		log.Printf("failed to create hash: %v", err)
		util.SendResponse(w, nil, "failed to prepare chat", http.StatusInternalServerError)

		return
	}

	tx, err := h.db.BeginTx()
	if err != nil {
		log.Printf("failed to begin transaction: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}
	defer tx.Rollback()

	newUser, err := h.db.CreateChatUser(tx, hashed, chatLanguage, config.GetLanguage(startChatRequest.SubtitleLanguage), voice)
	if err != nil {
		log.Printf("failed to create new chat: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}

	if _, err := h.db.CreateChats(tx, newUser.ID, []data.Entry{
		{
			Role: string(openai.ROLE_SYSTEM),
			Text: systemPrompt,
		},
		{
			Role:  string(openai.ROLE_ASSISTANT),
			Text:  initialText,
			Audio: initialAudio,
		},
	}); err != nil {
		log.Printf("failed to create chat: %v", err)
		util.SendResponse(w, nil, "failed to create chat", http.StatusInternalServerError)

		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit transaction: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}

	initialChat := model.StartChatResponse{
		ID:       newUser.ID,
		Secret:   plainSecret,
		Language: startChatRequest.Language,
		Chat: model.Chat{
			Text:     initialText,
			Audio:    initialAudio,
			Subtitle: initialSubtitle,
		},
	}

	util.SendResponse(w, initialChat, "a new chat created", http.StatusOK)
}

func (h *handler) AnswerChat(w http.ResponseWriter, req *http.Request) {
	userID := req.Context().Value(middleware.ContextKeyUserID).(string)
	userSecret := req.Context().Value(middleware.ContextKeyUserSecret).(string)

	if userID == "" || userSecret == "" {
		log.Println("user ID or secret is missing")
		util.SendResponse(w, nil, "missing required authentication", http.StatusUnauthorized)

		return
	}

	user, err := h.db.GetChatUser(userID)
	if err != nil {
		log.Printf("failed to get chat user: %v", err)
		util.SendResponse(w, nil, "failed to get chat user", http.StatusNotFound)

		return
	}

	if err := util.CompareHash(userSecret, user.Secret); err != nil {
		log.Println("invalid user secret")
		util.SendResponse(w, nil, "invalid user secret", http.StatusUnauthorized)

		return
	}

	entries, err := h.db.GetChatsByChatUserID(user.ID)
	if err != nil {
		log.Printf("failed to get chat: %v", err)
		util.SendResponse(w, nil, "failed to get chat", http.StatusInternalServerError)

		return
	}

	file, fileHeader, err := req.FormFile("file")
	if err != nil {
		log.Printf("failed to read file: %v", err)
		util.SendResponse(w, nil, "failed to read file", http.StatusInternalServerError)

		return
	}
	if fileHeader == nil {
		log.Println("required file is missing")
		util.SendResponse(w, nil, "required file is missing", http.StatusBadRequest)

		return
	}
	defer file.Close()

	// Read audio file into memory
	audioBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("failed to read audio file: %v", err)
		util.SendResponse(w, nil, "failed to read audio file", http.StatusInternalServerError)

		return
	}

	// Transcribe audio via Whisper for user's transcript display
	audioReader := io.NopCloser(bytes.NewReader(audioBytes))
	transcriptText, err := util.TranscribeSpeech(h.ai, audioReader, fileHeader.Filename, user.Language)
	if err != nil {
		log.Printf("failed to transcribe speech: %v", err)
		util.SendResponse(w, nil, "failed to transcribe speech", http.StatusInternalServerError)

		return
	}

	// Send audio to gpt-audio-mini for the AI response
	audioBase64 := base64.StdEncoding.EncodeToString(audioBytes)
	history := util.ConvertToChatMessage(entries)

	answerText, err := util.GenerateTextFromAudio(h.ai, history, audioBase64, "wav")
	if err != nil {
		log.Printf("failed to get chat completion: %v", err)
		util.SendResponse(w, nil, fmt.Sprintf("failed to get chat completion: %v", err), http.StatusInternalServerError)

		return
	}

	// Detect if AI signals end of conversation with [END] marker
	isLast := false
	if strings.HasPrefix(answerText, "[END]") {
		isLast = true
		answerText = strings.TrimPrefix(answerText, "[END]")
		answerText = strings.TrimSpace(answerText)
	}

	answerAudio, err := util.GenerateSpeech(h.el, answerText, user.Voice)
	if err != nil {
		log.Printf("failed to generate speech: %v", err)
		util.SendResponse(w, nil, "failed to generate speech", http.StatusInternalServerError)

		return
	}

	// Generate subtitle translation if enabled
	var answerSubtitle string
	var promptSubtitle string
	if user.SubtitleLanguage != "" {
		subtitleLangName := config.GetLanguageName(config.GetCode(user.SubtitleLanguage))
		answerSubtitle, err = util.GenerateSubtitle(h.ai, answerText, subtitleLangName)
		if err != nil {
			log.Printf("failed to generate answer subtitle: %v", err)
		}
		promptSubtitle, err = util.GenerateSubtitle(h.ai, transcriptText, subtitleLangName)
		if err != nil {
			log.Printf("failed to generate prompt subtitle: %v", err)
		}
	}

	tx, err := h.db.BeginTx()
	if err != nil {
		log.Printf("failed to begin transaction: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}
	defer tx.Rollback()

	if _, err := h.db.CreateChats(tx, userID, []data.Entry{
		{
			Role: string(openai.ROLE_USER),
			Text: transcriptText,
		},
		{
			Role:  string(openai.ROLE_ASSISTANT),
			Text:  answerText,
			Audio: answerAudio,
		},
	}); err != nil {
		log.Printf("failed to create chat: %v", err)
		util.SendResponse(w, nil, "failed to create chat", http.StatusInternalServerError)

		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit transaction: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}

	response := model.AnswerChatResponse{
		Language: config.GetCode(user.Language),
		IsLast:   isLast,
		Prompt: model.Chat{
			Text:     transcriptText,
			Subtitle: promptSubtitle,
		},
		Answer: model.Chat{
			Text:     answerText,
			Audio:    answerAudio,
			Subtitle: answerSubtitle,
		},
	}

	util.SendResponse(w, response, "success", http.StatusOK)
}

func (h *handler) EndChat(w http.ResponseWriter, req *http.Request) {
	userID := req.Context().Value(middleware.ContextKeyUserID).(string)
	userSecret := req.Context().Value(middleware.ContextKeyUserSecret).(string)

	if userID == "" || userSecret == "" {
		log.Println("user ID or secret is missing")
		util.SendResponse(w, nil, "missing required authentication", http.StatusUnauthorized)

		return
	}

	user, err := h.db.GetChatUser(userID)
	if err != nil {
		log.Printf("failed to get chat user: %v", err)
		util.SendResponse(w, nil, "failed to get chat user", http.StatusNotFound)

		return
	}

	if err := util.CompareHash(userSecret, user.Secret); err != nil {
		log.Println("invalid user secret")
		util.SendResponse(w, nil, "invalid user secret", http.StatusUnauthorized)

		return
	}

	entry, err := h.db.GetChatsByChatUserID(user.ID)
	if err != nil {
		log.Printf("failed to get chat: %v", err)
		util.SendResponse(w, nil, "failed to get chat", http.StatusInternalServerError)

		return
	}

	history := util.ConvertToChatMessage(entry)

	chatHistory := append(history, openai.ChatMessage{
		Role:    openai.ROLE_USER,
		Content: "[ENDCONV]",
	})

	answerText, err := util.GenerateText(h.ai, chatHistory)
	if err != nil {
		log.Printf("failed to get chat completion: %v", err)
		util.SendResponse(w, nil, "failed to get chat completion", http.StatusInternalServerError)

		return
	}

	// Strip [END] marker if the AI includes it
	if strings.HasPrefix(answerText, "[END]") {
		answerText = strings.TrimPrefix(answerText, "[END]")
		answerText = strings.TrimSpace(answerText)
	}

	answerAudio, err := util.GenerateSpeech(h.el, answerText, user.Voice)
	if err != nil {
		log.Printf("failed to generate speech: %v", err)
		util.SendResponse(w, nil, "failed to generate speech", http.StatusInternalServerError)

		return
	}

	// Generate subtitle for end message if enabled
	var answerSubtitle string
	if user.SubtitleLanguage != "" {
		subtitleLangName := config.GetLanguageName(config.GetCode(user.SubtitleLanguage))
		answerSubtitle, err = util.GenerateSubtitle(h.ai, answerText, subtitleLangName)
		if err != nil {
			log.Printf("failed to generate subtitle: %v", err)
		}
	}

	tx, err := h.db.BeginTx()
	if err != nil {
		log.Printf("failed to begin transaction: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}
	defer tx.Rollback()

	if _, err := h.db.CreateChat(tx, userID, string(openai.ROLE_ASSISTANT), answerText, answerAudio); err != nil {
		log.Printf("failed to create chat: %v", err)
		util.SendResponse(w, nil, "failed to create chat", http.StatusInternalServerError)

		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit transaction: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}

	response := model.AnswerChatResponse{
		Language: config.GetCode(user.Language),
		IsLast:   true,
		Answer: model.Chat{
			Text:     answerText,
			Audio:    answerAudio,
			Subtitle: answerSubtitle,
		},
	}

	util.SendResponse(w, response, "success", http.StatusOK)
}
