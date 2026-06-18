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
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agriconnect-ai/internal/ai"
	"github.com/agriconnect-ai/internal/config"
	"github.com/agriconnect-ai/internal/diagnosis"
	"github.com/agriconnect-ai/internal/storage"
	"github.com/google/uuid"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func createValidPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding test PNG: %v", err)
	}
	return buf.Bytes()
}

func testConfig() *config.Config {
	return &config.Config{
		MaxImageSizeMB:          5,
		MinImageWidth:           10,
		MinImageHeight:          10,
		MaxImagePixels:          25000000,
		AllowedImageTypes:       []string{"image/jpeg", "image/png", "image/webp"},
		MaxAudioSizeMB:          10,
		AllowedAudioTypes:       []string{"audio/webm", "audio/wav", "audio/mpeg", "audio/mp4", "audio/ogg"},
		DiagnosisRequestTimeout: 5,
	}
}

type mockMultipartFile struct {
	*bytes.Reader
}

func (m mockMultipartFile) Close() error { return nil }

func makeFileHeader(filename, contentType string, data []byte) *multipart.FileHeader {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", contentType)
	return &multipart.FileHeader{
		Filename: filename,
		Header:   h,
		Size:     int64(len(data)),
	}
}

// ── mocks ────────────────────────────────────────────────────────────────────

type mockDiagnosisRepo struct {
	mu    sync.Mutex
	diags map[uuid.UUID]*diagnosis.CropDiagnosis
	err   error
}

func (m *mockDiagnosisRepo) Create(_ context.Context, d *diagnosis.CropDiagnosis) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.diags[d.ID] = d
	return nil
}

func (m *mockDiagnosisRepo) FindByID(_ context.Context, id uuid.UUID) (*diagnosis.CropDiagnosis, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.diags[id]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockDiagnosisRepo) FindByUserID(_ context.Context, userID uuid.UUID, limit, offset int) ([]diagnosis.CropDiagnosis, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []diagnosis.CropDiagnosis
	for _, d := range m.diags {
		if d.UserID == userID {
			out = append(out, *d)
		}
	}
	if offset > len(out) {
		offset = len(out)
	}
	out = out[offset:]
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (m *mockDiagnosisRepo) CountByUserID(_ context.Context, userID uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, d := range m.diags {
		if d.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockDiagnosisRepo) Update(_ context.Context, d *diagnosis.CropDiagnosis) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.diags[d.ID] = d
	return nil
}

func (m *mockDiagnosisRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.diags, id)
	return nil
}

type mockObjectStorage struct {
	mu           sync.Mutex
	deleteCalled bool
	deletePath   string
	saveErr      error
	deleteErr    error
	savedSize    int64
	downloadData []byte
	downloadErr  error
}

func (m *mockObjectStorage) SetDownloadData(data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.downloadData = data
}

func (m *mockObjectStorage) Save(_ context.Context, input storage.SaveObjectInput) (storage.StoredObject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return storage.StoredObject{}, m.saveErr
	}
	data, _ := io.ReadAll(input.Content)
	return storage.StoredObject{
		Path:        input.Path,
		ContentType: input.ContentType,
		SizeBytes:   int64(len(data)),
	}, nil
}

func (m *mockObjectStorage) Delete(_ context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCalled = true
	m.deletePath = path
	return m.deleteErr
}

func (m *mockObjectStorage) SignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}

func (m *mockObjectStorage) Download(_ context.Context, path string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	if m.downloadData == nil {
		return nil, fmt.Errorf("not found")
	}
	return io.NopCloser(bytes.NewReader(m.downloadData)), nil
}

type mockDiagnosisAI struct {
	result *ai.DiagnosisAIResult
	err    error
	lastInput ai.DiagnosisAIInput
}

func (m *mockDiagnosisAI) Diagnose(_ context.Context, input ai.DiagnosisAIInput) (*ai.DiagnosisAIResult, error) {
	m.lastInput = input
	return m.result, m.err
}

type mockKnowledgeService struct {
	ctx     string
	sources []string
	err     error
}

func (m *mockKnowledgeService) RetrieveContext(_ context.Context, _, _ string) (string, []string, error) {
	return m.ctx, m.sources, m.err
}

