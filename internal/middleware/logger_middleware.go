package middleware

import (
	"api-gateway-service-ms/internal/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	X_REQUEST_ID = "X-Request-ID"
)

type LoggerMiddleware struct {
	logger *logger.Logger
}

func NewLoggerMiddleware(logger *logger.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{logger: logger}
}

// LoggerMiddleware adds request-specific logging to each request
func (lm *LoggerMiddleware) HandleLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Generate request ID if not present
		requestID := c.GetHeader(X_REQUEST_ID)
		if requestID == "" {
			requestID = uuid.New().String()
			c.Request.Header.Set(X_REQUEST_ID, requestID)
		}

		// Set request ID in the context
		c.Set(logger.FieldRequestID, requestID)

		// Set response header
		c.Writer.Header().Set(X_REQUEST_ID, requestID)

		// Create a logger entry with request info
		logEntry := logger.WithRequestID(requestID).WithFields(map[string]interface{}{
			logger.FieldMethod: c.Request.Method,
			logger.FieldPath:   c.Request.URL.Path,
			logger.FieldIP:     c.ClientIP(),
		})

		// Add user ID if available
		if userID, exists := c.Get("user_id"); exists {
			logEntry = logEntry.WithField(logger.FieldUserID, userID)
		}

		// Store the logger in the context
		c.Set("logger", logEntry)

		// Log request
		logEntry.Info("Request started")

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Update log entry with response info
		logEntry = logEntry.WithFields(map[string]interface{}{
			logger.FieldStatusCode: c.Writer.Status(),
			logger.FieldDuration:   duration.String(),
		})

		// Log based on status code
		statusCode := c.Writer.Status()
		switch {
		case statusCode >= 500:
			logEntry.Error("Server error")
		case statusCode >= 400:
			logEntry.Warn("Client error")
		case statusCode >= 300:
			logEntry.Info("Redirection")
		default:
			logEntry.Info("Request completed")
		}
	}
}
