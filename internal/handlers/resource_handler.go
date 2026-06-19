package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/agriconnect-ai/internal/middleware"
	"github.com/agriconnect-ai/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ResourceHandler struct {
	db *gorm.DB
}

func NewResourceHandler(db *gorm.DB) *ResourceHandler {
	return &ResourceHandler{db: db}
}

func (h *ResourceHandler) ResourcesPage(c *gin.Context) {
	var categories []string
	h.db.Model(&models.AgriculturalDocument{}).
		Select("DISTINCT category").Order("category ASC").
		Where("reviewed = ?", true).
		Pluck("category", &categories)

	var crops []string
	h.db.Model(&models.AgriculturalDocument{}).
		Select("DISTINCT crop").Order("crop ASC").
		Where("reviewed = ? AND crop IS NOT NULL AND crop != ''", true).
		Pluck("crop", &crops)

	if categories == nil {
		categories = []string{}
	}
	if crops == nil {
		crops = []string{}
	}

	data := gin.H{
		"Title":        "AgriConnect AI - Learning Resources",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentResources",
		"Categories":   categories,
		"Crops":        crops,
		"ActivePage":   "resources",
	}
	authUser, exists := c.Get(middleware.ContextKeyUser)
	if exists && authUser != nil {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			data["UserName"] = user.FullName
			data["UserRole"] = user.Role
			data["UserDistrict"] = user.District
			data["UserLanguage"] = user.PreferredLanguage
		}
	}
	c.HTML(http.StatusOK, "resources.html", data)
}

func (h *ResourceHandler) ResourceDetailPage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if 	err != nil {
		c.HTML(http.StatusNotFound, "resource_detail.html", gin.H{
			"Title":        "Resource Not Found",
			"Error":        "Invalid resource ID",
			"ContentBlock": "contentResourceDetail",
			"Year":         time.Now().Year(),
			"ActivePage":   "resource-detail",
		})
		return
	}

	var doc models.AgriculturalDocument
	if err := h.db.First(&doc, "id = ?", id).Error; err != nil {
		c.HTML(http.StatusNotFound, "resource_detail.html", gin.H{
			"Title":        "Resource Not Found",
			"Error":        "Resource not found",
			"ContentBlock": "contentResourceDetail",
			"Year":         time.Now().Year(),
			"ActivePage":   "resource-detail",
		})
		return
	}

	data := gin.H{
		"Title":        doc.Title + " - AgriConnect AI",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentResourceDetail",
		"Resource":     doc,
		"ActivePage":   "resource-detail",
	}
	authUser, exists := c.Get(middleware.ContextKeyUser)
	if exists && authUser != nil {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			data["UserName"] = user.FullName
			data["UserRole"] = user.Role
			data["UserDistrict"] = user.District
			data["UserLanguage"] = user.PreferredLanguage
		}
	}
	c.HTML(http.StatusOK, "resource_detail.html", data)
}

func (h *ResourceHandler) ListResources(c *gin.Context) {
	crop := c.Query("crop")
	category := c.Query("category")
	language := c.Query("language")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	query := h.db.Model(&models.AgriculturalDocument{}).
		Where("reviewed = ?", true)

	if crop != "" {
		query = query.Where("crop = ?", crop)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if language != "" {
		query = query.Where("language = ?", language)
	}

	var total int64
	query.Count(&total)

	var docs []models.AgriculturalDocument
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load resources"})
		return
	}

	views := make([]gin.H, 0, len(docs))
	for _, doc := range docs {
		summary := doc.Content
		runes := []rune(summary)
		if len(runes) > 200 {
			summary = string(runes[:200]) + "..."
		}
		views = append(views, gin.H{
			"id":         doc.ID.String(),
			"title":      doc.Title,
			"crop":       doc.Crop,
			"category":   doc.Category,
			"language":   doc.Language,
			"summary":    summary,
			"source":     doc.Source,
			"reviewed":   doc.Reviewed,
			"created_at": doc.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"resources": views,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *ResourceHandler) GetResource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	var doc models.AgriculturalDocument
	if err := h.db.First(&doc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         doc.ID.String(),
		"title":      doc.Title,
		"crop":       doc.Crop,
		"category":   doc.Category,
		"language":   doc.Language,
		"content":    doc.Content,
		"source":     doc.Source,
		"reviewed":   doc.Reviewed,
		"created_at": doc.CreatedAt,
	})
}

func (h *ResourceHandler) CreateResource(c *gin.Context) {
	var req struct {
		Title    string `json:"title"`
		Crop     string `json:"crop"`
		Category string `json:"category"`
		Language string `json:"language"`
		Content  string `json:"content"`
		Source   string `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Title == "" || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title and content are required"})
		return
	}

	if req.Language == "" {
		req.Language = "english"
	}

	validCategories := map[string]bool{
		"planting": true, "pests": true, "disease": true, "soil": true,
		"fertiliser": true, "irrigation": true, "harvesting": true, "storage": true,
	}
	if req.Category != "" && !validCategories[req.Category] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	doc := &models.AgriculturalDocument{
		Title:    req.Title,
		Crop:     req.Crop,
		Category: req.Category,
		Language: req.Language,
		Content:  req.Content,
		Source:   req.Source,
		Reviewed: false,
	}

	if err := h.db.Create(doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create resource"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      doc.ID.String(),
		"title":   doc.Title,
		"message": "Resource created successfully.",
	})
}

func (h *ResourceHandler) UpdateResource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	var doc models.AgriculturalDocument
	if err := h.db.First(&doc, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}

	var req struct {
		Title    *string `json:"title"`
		Content  *string `json:"content"`
		Category *string `json:"category"`
		Crop     *string `json:"crop"`
		Language *string `json:"language"`
		Source   *string `json:"source"`
		Reviewed *bool   `json:"reviewed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Crop != nil {
		updates["crop"] = *req.Crop
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}
	if req.Source != nil {
		updates["source"] = *req.Source
	}
	if req.Reviewed != nil {
		updates["reviewed"] = *req.Reviewed
	}

	h.db.Model(&doc).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "Resource updated"})
}
