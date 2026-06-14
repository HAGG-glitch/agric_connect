package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/agriconnect-ai/internal/ai"
	"github.com/agriconnect-ai/internal/auth"
	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/diagnosis"
	"github.com/agriconnect-ai/internal/handlers"
	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/storage"
	"github.com/agriconnect-ai/internal/transcription"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ── Mock service types ────────────────────────────────────────────────────────

type mockDiagnosisService struct {
	createFunc       func(context.Context, uuid.UUID, diagnosis.DiagnosisInput, multipart.File, *multipart.FileHeader) (*diagnosis.CropDiagnosis, error)
	getFunc          func(context.Context, uuid.UUID, uuid.UUID) (*diagnosis.CropDiagnosis, error)
	listFunc         func(context.Context, uuid.UUID, int, int) ([]diagnosis.CropDiagnosis, int64, error)
	deleteFunc       func(context.Context, uuid.UUID, uuid.UUID) error
	continueInChatFunc func(context.Context, uuid.UUID, uuid.UUID, services.ChatService) (uuid.UUID, error)
}

func (m *mockDiagnosisService) CreateDiagnosis(ctx context.Context, userID uuid.UUID, input diagnosis.DiagnosisInput, file multipart.File, header *multipart.FileHeader) (*diagnosis.CropDiagnosis, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, userID, input, file, header)
	}
	return nil, fmt.Errorf("unexpected CreateDiagnosis call")
}

func (m *mockDiagnosisService) GetDiagnosis(ctx context.Context, id, userID uuid.UUID) (*diagnosis.CropDiagnosis, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id, userID)
	}
	return nil, fmt.Errorf("unexpected GetDiagnosis call")
}

func (m *mockDiagnosisService) ListDiagnoses(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]diagnosis.CropDiagnosis, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, page, pageSize)
	}
	return nil, 0, fmt.Errorf("unexpected ListDiagnoses call")
}

func (m *mockDiagnosisService) DeleteDiagnosis(ctx context.Context, id, userID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return fmt.Errorf("unexpected DeleteDiagnosis call")
}

func (m *mockDiagnosisService) ContinueInChat(ctx context.Context, id, userID uuid.UUID, chatSvc services.ChatService) (uuid.UUID, error) {
	if m.continueInChatFunc != nil {
		return m.continueInChatFunc(ctx, id, userID, chatSvc)
	}
	return uuid.Nil, fmt.Errorf("unexpected ContinueInChat call")
}

type mockTranscriptionService struct {
	transcribeFunc func(context.Context, transcription.TranscriptionInput) (*transcription.TranscriptionResponse, error)
}

func (m *mockTranscriptionService) Transcribe(ctx context.Context, input transcription.TranscriptionInput) (*transcription.TranscriptionResponse, error) {
	if m.transcribeFunc != nil {
		return m.transcribeFunc(ctx, input)
	}
	return nil, fmt.Errorf("unexpected Transcribe call")
}

type mockAuthService struct {
	registerFunc             func(context.Context, auth.RegisterInput) (*auth.TokenPair, error)
	loginFunc                func(context.Context, auth.LoginInput) (*auth.TokenPair, error)
	refreshTokenFunc         func(context.Context, string) (*auth.TokenPair, error)
	logoutFunc               func(context.Context, uuid.UUID, uuid.UUID) error
	getUserFunc              func(context.Context, uuid.UUID) (*auth.UserView, error)
	transferAnonymousDataFunc func(context.Context, uuid.UUID, uuid.UUID) error
	normalizePhoneFunc       func(string) string
}

func (m *mockAuthService) Register(ctx context.Context, input auth.RegisterInput) (*auth.TokenPair, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, input)
	}
	return nil, fmt.Errorf("unexpected Register call")
}

func (m *mockAuthService) Login(ctx context.Context, input auth.LoginInput) (*auth.TokenPair, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, input)
	}
	return nil, fmt.Errorf("unexpected Login call")
}

func (m *mockAuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (*auth.TokenPair, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, refreshTokenStr)
	}
	return nil, fmt.Errorf("unexpected RefreshToken call")
}