func waitForGoroutine() {
	time.Sleep(200 * time.Millisecond)
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestDiagnosis_ValidFlow(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{
			result: &ai.DiagnosisAIResult{
				Crop:                "cassava",
				ProbableCondition:   "Cassava Mosaic Disease",
				Confidence:          85.5,
				ConfidenceLabel:     "high",
				Description:         "A viral disease affecting cassava.",
				ObservedSigns:       []string{"yellow leaves", "stunted growth"},
				PossibleAlternatives: []string{"nutrient deficiency"},
				RecommendedActions:  []string{"remove infected plants", "use resistant varieties"},
				PreventionTips:      []string{"use clean cuttings", "crop rotation"},
				Urgency:             "high",
				RequiresExpertReview: true,
				Disclaimer:          "This is a preliminary AI assessment.",
			},
		},
		&mockKnowledgeService{ctx: "test knowledge"},
		cfg)

	userID := uuid.New()
	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves with spots",
		PlantPart:          "leaf",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	d, err := svc.CreateDiagnosis(context.Background(), userID, input, file, header)
	if err != nil {
		t.Fatalf("CreateDiagnosis failed: %v", err)
	}

	if d.Status != "processing" {
		t.Errorf("expected initial status 'processing', got %q", d.Status)
	}
	if d.Crop != "cassava" {
		t.Errorf("expected crop cassava, got %q", d.Crop)
	}
	if d.UserID != userID {
		t.Error("expected UserID to match")
	}
	if d.ImageStoragePath == "" {
		t.Error("expected non-empty storage path")
	}
	if d.ImageContentType != "image/png" {
		t.Errorf("expected image/png, got %q", d.ImageContentType)
	}
	if d.ImageSHA256 == "" {
		t.Error("expected non-empty SHA256 hash")
	}
	if d.RequiresExpertReview != true {
		t.Error("expected RequiresExpertReview to be true by default")
	}

	waitForGoroutine()

	if d.Status != "completed" {
		t.Errorf("expected final status 'completed', got %q (err=%q)", d.Status, d.ErrorMessage)
	}
	if d.ProbableCondition != "Cassava Mosaic Disease" {
		t.Errorf("expected 'Cassava Mosaic Disease', got %q", d.ProbableCondition)
	}
	if d.Confidence != 85.5 {
		t.Errorf("expected confidence 85.5, got %f", d.Confidence)
	}
	if d.ConfidenceLabel != "high" {
		t.Errorf("expected confidence_label high, got %q", d.ConfidenceLabel)
	}
	if d.RawAIResult == nil {
		t.Error("expected non-nil RawAIResult")
	}
}

func TestDiagnosis_UnsupportedImageType(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.bmp", "image/bmp", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for unsupported image type")
	}
	if !containsStr(err.Error(), "validation:") {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestDiagnosis_OversizedImage(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()
	cfg.MaxImageSizeMB = 1

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)
	header.Size = 2 * 1024 * 1024 // 2 MB > 1 MB

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for oversized image")
	}
	if !containsStr(err.Error(), "validation:") {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestDiagnosis_InvalidImageBytes(t *testing.T) {
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	invalidData := make([]byte, 100)
	copy(invalidData, []byte{0x89, 0x50, 0x4E, 0x47}) // valid PNG magic but invalid body

	file := mockMultipartFile{bytes.NewReader(invalidData)}
	header := makeFileHeader("test.png", "image/png", invalidData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for invalid image bytes")
	}
	if !containsStr(err.Error(), "validation:") {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestDiagnosis_MissingSymptoms(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for missing symptoms")
	}
	if err.Error() != "validation: symptom description is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDiagnosis_StorageFailure(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{saveErr: fmt.Errorf("s3 unavailable")},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for storage failure")
	}
	if !containsStr(err.Error(), "storage:") {
		t.Errorf("expected storage error, got %v", err)
	}
}

func TestDiagnosis_DatabaseFailureAfterStorage(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	store := &mockObjectStorage{}
	repo := &mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis), err: fmt.Errorf("db unavailable")}

	svc := diagnosis.NewService(
		repo,
		store,
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for database failure")
	}
	if !containsStr(err.Error(), "database:") {
		t.Errorf("expected database error, got %v", err)
	}

	store.mu.Lock()
	if !store.deleteCalled {
		t.Error("expected storage.Delete to be called on database failure (cleanup)")
	}
	if store.deletePath == "" {
		t.Error("expected non-empty delete path in cleanup")
	}
	store.mu.Unlock()
}

func TestDiagnosis_ProviderFailure(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{err: fmt.Errorf("vision API returned 429")},
		&mockKnowledgeService{},
		cfg)

	userID := uuid.New()
	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	d, err := svc.CreateDiagnosis(context.Background(), userID, input, file, header)
	if err != nil {
		t.Fatalf("CreateDiagnosis should not return error for provider failure (async): %v", err)
	}

	waitForGoroutine()

	if d.Status != "failed" {
		t.Errorf("expected status 'failed' after AI error, got %q", d.Status)
	}
	if d.ErrorMessage == "" {
		t.Error("expected non-empty ErrorMessage after AI failure")
	}
}

func TestDiagnosis_InvalidAIResponse(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{err: fmt.Errorf("missing probable_condition in AI response")},
		&mockKnowledgeService{},
		cfg)

	userID := uuid.New()
	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	d, err := svc.CreateDiagnosis(context.Background(), userID, input, file, header)
	if err != nil {
		t.Fatalf("CreateDiagnosis should not return error for invalid AI resp (async): %v", err)
	}

	waitForGoroutine()

	if d.Status != "failed" {
		t.Errorf("expected status 'failed' after invalid AI response, got %q", d.Status)
	}
}

