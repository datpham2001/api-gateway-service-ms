package controller

import (
	"api-gateway-service-ms/config"
	"api-gateway-service-ms/internal/pkg/cache"
	"api-gateway-service-ms/internal/pkg/logger"
	"api-gateway-service-ms/internal/pkg/response"
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HealthController handles health check requests
type HealthController struct {
	config *config.Config
	cache  *cache.Cache
	logger *logger.Logger
}

func NewHealthController(cfg *config.Config, cache *cache.Cache, logger *logger.Logger) *HealthController {
	return &HealthController{
		config: cfg,
		cache:  cache,
		logger: logger,
	}
}

// CheckHealth handles the health check endpoint
func (h *HealthController) CheckHealth(c *gin.Context) {
	// Check Redis connection
	redisStatus := "up"
	redisError := ""

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.cache.Ping(ctx); err != nil {
		redisStatus = "down"
		redisError = err.Error()
		h.logger.Errorf("Redis health check failed: %v", err)
	}

	// Check backend services
	serviceStatuses := make(map[string]map[string]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for service, url := range h.config.FowardServiceUrl {
		wg.Add(1)
		go func(serviceName, serviceURL string) {
			defer wg.Done()

			status := "up"
			statusCode := 0
			responseTime := 0.0
			errorMsg := ""

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create request
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, serviceURL+"/health", nil)
			if err != nil {
				status = "down"
				errorMsg = err.Error()
				logrus.Errorf("Error creating request for service %s: %v", serviceName, err)
			} else {
				// Measure response time
				startTime := time.Now()

				// Send request
				client := &http.Client{}
				resp, err := client.Do(req)

				responseTime = time.Since(startTime).Seconds()

				if err != nil {
					status = "down"
					errorMsg = err.Error()
					logrus.Errorf("Error checking health of service %s: %v", serviceName, err)
				} else {
					defer resp.Body.Close()
					statusCode = resp.StatusCode

					if statusCode != http.StatusOK {
						status = "degraded"
						errorMsg = "Non-200 status code"
						logrus.Warnf("Service %s health check returned status %d", serviceName, statusCode)
					}
				}
			}

			// Store results
			mu.Lock()
			serviceStatuses[serviceName] = map[string]string{
				"status":        status,
				"statusCode":    http.StatusText(statusCode),
				"responseTime":  time.Duration(responseTime * float64(time.Second)).String(),
				"error":         errorMsg,
				"lastCheckedAt": time.Now().Format(time.RFC3339),
			}
			mu.Unlock()
		}(service, url)
	}

	// Wait for all checks to complete
	wg.Wait()

	// Determine overall status
	overallStatus := "up"
	if redisStatus != "up" {
		overallStatus = "degraded"
	}

	for _, status := range serviceStatuses {
		if status["status"] == "down" {
			overallStatus = "degraded"
			break
		}
	}

	response.Success(c, map[string]interface{}{
		"status":    overallStatus,
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{
			"redis": map[string]interface{}{
				"status": redisStatus,
				"error":  redisError,
			},
			"services": serviceStatuses,
		},
	})
}