func (m *mockAuthService) Logout(ctx context.Context, userID, refreshTokenID uuid.UUID) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, userID, refreshTokenID)
	}
	return fmt.Errorf("unexpected Logout call")
}

func (m *mockAuthService) GetUser(ctx context.Context, userID uuid.UUID) (*auth.UserView, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, userID)
	}
	return nil, fmt.Errorf("unexpected GetUser call")
}

func (m *mockAuthService) TransferAnonymousData(ctx context.Context, anonymousID, userID uuid.UUID) error {
	if m.transferAnonymousDataFunc != nil {
		return m.transferAnonymousDataFunc(ctx, anonymousID, userID)
	}
	return fmt.Errorf("unexpected TransferAnonymousData call")
}

func (m *mockAuthService) NormalizePhone(phone string) string {
	if m.normalizePhoneFunc != nil {
		return m.normalizePhoneFunc(phone)
	}
	return phone
}

type mockRoundTripper struct {
	fn func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func createPNGWithSize(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding test PNG: %v", err)
	}
	return buf.Bytes()
}

func diagnosisHandlerConfig() *config.Config {
	return &config.Config{
		MaxImageSizeMB:    5,
		MinImageWidth:     10,
		MinImageHeight:    10,
		MaxImagePixels:    25000000,
		AllowedImageTypes: []string{"image/jpeg", "image/png", "image/webp"},
	}
}

func decodeJSON(t *testing.T, body io.Reader, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("decoding JSON response: %v", err)
	}
}

// ── Diagnosis Handler Tests ───────────────────────────────────────────────────

func TestDiagnosisHandler_Create_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	diagID := uuid.New()
	imgData := createValidPNG(t)

	svc := &mockDiagnosisService{
		createFunc: func(_ context.Context, uid uuid.UUID, _ diagnosis.DiagnosisInput, _ multipart.File, _ *multipart.FileHeader) (*diagnosis.CropDiagnosis, error) {
			return &diagnosis.CropDiagnosis{
				ID:     diagID,
				UserID: uid,
				Status: "processing",
			}, nil
		},
	}

	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.POST("/api/v1/diagnoses", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.Create(c)
	})

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("symptom_description", "yellow leaves with spots")
	w.WriteField("crop", "cassava")
	w.WriteField("district", "bo")
	fw, _ := w.CreateFormFile("image", "test.png")
	fw.Write(imgData)
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/diagnoses", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["id"] != diagID.String() {
		t.Errorf("expected diagnosis ID %s, got %s", diagID.String(), body["id"])
	}
	if body["status"] != "processing" {
		t.Errorf("expected status 'processing', got %q", body["status"])
	}
}

func TestDiagnosisHandler_Create_MissingImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockDiagnosisService{}
	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.POST("/api/v1/diagnoses", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		handler.Create(c)
	})

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("symptom_description", "yellow leaves")
	w.WriteField("crop", "cassava")
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/diagnoses", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["error"] != "Image is required" {
		t.Errorf("expected 'Image is required', got %q", body["error"])
	}
}

func TestDiagnosisHandler_Create_InvalidDistrict(t *testing.T) {
	gin.SetMode(gin.TestMode)

	imgData := createValidPNG(t)
	svc := &mockDiagnosisService{}
	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.POST("/api/v1/diagnoses", func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		handler.Create(c)
	})

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("symptom_description", "yellow leaves")
	w.WriteField("crop", "cassava")
	w.WriteField("district", "nonexistent")
	fw, _ := w.CreateFormFile("image", "test.png")
	fw.Write(imgData)
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/diagnoses", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["error"] != "Unsupported district" {
		t.Errorf("expected 'Unsupported district', got %q", body["error"])
	}
}

func TestDiagnosisHandler_Get_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	diagID := uuid.New()

	svc := &mockDiagnosisService{
		getFunc: func(_ context.Context, id, uid uuid.UUID) (*diagnosis.CropDiagnosis, error) {
			return nil, fmt.Errorf("access denied")
		},
	}

	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.GET("/api/v1/diagnoses/:id", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.Get(c)
	})

	req := httptest.NewRequest("GET", "/api/v1/diagnoses/"+diagID.String(), nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["error"] != "Diagnosis not found" {
		t.Errorf("expected 'Diagnosis not found', got %q", body["error"])
	}
}

