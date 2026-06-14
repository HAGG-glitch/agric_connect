package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/agriconnect-ai/internal/ai"
	"github.com/agriconnect-ai/internal/transcription"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockAudioTranscriber struct {
	result *ai.TranscriptionResult
	err    error
}

func (m *mockAudioTranscriber) Transcribe(_ context.Context, _ ai.TranscriptionInput) (*ai.TranscriptionResult, error) {
	return m.result, m.err
}

// ── helpers ──────────────────────────────────────────────────────────────────

func testAudioData() []byte {
	return []byte("fake audio binary data that is not empty")
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestTranscription_Valid(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{
			Text:             "Cassava leaves are turning yellow",
			DetectedLanguage: "english",
		},
	})

	resp, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	if resp.Transcript != "Cassava leaves are turning yellow" {
		t.Errorf("expected transcript 'Cassava leaves are turning yellow', got %q", resp.Transcript)
	}
	if resp.DetectedLanguage != "english" {
		t.Errorf("expected language 'english', got %q", resp.DetectedLanguage)
	}
	if resp.RequiresConfirmation {
		t.Error("expected RequiresConfirmation=false for english")
	}
	if resp.ExperimentalKrio {
		t.Error("expected ExperimentalKrio=false for english")
	}
}

func TestTranscription_EmptyAudio(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{Text: "some text"},
	})

	_, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        []byte{},
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err == nil {
		t.Fatal("expected error for empty audio")
	}
	if !containsStr(err.Error(), "empty audio") {
		t.Errorf("expected 'empty audio' error, got %v", err)
	}
}

func TestTranscription_ProviderFailure(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		err: fmt.Errorf("transcription API returned 503"),
	})

	_, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err == nil {
		t.Fatal("expected error from provider failure")
	}
	if !containsStr(err.Error(), "transcription failed:") {
		t.Errorf("expected 'transcription failed:' wrapping, got %v", err)
	}
}

func TestTranscription_EmptyTranscriptFromProvider(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{
			Text:             "",
			DetectedLanguage: "english",
		},
	})

	_, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err == nil {
		t.Fatal("expected error for empty transcript")
	}
	if !containsStr(err.Error(), "empty transcript") {
		t.Errorf("expected 'empty transcript' error, got %v", err)
	}
}

func TestTranscription_WhitespaceOnlyTranscript(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{
			Text:             "   \n\t  ",
			DetectedLanguage: "english",
		},
	})

	_, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only transcript")
	}
	if !containsStr(err.Error(), "empty transcript") {
		t.Errorf("expected 'empty transcript' error, got %v", err)
	}
}

func TestTranscription_KrioLanguageHint(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{
			Text:             "Wetin de apun na di farm",
			DetectedLanguage: "krio",
		},
	})

	resp, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "krio",
	})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	if !resp.RequiresConfirmation {
		t.Error("expected RequiresConfirmation=true for krio language hint")
	}
	if !resp.ExperimentalKrio {
		t.Error("expected ExperimentalKrio=true for krio language hint")
	}
	if resp.Transcript != "Wetin de apun na di farm" {
		t.Errorf("unexpected transcript: %q", resp.Transcript)
	}
}

func TestTranscription_AutoLanguageHint(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{
			Text:             "Cassava leaves are turning yellow",
			DetectedLanguage: "english",
		},
	})

	resp, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "auto",
	})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	if resp.RequiresConfirmation {
		t.Error("expected RequiresConfirmation=false for auto language hint")
	}
	if resp.ExperimentalKrio {
		t.Error("expected ExperimentalKrio=false for auto language hint")
	}
	if resp.Transcript != "Cassava leaves are turning yellow" {
		t.Errorf("unexpected transcript: %q", resp.Transcript)
	}
}

func TestTranscription_EmptyLanguageHintDefaultsToAuto(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{
			Text:             "Cassava leaves are turning yellow",
			DetectedLanguage: "english",
		},
	})

	resp, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "",
	})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	if resp.RequiresConfirmation {
		t.Error("expected RequiresConfirmation=false when LanguageHint defaults to auto")
	}
}

func TestTranscription_KrioDetectedByProvider(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{
			Text:             "Wetin de apun",
			DetectedLanguage: "krio",
		},
	})

	resp, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	if !resp.RequiresConfirmation {
		t.Error("expected RequiresConfirmation=true when provider detects krio")
	}
	if !resp.ExperimentalKrio {
		t.Error("expected ExperimentalKrio=true when provider detects krio")
	}
}

