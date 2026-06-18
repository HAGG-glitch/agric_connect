package handlers

import (
	"net/http"
	"time"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/middleware"
	"github.com/agriconnect-ai/internal/weather"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PageHandler struct {
	cfg    *config.Config
	authSvc auth.Service
}

func NewPageHandler(cfg *config.Config, authSvc auth.Service) *PageHandler {
	return &PageHandler{cfg: cfg, authSvc: authSvc}
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

func (h *PageHandler) ProfilePage(c *gin.Context) {
	authUser, exists := c.Get(middleware.ContextKeyUser)
	if !exists || authUser == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	user, ok := authUser.(*middleware.AuthUser)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	userView, err := h.authSvc.GetUser(c.Request.Context(), user.ID)
	if err != nil {
		c.HTML(http.StatusNotFound, "profile.html", gin.H{
			"Title":  "AgriConnect AI - Profile",
			"Error":  "User not found",
			"Year":   time.Now().Year(),
		})
		return
	}

	c.HTML(http.StatusOK, "profile.html", gin.H{
		"Title":     "AgriConnect AI - Profile",
		"User":      userView,
		"Districts": weather.SupportedDistricts,
		"Year":      time.Now().Year(),
	})
}
