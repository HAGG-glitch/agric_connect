package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/transcription"
	"github.com/gin-gonic/gin"
)

type TranscriptionHandler struct {
	svc transcription.Service
	cfg *config.Config
}

func NewTranscriptionHandler(svc transcription.Service, cfg *config.Config) *TranscriptionHandler {
	return &TranscriptionHandler{svc: svc, cfg: cfg}
}

func (h *TranscriptionHandler) Transcribe(c *gin.Context) {
	maxSize := int64(h.cfg.MaxAudioSizeMB) * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize+1024)

	if err := c.Request.ParseMultipartForm(maxSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio too large or invalid form data"})
		return
	}

	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is required"})
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("reading audio: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read audio"})
		return
	}

	if len(audioData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty audio"})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if err := transcription.ValidateAudioContentType(contentType, h.cfg.AllowedAudioTypes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	langHint := c.PostForm("language_hint")
	if err := transcription.ValidateLanguageHint(langHint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.Transcribe(c.Request.Context(), transcription.TranscriptionInput{
		Audio:        audioData,
		AudioType:    contentType,
		LanguageHint: langHint,
		SizeBytes:    header.Size,
	})
	if err != nil {
		log.Printf("transcription failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transcription failed. Please try again."})
		return
	}

	c.JSON(http.StatusOK, result)
}