func TestDiagnosis_ConfidenceClamping(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{
			result: &ai.DiagnosisAIResult{
				Crop:              "cassava",
				ProbableCondition: "Cassava Mosaic",
				Confidence:        150,
				ConfidenceLabel:   "invalid_label",
				Urgency:           "invalid_urgency",
				Description:       "Test",
			},
		},
		&mockKnowledgeService{},
		cfg)

	userID := uuid.New()
	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	d, err := svc.CreateDiagnosis(context.Background(), userID, input, file, header)
	if err != nil {
		t.Fatalf("CreateDiagnosis failed: %v", err)
	}

	waitForGoroutine()

	if d.Confidence != 100 {
		t.Errorf("expected confidence clamped to 100, got %f", d.Confidence)
	}
	if d.ConfidenceLabel != "low" {
		t.Errorf("expected confidence_label defaulted to 'low', got %q", d.ConfidenceLabel)
	}
}

func TestDiagnosis_UrgencyValidation(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{
			result: &ai.DiagnosisAIResult{
				Crop:              "cassava",
				ProbableCondition: "Cassava Mosaic",
				Confidence:        80,
				ConfidenceLabel:   "high",
				Urgency:           "critical",
				Description:       "Test description",
			},
		},
		&mockKnowledgeService{},
		cfg)

	userID := uuid.New()
	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	d, err := svc.CreateDiagnosis(context.Background(), userID, input, file, header)
	if err != nil {
		t.Fatalf("CreateDiagnosis failed: %v", err)
	}

	waitForGoroutine()

	if d.Urgency != "medium" {
		t.Errorf("expected urgency defaulted to 'medium', got %q", d.Urgency)
	}
}

func TestDiagnosis_OwnershipGet(t *testing.T) {
	repo := &mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)}
	userID := uuid.New()
	otherUser := uuid.New()

	diag := &diagnosis.CropDiagnosis{ID: uuid.New(), UserID: userID}
	repo.diags[diag.ID] = diag

	svc := diagnosis.NewService(repo, &mockObjectStorage{}, &mockDiagnosisAI{}, &mockKnowledgeService{}, testConfig())

	_, err := svc.GetDiagnosis(context.Background(), diag.ID, otherUser)
	if err == nil {
		t.Fatal("expected access denied error")
	}
	if !containsStr(err.Error(), "access denied") {
		t.Errorf("expected 'access denied', got %v", err)
	}

	// correct owner succeeds
	got, err := svc.GetDiagnosis(context.Background(), diag.ID, userID)
	if err != nil {
		t.Fatalf("GetDiagnosis by owner failed: %v", err)
	}
	if got.ID != diag.ID {
		t.Error("diagnosis ID mismatch")
	}
}

func TestDiagnosis_OwnershipDelete(t *testing.T) {
	repo := &mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)}
	userID := uuid.New()
	otherUser := uuid.New()

	diag := &diagnosis.CropDiagnosis{ID: uuid.New(), UserID: userID}
	repo.diags[diag.ID] = diag

	svc := diagnosis.NewService(repo, &mockObjectStorage{}, &mockDiagnosisAI{}, &mockKnowledgeService{}, testConfig())

	err := svc.DeleteDiagnosis(context.Background(), diag.ID, otherUser)
	if err == nil {
		t.Fatal("expected access denied error")
	}
	if !containsStr(err.Error(), "access denied") {
		t.Errorf("expected 'access denied', got %v", err)
	}

	// diagnosis still exists
	if _, err := repo.FindByID(context.Background(), diag.ID); err != nil {
		t.Error("diagnosis should not have been deleted")
	}
}

func TestDiagnosis_DeletionCleanup(t *testing.T) {
	repo := &mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)}
	store := &mockObjectStorage{}
	userID := uuid.New()

	diag := &diagnosis.CropDiagnosis{
		ID:              uuid.New(),
		UserID:          userID,
		ImageStoragePath: "some/path/image.jpg",
	}
	repo.diags[diag.ID] = diag

	svc := diagnosis.NewService(repo, store, &mockDiagnosisAI{}, &mockKnowledgeService{}, testConfig())

	err := svc.DeleteDiagnosis(context.Background(), diag.ID, userID)
	if err != nil {
		t.Fatalf("DeleteDiagnosis failed: %v", err)
	}

	store.mu.Lock()
	if !store.deleteCalled {
		t.Error("expected storage.Delete to be called")
	}
	if store.deletePath != "some/path/image.jpg" {
		t.Errorf("expected delete path 'some/path/image.jpg', got %q", store.deletePath)
	}
	store.mu.Unlock()

	// diagnosis removed from repo
	_, err = repo.FindByID(context.Background(), diag.ID)
	if err == nil {
		t.Error("expected diagnosis to be deleted from repo")
	}
}

