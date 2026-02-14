package model

type StartChatRequest struct {
	Role             string `json:"role"`
	Topic            string `json:"topic"`
	Language         string `json:"language"`
	SubtitleLanguage string `json:"subtitleLanguage,omitempty"`
}
