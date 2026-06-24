package diagnosis

import (
	"testing"
)

func TestValidateCrop_Valid(t *testing.T) {
	valid := []string{"rice", "cassava", "maize", "groundnut", "cocoa", "coffee", "oil palm", "tomato", "pepper", "yam", "sweet potato", "plantain", "banana", "ginger", "okra", "cowpea", "sorghum", "other"}
	for _, c := range valid {
		if err := ValidateCrop(c); err != nil {
			t.Errorf("ValidateCrop(%q) = %v; want nil", c, err)
		}
	}
}

func TestValidateCrop_CaseInsensitive(t *testing.T) {
	if err := ValidateCrop("Rice"); err != nil {
		t.Errorf("ValidateCrop('Rice') = %v; want nil", err)
	}
	if err := ValidateCrop("CASSava"); err != nil {
		t.Errorf("ValidateCrop('CASSava') = %v; want nil", err)
	}
}

func TestValidateCrop_Invalid(t *testing.T) {
	invalid := []string{"wheat", "soybean", "barley", "", "  "}
	for _, c := range invalid {
		if err := ValidateCrop(c); err == nil {
			t.Errorf("ValidateCrop(%q) = nil; want error", c)
		}
	}
}

func TestValidatePlantPart_Valid(t *testing.T) {
	parts := []string{"whole plant", "leaf", "stem", "root", "fruit", "seed", "flower", "bark", "tuber", "pod", "other"}
	for _, p := range parts {
		if err := ValidatePlantPart(p); err != nil {
			t.Errorf("ValidatePlantPart(%q) = %v; want nil", p, err)
		}
	}
}

func TestValidatePlantPart_CaseInsensitive(t *testing.T) {
	if err := ValidatePlantPart("Leaf"); err != nil {
		t.Errorf("ValidatePlantPart('Leaf') = %v; want nil", err)
	}
	if err := ValidatePlantPart("ROOT"); err != nil {
		t.Errorf("ValidatePlantPart('ROOT') = %v; want nil", err)
	}
}

func TestValidatePlantPart_EmptyAllowed(t *testing.T) {
	if err := ValidatePlantPart(""); err != nil {
		t.Errorf("ValidatePlantPart('') = %v; want nil", err)
	}
}

func TestValidatePlantPart_Invalid(t *testing.T) {
	if err := ValidatePlantPart("branches"); err == nil {
		t.Error("ValidatePlantPart('branches') = nil; want error")
	}
}

func TestValidateConfidence_ClampsNegative(t *testing.T) {
	if got := ValidateConfidence(-5); got != 0 {
		t.Errorf("ValidateConfidence(-5) = %f; want 0", got)
	}
}

func TestValidateConfidence_ClampsAbove100(t *testing.T) {
	if got := ValidateConfidence(150); got != 100 {
		t.Errorf("ValidateConfidence(150) = %f; want 100", got)
	}
}

func TestValidateConfidence_ValidRange(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{0, 0},
		{50, 50},
		{100, 100},
		{99.9, 99.9},
	}
	for _, tt := range tests {
		if got := ValidateConfidence(tt.input); got != tt.want {
			t.Errorf("ValidateConfidence(%f) = %f; want %f", tt.input, got, tt.want)
		}
	}
}

func TestValidateConfidenceLabel_Valid(t *testing.T) {
	labels := []string{"low", "medium", "high"}
	for _, l := range labels {
		if got := ValidateConfidenceLabel(l); got != l {
			t.Errorf("ValidateConfidenceLabel(%q) = %q; want %q", l, got, l)
		}
	}
}

func TestValidateConfidenceLabel_InvalidDefaultsToLow(t *testing.T) {
	invalid := []string{"", "very high", "unknown"}
	for _, l := range invalid {
		if got := ValidateConfidenceLabel(l); got != "low" {
			t.Errorf("ValidateConfidenceLabel(%q) = %q; want 'low'", l, got)
		}
	}
}

