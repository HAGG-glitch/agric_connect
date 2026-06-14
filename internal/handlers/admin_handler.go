package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/agriconnect-ai/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) AdminPage(c *gin.Context) {
	var farmerCount, officerCount, adminCount, diagCount, pendingReviewCount int64
	h.db.Model(&auth.User{}).Where("role = ?", "farmer").Count(&farmerCount)
	h.db.Model(&auth.User{}).Where("role = ?", "officer").Count(&officerCount)
	h.db.Model(&auth.User{}).Where("role = ?", "admin").Count(&adminCount)
	h.db.Model(&diagnosisModel{}).Count(&diagCount)
	h.db.Model(&diagnosisModel{}).Where("status IN ?", []string{"ai_completed", "awaiting_review", "under_review"}).Count(&pendingReviewCount)

	c.HTML(http.StatusOK, "admin_users.html", gin.H{
		"Title":            "AgriConnect AI - Admin",
		"Year":             time.Now().Year(),
		"FarmerCount":      farmerCount,
		"OfficerCount":     officerCount,
		"AdminCount":       adminCount,
		"DiagnosisCount":   diagCount,
		"PendingReviews":   pendingReviewCount,
	})
}

type diagnosisModel struct{}

func (diagnosisModel) TableName() string { return "crop_diagnoses" }

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []auth.User
	if err := h.db.Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load users"})
		return
	}

	views := make([]gin.H, 0, len(users))
	for _, u := range users {
		views = append(views, gin.H{
			"id":                 u.ID.String(),
			"full_name":          u.FullName,
			"phone_number":       u.PhoneNumber,
			"district":           u.District,
			"preferred_language": u.PreferredLanguage,
			"role":               u.Role,
			"is_active":          u.IsActive,
			"created_at":         u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": views})
}

func (h *AdminHandler) UpdateRole(c *gin.Context) {
	actor := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	validRoles := map[string]bool{"farmer": true, "officer": true, "admin": true}
	if !validRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	var targetUser auth.User
	if err := h.db.First(&targetUser, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if targetUser.ID == actor.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change your own role"})
		return
	}

	// Prevent demoting the final active admin
	if targetUser.Role == "admin" && req.Role != "admin" {
		var adminCount int64
		h.db.Model(&auth.User{}).Where("role = ? AND is_active = ?", "admin", true).Count(&adminCount)
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot demote the last active admin"})
			return
		}
	}

	oldRole := targetUser.Role
	h.db.Model(&targetUser).Update("role", req.Role)

	metaBytes, _ := json.Marshal(map[string]interface{}{
		"old_role": oldRole,
		"new_role": req.Role,
		"target_user_id": userID.String(),
	})
	logEntry := &auth.AuditLog{
		ID:          uuid.New(),
		ActorUserID: &actor.ID,
		Action:      "role_change",
		EntityType:  "user",
		EntityID:    &userID,
		Metadata:    datatypes.JSON(metaBytes),
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.db.Create(logEntry).Error; err != nil {
		log.Printf("failed to write audit log: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}

func (h *AdminHandler) UpdateStatus(c *gin.Context) {
	actor := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		IsActive *bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.IsActive == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_active is required"})
		return
	}

	var targetUser auth.User
	if err := h.db.First(&targetUser, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if targetUser.ID == actor.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot change your own status"})
		return
	}

	// Prevent deactivating the last active admin
	if targetUser.Role == "admin" && !*req.IsActive {
		var adminCount int64
		h.db.Model(&auth.User{}).Where("role = ? AND is_active = ?", "admin", true).Count(&adminCount)
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot deactivate the last active admin"})
			return
		}
	}

	h.db.Model(&targetUser).Update("is_active", *req.IsActive)

	meta := map[string]interface{}{
		"is_active": *req.IsActive,
		"target_user_id": userID.String(),
	}
	metaJSON, _ := json.Marshal(meta)
	logEntry := &auth.AuditLog{
		ID:          uuid.New(),
		ActorUserID: &actor.ID,
		Action:      "status_change",
		EntityType:  "user",
		EntityID:    &userID,
		Metadata:    datatypes.JSON(metaJSON),
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.db.Create(logEntry).Error; err != nil {
		log.Printf("failed to write audit log: %v", err)
	}
}
