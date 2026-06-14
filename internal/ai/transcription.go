package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type TranscriptionInput struct {
	AudioData     []byte
	AudioType     string
	LanguageHint  string
}

type TranscriptionResult struct {
	Text             string
	DetectedLanguage string
	DurationSeconds  float64
}

type AudioTranscriber interface {
	Transcribe(ctx context.Context, input TranscriptionInput) (*TranscriptionResult, error)
}

type audioTranscriber struct {
	apiKey  string
	baseURL string
	model   string
}

func NewAudioTranscriber(apiKey, baseURL, model string) AudioTranscriber {
	return &audioTranscriber{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

func (t *audioTranscriber) Transcribe(ctx context.Context, input TranscriptionInput) (*TranscriptionResult, error) {
	if t.apiKey == "" {
		return nil, fmt.Errorf("transcription service is not configured")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("model", t.model); err != nil {
		return nil, fmt.Errorf("writing model field: %w", err)
	}

	if input.LanguageHint != "" && input.LanguageHint != "auto" {
		if err := w.WriteField("language", input.LanguageHint); err != nil {
			return nil, fmt.Errorf("writing language field: %w", err)
		}
	}

	fw, err := w.CreateFormFile("file", "recording."+extFromAudioType(input.AudioType))
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := fw.Write(input.AudioData); err != nil {
		return nil, fmt.Errorf("writing audio data: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/audio/transcriptions", &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	httpClient := &http.Client{Timeout: 90 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling transcription API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transcription API returned status %d: %s", resp.StatusCode, string(body))
	}

	var groqResp struct {
		Text string `json:"text"`
	}
	if err := jsonUnmarshal(body, &groqResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if groqResp.Text == "" {
		return nil, fmt.Errorf("empty transcript returned")
	}

	return &TranscriptionResult{
		Text:             groqResp.Text,
		DetectedLanguage: "",
		DurationSeconds:  0,
	}, nil
}

func extFromAudioType(audioType string) string {
	switch audioType {
	case "audio/webm":
		return "webm"
	case "audio/wav":
		return "wav"
	case "audio/mpeg":
		return "mp3"
	case "audio/mp4":
		return "mp4"
	case "audio/ogg":
		return "ogg"
	default:
		return "webm"
	}
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