func TestDiagnosisHandler_Get_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	diagID := uuid.New()
	now := time.Now().UTC()
	expectedDiag := &diagnosis.CropDiagnosis{
		ID:                   diagID,
		UserID:               userID,
		Crop:                 "cassava",
		District:             "bo",
		SymptomDescription:   "yellow leaves with spots",
		ProbableCondition:    "Cassava Mosaic Disease",
		Confidence:           85.5,
		ConfidenceLabel:      "high",
		Description:          "A viral disease.",
		Urgency:              "high",
		RequiresExpertReview: true,
		Disclaimer:           "Preliminary AI assessment.",
		Status:               "completed",
		CreatedAt:            now,
	}

	svc := &mockDiagnosisService{
		getFunc: func(_ context.Context, id, uid uuid.UUID) (*diagnosis.CropDiagnosis, error) {
			return expectedDiag, nil
		},
	}

	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.GET("/api/v1/diagnoses/:id", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.Get(c)
	})

	req := httptest.NewRequest("GET", "/api/v1/diagnoses/"+diagID.String(), nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]interface{}
	decodeJSON(t, resp.Body, &body)

	if body["id"] != diagID.String() {
		t.Errorf("expected id %s, got %v", diagID.String(), body["id"])
	}
	if body["crop"] != "cassava" {
		t.Errorf("expected crop 'cassava', got %v", body["crop"])
	}
	if body["probable_condition"] != "Cassava Mosaic Disease" {
		t.Errorf("expected 'Cassava Mosaic Disease', got %v", body["probable_condition"])
	}
}

func TestDiagnosisHandler_Delete_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	diagID := uuid.New()

	svc := &mockDiagnosisService{
		deleteFunc: func(_ context.Context, id, uid uuid.UUID) error {
			return fmt.Errorf("access denied")
		},
	}

	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.DELETE("/api/v1/diagnoses/:id", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.Delete(c)
	})

	req := httptest.NewRequest("DELETE", "/api/v1/diagnoses/"+diagID.String(), nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["error"] != "Diagnosis not found" {
		t.Errorf("expected 'Diagnosis not found', got %q", body["error"])
	}
}

func TestDiagnosisHandler_ServeImage_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	diagID := uuid.New()

	svc := &mockDiagnosisService{
		getFunc: func(_ context.Context, id, uid uuid.UUID) (*diagnosis.CropDiagnosis, error) {
			return &diagnosis.CropDiagnosis{
				ID:               diagID,
				UserID:           userID,
				ImageStoragePath: "",
			}, nil
		},
	}

	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.GET("/api/v1/diagnoses/:id/image", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.ServeImage(c)
	})

	req := httptest.NewRequest("GET", "/api/v1/diagnoses/"+diagID.String()+"/image", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["error"] != "Image not found" {
		t.Errorf("expected 'Image not found', got %q", body["error"])
	}
}

func TestDiagnosisHandler_ContinueInChat_Ownership(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	diagID := uuid.New()

	svc := &mockDiagnosisService{
		continueInChatFunc: func(_ context.Context, id, uid uuid.UUID, _ services.ChatService) (uuid.UUID, error) {
			return uuid.Nil, fmt.Errorf("access denied")
		},
	}

	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.POST("/api/v1/diagnoses/:id/continue-in-chat", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.ContinueInChat(c)
	})

	req := httptest.NewRequest("POST", "/api/v1/diagnoses/"+diagID.String()+"/continue-in-chat", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["error"] != "Diagnosis not found" {
		t.Errorf("expected 'Diagnosis not found', got %q", body["error"])
	}
}

