package util

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/madeindra/mock-conversation/server/internal/elevenlab"
	"github.com/madeindra/mock-conversation/server/internal/openai"
)

func GetChatAssets(ai openai.Client, role string, topic string, language string) (string, string, error) {
	if ai == nil {
		return "", "", fmt.Errorf("unsupported client")
	}

	systemPrompt, err := openai.GetSystemPrompt(role, topic, language)
	if err != nil {
		return "", "", err
	}

	// Generate initial greeting dynamically via AI so it's in the correct language
	messages := []openai.ChatMessage{
		{
			Role:    openai.ROLE_SYSTEM,
			Content: systemPrompt,
		},
		{
			Role:    openai.ROLE_USER,
			Content: "Start the conversation with a brief greeting and introduce the topic.",
		},
	}

	initialChat, err := ai.Chat(messages)
	if err != nil {
		return "", "", err
	}

	if initialChat == "" {
		return "", "", fmt.Errorf("empty initial chat response")
	}

	return systemPrompt, initialChat, nil
}

func TranscribeSpeech(ai openai.Client, file io.ReadCloser, filename, language string) (string, error) {
	if ai == nil {
		return "", fmt.Errorf("unsupported client")
	}

	transcript, err := ai.Transcribe(file, filename, language)
	if err != nil {
		return "", err
	}

	if transcript.Text == "" {
		return "", fmt.Errorf("empty transcript")
	}

	return transcript.Text, nil
}

func GenerateText(ai openai.Client, entries []openai.ChatMessage) (string, error) {
	if ai == nil {
		return "", fmt.Errorf("unsupported client")
	}

	chatCompletion, err := ai.Chat(entries)
	if err != nil {
		return "", err
	}

	if chatCompletion == "" {
		return "", fmt.Errorf("empty chat response")
	}

	return chatCompletion, nil
}

func GenerateSpeech(ai openai.Client, el elevenlab.Client, language, text, voice string) (string, error) {
	if ai == nil {
		return "", fmt.Errorf("unsupported client")
	}

	speechInput := SanitizeString(text)

	var speech io.ReadCloser
	if ai.IsSpeechAvailable(language) {
		tts, err := ai.TextToSpeech(speechInput)
		if err != nil {
			return "", err
		}

		speech = tts
	} else if el != nil {
		tts, err := el.TextToSpeech(speechInput, voice)
		if err != nil {
			return "", err
		}

		speech = tts
	} else {
		return "", nil // quietly ignore unsupported language when alternative api not available
	}

	speechByte, err := io.ReadAll(speech)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(speechByte), nil
}

func GenerateSSML(ai openai.Client, text string) (string, error) {
	if ai == nil {
		return "", fmt.Errorf("unsupported client")
	}

	ssml, err := ai.SSML(text)
	if err != nil {
		return "", err
	}

	if ssml == "" {
		return "", fmt.Errorf("empty ssml response")
	}

	sanitized, err := SanitizeSSML(ssml)
	if err != nil {
		return "", nil // quietly ignore improper formatted response
	}

	if err := ValidateIdentical(text, sanitized); err != nil {
		return "", nil // quietly ignore ssml that differs to the original text
	}

	return sanitized, nil
}

func GenerateSubtitle(ai openai.Client, text string, subtitleLanguage string) (string, error) {
	if ai == nil {
		return "", fmt.Errorf("unsupported client")
	}

	if subtitleLanguage == "" {
		return "", nil
	}

	prompt := fmt.Sprintf("Translate the following text to %s. Only respond with the translation, nothing else:\n\n%s", subtitleLanguage, text)

	messages := []openai.ChatMessage{
		{
			Role:    openai.ROLE_SYSTEM,
			Content: "You are a translator. You only respond with the translated text, no explanations or additional text.",
		},
		{
			Role:    openai.ROLE_USER,
			Content: prompt,
		},
	}

	translation, err := ai.Chat(messages)
	if err != nil {
		return "", err
	}

	return translation, nil
}
