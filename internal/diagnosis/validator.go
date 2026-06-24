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

// ValidateImageType checks that the given MIME type is in the allowed list.
// Case-insensitive comparison is used.
func ValidateImageType(contentType string, allowed []string) error {
	for _, a := range allowed {
		if strings.EqualFold(a, contentType) {
			return nil
		}
	}
	return fmt.Errorf("unsupported image type: %s", contentType)
}

// ValidateAudioType checks that the given audio MIME type is in the
// allowed list. Case-insensitive comparison is used.
func ValidateAudioType(contentType string, allowed []string) error {
	for _, a := range allowed {
		if strings.EqualFold(a, contentType) {
			return nil
		}
	}
	return fmt.Errorf("unsupported audio type: %s", contentType)
}

// ValidateCrop checks that the crop name is in the list of supported crops
// (rice, cassava, maize, etc.). Empty or unsupported names return an error.
func ValidateCrop(crop string) error {
	if crop == "" {
		return fmt.Errorf("crop is required")
	}
	if !validCrops[strings.ToLower(crop)] {
		return fmt.Errorf("unsupported crop: %s", crop)
	}
	return nil
}

// ValidatePlantPart checks that the plant part is one of the valid options
// (whole plant, leaf, stem, root, fruit, seed, flower, bark, tuber, pod,
// other). Empty values are accepted (field is optional).
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

// ValidateConfidence clamps the confidence value to the [0, 100] range.
func ValidateConfidence(confidence float64) float64 {
	if confidence < 0 {
		return 0
	}
	if confidence > 100 {
		return 100
	}
	return confidence
}

// ValidateConfidenceLabel returns the label if it is a known value (low,
// medium, high); otherwise defaults to "low".
func ValidateConfidenceLabel(label string) string {
	if ValidConfidenceLabels[label] {
		return label
	}
	return "low"
}

// ValidateUrgency returns the urgency if it is a known value (low, medium,
// high, urgent); otherwise defaults to "medium".
func ValidateUrgency(urgency string) string {
	if ValidUrgencies[urgency] {
		return urgency
	}
	return "medium"
}

// ValidateStringSlice truncates the slice to at most maxLen items and
// truncates each string to at most maxStrLen characters.
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

// EnsureDisclaimer returns the given disclaimer text, or a default
// disclaimer if one is not provided.
func EnsureDisclaimer(disclaimer string) string {
	if disclaimer == "" {
		return "This is a preliminary AI assessment and may be incorrect. Confirm serious or uncertain crop problems with a qualified agricultural extension officer."
	}
	return disclaimer
}