func TestDiagnosisHandler_List_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	diags := []diagnosis.CropDiagnosis{
		{ID: uuid.New(), UserID: userID, Crop: "cassava", Status: "completed", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), UserID: userID, Crop: "rice", Status: "completed", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), UserID: userID, Crop: "maize", Status: "processing", CreatedAt: time.Now().UTC()},
	}

	svc := &mockDiagnosisService{
		listFunc: func(_ context.Context, uid uuid.UUID, page, pageSize int) ([]diagnosis.CropDiagnosis, int64, error) {
			if page != 1 && pageSize != 2 {
				t.Errorf("expected page=1, pageSize=2, got page=%d, pageSize=%d", page, pageSize)
			}
			return diags, 3, nil
		},
	}

	handler := handlers.NewDiagnosisHandler(svc, diagnosisHandlerConfig(), &mockObjectStorage{}, nil)

	r := gin.New()
	r.GET("/api/v1/diagnoses", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.List(c)
	})

	req := httptest.NewRequest("GET", "/api/v1/diagnoses?page=1&page_size=2", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body struct {
		Diagnoses []map[string]interface{} `json:"diagnoses"`
		Total     int                      `json:"total"`
		Page      int                      `json:"page"`
		PageSize  int                      `json:"page_size"`
	}
	decodeJSON(t, resp.Body, &body)

	if body.Total != 3 {
		t.Errorf("expected total 3, got %d", body.Total)
	}
	if body.Page != 1 {
		t.Errorf("expected page 1, got %d", body.Page)
	}
	if body.PageSize != 2 {
		t.Errorf("expected page_size 2, got %d", body.PageSize)
	}
	if len(body.Diagnoses) != 3 {
		t.Errorf("expected 3 diagnoses, got %d", len(body.Diagnoses))
	}
}

// ── Transcription Handler Tests ───────────────────────────────────────────────

func transcribeHandlerConfig() *config.Config {
	return &config.Config{
		MaxAudioSizeMB:  10,
		AllowedAudioTypes: []string{"audio/webm", "audio/wav", "audio/mpeg", "audio/mp4", "audio/ogg"},
	}
}

func TestTranscriptionHandler_Transcribe_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	audioData := []byte("fake audio binary data")
	svc := &mockTranscriptionService{
		transcribeFunc: func(_ context.Context, input transcription.TranscriptionInput) (*transcription.TranscriptionResponse, error) {
			if len(input.Audio) == 0 {
				t.Error("expected non-empty audio data")
			}
			return &transcription.TranscriptionResponse{
				Transcript:         "Cassava leaves are turning yellow",
				DetectedLanguage:   "english",
				RequiresConfirmation: false,
				ExperimentalKrio:   false,
			}, nil
		},
	}

	handler := handlers.NewTranscriptionHandler(svc, transcribeHandlerConfig())

	r := gin.New()
	r.POST("/api/v1/ai/transcribe", handler.Transcribe)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("language_hint", "english")
	audioHdr := make(textproto.MIMEHeader)
	audioHdr.Set("Content-Disposition", `form-data; name="audio"; filename="test.webm"`)
	audioHdr.Set("Content-Type", "audio/webm")
	fw, _ := w.CreatePart(audioHdr)
	fw.Write(audioData)
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/ai/transcribe", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body transcription.TranscriptionResponse
	decodeJSON(t, resp.Body, &body)

	if body.Transcript != "Cassava leaves are turning yellow" {
		t.Errorf("expected transcript, got %q", body.Transcript)
	}
	if body.DetectedLanguage != "english" {
		t.Errorf("expected 'english', got %q", body.DetectedLanguage)
	}
}

func TestTranscriptionHandler_Transcribe_MissingAudio(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockTranscriptionService{}
	handler := handlers.NewTranscriptionHandler(svc, transcribeHandlerConfig())

	r := gin.New()
	r.POST("/api/v1/ai/transcribe", handler.Transcribe)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("language_hint", "english")
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/ai/transcribe", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if body["error"] != "Audio file is required" {
		t.Errorf("expected 'Audio file is required', got %q", body["error"])
	}
}

