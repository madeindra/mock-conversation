package openai

type ResponseFormat struct {
	Type string `json:"type"`
}

type ChatRequest struct {
	Messages       []ChatMessage   `json:"messages"`
	Model          string          `json:"model"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatMessage struct {
	Content string `json:"content"`
	Role    Role   `json:"role"`
}

type Role string

const (
	ROLE_SYSTEM    Role = "system"
	ROLE_ASSISTANT Role = "assistant"
	ROLE_USER      Role = "user"
)

type SpeechRequest struct {
	Model string  `json:"model"`
	Voice string  `json:"voice"`
	Input string  `json:"input"`
	Speed float64 `json:"speed,omitempty"`
}

// TranscriptResponse is the response from the transcription API.
type TranscriptResponse struct {
	Text string `json:"text"`
}

// AnswerChatResult is the JSON response from ChatGPT for all chat operations.
type AnswerChatResult struct {
	Transcript         string `json:"transcript,omitempty"`
	TranscriptSubtitle string `json:"transcriptSubtitle,omitempty"`
	Response           string `json:"response"`
	ResponseSubtitle   string `json:"responseSubtitle,omitempty"`
	IsLast             bool   `json:"isLast"`
}

type Status string

const (
	STATUS_OPERATIONAL          Status = "operational"
	STATUS_DEGRADED_PERFORMANCE Status = "degraded_performance"
	STATUS_PARTIAL_OUTAGE       Status = "partial_outage"
	STATUS_MAJOR_OUTAGE         Status = "major_outage"
	STATUS_UNKNOWN              Status = "unknown"
)

type ComponentStatusResponse struct {
	Components []Component `json:"components"`
}

type Component struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
}
