package handlers

import (
	"auth-service/internal/config"
	"auth-service/internal/services"
	"auth-service/utils"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Middleware struct {
	jwtService     *services.JWTService
	sessionService *services.SessionService
	config         *config.AuthConfig
}

func NewMiddleware(jwtService *services.JWTService, sessionService *services.SessionService, config *config.AuthConfig) *Middleware {
	return &Middleware{
		jwtService:     jwtService,
		sessionService: sessionService,
		config:         config,
	}
}

func (m *Middleware) RegisterRoutes(routes *gin.Engine) {
	routes.GET("/auth/validate", m.ValidateToken)
}

func (m *Middleware) ValidateToken(c *gin.Context) {
	log.Printf("ValidateToken called - Method: %s, Path: %s", c.Request.Method, c.Request.URL.Path)

	apiKey := c.GetHeader("API-KEY")
	if apiKey != "" && apiKey == m.config.APIKey {
		// Return success status for ForwardAuth middleware
		c.JSON(http.StatusOK, utils.SuccessResponse{
			Success: true,
			Data:    nil,
			Meta: &utils.Meta{
				Timestamp: time.Now(),
			},
		})
	}

	// Check for Authorization header
	authHeader := c.GetHeader("Authorization")
	log.Printf("Authorization header: %s", authHeader)

	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "MISSING_TOKEN",
				Message: "authorization header required",
			},
		})
		return
	}

	// Extract token from Bearer format
	tokenString := authHeader
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = authHeader[7:]
	}

	// Validate the token and extract claims
	claims, err := m.jwtService.VerifyToken(tokenString)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
		c.JSON(http.StatusUnauthorized, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "INVALID_TOKEN",
				Message: "token validation failed",
			},
		})
		return
	}

	// Check if session is valid
	sessions, err := m.sessionService.GetUserSessions(c, claims.UserID)
	if err != nil {
		log.Printf("Failed to retrieve user sessions: %v", err)
		c.JSON(http.StatusInternalServerError, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "SESSION_CHECK_FAILED",
				Message: "failed to check user session",
			},
		})
		return
	}

	isSessionValid := false
	for _, session := range sessions {
		if session.TokenHash == tokenString && session.IsActive {
			isSessionValid = true
			m.sessionService.RenewSession(c, session.ID)
			break
		}
	}

	if !isSessionValid {
		c.JSON(http.StatusUnauthorized, utils.ErrorResponse{
			Success: false,
			Error: utils.APIError{
				Code:    "SESSION_INVALID",
				Message: "no session found or session invalid",
			},
		})
		return
	}

	c.Header("X-User-ID", claims.UserID)
	c.Header("X-User-Email", claims.Email)

	// Return success status for ForwardAuth middleware
	c.JSON(http.StatusOK, utils.SuccessResponse{
		Success: true,
		Data:    nil,
		Meta: &utils.Meta{
			Timestamp: time.Now(),
		},
	})
}
