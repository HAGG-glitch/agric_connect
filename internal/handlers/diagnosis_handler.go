package handlers

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agriconnect-ai/internal/auth"
	"github.com/agriconnect-ai/internal/middleware"
	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/diagnosis"
	"github.com/agriconnect-ai/internal/models"
	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/storage"
	"github.com/agriconnect-ai/internal/weather"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DiagnosisHandler struct {
	svc      diagnosis.Service
	cfg      *config.Config
	objStore storage.ObjectStorage
	chatSvc  services.ChatService
	db       *gorm.DB
}

func NewDiagnosisHandler(svc diagnosis.Service, cfg *config.Config, objStore storage.ObjectStorage, chatSvc services.ChatService, db *gorm.DB) *DiagnosisHandler {
	return &DiagnosisHandler{svc: svc, cfg: cfg, objStore: objStore, chatSvc: chatSvc, db: db}
}

func (h *DiagnosisHandler) DiagnosePage(c *gin.Context) {
	data := gin.H{
		"Title":          "AgriConnect AI - Crop Diagnosis",
		"Districts":      weather.SupportedDistricts,
		"PlantParts":     diagnosis.ValidPlantParts,
		"MaxImageSizeMB": h.cfg.MaxImageSizeMB,
		"MinImageWidth":  h.cfg.MinImageWidth,
		"MinImageHeight": h.cfg.MinImageHeight,
		"AIAvailable":    h.cfg.AIAvailable(),
		"Year":           time.Now().Year(),
		"ContentBlock":   "contentDiagnose",
		"ActivePage":     "diagnose",
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
	c.HTML(http.StatusOK, "diagnose.html", data)
}

func (h *DiagnosisHandler) HistoryPage(c *gin.Context) {
	data := gin.H{
		"Title":        "AgriConnect AI - Diagnosis History",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentDiagnosisHistory",
		"ActivePage":   "diagnoses",
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
	c.HTML(http.StatusOK, "diagnosis_history.html", data)
}

func (h *DiagnosisHandler) DetailPage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.HTML(http.StatusNotFound, "diagnosis_detail.html", gin.H{"Error": "Invalid diagnosis ID", "ContentBlock": "contentDiagnosisDetail", "ActivePage": "diagnosis-detail"})
		return
	}

	userID := getUserID(c)
	d, err := h.svc.GetDiagnosis(c.Request.Context(), id, userID)
	if err != nil {
		c.HTML(http.StatusNotFound, "diagnosis_detail.html", gin.H{"Error": "Diagnosis not found", "ContentBlock": "contentDiagnosisDetail", "ActivePage": "diagnosis-detail"})
		return
	}

	confidence := int(math.Round(d.Confidence))
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 100 {
		confidence = 100
	}

	hasConfidence := confidence > 0

	type reviewWithOfficer struct {
		auth.DiagnosisReview
		OfficerName string `json:"officer_name"`
	}
	var reviews []reviewWithOfficer
	var uniqueOfficerCount int64 = 0
	if h.db != nil {
		h.db.Table("diagnosis_reviews").
			Select("diagnosis_reviews.*, COALESCE(users.full_name, 'Unknown') as officer_name").
			Joins("LEFT JOIN users ON users.id = diagnosis_reviews.officer_id").
			Where("diagnosis_reviews.diagnosis_id = ? AND diagnosis_reviews.is_hidden = false", id).
			Order("diagnosis_reviews.created_at DESC").
			Find(&reviews)
		h.db.Table("diagnosis_reviews").
			Where("diagnosis_id = ? AND is_hidden = false", id).
			Distinct("officer_id").
			Count(&uniqueOfficerCount)
	}

	var relatedDocs []models.AgriculturalDocument
	if d.Crop != "" && h.db != nil {
		h.db.Model(&models.AgriculturalDocument{}).
			Where("reviewed = ? AND LOWER(crop) = ?", true, strings.ToLower(d.Crop)).
			Order("created_at DESC").
			Limit(5).
			Find(&relatedDocs)
	}
	if relatedDocs == nil {
		relatedDocs = []models.AgriculturalDocument{}
	}

	data := gin.H{
		"Title":                 "AgriConnect AI - Diagnosis Detail",
		"Diagnosis":             d,
		"Reviews":               reviews,
		"UniqueOfficerCount":    uniqueOfficerCount,
		"RelatedResources":      relatedDocs,
		"Year":                  time.Now().Year(),
		"ContentBlock":          "contentDiagnosisDetail",
		"ConfidencePercent":     confidence,
		"HasConfidence":         hasConfidence,
		"ConfidenceDisplayText": strconv.Itoa(confidence) + "%",
		"ActivePage":            "diagnosis-detail",
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
	if data["UserName"] == nil {
		data["UserName"] = ""
	}
	if data["UserRole"] == nil {
		data["UserRole"] = ""
	}
	if data["UserDistrict"] == nil {
		data["UserDistrict"] = ""
	}
	if data["UserLanguage"] == nil {
		data["UserLanguage"] = "english"
	}
	if data["UnreadCount"] == nil {
		data["UnreadCount"] = 0
	}
	c.HTML(http.StatusOK, "diagnosis_detail.html", data)
}

func (h *DiagnosisHandler) Create(c *gin.Context) {
	userID := getUserID(c)

	maxSize := int64(h.cfg.MaxImageSizeMB+1) * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

	if err := c.Request.ParseMultipartForm(maxSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image too large or invalid form data"})
		return
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image is required"})
		return
	}
	defer file.Close()

	symptoms := c.PostForm("symptom_description")
	if symptoms == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symptom description is required"})
		return
	}

	affectedPct, _ := strconv.ParseFloat(c.PostForm("affected_percentage"), 64)
	if affectedPct < 0 || affectedPct > 100 {
		affectedPct = 0
	}
	affectedPct = math.Round(affectedPct*100) / 100

	district := c.PostForm("district")
	if district != "" && !weather.IsValidDistrict(district) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported district"})
		return
	}

	input := diagnosis.DiagnosisInput{
		Crop:               c.PostForm("crop"),
		District:           district,
		PreferredLanguage:  c.PostForm("preferred_language"),
		PlantPart:          c.PostForm("plant_part"),
		SymptomDescription: symptoms,
		SymptomsStartedAt:  c.PostForm("symptoms_started_at"),
		AffectedPercentage: affectedPct,
		RecentWeather:      c.PostForm("recent_weather"),
		FertiliserHistory:  c.PostForm("fertiliser_history"),
		PesticideHistory:   c.PostForm("pesticide_history"),
	}

	result, err := h.svc.CreateDiagnosis(c.Request.Context(), userID, input, file, header)
	if err != nil {
		log.Printf("diagnosis creation failed: %v", err)
		errMsg := err.Error()
		if strings.HasPrefix(errMsg, "validation:") {
			c.JSON(http.StatusBadRequest, gin.H{"error": strings.TrimPrefix(errMsg, "validation: ")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit diagnosis. Please try again."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     result.ID.String(),
		"status": result.Status,
	})

	// Notify officers and admins after AI completes
	go func() {
		diagID := result.ID
		for i := 0; i < 40; i++ {
			time.Sleep(2 * time.Second)
			var d diagnosis.CropDiagnosis
			if err := h.db.First(&d, "id = ?", diagID).Error; err != nil {
				return
			}
			if d.Status != "completed" {
				if d.Status == "failed" {
					return
				}
				continue
			}
			if d.RequiresExpertReview {
				var users []auth.User
				h.db.Where("role IN ?", []string{"officer", "admin"}).Find(&users)
				msg := fmt.Sprintf("A new %s diagnosis in %s needs your review.", d.Crop, d.District)
				if d.ProbableCondition != "" {
					msg = fmt.Sprintf("A new %s diagnosis (%s) in %s needs your review.", d.Crop, d.ProbableCondition, d.District)
				}
				for _, u := range users {
					h.db.Create(&auth.Notification{
						ID:               uuid.New(),
						UserID:           u.ID,
						Title:            "New Diagnosis Needs Review",
						Message:          msg,
						NotificationType: "diagnosis_needs_review",
						EntityType:       "crop_diagnosis",
						EntityID:         &d.ID,
						CreatedAt:        time.Now().UTC(),
					})
				}
			}
			return
		}
	}()
}

func (h *DiagnosisHandler) List(c *gin.Context) {
	userID := getUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	diags, count, err := h.svc.ListDiagnoses(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load diagnoses"})
		return
	}

	views := make([]gin.H, 0, len(diags))
	for _, d := range diags {
		views = append(views, diagnosisToView(&d))
	}

	c.JSON(http.StatusOK, gin.H{
		"diagnoses":  views,
		"total":      count,
		"page":       page,
		"page_size":  pageSize,
	})
}

func (h *DiagnosisHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	userID := getUserID(c)
	d, err := h.svc.GetDiagnosis(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	c.JSON(http.StatusOK, diagnosisToView(d))
}

func (h *DiagnosisHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	userID := getUserID(c)
	if err := h.svc.DeleteDiagnosis(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Diagnosis deleted"})
}

func (h *DiagnosisHandler) ServeImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	userID := getUserID(c)
	d, err := h.svc.GetDiagnosis(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	if d.ImageStoragePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	reader, err := h.objStore.Download(c.Request.Context(), d.ImageStoragePath)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Image file was not found in storage. The database record exists, but the stored file could not be retrieved."})
			return
		}
		log.Printf("failed to download image for diagnosis %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serve image"})
		return
	}
	defer reader.Close()

	contentType := d.ImageContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, reader)
}

