package handlers

import (
	"time"

	"github.com/agriconnect-ai/internal/auth"
	mw "github.com/agriconnect-ai/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func populateTemplateData(c *gin.Context, db *gorm.DB, data gin.H) {
	authUser, exists := c.Get(mw.ContextKeyUser)
	if !exists || authUser == nil {
		return
	}
	user, ok := authUser.(*mw.AuthUser)
	if !ok {
		return
	}

	// extractUser only sets ID and Role from JWT — FullName/District need a DB lookup
	if user.FullName == "" && db != nil {
		var rec struct {
			FullName string
			District string
		}
		db.Table("users").Select("full_name, district").Where("id = ?", user.ID).Scan(&rec)
		user.FullName = rec.FullName
		user.District = rec.District
	}

	data["UserName"] = user.FullName
	data["UserRole"] = user.Role
	data["UserDistrict"] = user.District
	data["UserLanguage"] = "english"

	if data["Year"] == nil {
		data["Year"] = time.Now().Year()
	}

	if data["UnreadCount"] == nil && db != nil {
		var unreadCount int64
		db.Model(&auth.Notification{}).
			Where("user_id = ? AND is_read = ?", user.ID, false).
			Count(&unreadCount)
		data["UnreadCount"] = unreadCount
	}

	if data["ActivePage"] == nil {
		data["ActivePage"] = c.Request.URL.Path
	}
}

func roleHomePath(role string) string {
	switch role {
	case "admin":
		return "/admin"
	case "officer":
		return "/officer"
	default:
		return "/dashboard"
	}
}
