package handlers

import (
	"net/http"
	"time"

	"github.com/agriconnect-ai/internal/auth"
	mw "github.com/agriconnect-ai/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
}

func NewDashboardHandler(db *gorm.DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

func (h *DashboardHandler) DashboardPage(c *gin.Context) {
	authUser, exists := c.Get(mw.ContextKeyUser)
	if !exists || authUser == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	user, ok := authUser.(*mw.AuthUser)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	var userRecord struct {
		FullName          string
		PhoneNumber       string
		District          string
		PreferredLanguage string
		Role              string
	}
	if err := h.db.Table("users").
		Select("full_name, phone_number, district, preferred_language, role").
		Where("id = ?", user.ID).
		Scan(&userRecord).Error; err != nil {
		userRecord.FullName = user.FullName
		userRecord.Role = user.Role
	}

	var recentDiags []map[string]interface{}
	h.db.Raw(`
		SELECT id, crop, probable_condition, status, urgency, created_at
		FROM crop_diagnoses
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 5
	`, user.ID).Scan(&recentDiags)

	var recentConvs []map[string]interface{}
	h.db.Raw(`
		SELECT id, title, created_at
		FROM ai_conversations
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 5
	`, user.ID).Scan(&recentConvs)

	var unreadCount int64
	h.db.Model(&auth.Notification{}).
		Where("user_id = ? AND is_read = ?", user.ID, false).
		Count(&unreadCount)

	if recentDiags == nil {
		recentDiags = []map[string]interface{}{}
	}
	if recentConvs == nil {
		recentConvs = []map[string]interface{}{}
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Title":          "AgriConnect AI - Dashboard",
		"Year":           time.Now().Year(),
		"ContentBlock":   "contentDashboard",
		"UserName":       userRecord.FullName,
		"UserPhone":      maskPhone(userRecord.PhoneNumber),
		"UserDistrict":   userRecord.District,
		"UserLanguage":   userRecord.PreferredLanguage,
		"UserRole":       userRecord.Role,
		"RecentDiagnoses":     recentDiags,
		"RecentConversations": recentConvs,
		"UnreadCount":    unreadCount,
		"ActivePage":     "dashboard",
	})
}

func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	runes := []rune(phone)
	return string(runes[:len(runes)-4]) + "****"
}
