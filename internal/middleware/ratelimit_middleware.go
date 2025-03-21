package middleware

import (
	"api-gateway-service-ms/config"
	"api-gateway-service-ms/internal/pkg/cache"
	"api-gateway-service-ms/internal/pkg/logger"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	RATELIMIT_HEADER    = "X-RateLimit-Limit"
	RATELIMIT_REMAINING = "X-RateLimit-Remaining"
	RATELIMIT_RESET     = "X-RateLimit-Reset"
)

type RateLimiterMiddleware struct {
	cache  *cache.Cache
	logger *logger.Logger
	cfg    *config.Config
}

func NewRateLimiterMiddleware(cache *cache.Cache, logger *logger.Logger, cfg *config.Config) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		cache:  cache,
		logger: logger,
		cfg:    cfg,
	}
}

func (rl *RateLimiterMiddleware) HandleRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP or user ID for rate limiting
		identifier := c.ClientIP()
		if userID, exists := c.Get("user_id"); exists {
			identifier = userID.(string)
		}

		key := fmt.Sprintf("ratelimit:%s", identifier)
		ctx := context.Background()

		count, isExceeded, err := rl.checkRateLimit(ctx, key)
		if err != nil {
			rl.logger.Errorf("Error checking rate limit: %v", err)

			c.Next()
			return
		}

		if isExceeded {
			ttl, err := rl.cache.TTL(ctx, key)
			if err != nil {
				rl.logger.Errorf("Error getting rate limit TTL: %v", err)
				ttl = 0
			}

			c.Header(RATELIMIT_HEADER, strconv.Itoa(rl.cfg.Ratelimit.Limit))
			c.Header(RATELIMIT_REMAINING, "0")
			c.Header(RATELIMIT_RESET, strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": ttl.Seconds(),
			})

			c.Abort()
			return
		}

		// Increment count
		_, err = rl.cache.Incr(ctx, key)
		if err != nil {
			rl.logger.Errorf("Error incrementing rate limit count: %v", err)
		}

		// Set rate limit headers
		c.Header(RATELIMIT_HEADER, strconv.Itoa(rl.cfg.Ratelimit.Limit))
		c.Header(RATELIMIT_REMAINING, strconv.Itoa(rl.cfg.Ratelimit.Limit-count-1))

		// Get TTL for the key
		ttl, err := rl.cache.TTL(ctx, key)
		if err != nil {
			rl.logger.Errorf("Error getting rate limit TTL: %v", err)
		} else {
			c.Header(RATELIMIT_RESET, strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
		}

		c.Next()
	}
}

func (rl *RateLimiterMiddleware) checkRateLimit(ctx context.Context, key string) (int, bool, error) {
	var count int
	err := rl.cache.Get(ctx, key, &count)
	if err != nil && err != redis.Nil {
		return 0, false, fmt.Errorf("error getting ratelimit count: %v", err)
	}

	if err == redis.Nil {
		if err = rl.cache.Set(ctx, key, 1, rl.cfg.Ratelimit.Period); err != nil {
			return 0, false, fmt.Errorf("error setting ratelimit count: %v", err)
		}

		count = 1
	}

	return count, count >= rl.cfg.Ratelimit.Limit, nil
}

func (rl *RateLimiterMiddleware) Close() error {
	return rl.cache.Close()
}