func (h *DiagnosisHandler) ContinueInChat(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}

	userID := getUserID(c)
	convID, err := h.svc.ContinueInChat(c.Request.Context(), id, userID, h.chatSvc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversation_id": convID.String()})
}

func (h *DiagnosisHandler) AcceptReview(c *gin.Context) {
	userID := getUserID(c)
	idStr := c.Param("id")
	reviewIDStr := c.Param("reviewId")

	diagID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review ID"})
		return
	}

	// Verify diagnosis belongs to this user
	d, err := h.svc.GetDiagnosis(c.Request.Context(), diagID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	var review auth.DiagnosisReview
	if err := h.db.First(&review, "id = ? AND diagnosis_id = ?", reviewID, diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	// Un-accept all other reviews for this diagnosis, then accept this one
	h.db.Model(&auth.DiagnosisReview{}).Where("diagnosis_id = ?", diagID).Update("is_accepted", false)
	h.db.Model(&auth.DiagnosisReview{}).Where("id = ?", reviewID).Update("is_accepted", true)

	// Update diagnosis status to reflect accepted review
	h.db.Model(&diagnosis.CropDiagnosis{}).Where("id = ?", diagID).Update("status", "reviewed")

	// Notify the officer
	h.db.Exec(`INSERT INTO notifications (id, user_id, title, message, notification_type, entity_type, entity_id, created_at)
		SELECT gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW()`,
		review.OfficerID, "Farmer Accepted Your Review",
		fmt.Sprintf("The farmer accepted your review on their %s diagnosis.", d.Crop),
		"review_accepted", "crop_diagnosis", diagID)

	c.JSON(http.StatusOK, gin.H{"message": "Review accepted"})
}

func (h *DiagnosisHandler) RejectReview(c *gin.Context) {
	userID := getUserID(c)
	idStr := c.Param("id")
	reviewIDStr := c.Param("reviewId")

	diagID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review ID"})
		return
	}

	// Verify diagnosis belongs to this user
	if _, err := h.svc.GetDiagnosis(c.Request.Context(), diagID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	var review auth.DiagnosisReview
	if err := h.db.First(&review, "id = ? AND diagnosis_id = ? AND is_accepted = true", reviewID, diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No accepted review to reject"})
		return
	}

	h.db.Model(&auth.DiagnosisReview{}).Where("id = ?", reviewID).Update("is_accepted", false)
	c.JSON(http.StatusOK, gin.H{"message": "Review rejected"})
}

func (h *DiagnosisHandler) ApproveRequest(c *gin.Context) {
	userID := getUserID(c)
	idStr := c.Param("id")
	reviewIDStr := c.Param("reviewId")

	diagID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review ID"})
		return
	}

	// Verify diagnosis belongs to this user
	d, err := h.svc.GetDiagnosis(c.Request.Context(), diagID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	var review auth.DiagnosisReview
	if err := h.db.First(&review, "id = ? AND diagnosis_id = ? AND request_status = 'pending'", reviewID, diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pending request not found"})
		return
	}

	// Approve the request
	h.db.Model(&review).Updates(map[string]interface{}{
		"request_status": "approved",
		"updated_at":     time.Now().UTC(),
	})

	// Update diagnosis status based on review status
	reviewStatus := review.ReviewStatus
	newDiagStatus := "under_review"
	if reviewStatus == "confirmed" || reviewStatus == "closed" {
		newDiagStatus = "reviewed"
	} else if reviewStatus == "needs_more_information" {
		newDiagStatus = "awaiting_review"
	} else if reviewStatus == "field_visit_required" {
		newDiagStatus = "under_review"
	}
	h.db.Model(&diagnosis.CropDiagnosis{}).Where("id = ?", diagID).Update("status", newDiagStatus)

	// Notify the officer
	h.db.Exec(`INSERT INTO notifications (id, user_id, title, message, notification_type, entity_type, entity_id, created_at)
		SELECT gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW()`,
		review.OfficerID, "Request Approved",
		fmt.Sprintf("The farmer approved your request regarding their %s diagnosis.", d.Crop),
		"request_approved", "crop_diagnosis", diagID)

	c.JSON(http.StatusOK, gin.H{"message": "Request approved"})
}

func (h *DiagnosisHandler) RejectRequest(c *gin.Context) {
	userID := getUserID(c)
	idStr := c.Param("id")
	reviewIDStr := c.Param("reviewId")

	diagID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid diagnosis ID"})
		return
	}
	reviewID, err := uuid.Parse(reviewIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review ID"})
		return
	}

	// Verify diagnosis belongs to this user
	d, err := h.svc.GetDiagnosis(c.Request.Context(), diagID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Diagnosis not found"})
		return
	}

	var review auth.DiagnosisReview
	if err := h.db.First(&review, "id = ? AND diagnosis_id = ? AND request_status = 'pending'", reviewID, diagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pending request not found"})
		return
	}

	// Reject the request — hide the review from view
	h.db.Model(&review).Updates(map[string]interface{}{
		"request_status": "rejected",
		"is_hidden":      true,
		"updated_at":     time.Now().UTC(),
	})

	// Notify the officer
	h.db.Exec(`INSERT INTO notifications (id, user_id, title, message, notification_type, entity_type, entity_id, created_at)
		SELECT gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW()`,
		review.OfficerID, "Request Rejected",
		fmt.Sprintf("The farmer declined your request regarding their %s diagnosis.", d.Crop),
		"request_rejected", "crop_diagnosis", diagID)

	c.JSON(http.StatusOK, gin.H{"message": "Request rejected"})
}

