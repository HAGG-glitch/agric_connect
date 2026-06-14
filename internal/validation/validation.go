package validation

import (
	"fmt"
	"strings"

	"github.com/agriconnect-ai/internal/weather"
)

var validLanguages = map[string]bool{
	"english": true,
	"krio":    true,
}

var validCrops = map[string]bool{
	"rice": true, "cassava": true, "maize": true, "groundnut": true,
	"yam": true, "sweet potato": true, "plantain": true, "banana": true,
	"cocoa": true, "coffee": true, "palm oil": true, "ginger": true,
	"pepper": true, "tomato": true, "okra": true, "eggplant": true,
	"cowpea": true, "sorghum": true, "other": true,
}

func Language(lang string) error {
	if lang == "" {
		return nil
	}
	if !validLanguages[strings.ToLower(lang)] {
		return fmt.Errorf("unsupported language: %s (must be english or krio)", lang)
	}
	return nil
}

func District(district string) error {
	if district == "" {
		return nil
	}
	if !weather.IsValidDistrict(district) {
		return fmt.Errorf("unsupported district: %s", district)
	}
	return nil
}

func Crop(crop string) error {
	if crop == "" {
		return nil
	}
	if !validCrops[strings.ToLower(crop)] {
		return fmt.Errorf("unsupported crop: %s", crop)
	}
	return nil
}

func MessageLength(msg string, min, max int) error {
	length := len(strings.TrimSpace(msg))
	if length < min {
		return fmt.Errorf("message too short (minimum %d characters)", min)
	}
	if length > max {
		return fmt.Errorf("message too long (maximum %d characters)", max)
	}
	return nil
}