func TestTranscriptionHandler_Transcribe_InvalidLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	audioData := []byte("fake audio")
	svc := &mockTranscriptionService{}
	handler := handlers.NewTranscriptionHandler(svc, transcribeHandlerConfig())

	r := gin.New()
	r.POST("/api/v1/ai/transcribe", handler.Transcribe)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("language_hint", "french")
	audioHdr := make(textproto.MIMEHeader)
	audioHdr.Set("Content-Disposition", `form-data; name="audio"; filename="test.webm"`)
	audioHdr.Set("Content-Type", "audio/webm")
	fw, _ := w.CreatePart(audioHdr)
	fw.Write(audioData)
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/ai/transcribe", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestTranscriptionHandler_Transcribe_KrioRequiresConfirmation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	audioData := []byte("fake krio audio")
	svc := &mockTranscriptionService{
		transcribeFunc: func(_ context.Context, input transcription.TranscriptionInput) (*transcription.TranscriptionResponse, error) {
			return &transcription.TranscriptionResponse{
				Transcript:           "Wetin de apun na di farm",
				DetectedLanguage:     "krio",
				RequiresConfirmation: true,
				ExperimentalKrio:     true,
			}, nil
		},
	}

	handler := handlers.NewTranscriptionHandler(svc, transcribeHandlerConfig())

	r := gin.New()
	r.POST("/api/v1/ai/transcribe", handler.Transcribe)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("language_hint", "krio")
	audioHdr := make(textproto.MIMEHeader)
	audioHdr.Set("Content-Disposition", `form-data; name="audio"; filename="test.webm"`)
	audioHdr.Set("Content-Type", "audio/webm")
	fw, _ := w.CreatePart(audioHdr)
	fw.Write(audioData)
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/ai/transcribe", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body transcription.TranscriptionResponse
	decodeJSON(t, resp.Body, &body)

	if !body.RequiresConfirmation {
		t.Error("expected RequiresConfirmation=true for krio")
	}
	if !body.ExperimentalKrio {
		t.Error("expected ExperimentalKrio=true for krio")
	}
	if body.Transcript != "Wetin de apun na di farm" {
		t.Errorf("unexpected transcript: %q", body.Transcript)
	}
}

func TestTranscriptionHandler_Transcribe_ProviderFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	audioData := []byte("fake audio")
	svc := &mockTranscriptionService{
		transcribeFunc: func(_ context.Context, input transcription.TranscriptionInput) (*transcription.TranscriptionResponse, error) {
			return nil, fmt.Errorf("transcription API returned 503")
		},
	}

	handler := handlers.NewTranscriptionHandler(svc, transcribeHandlerConfig())

	r := gin.New()
	r.POST("/api/v1/ai/transcribe", handler.Transcribe)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("language_hint", "english")
	audioHdr := make(textproto.MIMEHeader)
	audioHdr.Set("Content-Disposition", `form-data; name="audio"; filename="test.webm"`)
	audioHdr.Set("Content-Type", "audio/webm")
	fw, _ := w.CreatePart(audioHdr)
	fw.Write(audioData)
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/ai/transcribe", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if !containsStr(body["error"], "Transcription failed") {
		t.Errorf("expected safe error message, got %q", body["error"])
	}
}

// ── Image Validation Tests ────────────────────────────────────────────────────

func TestImageValidation_ValidDimensions(t *testing.T) {
	cfg := testConfig()
	imgData := createPNGWithSize(t, 100, 100)

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{
			result: aiResult(),
		},
		&mockKnowledgeService{},
		cfg,
	)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err != nil {
		t.Fatalf("expected no error for valid dimensions, got: %v", err)
	}
}

func TestImageValidation_WidthTooSmall(t *testing.T) {
	cfg := testConfig()
	cfg.MinImageWidth = 50
	imgData := createPNGWithSize(t, 30, 100)

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg,
	)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for width too small")
	}
	if !containsStr(err.Error(), "validation:") || !containsStr(err.Error(), "width") {
		t.Errorf("expected validation error about width, got: %v", err)
	}
}

func TestImageValidation_HeightTooSmall(t *testing.T) {
	cfg := testConfig()
	cfg.MinImageHeight = 50
	imgData := createPNGWithSize(t, 100, 30)

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg,
	)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for height too small")
	}
	if !containsStr(err.Error(), "validation:") || !containsStr(err.Error(), "height") {
		t.Errorf("expected validation error about height, got: %v", err)
	}
}

