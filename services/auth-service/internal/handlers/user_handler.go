package handlers

import (
	"auth-service/internal/services"
	"auth-service/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService services.IUserService
}

func NewUserHandler(userService services.IUserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) PingHandler(c *gin.Context) {
	// Simulate a successful response
	response := utils.CreateSuccessResponse("pong")
	c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers all routes for the user handler
func (u *UserHandler) RegisterRoutes(router *gin.Engine, userHandler *UserHandler) {
	// Add the ping route
	userAuthGrPub := router.Group("/auth/protected/api/v2/")
	userAuthGrPub.GET("/ping", userHandler.PingHandler)
	// Add the session init route
	userAuthGrPub.POST("/ocridcard", userHandler.OCRNationalIDCardHandler)
	userAuthGrPub.GET("/ekyc-progress/:i", userHandler.GetUserEkycProgressByUserID)
	userAuthGrPub.POST("/face-liveness", userHandler.VerifyFaceLiveness)

	// For testing API
	userAuthGrPub.POST("/testing/upload", userHandler.UploadFileTestHandler)
	userAuthGrPub.POST("/testing/upload-multiple", userHandler.UploadMultipleFilesTestHandler)
}

type InitSessionRequest struct {
	DeviceType string `json:"device-type"`
}

func (h *UserHandler) GetUserEkycProgressByUserID(c *gin.Context) {
	userID := c.Param("i")
	userEkycProgress, err := h.userService.GetUserEkycProgressByUserID(userID)
	if err != nil {
		if err.Error() == "user ekyc progress not found" {
			c.JSON(http.StatusNotFound, utils.CreateErrorResponse("NOT_FOUND", "User ekyc progress not found"))
			return
		}

		log.Println("internal error:", err)
		c.JSON(http.StatusInternalServerError, utils.CreateErrorResponse("INTERNAL_ERROR", "Internal server error"))
		return
	}
	c.JSON(http.StatusOK, utils.CreateSuccessResponse(userEkycProgress))
}

func (h *UserHandler) UploadFileTestHandler(c *gin.Context) {
	file, header, err := c.Request.FormFile("testingFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to get file from form: " + err.Error(),
		})
		return
	}
	defer file.Close()

	serviceName := c.Request.FormValue("serviceName")
	if serviceName == "" {
		serviceName = "auth-service" // default value
	}
	err = h.userService.UploadToMinIO(c, file, header, serviceName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to upload file to MinIO: " + err.Error(),
		})
		return
	}
	// Process the uploaded file (for testing purposes)
	log.Printf("Received file: %s", header.Filename)

	c.String(200, "File uploaded successfully: %s", header.Filename)
}

func (h *UserHandler) UploadMultipleFilesTestHandler(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to get multipart form: " + err.Error(),
		})
		return
	}
	uploadFiles, err := h.userService.ProcessAndUploadFiles(form.File, "auth-service", []string{".jpg", ".png", ".jpeg"}, 5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to upload files to MinIO: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, uploadFiles)
}

func (h *UserHandler) OCRNationalIDCardHandler(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	result, err := h.userService.OCRNationalIDCard(form)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process OCR: " + err.Error()})
		return
	}

	switch response := result.(type) {
	case utils.SuccessResponse:
		c.JSON(http.StatusOK, response)
		return
	case utils.ErrorResponse:
		var statusCode int
		if response.Error.Code == "INTERNAL_ERROR" {
			statusCode = http.StatusInternalServerError // 500
		} else {
			statusCode = http.StatusBadRequest // 400
		}

		c.JSON(statusCode, response)
	}
}

func (h *UserHandler) VerifyFaceLiveness(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form"})
		return
	}

	result, err := h.userService.VerifyFaceLiveness(form)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process face liveness: " + err.Error()})
		return
	}

	switch response := result.(type) {
	case utils.SuccessResponse:
		c.JSON(http.StatusOK, response)
		return
	case utils.ErrorResponse:
		var statusCode int
		if response.Error.Code == "INTERNAL_ERROR" || response.Error.Code == "EXTERNAL_API_ERROR" {
			statusCode = http.StatusInternalServerError
		} else {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, response)
	}
}
