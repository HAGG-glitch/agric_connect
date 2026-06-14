package diagnosis

import (
	"fmt"
	"strings"
)

var validCrops = map[string]bool{
	"rice": true, "cassava": true, "maize": true, "groundnut": true,
	"cocoa": true, "coffee": true, "oil palm": true, "palm oil": true,
	"tomato": true, "pepper": true, "yam": true, "sweet potato": true,
	"plantain": true, "banana": true, "ginger": true, "okra": true,
	"cowpea": true, "sorghum": true, "other": true,
}

func ValidateImageType(contentType string, allowed []string) error {
	for _, a := range allowed {
		if strings.EqualFold(a, contentType) {
			return nil
		}
	}
	return fmt.Errorf("unsupported image type: %s", contentType)
}

func ValidateAudioType(contentType string, allowed []string) error {
	for _, a := range allowed {
		if strings.EqualFold(a, contentType) {
			return nil
		}
	}
	return fmt.Errorf("unsupported audio type: %s", contentType)
}

func ValidateCrop(crop string) error {
	if crop == "" {
		return fmt.Errorf("crop is required")
	}
	if !validCrops[strings.ToLower(crop)] {
		return fmt.Errorf("unsupported crop: %s", crop)
	}
	return nil
}

func ValidatePlantPart(part string) error {
	if part == "" {
		return nil
	}
	lower := strings.ToLower(part)
	for _, valid := range ValidPlantParts {
		if lower == valid {
			return nil
		}
	}
	return fmt.Errorf("unsupported plant part: %s", part)
}

func ValidateConfidence(confidence float64) float64 {
	if confidence < 0 {
		return 0
	}
	if confidence > 100 {
		return 100
	}
	return confidence
}

func ValidateConfidenceLabel(label string) string {
	if ValidConfidenceLabels[label] {
		return label
	}
	return "low"
}

func ValidateUrgency(urgency string) string {
	if ValidUrgencies[urgency] {
		return urgency
	}
	return "medium"
}

func ValidateStringSlice(slice []string, maxLen, maxStrLen int) []string {
	if len(slice) > maxLen {
		slice = slice[:maxLen]
	}
	for i, s := range slice {
		if len(s) > maxStrLen {
			slice[i] = s[:maxStrLen]
		}
	}
	return slice
}

func EnsureDisclaimer(disclaimer string) string {
	if disclaimer == "" {
		return "This is a preliminary AI assessment and may be incorrect. Confirm serious or uncertain crop problems with a qualified agricultural extension officer."
	}
	return disclaimer
}
