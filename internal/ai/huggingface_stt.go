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

type HuggingFaceSTT struct {
	apiKey      string
	model       string
	httpClient  *http.Client
}

func NewHuggingFaceSTT(apiKey, model string, timeoutSecs int) *HuggingFaceSTT {
	if model == "" {
		model = "openai/whisper-large-v3"
	}
	if timeoutSecs <= 0 {
		timeoutSecs = 60
	}
	return &HuggingFaceSTT{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSecs) * time.Second,
		},
	}
}

func (t *HuggingFaceSTT) Transcribe(ctx context.Context, input TranscriptionInput) (*TranscriptionResult, error) {
	if t.apiKey == "" {
		return nil, fmt.Errorf("Hugging Face transcription is not configured: HUGGINGFACE_API_KEY is empty")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("data", "audio."+hfExtFromAudioType(input.AudioType))
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := fw.Write(input.AudioData); err != nil {
		return nil, fmt.Errorf("writing audio data: %w", err)
	}
	w.Close()

	apiURL := fmt.Sprintf("https://api-inference.huggingface.co/models/%s", t.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Hugging Face Inference API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading Hugging Face response: %w", err)
	}

	if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusBadGateway {
		return nil, fmt.Errorf("Hugging Face model is loading (cold start), please try again: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("Hugging Face rate limited, please try again later: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Hugging Face Inference API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var hfResp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &hfResp); err != nil {
		return nil, fmt.Errorf("decoding Hugging Face response: %w", err)
	}
	if hfResp.Text == "" {
		// Try a different response shape: some models return { "translation_text": "..." }
		var altResp struct {
			TranslationText string `json:"translation_text"`
		}
		if err := json.Unmarshal(body, &altResp); err == nil && altResp.TranslationText != "" {
			hfResp.Text = altResp.TranslationText
		}
	}
	if hfResp.Text == "" {
		return nil, fmt.Errorf("empty transcript from Hugging Face")
	}

	return &TranscriptionResult{
		Text:             hfResp.Text,
		DetectedLanguage: "",
		DurationSeconds:  0,
	}, nil
}

func hfExtFromAudioType(audioType string) string {
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
		return "wav"
	}
}
