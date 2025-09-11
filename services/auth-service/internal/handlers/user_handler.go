package handlers

import (
	"auth-service/internal/services"
	"auth-service/utils"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

var fptEkycApiKey = "QNTT3cYDOpRhkZzNu11DPSPaYWeF6PVI"
var baseURLProduction = "https://api.fpt.ai/vision/ekyc-be/"
var baseURLStaging = "https://api.fpt.ai/vision/ekyc/be-stag/"

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
	response := gin.H{"message": "pong"}
	c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers all routes for the user handler
func RegisterRoutes(router *gin.Engine, userHandler *UserHandler) {
	// Add the ping route
	router.GET("/api/v1/auth/ping", userHandler.PingHandler)
	// Add the session init route
	router.POST("/api/v1/auth/session/init", userHandler.SessionInitHandler)
	router.POST("/api/v1/auth/ocridcard", userHandler.OCRNationalIDCardHandler)
	router.GET("/api/v1/auth/ekyc-progress/:i", userHandler.GetUserEkycProgressByUserID)
	router.POST("/api/v1/auth/face-liveness", userHandler.VerifyFaceLiveness)

	// For testing API
	router.POST("/api/v1/auth/testing/upload", userHandler.UploadFileTestHandler)
	router.POST("/api/v1/auth/testing/upload-multiple", userHandler.UploadMultipleFilesTestHandler)

}

type InitSessionRequest struct {
	DeviceType string `json:"device-type"`
}

// SessionInitHandler handles POST /auth-service/session/init
func (h *UserHandler) SessionInitHandler(c *gin.Context) {
	var requestBody InitSessionRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	url := baseURLProduction + "session/init"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	req.Header.Set("api-key", fptEkycApiKey)
	req.Header.Set("device-type", requestBody.DeviceType)
	req.Header.Set("only-engine", "1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to call API A: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "external_api_error"})
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read API A response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.Data(resp.StatusCode, "application/json", body)
}

func (h *UserHandler) OcrHandler(c *gin.Context) {
	log.Println("Content-Type:", c.Request.Header.Get("Content-Type"))

	deviceType := c.GetHeader("device-type")
	documentType := c.GetHeader("document-type")
	sessionID := c.GetHeader("session-id")
	log.Printf("Device type: %s, Document type: %s, Session ID: %s", deviceType, documentType, sessionID)

	cccdFront, err := c.FormFile("cccd_front")
	if err != nil {
		c.String(400, "Error when accessing file: %v", err)
		log.Printf("Error when accessing file cccd_front: %v", err)
		return
	}

	cccdBack, err := c.FormFile("cccd_back")
	if err != nil {
		c.String(400, "Error when accessing file: %v", err)
		log.Printf("Error when accessing file cccd_back: %v", err)
		return
	}

	// Create body for multipart form-data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file cccdFront to field "files"
	f1, err := cccdFront.Open()
	if err != nil {
		c.String(400, "Error when opening cccd_front: %v", err)
		return
	}
	defer f1.Close()

	part1, err := writer.CreateFormFile("files", cccdFront.Filename)
	if err != nil {
		c.String(500, "Error when creating form file: %v", err)
		return
	}
	if _, err := io.Copy(part1, f1); err != nil {
		c.String(500, "Error when copying file: %v", err)
		return
	}

	// add file cccdBack to field "files"
	f2, err := cccdBack.Open()
	if err != nil {
		c.String(400, "Error when opening cccd_back: %v", err)
		return
	}
	defer f2.Close()

	part2, err := writer.CreateFormFile("files", cccdBack.Filename)
	if err != nil {
		c.String(500, "Error when creating form file: %v", err)
		return
	}
	if _, err := io.Copy(part2, f2); err != nil {
		c.String(500, "Error when copying file: %v", err)
		return
	}

	// end writer
	if err := writer.Close(); err != nil {
		c.String(500, "Error when closing writer: %v", err)
		return
	}

	url := baseURLProduction + "ocr"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// set headers
	req.Header.Set("api-key", fptEkycApiKey)
	req.Header.Set("device-type", deviceType)
	req.Header.Set("document-type", documentType)
	req.Header.Set("session-id", sessionID)
	log.Printf("Session id: %s", sessionID)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// add file to req

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to call OCR API: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "external_api_error"})
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyResp, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", bodyResp)

}

func (h *UserHandler) GetUserEkycProgressByUserID(c *gin.Context) {
	userID := c.Param("i")
	userEkycProgress, err := h.userService.GetUserEkycProgressByUserID(userID)

	if err != nil {
		log.Println("internal error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, userEkycProgress)
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
