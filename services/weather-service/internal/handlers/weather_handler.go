package handlers

import (
	"net/http"
	"utils"
	"weather-service/internal/services"

	"github.com/gin-gonic/gin"
)

type WeatherHandler struct {
	weatherService services.IWeatherService
}

func NewWeatherHandler(weatherService services.IWeatherService) *WeatherHandler {
	return &WeatherHandler{
		weatherService: weatherService,
	}
}

func (h *WeatherHandler) RegisterRoutes(router *gin.Engine) {
	weatherGroupPublic := router.Group("/weather/public/api/v2")
	weatherGroupPublic.GET("/current", h.GetWeatherByCoordinates)
}

func (h *WeatherHandler) GetWeather(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Weather data"})
}

func (h *WeatherHandler) GetWeatherByCoordinates(c *gin.Context) {
	lat := c.Query("lat")
	lon := c.Query("lon")
	if lat == "" || lon == "" {
		errorResponse := utils.CreateErrorResponse("Bad Request", "Latitude and Longitude are required")
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	exclude := c.Query("exclude")
	units := c.Query("units")
	lang := c.Query("lang")

	weatherResponse, err := h.weatherService.FetchWeatherData(lat, lon, exclude, units, lang)
	if err != nil {
		errorResponse := utils.CreateErrorResponse("Internal server error", "Failed to fetch weather data")
		c.JSON(http.StatusInternalServerError, errorResponse)
		return
	}

	c.JSON(http.StatusOK, weatherResponse)
}
