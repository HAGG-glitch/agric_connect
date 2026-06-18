package diagnosis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/agriconnect-ai/internal/ai"
	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/storage"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Service interface {
	CreateDiagnosis(ctx context.Context, userID uuid.UUID, input DiagnosisInput, file multipart.File, header *multipart.FileHeader) (*CropDiagnosis, error)
	GetDiagnosis(ctx context.Context, id, userID uuid.UUID) (*CropDiagnosis, error)
	ListDiagnoses(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]CropDiagnosis, int64, error)
	DeleteDiagnosis(ctx context.Context, id, userID uuid.UUID) error
	ContinueInChat(ctx context.Context, id, userID uuid.UUID, chatSvc services.ChatService) (uuid.UUID, error)
}

type service struct {
	repo         Repository
	storage      storage.ObjectStorage
	visionAI     ai.CropDiagnosisAI
	knowledgeSvc services.KnowledgeService
	cfg          *config.Config
}

func NewService(repo Repository, objStore storage.ObjectStorage, visionAI ai.CropDiagnosisAI, knowledgeSvc services.KnowledgeService, cfg *config.Config) Service {
	return &service{
		repo:         repo,
		storage:      objStore,
		visionAI:     visionAI,
		knowledgeSvc: knowledgeSvc,
		cfg:          cfg,
	}
}

