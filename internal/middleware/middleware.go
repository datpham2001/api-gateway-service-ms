package middleware

import (
	"github.com/gin-gonic/gin"
)

type Middleware struct {
	rateLimiter *RateLimiterMiddleware
	logger      *LoggerMiddleware
	auth        *AuthMiddleware
	idempotency *IdempotencyMiddleware
}

func NewMiddleware(
	rateLimiter *RateLimiterMiddleware,
	logger *LoggerMiddleware,
	auth *AuthMiddleware,
	idempotency *IdempotencyMiddleware,
) *Middleware {
	return &Middleware{
		rateLimiter: rateLimiter,
		logger:      logger,
		auth:        auth,
		idempotency: idempotency,
	}
}

func (m *Middleware) Logger() gin.HandlerFunc {
	return m.logger.HandleLogger()
}

func (m *Middleware) Authentication() gin.HandlerFunc {
	return m.auth.HandleAuth()
}

func (m *Middleware) Idempotency() gin.HandlerFunc {
	return m.idempotency.HandleIdempotency()
}

func (m *Middleware) RateLimiter() gin.HandlerFunc {
	return m.rateLimiter.HandleRateLimit()
}