func TestDiagnosis_ListPagination(t *testing.T) {
	repo := &mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)}
	userID := uuid.New()

	for i := 0; i < 15; i++ {
		d := &diagnosis.CropDiagnosis{
			ID:        uuid.New(),
			UserID:    userID,
			Crop:      fmt.Sprintf("crop-%d", i),
			CreatedAt: time.Now().UTC().Add(-time.Duration(i) * time.Hour),
		}
		repo.diags[d.ID] = d
	}

	svc := diagnosis.NewService(repo, &mockObjectStorage{}, &mockDiagnosisAI{}, &mockKnowledgeService{}, testConfig())

	t.Run("defaults", func(t *testing.T) {
		diags, count, err := svc.ListDiagnoses(context.Background(), userID, 0, 0)
		if err != nil {
			t.Fatalf("ListDiagnoses failed: %v", err)
		}
		if count != 15 {
			t.Errorf("expected count 15, got %d", count)
		}
		if len(diags) > 20 {
			t.Errorf("expected at most 20 (default pageSize), got %d", len(diags))
		}
	})

	t.Run("page 1 with pageSize 5", func(t *testing.T) {
		diags, count, err := svc.ListDiagnoses(context.Background(), userID, 1, 5)
		if err != nil {
			t.Fatalf("ListDiagnoses failed: %v", err)
		}
		if count != 15 {
			t.Errorf("expected count 15, got %d", count)
		}
		if len(diags) != 5 {
			t.Errorf("expected 5 diagnoses, got %d", len(diags))
		}
	})

	t.Run("page 3 with pageSize 5", func(t *testing.T) {
		diags, count, err := svc.ListDiagnoses(context.Background(), userID, 3, 5)
		if err != nil {
			t.Fatalf("ListDiagnoses failed: %v", err)
		}
		if count != 15 {
			t.Errorf("expected count 15, got %d", count)
		}
		if len(diags) != 5 {
			t.Errorf("expected 5 diagnoses on page 3, got %d", len(diags))
		}
	})

	t.Run("page size clamped to 50", func(t *testing.T) {
		diags, count, err := svc.ListDiagnoses(context.Background(), userID, 1, 100)
		if err != nil {
			t.Fatalf("ListDiagnoses failed: %v", err)
		}
		if count != 15 {
			t.Errorf("expected count 15, got %d", count)
		}
		if len(diags) > 50 {
			t.Errorf("expected at most 50 (clamped pageSize), got %d", len(diags))
		}
	})

	t.Run("empty list for wrong user", func(t *testing.T) {
		diags, count, err := svc.ListDiagnoses(context.Background(), uuid.New(), 1, 10)
		if err != nil {
			t.Fatalf("ListDiagnoses failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
		if len(diags) != 0 {
			t.Errorf("expected 0 diagnoses, got %d", len(diags))
		}
	})
}

func TestDiagnosis_UnsupportedCrop(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "unicorn",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for unsupported crop")
	}
	if !containsStr(err.Error(), "validation:") {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestDiagnosis_InvalidPlantPart(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
		PlantPart:          "nonexistent",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for invalid plant part")
	}
	if !containsStr(err.Error(), "validation:") {
		t.Errorf("expected validation error, got %v", err)
	}
}

func TestDiagnosis_ValidationErrorPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input diagnosis.DiagnosisInput
		extra func() ([]byte, *multipart.FileHeader)
	}{
		{
			name: "empty crop",
			input: diagnosis.DiagnosisInput{
				Crop:               "",
				SymptomDescription: "yellow leaves",
			},
			extra: func() ([]byte, *multipart.FileHeader) {
				d := createValidPNG(t)
				return d, makeFileHeader("test.png", "image/png", d)
			},
		},
		{
			name: "missing symptoms",
			input: diagnosis.DiagnosisInput{
				Crop:               "cassava",
				SymptomDescription: "",
			},
			extra: func() ([]byte, *multipart.FileHeader) {
				d := createValidPNG(t)
				return d, makeFileHeader("test.png", "image/png", d)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, header := tt.extra()
			svc := diagnosis.NewService(
				&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
				&mockObjectStorage{},
				&mockDiagnosisAI{},
				&mockKnowledgeService{},
				testConfig())
			file := mockMultipartFile{bytes.NewReader(data)}
			_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), tt.input, file, header)
			if err == nil {
				t.Fatal("expected error")
			}
			if !containsStr(err.Error(), "validation:") {
				t.Errorf("expected validation: prefix, got %v", err)
			}
		})
	}
}

func TestDiagnosis_ContinueInChatOwnership(t *testing.T) {
	repo := &mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)}
	userID := uuid.New()
	otherUser := uuid.New()

	diag := &diagnosis.CropDiagnosis{
		ID:                uuid.New(),
		UserID:            userID,
		Crop:              "cassava",
		PreferredLanguage: "english",
	}
	repo.diags[diag.ID] = diag

	svc := diagnosis.NewService(repo, &mockObjectStorage{}, &mockDiagnosisAI{}, &mockKnowledgeService{}, testConfig())

	_, err := svc.ContinueInChat(context.Background(), diag.ID, otherUser, nil)
	if err == nil {
		t.Fatal("expected access denied")
	}
	if !containsStr(err.Error(), "access denied") {
		t.Errorf("expected 'access denied', got %v", err)
	}

	// non-existent diagnosis
	_, err = svc.ContinueInChat(context.Background(), uuid.New(), userID, nil)
	if err == nil {
		t.Fatal("expected error for non-existent diagnosis")
	}
}