func diagnosisToView(d *diagnosis.CropDiagnosis) gin.H {
	view := gin.H{
		"id":                     d.ID.String(),
		"crop":                   d.Crop,
		"district":               d.District,
		"plant_part":             d.PlantPart,
		"symptom_description":    d.SymptomDescription,
		"probable_condition":     d.ProbableCondition,
		"confidence":             d.Confidence,
		"confidence_label":       d.ConfidenceLabel,
		"description":            d.Description,
		"observed_signs":         d.GetObservedSigns(),
		"possible_alternatives":  d.GetPossibleAlternatives(),
		"recommended_actions":    d.GetRecommendedActions(),
		"prevention_tips":        d.GetPreventionTips(),
		"urgency":                d.Urgency,
		"requires_expert_review": d.RequiresExpertReview,
		"disclaimer":             d.Disclaimer,
		"status":                 d.Status,
		"error_message":          d.ErrorMessage,
		"created_at":             d.CreatedAt,
		"affected_percentage":   d.AffectedPercentage,
		"affected_display":      d.GetAffectedPercentageDisplay(),
		"symptoms_started_at":   d.SymptomsStartedAt,
		"recent_weather":        d.RecentWeather,
		"fertiliser_history":    d.FertiliserHistory,
		"pesticide_history":     d.PesticideHistory,
	}

	if d.ConfidenceLabel != "" {
		view["confidence_label"] = d.ConfidenceLabel
	}

	return view
}
