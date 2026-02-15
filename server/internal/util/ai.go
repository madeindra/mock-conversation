package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/madeindra/mock-conversation/server/internal/openai"
)

func GenerateStartChat(ai openai.Client, role, topic, language, subtitleLanguage string) (string, openai.AnswerChatResult, error) {
	if ai == nil {
		return "", openai.AnswerChatResult{}, fmt.Errorf("unsupported client")
	}

	systemPrompt, err := openai.GetSystemPrompt(role, topic, language)
	if err != nil {
		return "", openai.AnswerChatResult{}, err
	}

	jsonInstruction := `Respond in JSON with: {"response": "your greeting"}`
	if subtitleLanguage != "" {
		jsonInstruction = fmt.Sprintf(`Respond in JSON with: {"response": "your greeting", "responseSubtitle": "complete and accurate translation of your entire greeting in %s"}`, subtitleLanguage)
	}

	messages := []openai.ChatMessage{
		{
			Role:    openai.ROLE_SYSTEM,
			Content: systemPrompt,
		},
		{
			Role:    openai.ROLE_USER,
			Content: "Start the conversation with a brief greeting and introduce the topic. " + jsonInstruction,
		},
	}

	rawJSON, err := ai.Chat(messages)
	if err != nil {
		return "", openai.AnswerChatResult{}, err
	}

	var result openai.AnswerChatResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return "", openai.AnswerChatResult{}, fmt.Errorf("failed to parse initial chat JSON: %w, raw: %s", err, rawJSON)
	}

	if result.Response == "" {
		return "", openai.AnswerChatResult{}, fmt.Errorf("empty initial chat response")
	}

	return systemPrompt, result, nil
}

func TranscribeSpeech(ai openai.Client, audio io.Reader, filename string, language string) (string, error) {
	if ai == nil {
		return "", fmt.Errorf("unsupported client")
	}

	return ai.Transcribe(audio, filename, language)
}

func GenerateAnswerChat(ai openai.Client, history []openai.ChatMessage, transcript string, subtitleLanguage string) (openai.AnswerChatResult, error) {
	if ai == nil {
		return openai.AnswerChatResult{}, fmt.Errorf("unsupported client")
	}

	jsonInstruction := `You MUST respond in JSON with: {"response": "your reply", "isLast": false}. Set isLast to true only when the conversation is ending (user says goodbye or you decide to end it). When isLast is true, respond with a natural farewell.`
	if subtitleLanguage != "" {
		jsonInstruction = fmt.Sprintf(`You MUST respond in JSON with: {"response": "your reply", "responseSubtitle": "complete and accurate translation of your entire reply in %s", "transcriptSubtitle": "complete and accurate translation of the user's entire message in %s", "isLast": false}. Set isLast to true only when the conversation is ending (user says goodbye or you decide to end it). When isLast is true, respond with a natural farewell.`, subtitleLanguage, subtitleLanguage)
	}

	// Copy history and inject JSON instruction into system prompt
	messages := make([]openai.ChatMessage, len(history))
	copy(messages, history)

	for i, msg := range messages {
		if msg.Role == openai.ROLE_SYSTEM {
			messages[i].Content = msg.Content + "\n\n" + jsonInstruction
			break
		}
	}

	// Add the user's transcript as a new user message
	messages = append(messages, openai.ChatMessage{
		Role:    openai.ROLE_USER,
		Content: transcript,
	})

	rawJSON, err := ai.Chat(messages)
	if err != nil {
		return openai.AnswerChatResult{}, err
	}

	var result openai.AnswerChatResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return openai.AnswerChatResult{}, fmt.Errorf("failed to parse answer chat JSON: %w, raw: %s", err, rawJSON)
	}

	if result.Response == "" {
		return openai.AnswerChatResult{}, fmt.Errorf("empty chat response")
	}

	return result, nil
}

func GenerateEndChat(ai openai.Client, history []openai.ChatMessage, subtitleLanguage string) (openai.AnswerChatResult, error) {
	if ai == nil {
		return openai.AnswerChatResult{}, fmt.Errorf("unsupported client")
	}

	jsonInstruction := `The user has decided to end the conversation. You MUST respond in JSON with: {"response": "your farewell", "isLast": true}. Provide a natural farewell message.`
	if subtitleLanguage != "" {
		jsonInstruction = fmt.Sprintf(`The user has decided to end the conversation. You MUST respond in JSON with: {"response": "your farewell", "responseSubtitle": "complete and accurate translation of your entire farewell in %s", "isLast": true}. Provide a natural farewell message.`, subtitleLanguage)
	}

	messages := make([]openai.ChatMessage, len(history))
	copy(messages, history)

	// Inject JSON instruction into system prompt
	for i, msg := range messages {
		if msg.Role == openai.ROLE_SYSTEM {
			messages[i].Content = msg.Content + "\n\n" + jsonInstruction
			break
		}
	}

	rawJSON, err := ai.Chat(messages)
	if err != nil {
		return openai.AnswerChatResult{}, err
	}

	var result openai.AnswerChatResult
	if err := json.Unmarshal([]byte(rawJSON), &result); err != nil {
		return openai.AnswerChatResult{}, fmt.Errorf("failed to parse end chat JSON: %w, raw: %s", err, rawJSON)
	}

	result.IsLast = true

	return result, nil
}

func GenerateSpeech(ai openai.Client, text, voice, language string) (string, error) {
	if ai == nil {
		return "", nil
	}

	speechInput := SanitizeString(text)

	speech, err := ai.Speech(speechInput, voice, language)
	if err != nil {
		return "", err
	}
	defer speech.Close()

	speechByte, err := io.ReadAll(speech)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(speechByte), nil
}