func TestImageValidation_PixelCountTooLarge(t *testing.T) {
	cfg := testConfig()
	cfg.MaxImagePixels = 50000
	imgData := createPNGWithSize(t, 300, 300)

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg,
	)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for pixel count too large")
	}
	if !containsStr(err.Error(), "validation:") || !containsStr(err.Error(), "pixels") {
		t.Errorf("expected validation error about pixel count, got: %v", err)
	}
}

func aiResult() *ai.DiagnosisAIResult {
	return &ai.DiagnosisAIResult{
		Crop:                "cassava",
		ProbableCondition:   "Test Condition",
		Confidence:          80,
		ConfidenceLabel:     "high",
		Description:         "Test description",
		Urgency:             "medium",
		RequiresExpertReview: false,
	}
}

// ── Auth Handler Tests ────────────────────────────────────────────────────────

func TestAuthHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	svc := &mockAuthService{
		registerFunc: func(_ context.Context, input auth.RegisterInput) (*auth.TokenPair, error) {
			if input.FullName == "" {
				t.Error("expected non-empty full name")
			}
			if input.PhoneNumber == "" {
				t.Error("expected non-empty phone number")
			}
			return &auth.TokenPair{
				AccessToken:  "access-token-123",
				RefreshToken: "refresh-token-456",
				User: auth.UserView{
					ID:                userID,
					FullName:          input.FullName,
					PhoneNumber:       input.PhoneNumber,
					District:          input.District,
					PreferredLanguage: input.PreferredLanguage,
					Role:              "farmer",
				},
			}, nil
		},
	}

	handler := handlers.NewAuthHandler(svc, false, "", "lax")

	r := gin.New()
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		c.Set("user_id", uuid.Nil.String())
		handler.Register(c)
	})

	formData := "full_name=John+Farmer&phone_number=%2B23276123456&district=bo&preferred_language=english&password=securepass123"
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]interface{}
	decodeJSON(t, resp.Body, &body)

	if body["access_token"] != "access-token-123" {
		t.Errorf("expected access token, got %v", body["access_token"])
	}
	if body["refresh_token"] != "refresh-token-456" {
		t.Errorf("expected refresh token, got %v", body["refresh_token"])
	}
}

func TestAuthHandler_Register_Duplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAuthService{
		registerFunc: func(_ context.Context, input auth.RegisterInput) (*auth.TokenPair, error) {
			return nil, fmt.Errorf("phone number already registered")
		},
	}

	handler := handlers.NewAuthHandler(svc, false, "", "lax")

	r := gin.New()
	r.POST("/api/v1/auth/register", func(c *gin.Context) {
		c.Set("user_id", uuid.Nil.String())
		handler.Register(c)
	})

	formData := "full_name=John+Farmer&phone_number=%2B23276123456&password=securepass123"
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if !containsStr(body["error"], "already registered") {
		t.Errorf("expected 'already registered' error, got %q", body["error"])
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	svc := &mockAuthService{
		loginFunc: func(_ context.Context, input auth.LoginInput) (*auth.TokenPair, error) {
			if input.PhoneNumber == "" {
				t.Error("expected non-empty phone number")
			}
			return &auth.TokenPair{
				AccessToken:  "access-token-789",
				RefreshToken: "refresh-token-000",
				User: auth.UserView{
					ID:                userID,
					FullName:          "John Farmer",
					PhoneNumber:       "+23276123456",
					Role:              "farmer",
				},
			}, nil
		},
	}

	handler := handlers.NewAuthHandler(svc, false, "", "lax")

	r := gin.New()
	r.POST("/api/v1/auth/login", handler.Login)

	formData := "phone_number=%2B23276123456&password=securepass123"
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]interface{}
	decodeJSON(t, resp.Body, &body)

	if body["access_token"] != "access-token-789" {
		t.Errorf("expected access token, got %v", body["access_token"])
	}
}

