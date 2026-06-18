package transcription

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/agriconnect-ai/internal/ai"
)

type Service interface {
	Transcribe(ctx context.Context, input TranscriptionInput) (*TranscriptionResponse, error)
}

type service struct {
	transcriber ai.AudioTranscriber
	krioSTT     ai.AudioTranscriber
}

func NewService(transcriber ai.AudioTranscriber) Service {
	return &service{transcriber: transcriber, krioSTT: transcriber}
}

func NewServiceWithKrio(transcriber, krioSTT ai.AudioTranscriber) Service {
	return &service{transcriber: transcriber, krioSTT: krioSTT}
}

func (s *service) Transcribe(ctx context.Context, input TranscriptionInput) (*TranscriptionResponse, error) {
	if len(input.Audio) == 0 {
		return nil, fmt.Errorf("empty audio")
	}

	langHint := input.LanguageHint
	if langHint == "" {
		langHint = "auto"
	}

	activeTranscriber := s.transcriber
	usingKrioProvider := false

	if strings.ToLower(langHint) == "krio" && s.krioSTT != nil && s.krioSTT != s.transcriber {
		activeTranscriber = s.krioSTT
		usingKrioProvider = true
		log.Printf("using Krio STT provider for transcription")
	}

	result, err := activeTranscriber.Transcribe(ctx, ai.TranscriptionInput{
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

	if input.LanguageHint == "krio" || result.DetectedLanguage == "krio" || usingKrioProvider {
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
