package handlers

import (
	"net/http"
	"time"

	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/weather"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PageHandler struct {
	cfg *config.Config
}

func NewPageHandler(cfg *config.Config) *PageHandler {
	return &PageHandler{cfg: cfg}
}

func (h *PageHandler) AssistantPage(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	c.HTML(http.StatusOK, "assistant.html", gin.H{
		"Title":        "AgriConnect AI - Agricultural Assistant",
		"UserID":       userID,
		"Districts":    weather.SupportedDistricts,
		"AIAvailable":  h.cfg.AIAvailable(),
		"Year":         time.Now().Year(),
		"ContentBlock": "contentAssistant",
	})
}
