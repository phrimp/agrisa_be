package handlers

import (
	"auth-service/internal/config"
	"auth-service/internal/models"
	"auth-service/internal/services"
	"auth-service/utils"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userService services.IUserService
	roleService *services.RoleService
}

func NewAuthHandler(userService services.IUserService, roleService *services.RoleService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		roleService: roleService,
	}
}

func (a *AuthHandler) RegisterRoutes(router *gin.Engine) {
	authGrPub := router.Group("/auth/public")

	// Public routes
	authGrPub.POST("/register", a.Register)
	authGrPub.POST("/login", a.Login)

	authGrPro := router.Group("/auth/protected/api/v2")
	accountGr := router.Group("/account")
	accountGr.POST("/new")
	sessionGr := authGrPro.Group("/session")
	// User manage their own session
	sessionGr.GET("/me", a.GetMySession)
	// Admin manage all sessions
	sessionGr.GET("/all", a.GetAllSessions)
}

func (a *AuthHandler) InitDefaultUser(cfg config.AuthServiceConfig) error {
	_, err := a.userService.RegisterNewUser("NOPHONE", "NOEMAIL", cfg.AuthCfg.AdminPWD, "NOID", true, true)
	return err
}

// Login handles user authentication
func (a *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	// Bind and validate JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid login request format: %v", err)
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INVALID_REQUEST_FORMAT",
				Message: "Invalid request format",
			},
		})
		return
	}

	// Validate request data
	if err := a.validateLoginRequest(&req); err != nil {
		log.Printf("Login validation failed: %v", err)
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	// Get client info for security tracking
	deviceInfo := a.getDeviceInfo(c)
	ipAddress := a.getClientIP(c)

	// Attempt login
	user, session, err := a.userService.Login(req.Email, req.Phone, req.Password, &deviceInfo, &ipAddress)
	if err != nil {
		log.Printf("Login failed for user %s/%s: %v", req.Email, req.Phone, err)

		// Map service errors to appropriate HTTP responses
		statusCode, errorCode := a.mapLoginError(err)
		c.JSON(statusCode, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    errorCode,
				Message: "Login failed",
			},
		})
		return
	}

	// Prepare successful login response
	responseData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":             user.ID,
			"email":          user.Email,
			"phone_number":   user.PhoneNumber,
			"status":         user.Status,
			"phone_verified": user.PhoneVerified,
			"kyc_verified":   user.KYCVerified,
		},
		"session": map[string]interface{}{
			"session_id":  session.ID,
			"expires_at":  session.ExpiresAt,
			"device_info": session.DeviceInfo,
			"ip_address":  session.IPAddress,
			"is_active":   session.IsActive,
		},
		"access_token": session.TokenHash,
	}

	log.Printf("Successful login for user %s/%s", user.ID, user.Email)
	c.JSON(http.StatusOK, utils.SuccessResponse{
		Success: true,
		Data:    responseData,
		Meta: &utils.Meta{
			Timestamp: time.Now(),
		},
	})
}

// validateLoginRequest validates the login request
func (a *AuthHandler) validateLoginRequest(req *models.LoginRequest) error {
	// Check if both email and phone are provided (security issue)
	if req.Email != "" && req.Phone != "" {
		return fmt.Errorf("provide either email or phone, not both")
	}

	// Check if neither email nor phone is provided
	if req.Email == "" && req.Phone == "" {
		return fmt.Errorf("email or phone is required")
	}

	// Validate password
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Basic email validation if provided
	if req.Email != "" {
		if !strings.Contains(req.Email, "@") || len(req.Email) < 5 {
			return fmt.Errorf("invalid email format")
		}
	}

	// Basic phone validation if provided
	if req.Phone != "" {
		if len(req.Phone) < 10 {
			return fmt.Errorf("invalid phone number format")
		}
	}

	return nil
}

