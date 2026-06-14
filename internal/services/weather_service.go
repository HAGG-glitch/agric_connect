package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agriconnect-ai/internal/repositories"
	"github.com/agriconnect-ai/internal/weather"
)

type WeatherService interface {
	GetWeather(ctx context.Context, district string, forecastDays int) (*weather.WeatherResponse, error)
}

type weatherService struct {
	repo        repositories.WeatherRepository
	client      *weather.Client
	cacheMins   int
}

func NewWeatherService(repo repositories.WeatherRepository, client *weather.Client, cacheMins int) WeatherService {
	return &weatherService{repo: repo, client: client, cacheMins: cacheMins}
}

func (s *weatherService) GetWeather(ctx context.Context, districtName string, forecastDays int) (*weather.WeatherResponse, error) {
	district, ok := weather.GetDistrict(districtName)
	if !ok {
		return nil, fmt.Errorf("unsupported district: %s", districtName)
	}

	// Check cache
	data, fetchedAt, err := s.repo.GetCache(ctx, district.Name)
	if err == nil && fetchedAt != nil {
		age := time.Since(*fetchedAt)
		if age < time.Duration(s.cacheMins)*time.Minute {
			result, err := mapToWeatherResponse(district.Name, data)
			if err == nil {
				result.Cached = true
				return result, nil
			}
		}
	}

	// Fetch fresh
	result, err := s.client.FetchWeather(ctx, district, forecastDays)
	if err != nil {
		return nil, fmt.Errorf("weather provider unavailable: %w", err)
	}

	// Cache it
	cacheData := map[string]interface{}{
		"district":   result.District,
		"current":    result.Current,
		"daily":      result.Daily,
		"fetched_at": result.FetchedAt,
	}
	_ = s.repo.SetCache(ctx, district.Name, cacheData)

	return result, nil
}

func mapToWeatherResponse(district string, data map[string]interface{}) (*weather.WeatherResponse, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var result struct {
		District  string               `json:"district"`
		Current   weather.CurrentWeather  `json:"current"`
		Daily     []weather.DailyForecast `json:"daily"`
		FetchedAt time.Time            `json:"fetched_at"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}

	return &weather.WeatherResponse{
		District:  district,
		Current:   result.Current,
		Daily:     result.Daily,
		FetchedAt: result.FetchedAt,
	}, nil
}