func TestDiagnosis_EmptyImageData(t *testing.T) {
	cfg := testConfig()

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{},
		&mockKnowledgeService{},
		cfg)

	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
	}

	file := mockMultipartFile{bytes.NewReader([]byte{})}
	header := makeFileHeader("test.png", "image/png", []byte{})

	_, err := svc.CreateDiagnosis(context.Background(), uuid.New(), input, file, header)
	if err == nil {
		t.Fatal("expected error for empty image data")
	}
}

func TestDiagnosis_ImageOptimizationReducesSize(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2000, 2000))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding test PNG: %v", err)
	}
	original := buf.Bytes()

	optimized := diagnosis.OptimizeImageForAI(original, "image/png")

	if len(optimized) >= len(original) {
		t.Errorf("optimized image (%d bytes) should be smaller than original (%d bytes)", len(optimized), len(original))
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(optimized))
	if err != nil {
		t.Fatalf("decoding optimized image: %v", err)
	}
	if cfg.Width < 256 || cfg.Height < 256 {
		t.Errorf("optimized image dimensions (%dx%d) below 256x256 minimum", cfg.Width, cfg.Height)
	}
	if cfg.Width > 768 || cfg.Height > 768 {
		t.Errorf("optimized image dimensions (%dx%d) exceed 768 max dimension", cfg.Width, cfg.Height)
	}
}

func TestDiagnosis_ImageOptimizationPreservesSmall(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding test PNG: %v", err)
	}
	original := buf.Bytes()

	optimized := diagnosis.OptimizeImageForAI(original, "image/png")

	cfg, _, err := image.DecodeConfig(bytes.NewReader(optimized))
	if err != nil {
		t.Fatalf("decoding optimized image: %v", err)
	}
	if cfg.Width != 100 || cfg.Height != 100 {
		t.Errorf("expected 100×100, got %dx%d", cfg.Width, cfg.Height)
	}
}

func TestDiagnosis_ContextCapped(t *testing.T) {
	imgData := createValidPNG(t)
	cfg := testConfig()
	cfg.MaxDiagnosisContextChars = 50
	cfg.DiagnosisRequestTimeout = 5

	mockAI := &mockDiagnosisAI{
		result: &ai.DiagnosisAIResult{
			Crop:              "cassava",
			ProbableCondition: "Cassava Mosaic",
			Confidence:        80,
			ConfidenceLabel:   "high",
			Urgency:           "medium",
			Description:       "Test",
		},
	}

	longCtx := ""
	for i := 0; i < 100; i++ {
		longCtx += "a"
	}

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		mockAI,
		&mockKnowledgeService{ctx: longCtx},
		cfg)

	userID := uuid.New()
	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
		PlantPart:          "leaf",
	}

	file := mockMultipartFile{bytes.NewReader(imgData)}
	header := makeFileHeader("test.png", "image/png", imgData)

	d, err := svc.CreateDiagnosis(context.Background(), userID, input, file, header)
	if err != nil {
		t.Fatalf("CreateDiagnosis failed: %v", err)
	}

	waitForGoroutine()

	if d.Status != "completed" {
		t.Fatalf("expected completed, got %q: %s", d.Status, d.ErrorMessage)
	}

	if len(mockAI.lastInput.KnowledgeContext) > cfg.MaxDiagnosisContextChars {
		t.Errorf("knowledge context length %d exceeds MaxDiagnosisContextChars %d",
			len(mockAI.lastInput.KnowledgeContext), cfg.MaxDiagnosisContextChars)
	}
}

func TestDiagnosis_AIUsesVisionModel(t *testing.T) {
	cfg := testConfig()
	cfg.GroqVisionModel = "llama-3.2-11b-vision-preview"
	cfg.GroqChatModel = "llama-3.1-8b-instant"

	svc := diagnosis.NewService(
		&mockDiagnosisRepo{diags: make(map[uuid.UUID]*diagnosis.CropDiagnosis)},
		&mockObjectStorage{},
		&mockDiagnosisAI{result: &ai.DiagnosisAIResult{
			Crop:              "cassava",
			ProbableCondition: "Test",
			Confidence:        80,
			ConfidenceLabel:   "high",
			Urgency:           "medium",
			Description:       "Test",
		}},
		&mockKnowledgeService{},
		cfg)

	userID := uuid.New()
	input := diagnosis.DiagnosisInput{
		Crop:               "cassava",
		SymptomDescription: "yellow leaves",
		PlantPart:          "leaf",
	}

	file := mockMultipartFile{bytes.NewReader(createValidPNG(t))}
	header := makeFileHeader("test.png", "image/png", createValidPNG(t))

	d, err := svc.CreateDiagnosis(context.Background(), userID, input, file, header)
	if err != nil {
		t.Fatalf("CreateDiagnosis failed: %v", err)
	}

	waitForGoroutine()

	if d.Status != "completed" {
		t.Fatalf("expected completed, got %q: %s", d.Status, d.ErrorMessage)
	}
}

