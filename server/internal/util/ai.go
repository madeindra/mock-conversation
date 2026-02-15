package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/madeindra/mock-conversation/server/internal/elevenlab"
	"github.com/madeindra/mock-conversation/server/internal/openai"
)

// extractJSON finds and returns the first JSON object in a string.
// Handles cases where the model wraps JSON in markdown code fences or adds extra text.
func extractJSON(raw string) string {
	s := strings.TrimSpace(raw)

	// Strip markdown code fences
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
		return strings.TrimSpace(s)
	}

	// Find the first { and last } to extract the JSON object
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}

	return s
}

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
		jsonInstruction = fmt.Sprintf(`Respond in JSON with: {"response": "your greeting", "responseSubtitle": "translation of your greeting in %s"}`, subtitleLanguage)
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

	rawResponse, err := ai.Chat(messages)
	if err != nil {
		return "", openai.AnswerChatResult{}, err
	}

	jsonStr := extractJSON(rawResponse)

	var result openai.AnswerChatResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", openai.AnswerChatResult{}, fmt.Errorf("failed to parse initial chat JSON: %w, raw: %s", err, rawResponse)
	}

	if result.Response == "" {
		return "", openai.AnswerChatResult{}, fmt.Errorf("empty initial chat response")
	}

	return systemPrompt, result, nil
}

func GenerateTextFromAudio(ai openai.Client, history []openai.ChatMessage, audioData, audioFormat, subtitleLanguage string) (openai.AnswerChatResult, error) {
	if ai == nil {
		return openai.AnswerChatResult{}, fmt.Errorf("unsupported client")
	}

	// Inject JSON instruction into the system prompt
	enriched := make([]openai.ChatMessage, len(history))
	copy(enriched, history)

	jsonInstruction := `You MUST respond in JSON with: {"transcript": "word-for-word transcription of the user's audio in the language they spoke", "response": "your reply", "isLast": false}. Set isLast to true only when the conversation is ending (user says goodbye or you decide to end it). When isLast is true, respond with a natural farewell.`
	if subtitleLanguage != "" {
		jsonInstruction = fmt.Sprintf(`You MUST respond in JSON with: {"transcript": "word-for-word transcription of the user's audio in the language they spoke", "transcriptSubtitle": "translation of transcript in %s", "response": "your reply", "responseSubtitle": "translation of your reply in %s", "isLast": false}. Set isLast to true only when the conversation is ending (user says goodbye or you decide to end it). When isLast is true, respond with a natural farewell.`, subtitleLanguage, subtitleLanguage)
	}

	for i, msg := range enriched {
		if msg.Role == openai.ROLE_SYSTEM {
			enriched[i].Content = msg.Content + "\n\n" + jsonInstruction
			break
		}
	}

	rawResponse, err := ai.ChatWithAudio(enriched, audioData, audioFormat)
	if err != nil {
		return openai.AnswerChatResult{}, err
	}

	// gpt-audio-mini doesn't support response_format: json_object, so extract JSON manually
	jsonStr := extractJSON(rawResponse)

	var result openai.AnswerChatResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return openai.AnswerChatResult{}, fmt.Errorf("failed to parse answer chat JSON: %w, raw: %s", err, rawResponse)
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
		jsonInstruction = fmt.Sprintf(`The user has decided to end the conversation. You MUST respond in JSON with: {"response": "your farewell", "responseSubtitle": "translation of your farewell in %s", "isLast": true}. Provide a natural farewell message.`, subtitleLanguage)
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

	rawResponse, err := ai.Chat(messages)
	if err != nil {
		return openai.AnswerChatResult{}, err
	}

	jsonStr := extractJSON(rawResponse)

	var result openai.AnswerChatResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return openai.AnswerChatResult{}, fmt.Errorf("failed to parse end chat JSON: %w, raw: %s", err, rawResponse)
	}

	result.IsLast = true

	return result, nil
}

func GenerateSpeech(el elevenlab.Client, text, voice string) (string, error) {
	if el == nil {
		return "", nil
	}

	speechInput := SanitizeString(text)

	speech, err := el.TextToSpeech(speechInput, voice)
	if err != nil {
		return "", err
	}

	speechByte, err := io.ReadAll(speech)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(speechByte), nil
}
