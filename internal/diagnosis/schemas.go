package diagnosis

import "time"

type DiagnosisInput struct {
	Crop               string
	District           string
	PreferredLanguage  string
	PlantPart          string
	SymptomDescription string
	SymptomsStartedAt  string
	AffectedPercentage float64
	RecentWeather      string
	FertiliserHistory  string
	PesticideHistory   string
}

type AIResult struct {
	Crop                string   `json:"crop"`
	ProbableCondition   string   `json:"probable_condition"`
	Confidence          float64  `json:"confidence"`
	ConfidenceLabel     string   `json:"confidence_label"`
	Description         string   `json:"description"`
	ObservedSigns       []string `json:"observed_signs"`
	PossibleAlternatives []string `json:"possible_alternatives"`
	RecommendedActions  []string `json:"recommended_actions"`
	PreventionTips      []string `json:"prevention_tips"`
	Urgency             string   `json:"urgency"`
	RequiresExpertReview bool    `json:"requires_expert_review"`
	Disclaimer          string   `json:"disclaimer"`
}

type DiagnosisView struct {
	ID                  string    `json:"id"`
	Crop                string    `json:"crop"`
	District            string    `json:"district"`
	PreferredLanguage   string    `json:"preferred_language"`
	PlantPart           string    `json:"plant_part"`
	SymptomDescription  string    `json:"symptom_description"`
	ProbableCondition   string    `json:"probable_condition"`
	Confidence          float64   `json:"confidence"`
	ConfidenceLabel     string    `json:"confidence_label"`
	Description         string    `json:"description"`
	ObservedSigns       []string  `json:"observed_signs"`
	PossibleAlternatives []string `json:"possible_alternatives"`
	RecommendedActions  []string  `json:"recommended_actions"`
	PreventionTips      []string  `json:"prevention_tips"`
	Urgency             string    `json:"urgency"`
	RequiresExpertReview bool     `json:"requires_expert_review"`
	Disclaimer          string    `json:"disclaimer"`
	ImageContentType    string    `json:"image_content_type"`
	Status              string    `json:"status"`
	ErrorMessage        string    `json:"error_message,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

var ValidPlantParts = []string{
	"whole plant", "leaf", "stem", "root", "fruit",
	"seed", "flower", "bark", "tuber", "pod", "other",
}

var ValidConfidenceLabels = map[string]bool{
	"low": true, "medium": true, "high": true,
}

var ValidUrgencies = map[string]bool{
	"low": true, "medium": true, "high": true, "urgent": true,
}

var AllowedStatuses = map[string]bool{
	"processing": true, "completed": true, "failed": true,
}