func (s *service) CreateDiagnosis(ctx context.Context, userID uuid.UUID, input DiagnosisInput, file multipart.File, header *multipart.FileHeader) (*CropDiagnosis, error) {
	if err := ValidateCrop(input.Crop); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}
	if input.SymptomDescription == "" {
		return nil, fmt.Errorf("validation: symptom description is required")
	}
	if err := ValidatePlantPart(input.PlantPart); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	contentType := header.Header.Get("Content-Type")
	if err := ValidateImageType(contentType, s.cfg.AllowedImageTypes); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	maxSize := int64(s.cfg.MaxImageSizeMB) * 1024 * 1024
	if header.Size > maxSize {
		return nil, fmt.Errorf("validation: image too large (%d MB max)", s.cfg.MaxImageSizeMB)
	}

	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	if len(fileData) < 12 {
		return nil, fmt.Errorf("validation: invalid image file")
	}

	sigOK, isWebP := detectImageSignature(fileData)
	if !sigOK {
		return nil, fmt.Errorf("validation: unsupported or invalid image format")
	}

	var imgWidth, imgHeight int
	if !isWebP {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(fileData))
		if err != nil {
			return nil, fmt.Errorf("validation: cannot decode image: %w", err)
		}
		imgWidth = cfg.Width
		imgHeight = cfg.Height
	}

	if imgWidth == 0 || imgHeight == 0 {
		return nil, fmt.Errorf("validation: invalid image dimensions")
	}
	if imgWidth < s.cfg.MinImageWidth {
		return nil, fmt.Errorf("validation: image width %d is too small (minimum %d pixels)", imgWidth, s.cfg.MinImageWidth)
	}
	if imgHeight < s.cfg.MinImageHeight {
		return nil, fmt.Errorf("validation: image height %d is too small (minimum %d pixels)", imgHeight, s.cfg.MinImageHeight)
	}
	pixels := int64(imgWidth) * int64(imgHeight)
	if pixels > s.cfg.MaxImagePixels {
		return nil, fmt.Errorf("validation: image has %d pixels (maximum %d)", pixels, s.cfg.MaxImagePixels)
	}

	sha := sha256.Sum256(fileData)
	shaHex := fmt.Sprintf("%x", sha)

	diagID := uuid.New()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	randomName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	storagePath := fmt.Sprintf("anonymous-users/%s/diagnoses/%s/%s", userID.String(), diagID.String(), randomName)

	obj, err := s.storage.Save(ctx, storage.SaveObjectInput{
		Content:     bytes.NewReader(fileData),
		ContentType: contentType,
		Path:        storagePath,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: %w", err)
	}

	diag := &CropDiagnosis{
		ID:                 diagID,
		UserID:             userID,
		Crop:               input.Crop,
		District:           input.District,
		PreferredLanguage:  input.PreferredLanguage,
		PlantPart:          input.PlantPart,
		SymptomDescription: input.SymptomDescription,
		RecentWeather:      input.RecentWeather,
		FertiliserHistory:  input.FertiliserHistory,
		PesticideHistory:   input.PesticideHistory,
		ImageStoragePath:   storagePath,
		ImageOriginalName:  header.Filename,
		ImageContentType:   contentType,
		ImageSizeBytes:     obj.SizeBytes,
		ImageSHA256:        shaHex,
		Status:             "processing",
		RequiresExpertReview: true,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if input.SymptomsStartedAt != "" {
		parsed, err := time.Parse("2006-01-02", input.SymptomsStartedAt)
		if err == nil {
			diag.SymptomsStartedAt = &parsed
		}
	}

	if input.AffectedPercentage > 0 {
		ap := math.Round(input.AffectedPercentage*100) / 100
		diag.AffectedPercentage = &ap
	}

	if err := s.repo.Create(ctx, diag); err != nil {
		s.cleanupImage(ctx, storagePath)
		return nil, fmt.Errorf("database: %w", err)
	}

	go func() {
		procCtx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.DiagnosisRequestTimeout)*time.Second)
		defer cancel()

		knowledgeCtx, _, err := s.knowledgeSvc.RetrieveContext(procCtx, input.SymptomDescription, input.Crop)
		if err != nil {
			log.Printf("knowledge retrieval for diagnosis %s: %v", diagID, err)
		}

		if s.cfg.MaxDiagnosisContextChars > 0 && len(knowledgeCtx) > s.cfg.MaxDiagnosisContextChars {
			knowledgeCtx = knowledgeCtx[:s.cfg.MaxDiagnosisContextChars]
		}

		optImageData := OptimizeImageForAI(fileData, contentType)
		aiContentType := "image/jpeg"

		aiInput := ai.DiagnosisAIInput{
			ImageData:          optImageData,
			ImageContentType:   aiContentType,
			Crop:               input.Crop,
			District:           input.District,
			PlantPart:          input.PlantPart,
			SymptomDescription: input.SymptomDescription,
			SymptomsStartedAt:  input.SymptomsStartedAt,
			AffectedPercentage: input.AffectedPercentage,
			RecentWeather:      input.RecentWeather,
			FertiliserHistory:  input.FertiliserHistory,
			PesticideHistory:   input.PesticideHistory,
			PreferredLanguage:  input.PreferredLanguage,
			KnowledgeContext:   knowledgeCtx,
		}

		if imageURL, err := s.storage.SignedURL(procCtx, storagePath, 10*time.Minute); err == nil {
			aiInput.ImageURL = imageURL
			aiInput.ImageData = nil
		}

		result, err := s.visionAI.Diagnose(procCtx, aiInput)
		if err != nil && strings.Contains(err.Error(), "input_length") {
			log.Printf("vision diagnosis input length error for %s, retrying with reduced image: %v", diagID, err)
			aiInput.ImageURL = ""
			aiInput.ImageData = compressImage(fileData, contentType)
			result, err = s.visionAI.Diagnose(procCtx, aiInput)
		}
		if err != nil {
			errStr := err.Error()
			log.Printf("vision diagnosis failed for %s: model=%s, error=%v", diagID, s.cfg.GroqVisionModel, err)
			if strings.Contains(errStr, "parsing") || strings.Contains(errStr, "invalid JSON") || strings.Contains(errStr, "missing") {
				errMsg := "The image was uploaded, but the AI returned a response that could not be processed. Please try again."
				diag.ErrorMessage = errMsg
			} else {
				diag.ErrorMessage = "AI diagnosis service encountered an error."
			}
			diag.Status = "failed"
			diag.UpdatedAt = time.Now().UTC()
			if uerr := s.repo.Update(procCtx, diag); uerr != nil {
				log.Printf("failed to update diagnosis %s: %v", diagID, uerr)
			}
			return
		}

		result.Confidence = ValidateConfidence(result.Confidence)
		result.ConfidenceLabel = ValidateConfidenceLabel(result.ConfidenceLabel)
		result.Urgency = ValidateUrgency(result.Urgency)
		result.Disclaimer = EnsureDisclaimer(result.Disclaimer)
		result.ObservedSigns = ValidateStringSlice(result.ObservedSigns, 10, 500)
		result.PossibleAlternatives = ValidateStringSlice(result.PossibleAlternatives, 10, 500)
		result.RecommendedActions = ValidateStringSlice(result.RecommendedActions, 10, 500)
		result.PreventionTips = ValidateStringSlice(result.PreventionTips, 10, 500)

		rawJSON, _ := json.Marshal(result)

		diag.ProbableCondition = truncate(result.ProbableCondition, 255)
		diag.Confidence = result.Confidence
		diag.ConfidenceLabel = result.ConfidenceLabel
		diag.Description = truncate(result.Description, 5000)
		diag.ObservedSigns = toJSONArray(result.ObservedSigns)
		diag.PossibleAlternatives = toJSONArray(result.PossibleAlternatives)
		diag.RecommendedActions = toJSONArray(result.RecommendedActions)
		diag.PreventionTips = toJSONArray(result.PreventionTips)
		diag.Urgency = result.Urgency
		diag.RequiresExpertReview = result.RequiresExpertReview
		diag.Disclaimer = result.Disclaimer
		diag.RawAIResult = rawJSON
		diag.Status = "completed"
		diag.UpdatedAt = time.Now().UTC()

		if uerr := s.repo.Update(procCtx, diag); uerr != nil {
			log.Printf("failed to update diagnosis result %s: %v", diagID, uerr)
		}
	}()

	return diag, nil
}

