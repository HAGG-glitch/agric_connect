package handlers

import (
	"net/http"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationHandler struct {
	db *gorm.DB
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := getUserID(c)

	var notifs []auth.Notification
	if err := h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&notifs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load notifications"})
		return
	}

	var unreadCount int64
	h.db.Model(&auth.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&unreadCount)

	views := make([]gin.H, 0, len(notifs))
	for _, n := range notifs {
		views = append(views, gin.H{
			"id":                n.ID.String(),
			"title":             n.Title,
			"message":           n.Message,
			"notification_type": n.NotificationType,
			"is_read":           n.IsRead,
			"entity_type":       n.EntityType,
			"entity_id":         n.EntityID,
			"created_at":        n.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": views,
		"unread_count":  unreadCount,
	})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID := getUserID(c)
	idStr := c.Param("id")
	notifID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	result := h.db.Model(&auth.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Update("is_read", true)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}
