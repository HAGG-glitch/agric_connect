package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
	h.db.Model(&diagnosisModel{}).Where("status IN ?", []string{"completed", "awaiting_review", "under_review"}).Count(&pendingReviewCount)

	data := gin.H{
		"Title":          "AgriConnect AI - Admin",
		"Year":           time.Now().Year(),
		"FarmerCount":    farmerCount,
		"OfficerCount":   officerCount,
		"AdminCount":     adminCount,
		"DiagnosisCount": diagCount,
		"PendingReviews": pendingReviewCount,
		"ContentBlock":   "contentAdminUsers",
		"ActivePage":     "admin",
	}
	h.addUserData(c, data)
	c.HTML(http.StatusOK, "admin_users.html", data)
}

func (h *AdminHandler) addUserData(c *gin.Context, data gin.H) {
	authUser, exists := c.Get(middleware.ContextKeyUser)
	if exists && authUser != nil {
		if user, ok := authUser.(*middleware.AuthUser); ok {
			data["UserName"] = user.FullName
			data["UserRole"] = user.Role
			data["UserDistrict"] = user.District
		}
	}
	if data["UnreadCount"] == nil {
		data["UnreadCount"] = int64(0)
	}
}

type diagnosisModel struct{}

func (diagnosisModel) TableName() string { return "crop_diagnoses" }

func (h *AdminHandler) AdminDiagnosesPage(c *gin.Context) {
	data := gin.H{
		"Title":        "AgriConnect AI - All Diagnoses",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentAdminDiagnoses",
		"ActivePage":   "admin-diagnoses",
	}
	h.addUserData(c, data)
	c.HTML(http.StatusOK, "admin_diagnoses.html", data)
}

func (h *AdminHandler) AdminReviewsPage(c *gin.Context) {
	data := gin.H{
		"Title":        "AgriConnect AI - All Reviews",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentAdminReviews",
		"ActivePage":   "admin-reviews",
	}
	h.addUserData(c, data)
	c.HTML(http.StatusOK, "admin_reviews.html", data)
}

func (h *AdminHandler) AdminAuditLogsPage(c *gin.Context) {
	data := gin.H{
		"Title":        "AgriConnect AI - Audit Logs",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentAdminAuditLogs",
		"ActivePage":   "admin-audit",
	}
	h.addUserData(c, data)
	c.HTML(http.StatusOK, "admin_audit_logs.html", data)
}

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

	// Notify the affected user
	notifTitle := "Account Activated"
	notifMsg := "Your AgriConnect account has been activated. You can now log in and access the platform."
	if !*req.IsActive {
		notifTitle = "Account Deactivated"
		notifMsg = "Your AgriConnect account has been deactivated. Please contact an administrator for more information."
	}
	notif := &auth.Notification{
		ID:               uuid.New(),
		UserID:           targetUser.ID,
		Title:            notifTitle,
		Message:          notifMsg,
		NotificationType: "account_status",
		EntityType:       "user",
		EntityID:         &targetUser.ID,
		CreatedAt:        time.Now().UTC(),
	}
	if err := h.db.Create(notif).Error; err != nil {
		log.Printf("failed to create notification: %v", err)
	}
}

func (h *AdminHandler) ListDiagnoses(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	crop := c.Query("crop")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	query := h.db.Table("crop_diagnoses")
	if crop != "" {
		query = query.Where("crop = ?", crop)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	type adminDiagView struct {
		ID                 string    `json:"id"`
		Crop               string    `json:"crop"`
		ProbableCondition  string    `json:"probable_condition"`
		Status             string    `json:"status"`
		Urgency            string    `json:"urgency"`
		District           string    `json:"district"`
		UserID             string    `json:"user_id"`
		ImageURL           string    `json:"image_url"`
		CreatedAt          time.Time `json:"created_at"`
	}
	var results []adminDiagView
	offset := (page - 1) * pageSize
	if err := query.Select("id::text, crop, COALESCE(probable_condition,'') as probable_condition, status, COALESCE(urgency,'') as urgency, COALESCE(district,'') as district, user_id::text, COALESCE(image_storage_path,'') as image_url, created_at").
		Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load diagnoses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"diagnoses": results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AdminHandler) ListReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	query := h.db.Table("diagnosis_reviews").
		Select(`diagnosis_reviews.id::text, diagnosis_reviews.diagnosis_id::text,
			diagnosis_reviews.officer_id::text, diagnosis_reviews.review_status,
			COALESCE(diagnosis_reviews.confirmed_condition,'') as confirmed_condition,
			diagnosis_reviews.created_at, diagnosis_reviews.updated_at,
			COALESCE(crop_diagnoses.crop,'') as crop_name,
			COALESCE(users.full_name,'') as officer_name`).
		Joins("LEFT JOIN crop_diagnoses ON crop_diagnoses.id::text = diagnosis_reviews.diagnosis_id::text").
		Joins("LEFT JOIN users ON users.id::text = diagnosis_reviews.officer_id::text")

	if status != "" {
		query = query.Where("diagnosis_reviews.review_status = ?", status)
	}

	var total int64
	query.Count(&total)

	type reviewView struct {
		ID                 string    `json:"id"`
		DiagnosisID        string    `json:"diagnosis_id"`
		CropName           string    `json:"crop_name"`
		OfficerID          string    `json:"officer_id"`
		OfficerName        string    `json:"officer_name"`
		ReviewStatus       string    `json:"review_status"`
		ConfirmedCondition string    `json:"confirmed_condition"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
	}
	var results []reviewView
	offset := (page - 1) * pageSize
	if err := query.Order("diagnosis_reviews.created_at DESC").Limit(pageSize).Offset(offset).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":   results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "30"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 30
	}

	query := h.db.Table("audit_logs").
		Select(`audit_logs.id::text, audit_logs.action, audit_logs.entity_type,
			audit_logs.entity_id::text, audit_logs.metadata, audit_logs.created_at,
			COALESCE(users.full_name,'') as actor_name`).
		Joins("LEFT JOIN users ON users.id::text = audit_logs.actor_user_id::text")

	var total int64
	query.Count(&total)

	type auditLogView struct {
		ID         string          `json:"id"`
		ActorName  string          `json:"actor_name"`
		Action     string          `json:"action"`
		EntityType string          `json:"entity_type"`
		EntityID   string          `json:"entity_id"`
		Metadata   json.RawMessage `json:"metadata"`
		CreatedAt  time.Time       `json:"created_at"`
	}
	var results []auditLogView
	offset := (page - 1) * pageSize
	if err := query.Order("audit_logs.created_at DESC").Limit(pageSize).Offset(offset).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      results,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
