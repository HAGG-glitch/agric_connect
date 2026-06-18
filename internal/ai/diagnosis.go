package ai

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed prompts/crop_diagnosis.txt
var cropDiagnosisPrompt string

type DiagnosisAIInput struct {
	ImageData          []byte
	ImageContentType   string
	Crop               string
	District           string
	PlantPart          string
	SymptomDescription string
	SymptomsStartedAt  string
	AffectedPercentage float64
	RecentWeather      string
	FertiliserHistory  string
	PesticideHistory   string
	PreferredLanguage  string
	KnowledgeContext   string
}

type DiagnosisAIResult struct {
	Crop                 string   `json:"crop"`
	ProbableCondition    string   `json:"probable_condition"`
	Confidence           float64  `json:"confidence"`
	ConfidenceLabel      string   `json:"confidence_label"`
	Description          string   `json:"description"`
	ObservedSigns        []string `json:"observed_signs"`
	PossibleAlternatives  []string `json:"possible_alternatives"`
	RecommendedActions   []string `json:"recommended_actions"`
	PreventionTips       []string `json:"prevention_tips"`
	Urgency              string   `json:"urgency"`
	RequiresExpertReview bool     `json:"requires_expert_review"`
	Disclaimer           string   `json:"disclaimer"`
}

type CropDiagnosisAI interface {
	Diagnose(ctx context.Context, input DiagnosisAIInput) (*DiagnosisAIResult, error)
}

type cropDiagnosisAI struct {
	client     *Client
	model      string
	maxTokens  int
}

func NewCropDiagnosisAI(client *Client, model string, maxTokens int) CropDiagnosisAI {
	if maxTokens <= 0 {
		maxTokens = 512
	}
	return &cropDiagnosisAI{client: client, model: model, maxTokens: maxTokens}
}

func (a *cropDiagnosisAI) Diagnose(ctx context.Context, input DiagnosisAIInput) (*DiagnosisAIResult, error) {
	if !a.client.Available() {
		return nil, fmt.Errorf("AI service is not configured")
	}

	systemMsg := buildDiagnosisSystemPrompt(input)

	b64Image := base64.StdEncoding.EncodeToString(input.ImageData)
	imageURL := fmt.Sprintf("data:%s;base64,%s", input.ImageContentType, b64Image)

	userContent := fmt.Sprintf(`Crop: %s
District: %s
Plant part affected: %s
Symptom description: %s
Symptoms started: %s
Affected percentage: %.1f%%
Recent weather: %s
Fertiliser history: %s
Pesticide history: %s
Language: %s`,
		input.Crop, input.District, input.PlantPart, input.SymptomDescription,
		input.SymptomsStartedAt, input.AffectedPercentage,
		input.RecentWeather, input.FertiliserHistory, input.PesticideHistory,
		input.PreferredLanguage)

	messages := []Message{
		{Role: "system", Content: systemMsg},
		{
			Role: "user",
			Content: fmt.Sprintf(`[Image: %s]
%s`, imageURL, userContent),
		},
	}

	req := ChatRequest{
		Model:       a.model,
		Messages:    messages,
		MaxTokens:   a.maxTokens,
		Temperature: 0.2,
	}

	resp, err := a.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vision API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("vision API returned no choices")
	}

	content := resp.Choices[0].Message.Content

	result, err := parseDiagnosisJSON(content)
	if err != nil {
		return nil, fmt.Errorf("parsing diagnosis result: %w", err)
	}

	return result, nil
}

func buildDiagnosisSystemPrompt(input DiagnosisAIInput) string {
	var sb strings.Builder
	sb.WriteString(cropDiagnosisPrompt)

	if input.KnowledgeContext != "" {
		sb.WriteString("\n\n---\nRELEVANT AGRICULTURAL KNOWLEDGE:\n")
		sb.WriteString(input.KnowledgeContext)
	}

	return sb.String()
}

func parseDiagnosisJSON(content string) (*DiagnosisAIResult, error) {
	content = strings.TrimSpace(content)

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

	var result DiagnosisAIResult
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