func TestTranscription_NilTranscriberResult(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: nil,
		err:    fmt.Errorf("nil result from transcriber"),
	})

	_, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        testAudioData(),
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err == nil {
		t.Fatal("expected error for nil result")
	}
	if !containsStr(err.Error(), "transcription failed:") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestTranscription_NilAudio(t *testing.T) {
	svc := transcription.NewService(&mockAudioTranscriber{
		result: &ai.TranscriptionResult{Text: "some text"},
	})

	_, err := svc.Transcribe(context.Background(), transcription.TranscriptionInput{
		Audio:        nil,
		AudioType:    "audio/webm",
		LanguageHint: "english",
	})
	if err == nil {
		t.Fatal("expected error for nil audio")
	}
}

// ── validator tests ──────────────────────────────────────────────────────────

func TestValidateLanguageHint(t *testing.T) {
	tests := []struct {
		hint    string
		wantErr bool
	}{
		{"", false},
		{"english", false},
		{"krio", false},
		{"auto", false},
		{"ENGLISH", false},
		{"KRIO", false},
		{"french", true},
		{"spanish", true},
		{"eng", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("hint=%q", tt.hint), func(t *testing.T) {
			err := transcription.ValidateLanguageHint(tt.hint)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for hint %q", tt.hint)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for hint %q: %v", tt.hint, err)
			}
		})
	}
}

func TestValidateAudioContentType(t *testing.T) {
	allowed := []string{"audio/webm", "audio/wav", "audio/mpeg", "audio/mp4", "audio/ogg"}

	tests := []struct {
		contentType string
		wantErr     bool
	}{
		{"audio/webm", false},
		{"audio/wav", false},
		{"audio/mpeg", false},
		{"audio/mp4", false},
		{"audio/ogg", false},
		{"AUDIO/WEBM", false},
		{"audio/mp3", true},
		{"video/mp4", true},
		{"application/octet-stream", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("type=%q", tt.contentType), func(t *testing.T) {
			err := transcription.ValidateAudioContentType(tt.contentType, allowed)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for type %q", tt.contentType)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for type %q: %v", tt.contentType, err)
			}
		})
	}
}

func TestValidateAudioContentType_EmptyAllowed(t *testing.T) {
	err := transcription.ValidateAudioContentType("audio/webm", []string{})
	if err == nil {
		t.Error("expected error when allowed list is empty")
	}
}

func TestValidateAudioContentType_MultipleAllowed(t *testing.T) {
	err := transcription.ValidateAudioContentType("audio/wav", []string{"audio/webm", "audio/wav", "audio/ogg"})
	if err != nil {
		t.Errorf("expected no error for audio/wav, got %v", err)
	}

	err = transcription.ValidateAudioContentType("audio/flac", []string{"audio/webm", "audio/wav", "audio/ogg"})
	if err == nil {
		t.Error("expected error for audio/flac")
	}
}

// ── integration of validator + service for unsupported audio ──────────────────

func TestTranscription_UnsupportedAudioType_Validator(t *testing.T) {
	allowed := []string{"audio/webm", "audio/wav", "audio/mpeg", "audio/mp4", "audio/ogg"}
	err := transcription.ValidateAudioContentType("audio/flac", allowed)
	if err == nil {
		t.Fatal("expected validation error for flac")
	}
	if !strings.Contains(err.Error(), "unsupported audio type") {
		t.Errorf("expected 'unsupported audio type' error, got %v", err)
	}
}

func TestValidateAudioContentType_NormalizesCodecParam(t *testing.T) {
	allowed := []string{"audio/webm", "audio/wav", "audio/mpeg"}
	err := transcription.ValidateAudioContentType("audio/webm; codecs=opus", allowed)
	if err != nil {
		t.Errorf("expected no error for audio/webm with codec param, got %v", err)
	}
}

func TestValidateAudioContentType_NormalizesWhitespace(t *testing.T) {
	allowed := []string{"audio/wav", "audio/webm"}
	err := transcription.ValidateAudioContentType("  audio/wav  ", allowed)
	if err != nil {
		t.Errorf("expected no error for audio/wav with whitespace, got %v", err)
	}
}

func TestValidateAudioContentType_RejectsTrulyUnknown(t *testing.T) {
	allowed := []string{"audio/webm", "audio/wav"}
	err := transcription.ValidateAudioContentType("audio/flac", allowed)
	if err == nil {
		t.Error("expected error for audio/flac")
	}
}
