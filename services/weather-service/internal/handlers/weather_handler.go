package handlers

import (
	"net/http"
	"utils"
	"weather-service/internal/models"
	"weather-service/internal/services"

	"github.com/gin-gonic/gin"
)

type WeatherHandler struct {
	weatherService services.IWeatherService
	agroService    services.IAgroService
}

func NewWeatherHandler(weatherService services.IWeatherService, agroService services.IAgroService) *WeatherHandler {
	return &WeatherHandler{
		weatherService: weatherService,
		agroService:    agroService,
	}
}

func (h *WeatherHandler) RegisterRoutes(router *gin.Engine) {
	weatherGroupPublic := router.Group("/weather/public/api/v2")
	weatherGroupPublic.GET("/current", h.GetWeatherByCoordinates)
	weatherGroupPublic.GET("/precipitation/polygon", h.GetPrecipitationByPolygon)
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

func (h *WeatherHandler) GetPrecipitationByPolygon(c *gin.Context) {
	var req models.PrecipitationRequest

	// Bind and validate query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		errorResponse := utils.CreateErrorResponse("Bad Request", err.Error())
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	// Validate time range
	if req.End <= req.Start {
		errorResponse := utils.CreateErrorResponse("Bad Request", "End time must be greater than start time")
		c.JSON(http.StatusBadRequest, errorResponse)
		return
	}

	// If polygon_id is not provided, validate that coordinates are present
	if req.PolygonID == "" {
		if req.Lat1 == 0 && req.Lon1 == 0 && req.Lat2 == 0 && req.Lon2 == 0 {
			errorResponse := utils.CreateErrorResponse("Bad Request", "Either polygon_id or coordinates (lat1, lon1, lat2, lon2, lat3, lon3, lat4, lon4) must be provided")
			c.JSON(http.StatusBadRequest, errorResponse)
			return
		}
	}

	// Construct coordinates array in [lon, lat] format for Agro API
	coordinates := [][2]float64{
		{req.Lon1, req.Lat1},
		{req.Lon2, req.Lat2},
		{req.Lon3, req.Lat3},
		{req.Lon4, req.Lat4},
	}

	// Call improved Agro service method that handles both scenarios
	precipitationResponse, err := h.agroService.GetPrecipitationWithPolygonID(req.PolygonID, coordinates, req.Start, req.End)
	if err != nil {
		errorResponse := utils.CreateErrorResponse("Internal server error", "Failed to fetch precipitation data: "+err.Error())
		c.JSON(http.StatusInternalServerError, errorResponse)
		return
	}

	c.JSON(http.StatusOK, precipitationResponse)
}

