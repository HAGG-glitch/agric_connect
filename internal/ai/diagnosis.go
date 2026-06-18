package ai

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

//go:embed prompts/crop_diagnosis.txt
var cropDiagnosisPrompt string

type DiagnosisAIInput struct {
	ImageData          []byte
	ImageContentType   string
	ImageURL           string
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

func buildJSONSchemaFormat() *ResponseFormat {
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
			"probable_condition", "confidence", "description",
			"observed_signs", "possible_alternatives",
			"recommended_actions", "prevention_tips",
			"urgency", "requires_expert_review", "disclaimer",
		},
		"additionalProperties": false,
	}
	schemaBytes, _ := json.Marshal(schema)
	return &ResponseFormat{
		Type: "json_schema",
		JSONSchema: &JSONSchema{
			Name:   "crop_diagnosis_result",
			Schema: schemaBytes,
			Strict: true,
		},
	}
}

func (a *cropDiagnosisAI) Diagnose(ctx context.Context, input DiagnosisAIInput) (*DiagnosisAIResult, error) {
	if !a.client.Available() {
		return nil, fmt.Errorf("AI service is not configured")
	}

	systemMsg := buildDiagnosisSystemPrompt(input)

	var imageRef string
	if input.ImageURL != "" {
		imageRef = input.ImageURL
	} else {
		b64Image := base64.StdEncoding.EncodeToString(input.ImageData)
		imageRef = fmt.Sprintf("data:%s;base64,%s", input.ImageContentType, b64Image)
	}

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
%s`, imageRef, userContent),
		},
	}

	req := ChatRequest{
		Model:          a.model,
		Messages:       messages,
		MaxTokens:      a.maxTokens,
		Temperature:    0.2,
		ResponseFormat: buildJSONSchemaFormat(),
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
	if err == nil {
		return result, nil
	}

	log.Printf("diagnosis JSON parse failed: response_length=%d, first_120=%q, parse_error=%v",
		len(content), truncateFirst(content, 120), err)

	if strings.HasPrefix(content, "Here") || strings.HasPrefix(content, "```") || !strings.HasPrefix(content, "{") {
		result, retryErr := a.retryDiagnose(ctx, input)
		if retryErr == nil {
			return result, nil
		}
		return nil, fmt.Errorf("parsing diagnosis result: %w (retry error: %v)", err, retryErr)
	}

	return nil, fmt.Errorf("parsing diagnosis result: %w", err)
}

var repairPrompt = `Your previous answer was not valid JSON.
Return only a valid JSON object matching the required schema.
No markdown. No code fences. No explanation. No text before or after JSON.
The first character must be "{". The last character must be "}".`

func (a *cropDiagnosisAI) retryDiagnose(ctx context.Context, input DiagnosisAIInput) (*DiagnosisAIResult, error) {
	var imageRef string
	if input.ImageURL != "" {
		imageRef = input.ImageURL
	} else {
		b64Image := base64.StdEncoding.EncodeToString(input.ImageData)
		imageRef = fmt.Sprintf("data:%s;base64,%s", input.ImageContentType, b64Image)
	}

	messages := []Message{
		{Role: "system", Content: repairPrompt},
		{
			Role: "user",
			Content: fmt.Sprintf(`Crop: %s
Symptom: %s
[Image: %s]`,
				input.Crop, input.SymptomDescription, imageRef),
		},
	}

	req := ChatRequest{
		Model:          a.model,
		Messages:       messages,
		MaxTokens:      a.maxTokens,
		Temperature:    0.1,
		ResponseFormat: buildJSONSchemaFormat(),
	}

	resp, err := a.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("retry vision call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("retry vision returned no choices")
	}

	content := resp.Choices[0].Message.Content
	result, err := parseDiagnosisJSON(content)
	if err != nil {
		log.Printf("diagnosis retry parse failed: response_length=%d, first_120=%q, parse_error=%v",
			len(content), truncateFirst(content, 120), err)
		return nil, fmt.Errorf("retry parse failed: %w", err)
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
	if content == "" {
		return nil, fmt.Errorf("empty AI response")
	}

	// Remove markdown code fences
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

	// Extract JSON object from surrounding text
	if braceStart := strings.Index(content, "{"); braceStart >= 0 {
		if braceEnd := strings.LastIndex(content, "}"); braceEnd > braceStart {
			content = content[braceStart : braceEnd+1]
		}
	}
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "{") {
		return nil, fmt.Errorf("response does not contain a JSON object")
	}

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

func truncateFirst(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
