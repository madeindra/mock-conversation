package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client interface {
	IsKeyValid() (bool, error)
	Status() (Status, error)
	Chat([]ChatMessage) (string, error)
	ChatWithAudio(history []ChatMessage, audioData string, audioFormat string) (string, error)

	GetDefaultTranscriptLanguage() string
}

type OpenAI struct {
	apiKey             string
	baseURL            string
	chatModel          string
	transcriptLanguage string
}

const (
	baseURL            = "https://api.openai.com/v1"
	statusURL          = "https://status.openai.com/api/v2"
	chatModel          = "gpt-4o-mini"
	audioChatModel     = "gpt-4o-mini-audio-preview"
	transcriptLanguage = "en"
)

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		apiKey:             apiKey,
		baseURL:            baseURL,
		chatModel:          chatModel,
		transcriptLanguage: transcriptLanguage,
	}
}

func (c *OpenAI) IsKeyValid() (bool, error) {
	url, err := url.JoinPath(c.baseURL, "/models")
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}

func (c *OpenAI) Status() (Status, error) {
	url, err := url.JoinPath(statusURL, "/components.json")
	if err != nil {
		return STATUS_UNKNOWN, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return STATUS_UNKNOWN, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return STATUS_UNKNOWN, err
	}

	if resp.StatusCode != http.StatusOK {
		return STATUS_UNKNOWN, nil
	}

	var statusResp ComponentStatusResponse
	err = unmarshalJSONResponse(resp, &statusResp)
	if err != nil {
		return STATUS_UNKNOWN, err
	}

	targetComponents := map[string]bool{
		"Chat Completions": true,
		"Audio":            true,
	}

	worstStatus := STATUS_OPERATIONAL
	foundAny := false

	for _, component := range statusResp.Components {
		if !targetComponents[component.Name] {
			continue
		}
		foundAny = true
		switch component.Status {
		case "major_outage":
			return STATUS_MAJOR_OUTAGE, nil
		case "partial_outage":
			if worstStatus != STATUS_MAJOR_OUTAGE {
				worstStatus = STATUS_PARTIAL_OUTAGE
			}
		case "degraded_performance":
			if worstStatus == STATUS_OPERATIONAL {
				worstStatus = STATUS_DEGRADED_PERFORMANCE
			}
		}
	}

	if !foundAny {
		return STATUS_UNKNOWN, nil
	}

	return worstStatus, nil
}

func (c *OpenAI) Chat(messages []ChatMessage) (string, error) {
	url, err := url.JoinPath(c.baseURL, "/chat/completions")
	if err != nil {
		return "", err
	}

	chatReq := ChatRequest{
		Model:          c.chatModel,
		Messages:       messages,
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	var chatResp ChatResponse
	err = unmarshalJSONResponse(resp, &chatResp)
	if err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no valid response returned")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *OpenAI) ChatWithAudio(history []ChatMessage, audioData string, audioFormat string) (string, error) {
	url, err := url.JoinPath(c.baseURL, "/chat/completions")
	if err != nil {
		return "", err
	}

	messages := make([]AudioChatMessage, 0, len(history)+1)
	for _, msg := range history {
		messages = append(messages, AudioChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	messages = append(messages, AudioChatMessage{
		Role: ROLE_USER,
		Content: []AudioContentPart{
			{
				Type: "input_audio",
				InputAudio: &InputAudio{
					Data:   audioData,
					Format: audioFormat,
				},
			},
		},
	})

	chatReq := AudioChatRequest{
		Model:      audioChatModel,
		Modalities: []string{"text"},
		Messages:   messages,
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	var chatResp AudioChatResponse
	err = unmarshalJSONResponse(resp, &chatResp)
	if err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no valid response returned")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *OpenAI) GetDefaultTranscriptLanguage() string {
	return string(c.transcriptLanguage)
}

func getResponseBody(resp *http.Response) (io.ReadCloser, error) {
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("response is nil")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

func unmarshalJSONResponse(resp *http.Response, v interface{}) error {
	respBody, err := getResponseBody(resp)
	if err != nil {
		return err
	}
	if respBody == nil {
		return fmt.Errorf("response body is nil")
	}
	defer respBody.Close()

	respByte, err := io.ReadAll(respBody)
	if err != nil {
		return err
	}

	return json.Unmarshal(respByte, v)
}
