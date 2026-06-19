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
	cfg     *config.Config
	authSvc auth.Service
}

func NewPageHandler(cfg *config.Config, authSvc auth.Service) *PageHandler {
	return &PageHandler{cfg: cfg, authSvc: authSvc}
}

func (h *PageHandler) Home(c *gin.Context) {
	authUser, exists := c.Get(middleware.ContextKeyUser)
	if !exists || authUser == nil {
		c.HTML(http.StatusOK, "landing.html", gin.H{
			"Title": "AgriConnect AI - Agricultural Assistant for Sierra Leone",
			"Year":  time.Now().Year(),
		})
		return
	}
	user, ok := authUser.(*middleware.AuthUser)
	if !ok {
		c.HTML(http.StatusOK, "landing.html", gin.H{
			"Title": "AgriConnect AI - Agricultural Assistant for Sierra Leone",
			"Year":  time.Now().Year(),
		})
		return
	}
	switch user.Role {
	case "admin":
		c.Redirect(http.StatusSeeOther, "/admin")
	case "officer":
		c.Redirect(http.StatusSeeOther, "/officer")
	default:
		c.Redirect(http.StatusSeeOther, "/dashboard")
	}
}

func (h *PageHandler) AssistantPage(c *gin.Context) {
	if !h.cfg.AllowAnonymousAssistant {
		authUser, exists := c.Get(middleware.ContextKeyUser)
		if !exists || authUser == nil {
			c.Redirect(http.StatusSeeOther, "/register")
			return
		}
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	userDistrict := ""
	if authUser, exists := c.Get(middleware.ContextKeyUser); exists {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			userDistrict = user.District
		}
	}

	data := gin.H{
		"Title":         "AgriConnect AI - Agricultural Assistant",
		"UserID":        userID,
		"Districts":     weather.SupportedDistricts,
		"UserDistrict":  userDistrict,
		"AIAvailable":   h.cfg.AIAvailable(),
		"Year":          time.Now().Year(),
		"ContentBlock":  "contentAssistant",
		"ActivePage":    "assistant",
	}
	authUser, exists := c.Get(middleware.ContextKeyUser)
	if exists && authUser != nil {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			data["UserName"] = user.FullName
			data["UserRole"] = user.Role
			data["UserLanguage"] = user.PreferredLanguage
			if data["UserDistrict"] == "" {
				data["UserDistrict"] = user.District
			}
		}
	}
	c.HTML(http.StatusOK, "assistant.html", data)
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
		"Title":      "AgriConnect AI - Profile",
		"Error":      "User not found",
		"Year":       time.Now().Year(),
		"ActivePage": "profile",
		"ContentBlock": "contentProfile",
	})
		return
	}

	backURL := "/dashboard"
	switch user.Role {
	case "admin":
		backURL = "/admin"
	case "officer":
		backURL = "/officer"
	}

	data := gin.H{
		"Title":      "AgriConnect AI - Profile",
		"User":       userView,
		"Districts":  weather.SupportedDistricts,
		"Year":       time.Now().Year(),
		"ActivePage": "profile",
		"BackURL":    backURL,
		"ContentBlock": "contentProfile",
		"UserName":   user.FullName,
		"UserRole":   user.Role,
		"UserDistrict": user.District,
		"UserLanguage": user.PreferredLanguage,
	}
	c.HTML(http.StatusOK, "profile.html", data)
}
