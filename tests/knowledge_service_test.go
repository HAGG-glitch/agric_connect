package tests

import (
	"context"
	"testing"

	"github.com/agriconnect-ai/internal/models"
	"github.com/agriconnect-ai/internal/services"
)

type mockKnowledgeRepo struct {
	docs []models.AgriculturalDocument
}

func (m *mockKnowledgeRepo) Search(_ context.Context, crop, category string, limit int) ([]models.AgriculturalDocument, error) {
	var result []models.AgriculturalDocument
	for _, d := range m.docs {
		if crop != "" && d.Crop != crop {
			continue
		}
		if category != "" && d.Category != category {
			continue
		}
		result = append(result, d)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *mockKnowledgeRepo) SeedDocuments(_ context.Context, _ []models.AgriculturalDocument) error {
	return nil
}

func TestKnowledgeContextRetrieval(t *testing.T) {
	repo := &mockKnowledgeRepo{
		docs: []models.AgriculturalDocument{
			{Crop: "cassava", Category: "disease", Title: "Cassava Mosaic", Content: "Cassava mosaic disease is caused by a virus.", Source: "Test Source"},
			{Crop: "cassava", Category: "planting", Title: "Cassava Planting", Content: "Plant cassava at the start of rains."},
			{Crop: "rice", Category: "disease", Title: "Rice Blast", Content: "Rice blast affects leaves."},
			{Crop: "maize", Category: "disease", Title: "Maize Streak", Content: "Maize streak virus is transmitted by leafhoppers."},
		},
	}
	svc := services.NewKnowledgeService(repo)

	// Cassava disease question
	ctx, sources, err := svc.RetrieveContext(context.Background(), "My cassava leaves are turning yellow and have spots", "")
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}
	if ctx == "" {
		t.Error("expected non-empty context for cassava disease question")
	}
	if len(sources) == 0 {
		t.Log("no sources returned")
	}
	if !containsStr(ctx, "Cassava Mosaic") {
		t.Error("expected cassava disease document in context")
	}

	// Generic disease question (no crop)
	ctx, _, err = svc.RetrieveContext(context.Background(), "My plant leaves have brown spots", "")
	if err != nil {
		t.Fatalf("RetrieveContext failed: %v", err)
	}
	if ctx == "" {
		t.Error("expected non-empty context for disease question")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
