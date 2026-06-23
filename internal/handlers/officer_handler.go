package handlers

import (
	"fmt"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/agriconnect-ai/internal/diagnosis"
	"github.com/agriconnect-ai/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OfficerHandler struct {
	db           *gorm.DB
	diagnosisSvc diagnosis.Service
}

func NewOfficerHandler(db *gorm.DB, diagnosisSvc diagnosis.Service) *OfficerHandler {
	return &OfficerHandler{db: db, diagnosisSvc: diagnosisSvc}
}

func (h *OfficerHandler) OfficerPage(c *gin.Context) {
	user := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)

	var pendingCount, inReviewCount, completedCount int64

	h.db.Model(&diagnosis.CropDiagnosis{}).
		Where("status IN ?", []string{"completed", "awaiting_review"}).
		Count(&pendingCount)

	h.db.Model(&diagnosis.CropDiagnosis{}).
		Where("status = ?", "under_review").
		Count(&inReviewCount)

	h.db.Model(&diagnosis.CropDiagnosis{}).
		Where("status = ?", "reviewed").
		Count(&completedCount)

	var userRecord struct {
		FullName string
		District string
	}
	h.db.Table("users").Select("full_name, district").
		Where("id = ?", user.ID).Scan(&userRecord)

	c.HTML(http.StatusOK, "officer_dashboard.html", gin.H{
		"Title":         "AgriConnect AI - Officer Dashboard",
		"Year":          time.Now().Year(),
		"ContentBlock":  "contentOfficerDashboard",
		"PendingCount":  pendingCount,
		"InReviewCount": inReviewCount,
		"CompletedCount": completedCount,
		"UserName":      userRecord.FullName,
		"OfficerDistrict": userRecord.District,
		"UserRole":      "officer",
		"ActivePage":    "officer",
	})
}

func (h *OfficerHandler) OfficerDiagnosesPage(c *gin.Context) {
	data := gin.H{
		"Title":        "AgriConnect AI - Diagnosis Queue",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentOfficerDiagnoses",
		"ActivePage":   "officer-diagnoses",
	}
	populateTemplateData(c, h.db, data)
	c.HTML(http.StatusOK, "officer_diagnoses.html", data)
}

func (h *OfficerHandler) OfficerDiagnosisDetailPage(c *gin.Context) {
	data := gin.H{
		"Title":        "AgriConnect AI - Review Diagnosis",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentOfficerDiagnosisDetail",
		"ActivePage":   "officer-diagnoses",
	}
	populateTemplateData(c, h.db, data)
	c.HTML(http.StatusOK, "officer_diagnosis_detail.html", data)
}