func TestAuthHandler_Login_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockAuthService{
		loginFunc: func(_ context.Context, input auth.LoginInput) (*auth.TokenPair, error) {
			return nil, fmt.Errorf("invalid phone number or password")
		},
	}

	handler := handlers.NewAuthHandler(svc, false, "", "lax")

	r := gin.New()
	r.POST("/api/v1/auth/login", handler.Login)

	formData := "phone_number=%2B23276123456&password=wrongpass"
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	decodeJSON(t, resp.Body, &body)
	if !containsStr(body["error"], "invalid") {
		t.Errorf("expected 'invalid' error, got %q", body["error"])
	}
}

// ── Supabase Storage Tests ────────────────────────────────────────────────────

func TestSupabaseStorage_Upload_Non2xx(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{
			fn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid request"}`)),
					Header:     make(http.Header),
				}, nil
			},
		},
	}
	store := storage.NewSupabaseStorageWithClient("https://test.supabase.co", "test-key", "test-bucket", client)

	_, err := store.Save(context.Background(), storage.SaveObjectInput{
		Content:     strings.NewReader("test data"),
		ContentType: "text/plain",
		Path:        "test/object.txt",
	})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
	if !containsStr(err.Error(), "HTTP 400") {
		t.Errorf("expected HTTP 400 error, got: %v", err)
	}
}

func TestSupabaseStorage_Delete_Non2xx(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{
			fn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
					Header:     make(http.Header),
				}, nil
			},
		},
	}
	store := storage.NewSupabaseStorageWithClient("https://test.supabase.co", "test-key", "test-bucket", client)

	err := store.Delete(context.Background(), "test/object.txt")
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
	if !containsStr(err.Error(), "HTTP 404") {
		t.Errorf("expected HTTP 404 error, got: %v", err)
	}
}

func TestSupabaseStorage_SignedURL_Non2xx(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{
			fn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"server error"}`)),
					Header:     make(http.Header),
				}, nil
			},
		},
	}
	store := storage.NewSupabaseStorageWithClient("https://test.supabase.co", "test-key", "test-bucket", client)

	_, err := store.SignedURL(context.Background(), "test/object.txt", 5*time.Minute)
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
	if !containsStr(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 error, got: %v", err)
	}
}

func TestSupabaseStorage_EmptyObjectPath(t *testing.T) {
	client := &http.Client{}
	store := storage.NewSupabaseStorageWithClient("https://test.supabase.co", "test-key", "test-bucket", client)

	t.Run("Save empty path", func(t *testing.T) {
		_, err := store.Save(context.Background(), storage.SaveObjectInput{
			Content:     strings.NewReader("test"),
			ContentType: "text/plain",
			Path:        "",
		})
		if err == nil {
			t.Fatal("expected error for empty path")
		}
		if !containsStr(err.Error(), "object path is required") {
			t.Errorf("expected 'object path is required', got: %v", err)
		}
	})

	t.Run("Delete empty path", func(t *testing.T) {
		err := store.Delete(context.Background(), "")
		if err == nil {
			t.Fatal("expected error for empty path")
		}
		if !containsStr(err.Error(), "object path is required") {
			t.Errorf("expected 'object path is required', got: %v", err)
		}
	})

	t.Run("SignedURL empty path", func(t *testing.T) {
		_, err := store.SignedURL(context.Background(), "", 5*time.Minute)
		if err == nil {
			t.Fatal("expected error for empty path")
		}
		if !containsStr(err.Error(), "object path is required") {
			t.Errorf("expected 'object path is required', got: %v", err)
		}
	})
}

func TestSupabaseStorage_ContextCancellation(t *testing.T) {
	client := &http.Client{
		Transport: &mockRoundTripper{
			fn: func(req *http.Request) (*http.Response, error) {
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				default:
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
						Header:     make(http.Header),
					}, nil
				}
			},
		},
	}
	store := storage.NewSupabaseStorageWithClient("https://test.supabase.co", "test-key", "test-bucket", client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Save(ctx, storage.SaveObjectInput{
		Content:     strings.NewReader("test"),
		ContentType: "text/plain",
		Path:        "test/object.txt",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !containsStr(err.Error(), "context canceled") {
		t.Errorf("expected context cancellation error, got: %v", err)
	}
}
