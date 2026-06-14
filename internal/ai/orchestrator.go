package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/agriconnect-ai/internal/weather"
)

const maxToolIterations = 3

type OrchestratorResult struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Sources      []string
}

type KnowledgeProvider interface {
	RetrieveContext(ctx context.Context, question, crop string) (string, []string, error)
}

type WeatherProvider interface {
	GetWeather(ctx context.Context, district string, forecastDays int) (*weather.WeatherResponse, error)
}

type Orchestrator struct {
	client       *Client
	knowledgeSvc KnowledgeProvider
	weatherSvc   WeatherProvider
}

func NewOrchestrator(client *Client, knowledge KnowledgeProvider, weatherSvc WeatherProvider) *Orchestrator {
	return &Orchestrator{
		client:       client,
		knowledgeSvc: knowledge,
		weatherSvc:   weatherSvc,
	}
}

func (o *Orchestrator) Run(ctx context.Context, messages []Message, language, district, crop string) (*OrchestratorResult, error) {
	question := lastUserMessage(messages)

	knowledgeCtx, sources, err := o.knowledgeSvc.RetrieveContext(ctx, question, crop)
	if err != nil {
		log.Printf("knowledge retrieval warning: %v", err)
	}

	systemPrompt := BuildSystemPrompt(language, district, crop, knowledgeCtx)

	var weatherCtx string
	if district != "" && DetectWeatherIntent(question) {
		wr, werr := o.weatherSvc.GetWeather(ctx, district, 7)
		if werr == nil {
			weatherCtx = BuildWeatherContext(wr)
		} else {
			log.Printf("weather fetch warning: %v", werr)
		}
	}

	if weatherCtx != "" {
		systemPrompt = systemPrompt + "\n\n---\n" + weatherCtx
	}

	allMessages := []Message{{Role: "system", Content: systemPrompt}}
	allMessages = append(allMessages, messages...)

	tools := []Tool{WeatherToolDefinition()}
	req := ChatRequest{
		Messages:    allMessages,
		Tools:       tools,
		MaxTokens:   1500,
		Temperature: 0.3,
	}

	for i := 0; i < maxToolIterations; i++ {
		resp, err := o.client.Chat(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("groq chat failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("groq returned no choices")
		}

		choice := resp.Choices[0]

		if len(choice.Message.ToolCalls) == 0 {
			return &OrchestratorResult{
				Content:      choice.Message.Content,
				InputTokens:  resp.Usage.PromptTokens,
				OutputTokens: resp.Usage.CompletionTokens,
				Sources:      sources,
			}, nil
		}

		req.Messages = append(req.Messages, choice.Message)

		for _, tc := range choice.Message.ToolCalls {
			toolResult, err := o.executeToolCall(ctx, tc)
			if err != nil {
				toolResult = fmt.Sprintf("Tool error: %s", err.Error())
			}
			req.Messages = append(req.Messages, Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
			})
		}
	}

	return nil, fmt.Errorf("exceeded maximum tool iterations")
}

func (o *Orchestrator) RunStream(ctx context.Context, messages []Message, language, district, crop string, tokenCh chan<- string, statusCh chan<- string) (*OrchestratorResult, error) {
	question := lastUserMessage(messages)

	sendStatus(statusCh, "Searching agricultural knowledge base")
	knowledgeCtx, sources, err := o.knowledgeSvc.RetrieveContext(ctx, question, crop)
	if err != nil {
		log.Printf("knowledge retrieval warning: %v", err)
	}

	systemPrompt := BuildSystemPrompt(language, district, crop, knowledgeCtx)

	if district != "" && DetectWeatherIntent(question) {
		sendStatus(statusCh, fmt.Sprintf("Checking weather for %s", district))
		wr, werr := o.weatherSvc.GetWeather(ctx, district, 7)
		if werr == nil {
			systemPrompt = systemPrompt + "\n\n---\n" + BuildWeatherContext(wr)
		} else {
			log.Printf("weather fetch warning: %v", werr)
		}
	}

	sendStatus(statusCh, "Generating response")

	allMessages := []Message{{Role: "system", Content: systemPrompt}}
	allMessages = append(allMessages, messages...)

	req := ChatRequest{
		Messages:    allMessages,
		MaxTokens:   1500,
		Temperature: 0.3,
	}

	content, inputTok, outputTok, err := o.client.ChatStream(ctx, req, tokenCh)
	if err != nil {
		return nil, err
	}

	return &OrchestratorResult{
		Content:      content,
		InputTokens:  inputTok,
		OutputTokens: outputTok,
		Sources:      sources,
	}, nil
}

func (o *Orchestrator) executeToolCall(ctx context.Context, tc ToolCall) (string, error) {
	switch tc.Function.Name {
	case "get_weather_forecast":
		var args struct {
			District     string `json:"district"`
			ForecastDays int    `json:"forecast_days"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments: %w", err)
		}

		wr, err := o.weatherSvc.GetWeather(ctx, args.District, args.ForecastDays)
		if err != nil {
			return fmt.Sprintf("Weather data unavailable for %s: %v", args.District, err), nil
		}

		return weatherResponseToText(wr), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", tc.Function.Name)
	}
}

func weatherResponseToText(wr *weather.WeatherResponse) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Weather for %s:\n", wr.District))
	sb.WriteString(fmt.Sprintf("Current: %.1f°C, humidity %d%%, precipitation %.1f mm, wind %.1f km/h\n",
		wr.Current.TemperatureC, wr.Current.HumidityPercent,
		wr.Current.PrecipitationMM, wr.Current.WindSpeedKMH))
	for _, d := range wr.Daily {
		sb.WriteString(fmt.Sprintf("%s: %.0f-%.0f°C, rain %d%%, %.1f mm\n",
			d.Date, d.MinTemperatureC, d.MaxTemperatureC,
			d.RainProbabilityPercent, d.PrecipitationMM))
	}
	return sb.String()
}

func lastUserMessage(messages []Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

func sendStatus(ch chan<- string, msg string) {
	if ch != nil {
		select {
		case ch <- msg:
		default:
		}
	}
}
