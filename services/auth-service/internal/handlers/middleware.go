package handlers

import (
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
}

func NewMiddleware(jwtService *services.JWTService, sessionService *services.SessionService) *Middleware {
	return &Middleware{
		jwtService:     jwtService,
		sessionService: sessionService,
	}
}

func (m *Middleware) RegisterRoutes(routes *gin.Engine) {
	routes.GET("/auth/validate", m.ValidateToken)
}

func (m *Middleware) ValidateToken(c *gin.Context) {
	log.Printf("ValidateToken called - Method: %s, Path: %s", c.Request.Method, c.Request.URL.Path)

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
