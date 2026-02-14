package openai

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed templates/system.txt
var systemPromptTemplate string

func GetSystemPrompt(role string, topic string, language string) (string, error) {
	t, err := template.New("prompt").Parse(systemPromptTemplate)
	if err != nil {
		return "", err
	}

	data := struct {
		Role     string
		Topic    string
		Language string
	}{
		Role:     role,
		Topic:    topic,
		Language: language,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