func (h *OfficerHandler) ListDiagnoses(c *gin.Context) {
	user := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	crop := c.Query("crop")
	urgency := c.Query("urgency")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}

	query := h.db.Model(&diagnosis.CropDiagnosis{}).
		Where("status IN ?", []string{"completed", "awaiting_review", "under_review", "reviewed"})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if crop != "" {
		query = query.Where("crop = ?", crop)
	}
	if urgency != "" {
		query = query.Where("urgency = ?", urgency)
	}

	var total int64
	query.Count(&total)

	var diags []diagnosis.CropDiagnosis
	offset := (page - 1) * pageSize
	var orderExpr interface{} = "created_at DESC"
	if user.District != "" {
		orderExpr = gorm.Expr("CASE WHEN district = ? THEN 0 ELSE 1 END, created_at DESC", user.District)
	}
	if err := query.Order(orderExpr).Limit(pageSize).Offset(offset).Find(&diags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load diagnoses"})
		return
	}

	views := make([]gin.H, 0, len(diags))
	for _, d := range diags {
		view := diagnosisToView(&d)
		if user.District != "" {
			view["is_own_district"] = d.District == user.District
		} else {
			view["is_own_district"] = false
		}
		views = append(views, view)
	}

	c.JSON(http.StatusOK, gin.H{
		"diagnoses": views,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *OfficerHandler) GetDiagnosis(c *gin.Context) {
	idStr := c.Param("id")
	diagID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	var d diagnosis.CropDiagnosis
	if err := h.db.First(&d, "id = ?", diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	type reviewWithOfficer struct {
		auth.DiagnosisReview
		OfficerName string `json:"officer_name"`
	}
	var reviews []reviewWithOfficer
	h.db.Table("diagnosis_reviews").
		Select("diagnosis_reviews.*, COALESCE(users.full_name, 'Unknown') as officer_name").
		Joins("LEFT JOIN users ON users.id = diagnosis_reviews.officer_id").
		Where("diagnosis_reviews.diagnosis_id = ? AND diagnosis_reviews.is_hidden = false", diagID).
		Order("diagnosis_reviews.created_at DESC").
		Find(&reviews)

	view := diagnosisToView(&d)
	view["reviews"] = reviews

	c.JSON(http.StatusOK, view)
}

func (h *OfficerHandler) CreateReview(c *gin.Context) {
	user := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)
	idStr := c.Param("id")
	diagID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	var d diagnosis.CropDiagnosis
	if err := h.db.First(&d, "id = ?", diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	var req struct {
		ReviewStatus       string `json:"review_status"`
		ConfirmedCondition string `json:"confirmed_condition"`
		OfficerComment     string `json:"officer_comment"`
		Recommendation     string `json:"recommendation"`
		Urgency            string `json:"urgency"`
		RequiresFieldVisit bool   `json:"requires_field_visit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	validStatuses := map[string]bool{
		"pending": true, "in_review": true, "confirmed": true,
		"needs_more_information": true, "field_visit_required": true, "closed": true,
	}

	reviewStatus := req.ReviewStatus
	if reviewStatus == "" || !validStatuses[reviewStatus] {
		reviewStatus = "pending"
	}

	review := &auth.DiagnosisReview{
		ID:                 uuid.New(),
		DiagnosisID:        diagID,
		OfficerID:          user.ID,
		ReviewStatus:       reviewStatus,
		ConfirmedCondition: req.ConfirmedCondition,
		OfficerComment:     req.OfficerComment,
		Recommendation:     req.Recommendation,
		Urgency:            req.Urgency,
		RequiresFieldVisit: req.RequiresFieldVisit,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if err := h.db.Create(review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create review"})
		return
	}

	// Don't lock diagnosis to a single review anymore — just set to under_review
	h.db.Model(&diagnosis.CropDiagnosis{}).Where("id = ?", diagID).Update("status", "under_review")

	// Create notification for farmer
	h.createNotification(d.UserID, "New Diagnosis Review",
		fmt.Sprintf("An extension officer has shared their opinion on your %s diagnosis.", d.Crop),
		"review_created", "crop_diagnosis", diagID)

	// Audit log
	h.writeAuditLog(&user.ID, "review_created", "diagnosis_review", &review.ID, "diagnosis_id", diagID.String())

	c.JSON(http.StatusCreated, review)
}

func (h *OfficerHandler) UpdateReview(c *gin.Context) {
	user := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)
	diagIDStr := c.Param("id")
	reviewIDStr := c.Param("reviewID")

	diagID, err := uuid.Parse(diagIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review ID"})
		return
	}

	var d diagnosis.CropDiagnosis
	if err := h.db.First(&d, "id = ?", diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	var review auth.DiagnosisReview
	if err := h.db.First(&review, "id = ? AND diagnosis_id = ?", reviewID, diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	var req struct {
		ReviewStatus       string `json:"review_status"`
		ConfirmedCondition string `json:"confirmed_condition"`
		OfficerComment     string `json:"officer_comment"`
		Recommendation     string `json:"recommendation"`
		Urgency            string `json:"urgency"`
		RequiresFieldVisit bool   `json:"requires_field_visit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	validStatuses := map[string]bool{
		"pending": true, "in_review": true, "confirmed": true,
		"needs_more_information": true, "field_visit_required": true, "closed": true,
	}

	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}

	if req.ReviewStatus != "" && validStatuses[req.ReviewStatus] {
		updates["review_status"] = req.ReviewStatus
	}
	if req.ConfirmedCondition != "" {
		updates["confirmed_condition"] = req.ConfirmedCondition
	}
	updates["officer_comment"] = req.OfficerComment
	updates["recommendation"] = req.Recommendation
	updates["urgency"] = req.Urgency
	updates["requires_field_visit"] = req.RequiresFieldVisit

	h.db.Model(&review).Updates(updates)

	newDiagStatus := "under_review"
	if req.ReviewStatus == "confirmed" || req.ReviewStatus == "closed" {
		newDiagStatus = "reviewed"
	} else if req.ReviewStatus == "needs_more_information" {
		newDiagStatus = "awaiting_review"
	}
	h.db.Model(&diagnosis.CropDiagnosis{}).Where("id = ?", diagID).Update("status", newDiagStatus)

	h.db.First(&review, "id = ?", reviewID)

	// Create notification
	if req.ReviewStatus == "needs_more_information" {
		h.createNotification(d.UserID, "More Information Needed",
			"The extension officer needs more information about your crop diagnosis.",
			"info_requested", "crop_diagnosis", diagID)
	} else if req.ReviewStatus == "confirmed" || req.ReviewStatus == "closed" {
		h.createNotification(d.UserID, "Diagnosis Review Completed",
			"The extension officer has completed reviewing your crop diagnosis.",
			"review_completed", "crop_diagnosis", diagID)
	}

	h.writeAuditLog(&user.ID, "review_updated", "diagnosis_review", &reviewID, "diagnosis_id", diagID.String())

	c.JSON(http.StatusOK, gin.H{"review": review})
}

func (h *OfficerHandler) ClaimCase(c *gin.Context) {
	user := c.MustGet(middleware.ContextKeyUser).(*middleware.AuthUser)
	idStr := c.Param("id")
	diagID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	var d diagnosis.CropDiagnosis
	if err := h.db.First(&d, "id = ?", diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	var existing auth.DiagnosisReview
	if err := h.db.Where("diagnosis_id = ? AND review_status NOT IN ('closed')", diagID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This diagnosis is already claimed by another officer"})
		return
	}

	review := &auth.DiagnosisReview{
		ID:           uuid.New(),
		DiagnosisID:  diagID,
		OfficerID:    user.ID,
		ReviewStatus: "in_review",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := h.db.Create(review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim case"})
		return
	}

	h.db.Model(&diagnosis.CropDiagnosis{}).Where("id = ?", diagID).Update("status", "under_review")

	h.createNotification(d.UserID, "Diagnosis Claimed",
		fmt.Sprintf("An extension officer has started reviewing your %s diagnosis.", d.Crop),
		"review_started", "crop_diagnosis", diagID)

	h.writeAuditLog(&user.ID, "case_claimed", "crop_diagnosis", &diagID, "officer_id", user.ID.String())

	c.JSON(http.StatusOK, gin.H{"message": "Case claimed", "review_id": review.ID.String()})
}

func (h *OfficerHandler) createNotification(userID uuid.UUID, title, message, notifType, entityType string, entityID uuid.UUID) {
	notif := &auth.Notification{
		ID:               uuid.New(),
		UserID:           userID,
		Title:            title,
		Message:          message,
		NotificationType: notifType,
		EntityType:       entityType,
		EntityID:         &entityID,
		CreatedAt:        time.Now().UTC(),
	}
	if err := h.db.Create(notif).Error; err != nil {
		log.Printf("failed to create notification: %v", err)
	}
}

func (h *OfficerHandler) writeAuditLog(actorID *uuid.UUID, action, entityType string, entityID *uuid.UUID, metaKey, metaValue string) {
	metaBytes, _ := json.Marshal(map[string]interface{}{metaKey: metaValue})

	auditEntry := &auth.AuditLog{
		ID:          uuid.New(),
		ActorUserID: actorID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Metadata:    datatypes.JSON(metaBytes),
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.db.Create(auditEntry).Error; err != nil {
		log.Printf("failed to write audit log: %v", err)
	}
}
