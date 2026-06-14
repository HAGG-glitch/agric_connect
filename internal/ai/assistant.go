package ai

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/agriconnect-ai/internal/weather"
)

//go:embed prompts/agricultural_assistant.txt
var agriculturalPrompt string

//go:embed prompts/krio_rules.txt
var krioRules string

func BuildSystemPrompt(language, district, crop, knowledgeContext string) string {
	var sb strings.Builder
	sb.WriteString(agriculturalPrompt)

	if language == "krio" {
		sb.WriteString("\n\n---\nKRIO LANGUAGE RULES:\n")
		sb.WriteString(krioRules)
	}

	if district != "" {
		sb.WriteString(fmt.Sprintf("\n\nFARMER CONTEXT:\n- District: %s, Sierra Leone\n", district))
	}
	if crop != "" {
		sb.WriteString(fmt.Sprintf("- Primary crop: %s\n", crop))
	}
	if language != "" {
		sb.WriteString(fmt.Sprintf("- Preferred language: %s\n", language))
	}

	if knowledgeContext != "" {
		sb.WriteString("\n\n---\nRELEVANT AGRICULTURAL KNOWLEDGE (use this to inform your answer):\n")
		sb.WriteString(knowledgeContext)
	}

	return sb.String()
}

func BuildWeatherContext(w *weather.WeatherResponse) string {
	if w == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CURRENT WEATHER FOR %s (Sierra Leone):\n", strings.ToUpper(w.District)))
	sb.WriteString(fmt.Sprintf("- Temperature: %.1f°C\n", w.Current.TemperatureC))
	sb.WriteString(fmt.Sprintf("- Humidity: %d%%\n", w.Current.HumidityPercent))
	sb.WriteString(fmt.Sprintf("- Current precipitation: %.1f mm\n", w.Current.PrecipitationMM))
	sb.WriteString(fmt.Sprintf("- Wind speed: %.1f km/h\n", w.Current.WindSpeedKMH))

	if len(w.Daily) > 0 {
		sb.WriteString("\n7-DAY FORECAST:\n")
		for _, day := range w.Daily {
			sb.WriteString(fmt.Sprintf("- %s: %.0f-%.0f°C, Rain probability: %d%%, Precipitation: %.1f mm\n",
				day.Date, day.MinTemperatureC, day.MaxTemperatureC,
				day.RainProbabilityPercent, day.PrecipitationMM))
		}
	}

	return sb.String()
}

func DetectWeatherIntent(question string) bool {
	lower := strings.ToLower(question)
	weatherKeywords := []string{
		"weather", "rain", "temperature", "forecast", "ren", "hot", "dry",
		"wet season", "dry season", "humidity", "wind", "climate",
		"plant this week", "good time to plant", "wetin di weather",
	}
	for _, kw := range weatherKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func WeatherToolDefinition() Tool {
	return Tool{
		Type: "function",
		Function: ToolFunction{
			Name:        "get_weather_forecast",
			Description: "Get current weather and 7-day forecast for a Sierra Leone district. Use this when the farmer asks about weather conditions for farming decisions.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"district": map[string]interface{}{
						"type":        "string",
						"description": "The Sierra Leone district name (e.g., Bo, Bombali, Kenema)",
					},
					"forecast_days": map[string]interface{}{
						"type":        "integer",
						"description": "Number of forecast days (1-7)",
						"default":     7,
					},
				},
				"required": []string{"district"},
			},
		},
	}
}
