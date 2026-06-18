package handlers

import (
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/diagnosis"
	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/storage"
	"github.com/agriconnect-ai/internal/weather"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DiagnosisHandler struct {
	svc      diagnosis.Service
	cfg      *config.Config
	objStore storage.ObjectStorage
	chatSvc  services.ChatService
}

func NewDiagnosisHandler(svc diagnosis.Service, cfg *config.Config, objStore storage.ObjectStorage, chatSvc services.ChatService) *DiagnosisHandler {
	return &DiagnosisHandler{svc: svc, cfg: cfg, objStore: objStore, chatSvc: chatSvc}
}

func (h *DiagnosisHandler) DiagnosePage(c *gin.Context) {
	c.HTML(http.StatusOK, "diagnose.html", gin.H{
		"Title":          "AgriConnect AI - Crop Diagnosis",
		"Districts":      weather.SupportedDistricts,
		"PlantParts":     diagnosis.ValidPlantParts,
		"MaxImageSizeMB": h.cfg.MaxImageSizeMB,
		"MinImageWidth":  h.cfg.MinImageWidth,
		"MinImageHeight": h.cfg.MinImageHeight,
		"AIAvailable":    h.cfg.AIAvailable(),
		"Year":           time.Now().Year(),
		"ContentBlock":   "contentDiagnose",
	})
}

func (h *DiagnosisHandler) HistoryPage(c *gin.Context) {
	c.HTML(http.StatusOK, "diagnosis_history.html", gin.H{
		"Title":        "AgriConnect AI - Diagnosis History",
		"Year":         time.Now().Year(),
		"ContentBlock": "contentDiagnosisHistory",
	})
}

func (h *DiagnosisHandler) DetailPage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.HTML(http.StatusNotFound, "diagnosis_detail.html", gin.H{"Error": "Invalid diagnosis ID", "ContentBlock": "contentDiagnosisDetail"})
		return
	}

	userID := getUserID(c)
	d, err := h.svc.GetDiagnosis(c.Request.Context(), id, userID)
	if err != nil {
		c.HTML(http.StatusNotFound, "diagnosis_detail.html", gin.H{"Error": "Diagnosis not found", "ContentBlock": "contentDiagnosisDetail"})
		return
	}

	confidence := int(math.Round(d.Confidence))
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 100 {
		confidence = 100
	}

	hasConfidence := d.Status == "completed" && confidence > 0

	var confidenceBarClass string
	switch {
	case confidence >= 70:
		confidenceBarClass = "bg-green-600"
	case confidence >= 40:
		confidenceBarClass = "bg-amber-500"
	default:
		confidenceBarClass = "bg-red-500"
	}

	c.HTML(http.StatusOK, "diagnosis_detail.html", gin.H{
		"Title":                 "AgriConnect AI - Diagnosis Detail",
		"Diagnosis":             d,
		"Year":                  time.Now().Year(),
		"ContentBlock":          "contentDiagnosisDetail",
		"ConfidencePercent":     confidence,
		"ConfidenceWidth":       strconv.Itoa(confidence) + "%",
		"ConfidenceBarClass":    confidenceBarClass,
		"HasConfidence":         hasConfidence,
		"ConfidenceDisplayText": strconv.Itoa(confidence) + "%",
	})
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
	}

	if d.ConfidenceLabel != "" {
		view["confidence_label"] = d.ConfidenceLabel
	}

	return view
}
