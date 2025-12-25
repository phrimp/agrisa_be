package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
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
	weatherGroupPublic.GET("/current/polygon", h.GetCurrentWeatherByPolygon)
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

func (h *WeatherHandler) GetCurrentWeatherByPolygon(c *gin.Context) {
	// Simple endpoint: only polygon_id required, no time ranges
	polygonID := c.Query("polygon_id")

	if polygonID == "" {
		// If no polygon_id, require coordinates to create one
		lat1Str := c.Query("lat1")
		lon1Str := c.Query("lon1")
		lat2Str := c.Query("lat2")
		lon2Str := c.Query("lon2")
		lat3Str := c.Query("lat3")
		lon3Str := c.Query("lon3")
		lat4Str := c.Query("lat4")
		lon4Str := c.Query("lon4")

		if lat1Str == "" || lon1Str == "" || lat2Str == "" || lon2Str == "" {
			errorResponse := utils.CreateErrorResponse("Bad Request", "Either polygon_id or coordinates (lat1, lon1, lat2, lon2, lat3, lon3, lat4, lon4) are required")
			c.JSON(http.StatusBadRequest, errorResponse)
			return
		}

		// Parse coordinates
		lat1, _ := strconv.ParseFloat(lat1Str, 64)
		lon1, _ := strconv.ParseFloat(lon1Str, 64)
		lat2, _ := strconv.ParseFloat(lat2Str, 64)
		lon2, _ := strconv.ParseFloat(lon2Str, 64)

		coords := [][2]float64{
			{lon1, lat1},
			{lon2, lat2},
		}

		if lat3Str != "" && lon3Str != "" {
			lat3, _ := strconv.ParseFloat(lat3Str, 64)
			lon3, _ := strconv.ParseFloat(lon3Str, 64)
			coords = append(coords, [2]float64{lon3, lat3})
		}
		if lat4Str != "" && lon4Str != "" {
			lat4, _ := strconv.ParseFloat(lat4Str, 64)
			lon4, _ := strconv.ParseFloat(lon4Str, 64)
			coords = append(coords, [2]float64{lon4, lat4})
		}

		// Create polygon
		polygonName := fmt.Sprintf("temp_polygon_%d", time.Now().Unix())
		polygon, err := h.agroService.CreatePolygon(polygonName, coords)
		if err != nil {
			errorResponse := utils.CreateErrorResponse("Internal server error", "Failed to create polygon: "+err.Error())
			c.JSON(http.StatusInternalServerError, errorResponse)
			return
		}
		polygonID = polygon.ID
	}

	// Get current weather for the polygon - real-time data only
	currentWeather, err := h.agroService.GetCurrentWeather(polygonID)
	if err != nil {
		errorResponse := utils.CreateErrorResponse("Internal server error", "Failed to fetch current weather: "+err.Error())
		c.JSON(http.StatusInternalServerError, errorResponse)
		return
	}

	c.JSON(http.StatusOK, currentWeather)
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

