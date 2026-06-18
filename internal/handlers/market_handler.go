package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/agriconnect-ai/internal/middleware"
	"github.com/agriconnect-ai/internal/models"
	"github.com/agriconnect-ai/internal/weather"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MarketHandler struct {
	db *gorm.DB
}

func NewMarketHandler(db *gorm.DB) *MarketHandler {
	return &MarketHandler{db: db}
}

func (h *MarketHandler) MarketPricesPage(c *gin.Context) {
	var commodities []string
	h.db.Model(&models.MarketPrice{}).
		Select("DISTINCT commodity").Order("commodity ASC").
		Pluck("commodity", &commodities)

	var districts []string
	h.db.Model(&models.MarketPrice{}).
		Select("DISTINCT district").Order("district ASC").
		Pluck("district", &districts)

	if commodities == nil {
		commodities = []string{}
	}
	if districts == nil {
		districts = []string{}
	}

	c.HTML(http.StatusOK, "market_prices.html", gin.H{
		"Title":        "AgriConnect AI - Market Prices",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentMarketPrices",
		"Commodities":  commodities,
		"Districts":    districts,
	})
}

func (h *MarketHandler) ListPrices(c *gin.Context) {
	commodity := c.Query("commodity")
	district := c.Query("district")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	query := h.db.Model(&models.MarketPrice{})

	if commodity != "" {
		query = query.Where("commodity = ?", commodity)
	}
	if district != "" {
		query = query.Where("district = ?", district)
	}

	var total int64
	query.Count(&total)

	var prices []models.MarketPrice
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&prices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load prices"})
		return
	}

	views := make([]gin.H, 0, len(prices))
	for _, p := range prices {
		views = append(views, gin.H{
			"id":          p.ID.String(),
			"commodity":   p.Commodity,
			"market_name": p.MarketName,
			"district":    p.District,
			"price":       p.Price,
			"currency":    p.Currency,
			"unit":        p.Unit,
			"source":      p.Source,
			"is_verified": p.IsVerified,
			"created_at":  p.CreatedAt,
			"updated_at":  p.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"prices": views,
		"total":  total,
		"page":   page,
		"page_size": pageSize,
	})
}

func (h *MarketHandler) CreatePrice(c *gin.Context) {
	user := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)

	var req struct {
		Commodity  string  `json:"commodity"`
		MarketName string  `json:"market_name"`
		District   string  `json:"district"`
		Price      float64 `json:"price"`
		Unit       string  `json:"unit"`
		Source     string  `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Commodity == "" || req.MarketName == "" || req.District == "" || req.Price <= 0 || req.Unit == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "commodity, market_name, district, price, and unit are required"})
		return
	}

	if !weather.IsValidDistrict(req.District) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid district"})
		return
	}

	price := &models.MarketPrice{
		ID:         uuid.New(),
		Commodity:  req.Commodity,
		MarketName: req.MarketName,
		District:   req.District,
		Price:      req.Price,
		Currency:   "SLE",
		Unit:       req.Unit,
		Source:     req.Source,
		IsVerified: user.Role == "admin",
		CreatedBy:  &user.ID,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := h.db.Create(price).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create price"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          price.ID.String(),
		"commodity":   price.Commodity,
		"market_name": price.MarketName,
		"district":    price.District,
		"price":       price.Price,
		"currency":    price.Currency,
		"unit":        price.Unit,
		"is_verified": price.IsVerified,
	})
}

func (h *MarketHandler) UpdatePrice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price ID"})
		return
	}

	var price models.MarketPrice
	if err := h.db.First(&price, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Price not found"})
		return
	}

	var req struct {
		Price      *float64 `json:"price"`
		MarketName *string  `json:"market_name"`
		IsVerified *bool    `json:"is_verified"`
		Source     *string  `json:"source"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}
	if req.Price != nil {
		updates["price"] = *req.Price
	}
	if req.MarketName != nil {
		updates["market_name"] = *req.MarketName
	}
	if req.IsVerified != nil {
		updates["is_verified"] = *req.IsVerified
	}
	if req.Source != nil {
		updates["source"] = *req.Source
	}

	h.db.Model(&price).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"message": "Price updated"})
}

func (h *MarketHandler) DeletePrice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price ID"})
		return
	}

	result := h.db.Where("id = ?", id).Delete(&models.MarketPrice{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Price not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Price deleted"})
}

// Commodities returns the list of supported MVP commodities
func SupportedCommodities() []string {
	return []string{"rice", "cassava", "groundnut", "palm oil", "cocoa", "coffee"}
}

func (h *MarketHandler) WriteAuditLog(actorID *uuid.UUID, action, entityType string, entityID *uuid.UUID, metaInfo gin.H) {
	importsForAudit := struct {
		EncodingJSON string
		AuditLog     string
		Datatypes    string
	}{}
	_ = importsForAudit
}
