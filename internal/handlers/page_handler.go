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
	"gorm.io/gorm"
)

type PageHandler struct {
	cfg     *config.Config
	authSvc auth.Service
	db      *gorm.DB
}

func NewPageHandler(cfg *config.Config, authSvc auth.Service, db *gorm.DB) *PageHandler {
	return &PageHandler{cfg: cfg, authSvc: authSvc, db: db}
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

func (h *PageHandler) WeatherPage(c *gin.Context) {
	userDistrict := ""
	userName := ""
	userRole := ""
	var userID uuid.UUID
	authUser, exists := c.Get(middleware.ContextKeyUser)
	if exists && authUser != nil {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			userID = user.ID
			userDistrict = user.District
			userName = user.FullName
			userRole = user.Role
		}
	}

	// Fallback: query DB if JWT claims are empty
	if (userDistrict == "" || userName == "") && h.db != nil && userID != uuid.Nil {
		var rec struct {
			FullName string
			District string
		}
		h.db.Table("users").Select("full_name, district").Where("id = ?", userID).Scan(&rec)
		if userDistrict == "" && rec.District != "" {
			userDistrict = rec.District
		}
		if userName == "" && rec.FullName != "" {
			userName = rec.FullName
		}
	}

	c.HTML(http.StatusOK, "weather_forecast.html", gin.H{
		"Title":        "AgriConnect AI - Weather Forecast",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentWeather",
		"ActivePage":   "weather-forecast",
		"Districts":    weather.SupportedDistricts,
		"UserDistrict": userDistrict,
		"UserName":     userName,
		"UserRole":     userRole,
	})
}

func (h *PageHandler) AssistantPage(c *gin.Context) {
	if !h.cfg.AllowAnonymousAssistant {
		authUser, exists := c.Get(middleware.ContextKeyUser)
		if !exists || authUser == nil {
			c.Redirect(http.StatusSeeOther, "/register")
			return
		}
	}

	userID, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
	if idVal, exists := c.Get("user_id"); exists && idVal != nil {
		if s, ok := idVal.(string); ok {
			userID, _ = uuid.Parse(s)
		}
	}

	userDistrict := ""
	userName := ""
	userRole := ""
	userLanguage := ""
	if authUser, exists := c.Get(middleware.ContextKeyUser); exists {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			userDistrict = user.District
			userName = user.FullName
			userRole = user.Role
			userLanguage = user.PreferredLanguage
		}
	}

	// Fallback: query DB if JWT claims are empty (old tokens)
	if (userDistrict == "" || userName == "") && h.db != nil {
		var rec struct {
			FullName          string
			District          string
			PreferredLanguage string
		}
		h.db.Table("users").Select("full_name, district, preferred_language").Where("id = ?", userID).Scan(&rec)
		if userDistrict == "" && rec.District != "" {
			userDistrict = rec.District
		}
		if userName == "" && rec.FullName != "" {
			userName = rec.FullName
		}
		if userLanguage == "" && rec.PreferredLanguage != "" {
			userLanguage = rec.PreferredLanguage
		}
	}

	data := gin.H{
		"Title":         "AgriConnect AI - Agricultural Assistant",
		"UserID":        userID,
		"Districts":     weather.SupportedDistricts,
		"UserDistrict":  userDistrict,
		"UserName":      userName,
		"UserRole":      userRole,
		"UserLanguage":  userLanguage,
		"AIAvailable":   h.cfg.AIAvailable(),
		"Year":          time.Now().Year(),
		"ContentBlock":  "contentAssistant",
		"ActivePage":    "assistant",
	}
	c.HTML(http.StatusOK, "assistant.html", data)
}

func (h *PageHandler) ErrorPage(c *gin.Context) {
	codeStr := c.DefaultQuery("code", "403")
	errMsg := c.DefaultQuery("message", "You don't have access to this page.")
	statusMap := map[string]int{"400": 400, "401": 401, "403": 403, "404": 404, "429": 429, "500": 500}
	status := statusMap[codeStr]
	if status == 0 {
		status = http.StatusForbidden
	}
	c.HTML(status, "error.html", gin.H{
		"ErrorCode":    codeStr,
		"ErrorMessage": errMsg,
		"Title":        "AgriConnect - " + codeStr,
		"Year":         time.Now().Year(),
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
