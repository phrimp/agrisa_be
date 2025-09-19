package services

import (
	utils "agrisa_utils"
	"auth-service/internal/models"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	JWTSecret string
}

func NewJWTService(jwtSecret string) *JWTService {
	return &JWTService{
		JWTSecret: jwtSecret,
	}
}

func (jwt_s *JWTService) GenerateNewToken(roles []string, phone, email, userID string) (string, error) {
	claim_id := "C-" + utils.GenerateRandomStringWithLength(6)
	claim := models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   "auth-service",
		},
		Id:     claim_id,
		UserID: userID,
		Phone:  phone,
		Email:  email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte(jwt_s.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("error generate token string: %s", err)
	}
	return tokenString, nil
}

func (jwt_s *JWTService) VerifyToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&models.Claims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwt_s.JWTSecret, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*models.Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