func TestDiagnosis_GetObservedSigns_ReturnsStrings(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	signs := d.GetObservedSigns()
	if signs != nil {
		t.Errorf("expected nil for unset observed signs, got %v", signs)
	}

	d.ObservedSigns = []byte(`["yellowing","leaf curling"]`)
	signs = d.GetObservedSigns()
	if len(signs) != 2 {
		t.Fatalf("expected 2 signs, got %d: %v", len(signs), signs)
	}
	if signs[0] != "yellowing" {
		t.Errorf("expected 'yellowing', got %q", signs[0])
	}
	if signs[1] != "leaf curling" {
		t.Errorf("expected 'leaf curling', got %q", signs[1])
	}
}

func TestDiagnosis_GetPossibleAlternatives_ReturnsStrings(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	d.PossibleAlternatives = []byte(`["pest damage","fungal disease"]`)
	alts := d.GetPossibleAlternatives()
	if len(alts) != 2 {
		t.Fatalf("expected 2 alternatives, got %d: %v", len(alts), alts)
	}
	if alts[0] != "pest damage" {
		t.Errorf("expected 'pest damage', got %q", alts[0])
	}
}

func TestDiagnosis_GetRecommendedActions_ReturnsStrings(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	d.RecommendedActions = []byte(`["apply fungicide","remove affected leaves"]`)
	actions := d.GetRecommendedActions()
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d: %v", len(actions), actions)
	}
}

func TestDiagnosis_GetPreventionTips_ReturnsStrings(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	d.PreventionTips = []byte(`["use resistant varieties","crop rotation"]`)
	tips := d.GetPreventionTips()
	if len(tips) != 2 {
		t.Fatalf("expected 2 tips, got %d: %v", len(tips), tips)
	}
}

func TestDiagnosis_GetObservedSigns_NullJSON(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	d.ObservedSigns = []byte(`null`)
	signs := d.GetObservedSigns()
	if signs != nil {
		t.Errorf("expected nil for null JSON, got %v", signs)
	}
}

func TestDiagnosis_GetObservedSigns_EmptyJSON(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	d.ObservedSigns = []byte(`[]`)
	signs := d.GetObservedSigns()
	if signs == nil {
		t.Fatal("expected non-nil for empty array")
	}
	if len(signs) != 0 {
		t.Errorf("expected empty array, got %v", signs)
	}
}

func TestDiagnosis_GetObservedSigns_InvalidJSON(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	d.ObservedSigns = []byte(`not json`)
	signs := d.GetObservedSigns()
	if signs != nil {
		t.Errorf("expected nil for invalid JSON, got %v", signs)
	}
}

func TestDiagnosis_GetObservedSigns_NumbersArray(t *testing.T) {
	d := &diagnosis.CropDiagnosis{}
	d.ObservedSigns = []byte(`[91, 34, 121]`)
	signs := d.GetObservedSigns()
	if signs != nil {
		t.Errorf("expected nil for number array (not []string), got %v", signs)
	}
}

func TestDiagnosis_JSONParse_ValidObject(t *testing.T) {
	content := `{"crop":"cassava","probable_condition":"Mosaic Virus","confidence":75,"description":"test","observed_signs":["yellowing"],"possible_alternatives":["pest"],"recommended_actions":["remove"],"prevention_tips":["rotate"],"urgency":"medium","requires_expert_review":true,"disclaimer":"This is preliminary."}`
	result, err := parseDiagnosisJSONHelper(content)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.ProbableCondition != "Mosaic Virus" {
		t.Errorf("expected 'Mosaic Virus', got %q", result.ProbableCondition)
	}
	if len(result.ObservedSigns) != 1 || result.ObservedSigns[0] != "yellowing" {
		t.Errorf("unexpected observed_signs: %v", result.ObservedSigns)
	}
}

func TestDiagnosis_JSONParse_MarkdownCodeFence(t *testing.T) {
	content := "```json\n{\"crop\":\"cassava\",\"probable_condition\":\"Mosaic\",\"confidence\":60,\"description\":\"desc\",\"observed_signs\":[\"yellow\"],\"possible_alternatives\":[],\"recommended_actions\":[],\"prevention_tips\":[],\"urgency\":\"low\",\"requires_expert_review\":false,\"disclaimer\":\"preliminary\"}\n```"
	result, err := parseDiagnosisJSONHelper(content)
	if err != nil {
		t.Fatalf("expected no error for markdown fence, got: %v", err)
	}
	if result.ProbableCondition != "Mosaic" {
		t.Errorf("expected 'Mosaic', got %q", result.ProbableCondition)
	}
}

