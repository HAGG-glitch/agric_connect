package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type DailyForecast struct {
	Date                  string  `json:"date"`
	MinTemperatureC       float64 `json:"minimum_temperature_c"`
	MaxTemperatureC       float64 `json:"maximum_temperature_c"`
	RainProbabilityPercent int    `json:"rain_probability_percent"`
	PrecipitationMM       float64 `json:"precipitation_mm"`
}

type CurrentWeather struct {
	TemperatureC     float64 `json:"temperature_c"`
	HumidityPercent  int     `json:"humidity_percent"`
	PrecipitationMM  float64 `json:"precipitation_mm"`
	WindSpeedKMH     float64 `json:"wind_speed_kmh"`
}

type WeatherResponse struct {
	District  string          `json:"district"`
	Current   CurrentWeather  `json:"current"`
	Daily     []DailyForecast `json:"daily"`
	FetchedAt time.Time       `json:"fetched_at"`
	Cached    bool            `json:"cached"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type openMeteoResponse struct {
	Current struct {
		Temperature2m         float64 `json:"temperature_2m"`
		RelativeHumidity2m    int     `json:"relative_humidity_2m"`
		Precipitation         float64 `json:"precipitation"`
		WindSpeed10m          float64 `json:"wind_speed_10m"`
	} `json:"current"`
	Daily struct {
		Time                       []string  `json:"time"`
		Temperature2mMin           []float64 `json:"temperature_2m_min"`
		Temperature2mMax           []float64 `json:"temperature_2m_max"`
		PrecipitationProbabilityMax []int    `json:"precipitation_probability_max"`
		PrecipitationSum           []float64 `json:"precipitation_sum"`
	} `json:"daily"`
}

func (c *Client) FetchWeather(ctx context.Context, district DistrictCoordinates, forecastDays int) (*WeatherResponse, error) {
	if forecastDays <= 0 || forecastDays > 16 {
		forecastDays = 7
	}

	url := fmt.Sprintf(
		"%s/forecast?latitude=%.4f&longitude=%.4f"+
			"&current=temperature_2m,relative_humidity_2m,precipitation,wind_speed_10m"+
			"&daily=temperature_2m_max,temperature_2m_min,precipitation_sum,precipitation_probability_max"+
			"&forecast_days=%d&timezone=Africa%%2FFreetown",
		c.baseURL, district.Latitude, district.Longitude, forecastDays,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling open-meteo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var raw openMeteoResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	result := &WeatherResponse{
		District:  district.Name,
		FetchedAt: time.Now().UTC(),
		Current: CurrentWeather{
			TemperatureC:    raw.Current.Temperature2m,
			HumidityPercent: raw.Current.RelativeHumidity2m,
			PrecipitationMM: raw.Current.Precipitation,
			WindSpeedKMH:    raw.Current.WindSpeed10m,
		},
	}

	for i, date := range raw.Daily.Time {
		day := DailyForecast{Date: date}
		if i < len(raw.Daily.Temperature2mMin) {
			day.MinTemperatureC = raw.Daily.Temperature2mMin[i]
		}
		if i < len(raw.Daily.Temperature2mMax) {
			day.MaxTemperatureC = raw.Daily.Temperature2mMax[i]
		}
		if i < len(raw.Daily.PrecipitationProbabilityMax) {
			day.RainProbabilityPercent = raw.Daily.PrecipitationProbabilityMax[i]
		}
		if i < len(raw.Daily.PrecipitationSum) {
			day.PrecipitationMM = raw.Daily.PrecipitationSum[i]
		}
		result.Daily = append(result.Daily, day)
	}

	return result, nil
}
