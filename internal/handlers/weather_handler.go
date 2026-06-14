package handlers

import (
	"net/http"

	"github.com/agriconnect-ai/internal/services"
	"github.com/agriconnect-ai/internal/validation"
	"github.com/gin-gonic/gin"
)

type WeatherHandler struct {
	weatherSvc services.WeatherService
}

func NewWeatherHandler(weatherSvc services.WeatherService) *WeatherHandler {
	return &WeatherHandler{weatherSvc: weatherSvc}
}

func (h *WeatherHandler) GetWeather(c *gin.Context) {
	district := c.Query("district")
	if district == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "district query parameter is required"})
		return
	}

	if err := validation.District(district); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.weatherSvc.GetWeather(c.Request.Context(), district, 7)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Weather data temporarily unavailable"})
		return
	}

	c.JSON(http.StatusOK, result)
}
