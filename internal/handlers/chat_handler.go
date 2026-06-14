package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/agriconnect-ai/internal/ai"
	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/validation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatSvc      services.ChatService
	orchestrator *ai.Orchestrator
	cfg          *config.Config
}

func NewChatHandler(chatSvc services.ChatService, orchestrator *ai.Orchestrator, cfg *config.Config) *ChatHandler {
	return &ChatHandler{chatSvc: chatSvc, orchestrator: orchestrator, cfg: cfg}
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID := getUserID(c)
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := validation.MessageLength(req.Message, 2, h.cfg.MaxMessageLength); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.cfg.AIAvailable() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service is not configured. Please set GROQ_API_KEY."})
		return
	}

	result, err := h.chatSvc.SendMessage(c.Request.Context(), convID, userID, req.Message, h.orchestrator, h.cfg.MaxContextMessages)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate response. Please try again."})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": result.Message,
		"sources": result.Sources,
	})
}

func (h *ChatHandler) StreamMessage(c *gin.Context) {
	userID := getUserID(c)
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := validation.MessageLength(req.Message, 2, h.cfg.MaxMessageLength); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.cfg.AIAvailable() {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		writeSSEEvent(c, "error", map[string]string{"message": "AI service is not configured. Please set GROQ_API_KEY."})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ctx := c.Request.Context()
	tokenCh := make(chan string, 100)
	statusCh := make(chan string, 10)
	errCh := make(chan error, 1)
	var msgID string

	go func() {
		result, err := h.chatSvc.SendMessageStream(ctx, convID, userID, req.Message, h.orchestrator, h.cfg.MaxContextMessages, tokenCh, statusCh)
		if result != nil {
			msgID = result.Message.ID.String()
		}
		errCh <- err
		close(tokenCh)
		close(statusCh)
	}()

	flusher, _ := c.Writer.(http.Flusher)

	// Drain status channel first in a non-blocking way
	drain := func() {
		for {
			select {
			case status, ok := <-statusCh:
				if !ok {
					return
				}
				writeSSEEvent(c, "status", map[string]string{"message": status})
				if flusher != nil {
					flusher.Flush()
				}
			default:
				return
			}
		}
	}

	for {
		drain()
		select {
		case token, ok := <-tokenCh:
			if !ok {
				goto done
			}
			writeSSEEvent(c, "token", map[string]string{"text": token})
			if flusher != nil {
				flusher.Flush()
			}
		case <-ctx.Done():
			goto done
		}
	}

done:
	if err := <-errCh; err != nil {
		writeSSEEvent(c, "error", map[string]string{"message": "The AI service encountered an error. Please try again."})
	} else {
		writeSSEEvent(c, "complete", map[string]string{"message_id": msgID})
	}
	if flusher != nil {
		flusher.Flush()
	}
}

func writeSSEEvent(c *gin.Context, event string, data interface{}) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(b))
}
