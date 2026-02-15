package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

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

	systemPrompt, initialResult, err := util.GenerateStartChat(h.ai, startChatRequest.Role, startChatRequest.Topic, config.GetLanguageName(startChatRequest.Language), subtitleLanguage)
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

	initialAudio, err := util.GenerateSpeech(h.el, initialResult.Response, voice)
	if err != nil {
		log.Printf("failed to generate speech: %v", err)
		util.SendResponse(w, nil, "failed to generate speech", http.StatusInternalServerError)

		return
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
			Text:  initialResult.Response,
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
			Text:     initialResult.Response,
			Audio:    initialAudio,
			Subtitle: initialResult.ResponseSubtitle,
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

	// Resolve subtitle language name
	subtitleLanguage := ""
	if user.SubtitleLanguage != "" {
		subtitleLanguage = config.GetLanguageName(config.GetCode(user.SubtitleLanguage))
	}

	// Step 1: Transcribe audio using gpt-4o-mini-transcribe
	audioReader := io.NopCloser(bytes.NewReader(audioBytes))
	transcript, err := util.TranscribeSpeech(h.ai, audioReader, fileHeader.Filename, user.Language)
	if err != nil {
		log.Printf("failed to transcribe speech: %v", err)
		util.SendResponse(w, nil, "failed to transcribe speech", http.StatusInternalServerError)

		return
	}

	// Step 2: Generate response using gpt-4o-mini with JSON format
	history := util.ConvertToChatMessage(entries)

	answerResult, err := util.GenerateAnswerChat(h.ai, history, transcript, subtitleLanguage)
	if err != nil {
		log.Printf("failed to get chat completion: %v", err)
		util.SendResponse(w, nil, fmt.Sprintf("failed to get chat completion: %v", err), http.StatusInternalServerError)

		return
	}

	// Set the transcript from Whisper (not from the chat model)
	answerResult.Transcript = transcript

	answerAudio, err := util.GenerateSpeech(h.el, answerResult.Response, user.Voice)
	if err != nil {
		log.Printf("failed to generate speech: %v", err)
		util.SendResponse(w, nil, "failed to generate speech", http.StatusInternalServerError)

		return
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
			Text: answerResult.Transcript,
		},
		{
			Role:  string(openai.ROLE_ASSISTANT),
			Text:  answerResult.Response,
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
		IsLast:   answerResult.IsLast,
		Prompt: model.Chat{
			Text:     answerResult.Transcript,
			Subtitle: answerResult.TranscriptSubtitle,
		},
		Answer: model.Chat{
			Text:     answerResult.Response,
			Audio:    answerAudio,
			Subtitle: answerResult.ResponseSubtitle,
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

	entries, err := h.db.GetChatsByChatUserID(user.ID)
	if err != nil {
		log.Printf("failed to get chat: %v", err)
		util.SendResponse(w, nil, "failed to get chat", http.StatusInternalServerError)

		return
	}

	// Resolve subtitle language name
	subtitleLanguage := ""
	if user.SubtitleLanguage != "" {
		subtitleLanguage = config.GetLanguageName(config.GetCode(user.SubtitleLanguage))
	}

	history := util.ConvertToChatMessage(entries)

	endResult, err := util.GenerateEndChat(h.ai, history, subtitleLanguage)
	if err != nil {
		log.Printf("failed to get chat completion: %v", err)
		util.SendResponse(w, nil, "failed to get chat completion", http.StatusInternalServerError)

		return
	}

	answerAudio, err := util.GenerateSpeech(h.el, endResult.Response, user.Voice)
	if err != nil {
		log.Printf("failed to generate speech: %v", err)
		util.SendResponse(w, nil, "failed to generate speech", http.StatusInternalServerError)

		return
	}

	tx, err := h.db.BeginTx()
	if err != nil {
		log.Printf("failed to begin transaction: %v", err)
		util.SendResponse(w, nil, "failed to create new chat", http.StatusInternalServerError)

		return
	}
	defer tx.Rollback()

	if _, err := h.db.CreateChat(tx, userID, string(openai.ROLE_ASSISTANT), endResult.Response, answerAudio); err != nil {
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
			Text:     endResult.Response,
			Audio:    answerAudio,
			Subtitle: endResult.ResponseSubtitle,
		},
	}

	util.SendResponse(w, response, "success", http.StatusOK)
}
