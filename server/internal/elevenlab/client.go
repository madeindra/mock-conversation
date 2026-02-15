package elevenlab

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
)

type Client interface {
	TextToSpeech(input string, voice string) (io.ReadCloser, error)
	RandomVoice() string
}

type ElevenLab struct {
	apiKey   string
	baseURL  string
	ttsModel string
	ttsVoice string
}

const (
	baseURL  = "https://api.elevenlabs.io/v1"
	ttsModel = "eleven_multilingual_v2"
	ttsVoice = "cgSgspJ2msm6clMCkdW9"
)

var voiceIDs = []string{
	"hpp4J3VqNfWAUOO0d1Us", // Bella
	"CwhRBWXzGAHq8TQ4Fs17", // Roger
	"EXAVITQu4vr4xnSDxMaL", // Sarah
	"FGY2WhTYpPnrIDTdsKH5", // Laura
	"IKne3meq5aSn9XLyUdCD", // Charlie
	"JBFqnCBsd6RMkjVDRZzb", // George
	"N2lVS1w4EtoT3dr4eOWO", // Callum
	"SAz9YHcvj6GT2YYXdXww", // River
	"SOYHLrjzK2X1ezoPC6cr", // Harry
	"TX3LPaxmHKxFdv7VOQHJ", // Liam
	"Xb7hH8MSUJpSbSDYk0k2", // Alice
	"XrExE9yKIg1WjnnlVkGX", // Matilda
	"bIHbv24MWmeRgasZH58o", // Will
	"cgSgspJ2msm6clMCkdW9", // Jessica
	"cjVigY5qzO86Huf0OWal", // Eric
	"iP95p4xoKVk53GoZ742B", // Chris
	"nPczCjzI2devNBz1zQrb", // Brian
	"onwK4e9ZLuTAKqWW03F9", // Daniel
	"pFZP5JQG7iQjIQuC4Bku", // Lily
	"pNInz6obpgDQGcFmaJgB", // Adam
	"pqHfZKP75CvOlQylNhV4", // Bill
}

var defaultVoiceSetting = VoiceSetting{
	Stability:       0.5,
	SimilarityBoost: 0.75,
}

func NewElevenLab(apiKey string) *ElevenLab {
	return &ElevenLab{
		apiKey:   apiKey,
		baseURL:  baseURL,
		ttsModel: ttsModel,
		ttsVoice: ttsVoice,
	}
}

func (c *ElevenLab) RandomVoice() string {
	return voiceIDs[rand.Intn(len(voiceIDs))]
}

func (c *ElevenLab) TextToSpeech(input string, voice string) (io.ReadCloser, error) {
	if voice == "" {
		voice = c.ttsVoice
	}

	url, err := url.JoinPath(c.baseURL, "text-to-speech", voice)
	if err != nil {
		return nil, err
	}

	ttsReq := TTSRequest{
		Text:         input,
		ModelID:      c.ttsModel,
		VoiceSetting: defaultVoiceSetting,
	}

	body, err := json.Marshal(ttsReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("xi-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	respBody, err := getResponseBody(resp)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func getResponseBody(resp *http.Response) (io.ReadCloser, error) {
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("response is nil")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
