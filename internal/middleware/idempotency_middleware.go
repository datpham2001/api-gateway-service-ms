package middleware

import (
	"api-gateway-service-ms/internal/pkg/cache"
	"api-gateway-service-ms/internal/pkg/logger"
	"api-gateway-service-ms/internal/pkg/response"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	X_IDEMPOTENCY_KEY = "X-Idempotency-Key"
	IDEMPOTENCY_TTL   = 24 * time.Hour
)

type CachedResponse struct {
	StatusCode int               `json:"status_code"`
	Body       []byte            `json:"body"`
	Headers    map[string]string `json:"headers"`
}

// responseBodyWriter is a custom response writer that captures the response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

type IdempotencyMiddleware struct {
	cache  *cache.Cache
	logger *logger.Logger
}

func NewIdempotencyMiddleware(cache *cache.Cache, logger *logger.Logger) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{
		cache:  cache,
		logger: logger,
	}
}

func (im *IdempotencyMiddleware) HandleIdempotency() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isIdempotentMethod(c.Request.Method) {
			c.Next()
			return
		}

		idempotencyKey := c.Request.Header.Get(X_IDEMPOTENCY_KEY)
		if idempotencyKey == "" {
			var err error
			idempotencyKey, err = generateIdempotencyKey(c)
			if err != nil {
				im.logger.Errorf("Failed to generate idempotency key: %v", err)
				c.Next()
				return
			}
		}

		cacheKey := fmt.Sprintf("idempotency:%s:%s", c.Request.Method, idempotencyKey)
		ctx := context.Background()

		var cachedResponse CachedResponse
		if err := im.cache.Get(ctx, cacheKey, &cachedResponse); err == nil {
			im.logger.Infof("Cache hit for idempotency key: %s", idempotencyKey)
			for k, v := range cachedResponse.Headers {
				c.Header(k, v)
			}

			c.Header("X-Idempotency-Hit", "true")

			c.Data(cachedResponse.StatusCode, "application/json", cachedResponse.Body)
			c.Abort()
			return
		}

		// Use a lock to prevent race conditions with concurrent requests using the same idempotency key
		lockKey := fmt.Sprintf("idempotency_lock:%s:%s", c.Request.Method, idempotencyKey)
		if err := im.cache.Set(ctx, lockKey, true, 10*time.Second); err != nil {
			im.logger.Warnf("Failed to set lock for idempotency key: %v", err)

			response.Error(
				c,
				http.StatusConflict,
				"A request with the same idempotency key is already being processed",
			)

			c.Abort()
			return
		}
		defer im.cache.Delete(ctx, lockKey)

		// Create a response writer that captures the response
		writer := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = writer

		// Process the request
		c.Next()

		// Don't cache failed requests
		if c.Writer.Status() >= http.StatusInternalServerError {
			im.logger.Warnf(
				"Not caching response with status %d for idempotency key: %s",
				c.Writer.Status(), idempotencyKey,
			)

			return
		}

		// After the request is processed, cache the response
		headers := make(map[string]string)
		for k, v := range c.Writer.Header() {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}

		cachedResponse = CachedResponse{
			StatusCode: c.Writer.Status(),
			Headers:    headers,
			Body:       writer.body.Bytes(),
		}

		// Store the response in cache
		if err := im.cache.Set(ctx, cacheKey, cachedResponse, IDEMPOTENCY_TTL); err != nil {
			im.logger.Errorf("Failed to cache response for idempotency key %s: %v",
				idempotencyKey, err)
		} else {
			im.logger.Infof("Cached response for idempotency key: %s", idempotencyKey)
		}

		c.Next()
	}
}

func isIdempotentMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete
}

func generateIdempotencyKey(c *gin.Context) (string, error) {
	hasher := sha256.New()

	// Add method and path
	hasher.Write([]byte(c.Request.Method + ":" + c.Request.URL.Path))

	if userId, exists := c.Get("user_id"); exists {
		hasher.Write([]byte(fmt.Sprintf("user_id:%s", userId)))
	}

	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return "", err
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		hasher.Write(bodyBytes)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