// validateRegisterRequest validates the register request
func (a *AuthHandler) validateRegisterRequest(req *models.RegisterRequest) error {
	// Validate email
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !strings.Contains(req.Email, "@") || len(req.Email) < 5 {
		return fmt.Errorf("invalid email format")
	}

	// Validate phone
	if req.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if len(req.Phone) < 10 {
		return fmt.Errorf("invalid phone number format")
	}

	// Validate password
	if req.Password == "" {
		return fmt.Errorf("password is required")
	}
	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Validate national ID
	if req.NationalID == "" {
		return fmt.Errorf("national ID is required")
	}

	return nil
}

// getDeviceInfo extracts device information from request
func (a *AuthHandler) getDeviceInfo(c *gin.Context) string {
	userAgent := c.GetHeader("User-Agent")
	if userAgent == "" {
		userAgent = "Unknown Device"
	}
	return userAgent
}

// getClientIP extracts client IP address
func (a *AuthHandler) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first (for load balancers/proxies)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple are present
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr
	return c.ClientIP()
}

// mapLoginError maps service layer errors to HTTP responses
func (a *AuthHandler) mapLoginError(err error) (int, string) {
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "action forbidden"):
		return http.StatusForbidden, "ACTION_FORBIDDEN"
	case strings.Contains(errorMsg, "account blocked"):
		return http.StatusForbidden, "ACCOUNT_BLOCKED"
	case strings.Contains(errorMsg, "invalid password"):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS"
	case strings.Contains(errorMsg, "email or password incorrect"):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS"
	case strings.Contains(errorMsg, "phone number or password incorrect"):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS"
	case strings.Contains(errorMsg, "user found but still null"):
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

// mapRegisterError maps service layer errors to HTTP responses
func (a *AuthHandler) mapRegisterError(err error) (int, string) {
	errorMsg := err.Error()

	switch {
	case strings.Contains(errorMsg, "email"):
		return http.StatusBadRequest, "INVALID_EMAIL"
	case strings.Contains(errorMsg, "phone"):
		return http.StatusBadRequest, "INVALID_PHONE"
	case strings.Contains(errorMsg, "password format"):
		return http.StatusBadRequest, "INVALID_PASSWORD_FORMAT"
	case strings.Contains(errorMsg, "cccd format"):
		return http.StatusBadRequest, "INVALID_NATIONAL_ID"
	case strings.Contains(errorMsg, "creating new user"):
		return http.StatusConflict, "USER_ALREADY_EXISTS"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}

// Register handles user registration
func (a *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest

	// Bind and validate JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid register request format: %v", err)
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INVALID_REQUEST_FORMAT",
				Message: "Invalid request format",
			},
		})
		return
	}

	// Validate request data
	if err := a.validateRegisterRequest(&req); err != nil {
		log.Printf("Register validation failed: %v", err)
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	// Attempt registration
	user, err := a.userService.RegisterNewUser(req.Phone, req.Email, req.Password, req.NationalID, false, false)
	if err != nil {
		log.Printf("Registration failed for user %s/%s: %v", req.Email, req.Phone, err)

		// Map service errors to appropriate HTTP responses
		statusCode, errorCode := a.mapRegisterError(err)
		c.JSON(statusCode, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    errorCode,
				Message: "Registration failed",
			},
		})
		return
	}

	// Assign default user role

	// Prepare successful registration response
	responseData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":             user.ID,
			"email":          user.Email,
			"phone_number":   user.PhoneNumber,
			"status":         user.Status,
			"phone_verified": user.PhoneVerified,
			"kyc_verified":   user.KYCVerified,
		},
	}

	log.Printf("Successful registration for user %s", user.ID)
	c.JSON(http.StatusCreated, utils.SuccessResponse{
		Success: true,
		Data:    responseData,
		Meta: &utils.Meta{
			Timestamp: time.Now(),
		},
	})
}

func (a *AuthHandler) GetMySession(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, utils.ErrorResponse{
		Success: false,
		Error: utils.APIError{
			Code:    "NOT_IMPLEMENTED",
			Message: "Get my session endpoint not yet implemented",
		},
	})
}

func (a *AuthHandler) GetAllSessions(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, utils.ErrorResponse{
		Success: false,
		Error: utils.APIError{
			Code:    "NOT_IMPLEMENTED",
			Message: "Get all sessions endpoint not yet implemented",
		},
	})
}
