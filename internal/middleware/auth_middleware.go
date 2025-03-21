package middleware

import (
	"api-gateway-service-ms/config"
	"api-gateway-service-ms/internal/pkg/logger"
	"api-gateway-service-ms/internal/pkg/response"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	BEARER_PREFIX = "Bearer"
)

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthMiddleware struct {
	cfg    *config.Config
	logger *logger.Logger
}

func NewAuthMiddleware(cfg *config.Config, logger *logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		cfg:    cfg,
		logger: logger,
	}
}

// AuthMiddleware creates a middleware for JWT authentication
func (am *AuthMiddleware) HandleAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "Authorization header is required")
			c.Abort()
			return
		}

		tokenString, err := extractToken(authHeader)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}

		claims, err := am.ValidateToken(tokenString)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}

		// Set claims in the context for later use
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

func extractToken(authHeader string) (string, error) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != BEARER_PREFIX {
		return "", fmt.Errorf("invalid authorization header format")
	}

	if parts[1] == "" {
		return "", fmt.Errorf("token is required")
	}

	return parts[1], nil
}

func (am *AuthMiddleware) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(am.cfg.Auth.JWTSecret), nil
	})

	if err != nil {
		am.logger.Errorf("Error parsing JWT token: %v", err)
		return nil, fmt.Errorf("invalid or expired token")
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract token claims")
	}

	return claims, nil
}
