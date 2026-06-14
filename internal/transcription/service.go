package transcription

import (
	"context"
	"fmt"
	"strings"

	"github.com/agriconnect-ai/internal/ai"
)

type Service interface {
	Transcribe(ctx context.Context, input TranscriptionInput) (*TranscriptionResponse, error)
}

type service struct {
	transcriber ai.AudioTranscriber
}

func NewService(transcriber ai.AudioTranscriber) Service {
	return &service{transcriber: transcriber}
}

func (s *service) Transcribe(ctx context.Context, input TranscriptionInput) (*TranscriptionResponse, error) {
	if len(input.Audio) == 0 {
		return nil, fmt.Errorf("empty audio")
	}

	langHint := input.LanguageHint
	if langHint == "" {
		langHint = "auto"
	}

	result, err := s.transcriber.Transcribe(ctx, ai.TranscriptionInput{
		AudioData:    input.Audio,
		AudioType:    input.AudioType,
		LanguageHint: langHint,
	})
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	if strings.TrimSpace(result.Text) == "" {
		return nil, fmt.Errorf("empty transcript")
	}

	requiresConfirmation := false
	experimentalKrio := false

	if input.LanguageHint == "krio" || result.DetectedLanguage == "krio" {
		requiresConfirmation = true
		experimentalKrio = true
	}

	return &TranscriptionResponse{
		Transcript:           result.Text,
		DetectedLanguage:     result.DetectedLanguage,
		RequiresConfirmation: requiresConfirmation,
		ExperimentalKrio:     experimentalKrio,
	}, nil
}
