package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/agriconnect-ai/internal/models"
	"github.com/agriconnect-ai/internal/repositories"
)

const maxContextChars = 8000

var knownCrops = []string{
	"rice", "cassava", "maize", "groundnut", "yam", "sweet potato",
	"plantain", "banana", "cocoa", "coffee", "palm oil", "ginger",
	"pepper", "tomato", "okra", "eggplant", "cowpea", "sorghum",
}

var categoryKeywords = map[string][]string{
	"disease":    {"disease", "sick", "sik", "yellow", "brown", "rot", "blight", "wilt", "spot", "lesion", "fungus", "virus", "bacteria"},
	"pests":      {"pest", "insect", "bug", "worm", "caterpillar", "aphid", "weevil", "rat", "bird", "bad insect"},
	"planting":   {"plant", "planting", "seed", "sow", "nursery", "germinate", "spacing", "transplant"},
	"soil":       {"soil", "gron", "fertility", "pH", "compost", "organic", "clay", "sandy", "loam"},
	"fertiliser": {"fertiliser", "fertilizer", "npk", "nitrogen", "phosphorus", "potassium", "manure", "compost", "plant food"},
	"harvesting": {"harvest", "pull", "gather", "maturity", "yield", "pick", "cut"},
	"storage":    {"store", "storage", "keep", "preserve", "dry", "silo", "bag", "aflatoxin"},
	"irrigation": {"water", "irrigat", "rain", "drought", "moisture", "ren"},
}

type KnowledgeService interface {
	RetrieveContext(ctx context.Context, question, crop string) (string, []string, error)
}

type knowledgeService struct {
	repo repositories.KnowledgeRepository
}

func NewKnowledgeService(repo repositories.KnowledgeRepository) KnowledgeService {
	return &knowledgeService{repo: repo}
}

func (s *knowledgeService) RetrieveContext(ctx context.Context, question, cropHint string) (string, []string, error) {
	lower := strings.ToLower(question)

	detectedCrop := cropHint
	if detectedCrop == "" {
		for _, crop := range knownCrops {
			if strings.Contains(lower, crop) {
				detectedCrop = crop
				break
			}
		}
	}

	detectedCategory := ""
	for category, keywords := range categoryKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				detectedCategory = category
				break
			}
		}
		if detectedCategory != "" {
			break
		}
	}

	docs, err := s.repo.Search(ctx, detectedCrop, detectedCategory, 6)
	if err != nil {
		return "", nil, fmt.Errorf("knowledge search failed: %w", err)
	}

	if len(docs) == 0 && detectedCrop != "" {
		// Fallback: search by crop only
		docs, err = s.repo.Search(ctx, detectedCrop, "", 4)
		if err != nil {
			return "", nil, err
		}
	}

	return buildContext(docs)
}

func buildContext(docs []models.AgriculturalDocument) (string, []string, error) {
	var parts []string
	var sources []string
	total := 0

	for _, doc := range docs {
		entry := fmt.Sprintf("[%s - %s]\n%s", doc.Title, doc.Category, doc.Content)
		if total+len(entry) > maxContextChars {
			break
		}
		parts = append(parts, entry)
		total += len(entry)
		if doc.Source != "" {
			sources = append(sources, doc.Source)
		}
	}

	return strings.Join(parts, "\n\n---\n\n"), sources, nil
}
