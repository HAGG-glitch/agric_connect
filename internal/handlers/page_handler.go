package handlers

import (
	"net/http"
	"time"

	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/middleware"
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

	userDistrict := ""
	if authUser, exists := c.Get(middleware.ContextKeyUser); exists {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			userDistrict = user.District
		}
	}

	c.HTML(http.StatusOK, "assistant.html", gin.H{
		"Title":         "AgriConnect AI - Agricultural Assistant",
		"UserID":        userID,
		"Districts":     weather.SupportedDistricts,
		"UserDistrict":  userDistrict,
		"AIAvailable":   h.cfg.AIAvailable(),
		"Year":          time.Now().Year(),
		"ContentBlock":  "contentAssistant",
	})
}
