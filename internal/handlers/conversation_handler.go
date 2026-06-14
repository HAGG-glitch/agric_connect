package handlers

import (
	"net/http"

	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/validation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ConversationHandler struct {
	chatSvc services.ChatService
}

func NewConversationHandler(chatSvc services.ChatService) *ConversationHandler {
	return &ConversationHandler{chatSvc: chatSvc}
}

func getUserID(c *gin.Context) uuid.UUID {
	uid, _ := c.Get("user_id")
	id, _ := uuid.Parse(uid.(string))
	return id
}

func (h *ConversationHandler) Create(c *gin.Context) {
	userID := getUserID(c)

	var req struct {
		PreferredLanguage string `json:"preferred_language"`
		District          string `json:"district"`
		Crop              string `json:"crop"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := validation.Language(req.PreferredLanguage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.District(req.District); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conv, err := h.chatSvc.CreateConversation(c.Request.Context(), userID, req.PreferredLanguage, req.District, req.Crop)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	c.JSON(http.StatusCreated, conv)
}

func (h *ConversationHandler) List(c *gin.Context) {
	userID := getUserID(c)
	convs, err := h.chatSvc.ListConversations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load conversations"})
		return
	}
	c.JSON(http.StatusOK, convs)
}

func (h *ConversationHandler) Get(c *gin.Context) {
	userID := getUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	conv, err := h.chatSvc.GetConversation(c.Request.Context(), id, userID)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	c.JSON(http.StatusOK, conv)
}

func (h *ConversationHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
		return
	}

	if err := h.chatSvc.DeleteConversation(c.Request.Context(), id, userID); err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation deleted"})
}
