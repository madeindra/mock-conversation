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

type AudioChatRequest struct {
	Model      string             `json:"model"`
	Modalities []string           `json:"modalities"`
	Messages   []AudioChatMessage `json:"messages"`
}

type AudioChatMessage struct {
	Role    Role        `json:"role"`
	Content interface{} `json:"content"`
}

type AudioContentPart struct {
	Type       string      `json:"type"`
	Text       string      `json:"text,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
}

type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type AudioChatResponse struct {
	Choices []AudioChoice `json:"choices"`
}

type AudioChoice struct {
	Index        int              `json:"index"`
	Message      AudioRespMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

type AudioRespMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
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