func (s *service) GetDiagnosis(ctx context.Context, id, userID uuid.UUID) (*CropDiagnosis, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("diagnosis not found")
	}
	if d.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}
	return d, nil
}

func (s *service) ListDiagnoses(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]CropDiagnosis, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	diags, err := s.repo.FindByUserID(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	return diags, count, nil
}

func (s *service) DeleteDiagnosis(ctx context.Context, id, userID uuid.UUID) error {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("diagnosis not found")
	}
	if d.UserID != userID {
		return fmt.Errorf("access denied")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting diagnosis: %w", err)
	}

	s.cleanupImage(ctx, d.ImageStoragePath)
	return nil
}

func (s *service) ContinueInChat(ctx context.Context, id, userID uuid.UUID, chatSvc services.ChatService) (uuid.UUID, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("diagnosis not found")
	}
	if d.UserID != userID {
		return uuid.Nil, fmt.Errorf("access denied")
	}

	lang := d.PreferredLanguage
	if lang == "" {
		lang = "english"
	}

	conv, err := chatSvc.CreateConversation(ctx, userID, lang, d.District, d.Crop)
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating conversation: %w", err)
	}

	conv.Title = truncate("Crop diagnosis: "+d.Crop+" - "+d.ProbableCondition, 200)

	return conv.ID, nil
}

func (s *service) cleanupImage(ctx context.Context, path string) {
	if path == "" {
		return
	}
	if err := s.storage.Delete(ctx, path); err != nil {
		log.Printf("failed to clean up image %s: %v", path, err)
	}
}

func OptimizeImageForAI(data []byte, contentType string) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}

	maxDim := 768
	img = resizeImageMaxDim(img, maxDim)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}); err != nil {
		return data
	}
	return buf.Bytes()
}

func compressImage(data []byte, contentType string) []byte {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}

	maxDim := 512
	img = resizeImageMaxDim(img, maxDim)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return data
	}
	return buf.Bytes()
}

func resizeImageMaxDim(img image.Image, maxDim int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}

	ratio := float64(maxDim) / float64(w)
	if float64(h)*ratio > float64(maxDim) {
		ratio = float64(maxDim) / float64(h)
	}

	nw := int(float64(w) * ratio)
	nh := int(float64(h) * ratio)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	scaleImage(dst, img)
	return dst
}

func scaleImage(dst *image.RGBA, src image.Image) {
	bounds := dst.Bounds()
	dw := bounds.Dx()
	dh := bounds.Dy()
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()

	for dy := 0; dy < dh; dy++ {
		for dx := 0; dx < dw; dx++ {
			sx := dx * sw / dw
			sy := dy * sh / dh
			dst.Set(dx, dy, src.At(sx, sy))
		}
	}
}

func detectImageSignature(data []byte) (ok bool, isWebP bool) {
	if len(data) >= 4 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 {
		return true, true
	}
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return true, false
	}
	if len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return true, false
	}
	return false, false
}

func toJSONArray(strs []string) datatypes.JSON {
	if strs == nil {
		strs = []string{}
	}
	b, _ := json.Marshal(strs)
	return datatypes.JSON(b)
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
