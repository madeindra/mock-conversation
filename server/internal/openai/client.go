package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
)

type Client interface {
	IsKeyValid() (bool, error)
	Status() (Status, error)
	Chat([]ChatMessage) (string, error)
	Transcribe(audio io.Reader, filename string, language string) (string, error)
	Speech(text string, voice string) (io.ReadCloser, error)
	RandomVoice() string

	GetDefaultTranscriptLanguage() string
}

type OpenAI struct {
	apiKey             string
	baseURL            string
	chatModel          string
	transcriptModel    string
	ttsModel           string
	transcriptLanguage string
}

const (
	baseURL            = "https://api.openai.com/v1"
	statusURL          = "https://status.openai.com/api/v2"
	chatModel          = "gpt-4o-mini"
	transcriptModel    = "whisper-1"
	ttsModel           = "gpt-4o-mini-tts"
	transcriptLanguage = "en"
)

var ttsVoices = []string{
	"alloy",
	"ash",
	"ballad",
	"coral",
	"echo",
	"fable",
	"marin",
	"nova",
	"onyx",
	"sage",
	"shimmer",
	"verse",
}

func NewOpenAI(apiKey string) *OpenAI {
	return &OpenAI{
		apiKey:             apiKey,
		baseURL:            baseURL,
		chatModel:          chatModel,
		transcriptModel:    transcriptModel,
		ttsModel:           ttsModel,
		transcriptLanguage: transcriptLanguage,
	}
}

func (c *OpenAI) RandomVoice() string {
	return ttsVoices[rand.Intn(len(ttsVoices))]
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

func (c *OpenAI) Transcribe(audio io.Reader, filename string, language string) (string, error) {
	url, err := url.JoinPath(c.baseURL, "/audio/transcriptions")
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(part, audio); err != nil {
		return "", err
	}

	if err := writer.WriteField("model", c.transcriptModel); err != nil {
		return "", err
	}

	if language != "" {
		if err := writer.WriteField("language", language); err != nil {
			return "", err
		}
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, &buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	var transcriptResp TranscriptResponse
	err = unmarshalJSONResponse(resp, &transcriptResp)
	if err != nil {
		return "", err
	}

	return transcriptResp.Text, nil
}

func (c *OpenAI) Speech(text string, voice string) (io.ReadCloser, error) {
	url, err := url.JoinPath(c.baseURL, "/audio/speech")
	if err != nil {
		return nil, err
	}

	speechReq := SpeechRequest{
		Model: c.ttsModel,
		Voice: voice,
		Input: text,
		Speed: 1.00,
	}

	body, err := json.Marshal(speechReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return getResponseBody(resp)
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