func TestValidateUrgency_Valid(t *testing.T) {
	urgencies := []string{"low", "medium", "high", "urgent"}
	for _, u := range urgencies {
		if got := ValidateUrgency(u); got != u {
			t.Errorf("ValidateUrgency(%q) = %q; want %q", u, got, u)
		}
	}
}

func TestValidateUrgency_InvalidDefaultsToMedium(t *testing.T) {
	invalid := []string{"", "critical", "asap"}
	for _, u := range invalid {
		if got := ValidateUrgency(u); got != "medium" {
			t.Errorf("ValidateUrgency(%q) = %q; want 'medium'", u, got)
		}
	}
}

func TestValidateStringSlice_TruncatesSlice(t *testing.T) {
	input := []string{"a", "b", "c", "d", "e"}
	got := ValidateStringSlice(input, 3, 100)
	if len(got) != 3 {
		t.Errorf("ValidateStringSlice length = %d; want 3", len(got))
	}
}

func TestValidateStringSlice_TruncatesStrings(t *testing.T) {
	input := []string{"hello world this is a long string"}
	got := ValidateStringSlice(input, 10, 10)
	if len(got[0]) > 10 {
		t.Errorf("ValidateStringSlice string length = %d; want <= 10", len(got[0]))
	}
}

func TestValidateStringSlice_UnderLimit(t *testing.T) {
	input := []string{"short", "list"}
	got := ValidateStringSlice(input, 10, 500)
	if len(got) != 2 {
		t.Errorf("ValidateStringSlice length = %d; want 2", len(got))
	}
}

func TestEnsureDisclaimer_Empty(t *testing.T) {
	got := EnsureDisclaimer("")
	if got == "" {
		t.Error("EnsureDisclaimer('') returned empty; expected default disclaimer")
	}
}

func TestEnsureDisclaimer_Provided(t *testing.T) {
	input := "Custom disclaimer"
	got := EnsureDisclaimer(input)
	if got != input {
		t.Errorf("EnsureDisclaimer(%q) = %q; want %q", input, got, input)
	}
}

func TestValidateImageType_Valid(t *testing.T) {
	allowed := []string{"image/jpeg", "image/png", "image/webp"}
	valid := []string{"image/jpeg", "image/png", "image/webp"}
	for _, v := range valid {
		if err := ValidateImageType(v, allowed); err != nil {
			t.Errorf("ValidateImageType(%q) = %v; want nil", v, err)
		}
	}
}

func TestValidateImageType_CaseInsensitive(t *testing.T) {
	allowed := []string{"image/jpeg"}
	if err := ValidateImageType("Image/JPEG", allowed); err != nil {
		t.Errorf("ValidateImageType('Image/JPEG') = %v; want nil", err)
	}
}

func TestValidateImageType_Invalid(t *testing.T) {
	allowed := []string{"image/jpeg", "image/png"}
	if err := ValidateImageType("image/gif", allowed); err == nil {
		t.Error("ValidateImageType('image/gif') = nil; want error")
	}
}

func TestValidateAudioType_Valid(t *testing.T) {
	allowed := []string{"audio/webm", "audio/wav", "audio/mpeg"}
	valid := []string{"audio/webm", "audio/wav", "audio/mpeg"}
	for _, v := range valid {
		if err := ValidateAudioType(v, allowed); err != nil {
			t.Errorf("ValidateAudioType(%q) = %v; want nil", v, err)
		}
	}
}

func TestValidateAudioType_Invalid(t *testing.T) {
	allowed := []string{"audio/webm", "audio/wav"}
	if err := ValidateAudioType("audio/mp3", allowed); err == nil {
		t.Error("ValidateAudioType('audio/mp3') = nil; want error")
	}
}

func TestValidateAudioType_EmptyAllowed(t *testing.T) {
	allowed := []string{"audio/webm", "audio/wav"}
	if err := ValidateAudioType("", allowed); err == nil {
		t.Error("ValidateAudioType('') = nil; want error")
	}
}