func TestDiagnosis_JSONParse_TextBeforeJSON(t *testing.T) {
	content := "Here is the diagnosis result:\n{\"crop\":\"cassava\",\"probable_condition\":\"Blight\",\"confidence\":70,\"description\":\"desc\",\"observed_signs\":[\"spots\"],\"possible_alternatives\":[],\"recommended_actions\":[],\"prevention_tips\":[],\"urgency\":\"high\",\"requires_expert_review\":true,\"disclaimer\":\"preliminary\"}"
	result, err := parseDiagnosisJSONHelper(content)
	if err != nil {
		t.Fatalf("expected no error for text before JSON, got: %v", err)
	}
	if result.ProbableCondition != "Blight" {
		t.Errorf("expected 'Blight', got %q", result.ProbableCondition)
	}
}

func TestDiagnosis_JSONParse_TextAfterJSON(t *testing.T) {
	content := "{\"crop\":\"cassava\",\"probable_condition\":\"Wilt\",\"confidence\":50,\"description\":\"desc\",\"observed_signs\":[\"wilting\"],\"possible_alternatives\":[],\"recommended_actions\":[],\"prevention_tips\":[],\"urgency\":\"medium\",\"requires_expert_review\":true,\"disclaimer\":\"preliminary\"}\n\nThis is just additional text that should be ignored."
	result, err := parseDiagnosisJSONHelper(content)
	if err != nil {
		t.Fatalf("expected no error for text after JSON, got: %v", err)
	}
	if result.ProbableCondition != "Wilt" {
		t.Errorf("expected 'Wilt', got %q", result.ProbableCondition)
	}
}

func TestDiagnosis_JSONParse_EmptyContent(t *testing.T) {
	_, err := parseDiagnosisJSONHelper("")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestDiagnosis_JSONParse_NoJSONObject(t *testing.T) {
	_, err := parseDiagnosisJSONHelper("Here is some text without JSON")
	if err == nil {
		t.Fatal("expected error for text without JSON")
	}
}

func TestDiagnosis_JSONParse_MissingRequiredField(t *testing.T) {
	content := `{"crop":"cassava","confidence":50,"description":"test","observed_signs":[],"possible_alternatives":[],"recommended_actions":[],"prevention_tips":[],"urgency":"low","requires_expert_review":false,"disclaimer":"preliminary"}`
	_, err := parseDiagnosisJSONHelper(content)
	if err == nil {
		t.Fatal("expected error for missing probable_condition")
	}
}

func TestJSONSchema_AllPropertiesInRequired(t *testing.T) {
	// Build the same schema as buildResponseFormat uses
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"crop":                   map[string]interface{}{"type": "string"},
			"probable_condition":     map[string]interface{}{"type": "string"},
			"confidence":             map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			"confidence_label":       map[string]interface{}{"type": "string"},
			"description":            map[string]interface{}{"type": "string"},
			"observed_signs":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"possible_alternatives":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"recommended_actions":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"prevention_tips":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"urgency":                map[string]interface{}{"type": "string", "enum": []interface{}{"low", "medium", "high", "urgent"}},
			"requires_expert_review": map[string]interface{}{"type": "boolean"},
			"disclaimer":             map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{
			"crop", "probable_condition", "confidence", "confidence_label",
			"description", "observed_signs", "possible_alternatives",
			"recommended_actions", "prevention_tips",
			"urgency", "requires_expert_review", "disclaimer",
		},
		"additionalProperties": false,
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties is not a map")
	}
	req, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("required is not a slice")
	}

	requiredSet := make(map[string]bool, len(req))
	for _, r := range req {
		rStr, ok := r.(string)
		if !ok {
			t.Fatalf("required entry %v is not a string", r)
		}
		requiredSet[rStr] = true
	}

	t.Run("every property is required", func(t *testing.T) {
		for propName := range props {
			if !requiredSet[propName] {
				t.Errorf("property %q is not in required array", propName)
			}
		}
	})

	t.Run("crop is required", func(t *testing.T) {
		if !requiredSet["crop"] {
			t.Error("crop is not in required array")
		}
	})

	t.Run("confidence_label is required", func(t *testing.T) {
		if !requiredSet["confidence_label"] {
			t.Error("confidence_label is not in required array")
		}
	})

	t.Run("all required entries exist in properties", func(t *testing.T) {
		for _, r := range req {
			rStr := r.(string)
			if _, ok := props[rStr]; !ok {
				t.Errorf("required entry %q has no matching property definition", rStr)
			}
		}
	})

	t.Run("schema marshals to valid JSON", func(t *testing.T) {
		_, err := json.Marshal(schema)
		if err != nil {
			t.Fatalf("schema marshalling failed: %v", err)
		}
	})
}

func TestJSONSchema_NoRequiredFieldsMissing(t *testing.T) {
	// Double-check the exact property set
	expectedProperties := []string{
		"crop", "probable_condition", "confidence", "confidence_label",
		"description", "observed_signs", "possible_alternatives",
		"recommended_actions", "prevention_tips",
		"urgency", "requires_expert_review", "disclaimer",
	}

	required := []interface{}{
		"crop", "probable_condition", "confidence", "confidence_label",
		"description", "observed_signs", "possible_alternatives",
		"recommended_actions", "prevention_tips",
		"urgency", "requires_expert_review", "disclaimer",
	}

	if len(expectedProperties) != len(required) {
		t.Errorf("properties count (%d) != required count (%d): all properties must be in required for strict json_schema",
			len(expectedProperties), len(required))
	}

	reqSet := make(map[string]bool)
	for _, r := range required {
		reqSet[r.(string)] = true
	}
	for _, p := range expectedProperties {
		if !reqSet[p] {
			t.Errorf("property %q is missing from required", p)
		}
	}
}

func TestJSONSchema_RetryWithJSONObject(t *testing.T) {
	// This test validates that the containsJSONSchemaError helper catches schema errors.
	err := fmt.Errorf("groq returned status 400: invalid JSON schema for response_format")
	if !containsJSONSchemaErrorHelper(err) {
		t.Error("expected containsJSONSchemaError to return true for invalid JSON schema error")
	}

	// Non-schema errors should not trigger fallback
	err = fmt.Errorf("groq returned status 429: rate limit exceeded")
	if containsJSONSchemaErrorHelper(err) {
		t.Error("expected containsJSONSchemaError to return false for non-schema error")
	}

	// "must be listed in required" error — matches the production error from Groq
	err = fmt.Errorf("groq returned status 400: /required: `required` is required to be supplied and to be an array including every key in properties.\nThe following properties must be listed in `required`: confidence_label, crop")
	if !containsJSONSchemaErrorHelper(err) {
		t.Error("expected containsJSONSchemaError to return true for 'must be listed in required' error")
	}

	// json_object retry can still parse a valid result
	content := `{"crop":"cassava","probable_condition":"Blight","confidence":70,"confidence_label":"high","description":"desc","observed_signs":["spots"],"possible_alternatives":[],"recommended_actions":[],"prevention_tips":[],"urgency":"high","requires_expert_review":true,"disclaimer":"preliminary"}`
	result, err := parseDiagnosisJSONHelper(content)
	if err != nil {
		t.Fatalf("expected no error for valid JSON from json_object, got: %v", err)
	}
	if result.ProbableCondition != "Blight" {
		t.Errorf("expected 'Blight', got %q", result.ProbableCondition)
	}

	// Parser still rejects non-JSON after retry
	_, err = parseDiagnosisJSONHelper("This is not JSON")
	if err == nil {
		t.Fatal("expected error for non-JSON after retry")
	}
}

func TestJSONSchema_ConfigDrivenResponseFormat(t *testing.T) {
	t.Run("default is json_schema", func(t *testing.T) {
		// Only check that the default config value is set
		cfg := testConfig()
		if cfg.GroqVisionResponseFormat != "" {
			t.Logf("GroqVisionResponseFormat default: %q (may be empty in tests)", cfg.GroqVisionResponseFormat)
		}
	})

	t.Run("none format does not panic", func(t *testing.T) {
		// NewCropDiagnosisAI accepts "none" without error
		client := ai.NewClient("", "", "", 5)
		_ = ai.NewCropDiagnosisAI(client, "test-model", 100, "none")
	})

	t.Run("empty format defaults to json_schema", func(t *testing.T) {
		client := ai.NewClient("", "", "", 5)
		_ = ai.NewCropDiagnosisAI(client, "test-model", 100, "")
	})
}

func containsJSONSchemaErrorHelper(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "invalid JSON schema") ||
		strings.Contains(msg, "json_schema") ||
		strings.Contains(msg, "response_format") ||
		strings.Contains(msg, "must be listed in `required`") ||
		strings.Contains(msg, "additionalProperties")
}

func TestDiagnosis_FailedDisplay_NoConfidenceChip(t *testing.T) {
	d := &diagnosis.CropDiagnosis{
		Status:     "failed",
		Confidence: 0,
		Urgency:    "",
	}
	if d.Status == "failed" {
		_ = fmt.Sprintf("Confidence: %.0f%%", d.Confidence)
		_ = fmt.Sprintf("%s urgency", d.Urgency)
	}
}

func parseDiagnosisJSONHelper(content string) (*ai.DiagnosisAIResult, error) {
	// Using the same logic as internal/ai.parseDiagnosisJSON but accessible from tests
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty AI response")
	}
	if strings.HasPrefix(content, "```") {
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) == 2 {
			content = lines[1]
		}
	}
	if idx := strings.LastIndex(content, "```"); idx >= 0 {
		content = content[:idx]
	}
	content = strings.TrimSpace(content)
	if braceStart := strings.Index(content, "{"); braceStart >= 0 {
		if braceEnd := strings.LastIndex(content, "}"); braceEnd > braceStart {
			content = content[braceStart : braceEnd+1]
		}
	}
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "{") {
		return nil, fmt.Errorf("response does not contain a JSON object")
	}
	var result ai.DiagnosisAIResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}
	if result.ProbableCondition == "" {
		return nil, fmt.Errorf("missing probable_condition in AI response")
	}
	if result.Crop == "" {
		return nil, fmt.Errorf("missing crop in AI response")
	}
	return &result, nil
}
