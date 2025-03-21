package proxy

import (
	"api-gateway-service-ms/config"
	"api-gateway-service-ms/internal/pkg/logger"
	"api-gateway-service-ms/internal/pkg/response"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ServiceProxy handles proxying requests to backend services
type ServiceProxy struct {
	config     *config.Config
	httpClient *http.Client
	logger     *logger.Logger
}

// NewServiceProxy creates a new service proxy
func NewServiceProxy(cfg *config.Config, logger *logger.Logger) *ServiceProxy {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &ServiceProxy{
		config:     cfg,
		httpClient: httpClient,
		logger:     logger,
	}
}

// ForwardRequest forwards a request to a backend service
func (sp *ServiceProxy) ForwardRequest(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceURL, exists := sp.config.FowardServiceUrl[serviceName]
		if !exists || serviceURL == "" {
			response.Error(
				c,
				http.StatusNotFound,
				fmt.Sprintf("Service '%s' not found", serviceName),
			)

			c.Abort()
			return
		}

		target, err := url.Parse(serviceURL)
		if err != nil {
			sp.logger.Errorf("Error parsing service URL: %v", err)
			response.Error(
				c,
				http.StatusInternalServerError,
				"Internal server error",
			)

			c.Abort()
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(target)

		// Set custom director to modify the request
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)

			// Update request URL
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host

			// Strip the service prefix from the path if needed
			// For example, if the request is to /user/profile and the service is "user",
			// the path becomes /profile
			if strings.HasPrefix(req.URL.Path, "/"+serviceName) {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, "/"+serviceName)
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
			}

			// Forward user information if available
			if userID, exists := c.Get("user_id"); exists {
				req.Header.Set("X-User-ID", userID.(string))
			}

			// Set X-Forwarded headers
			req.Header.Set("X-Forwarded-For", c.ClientIP())
			req.Header.Set("X-Forwarded-Host", c.Request.Host)
			req.Header.Set("X-Forwarded-Proto", c.Request.URL.Scheme)

			// Set X-Request-ID for tracing
			requestID := c.GetHeader("X-Request-ID")
			if requestID == "" {
				requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			}
			req.Header.Set("X-Request-ID", requestID)
		}

		// Set custom error handler
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			sp.logger.Errorf("Proxy error: %v", err)
			response.Error(c, http.StatusBadGateway, "Bad gateway")
			c.Abort()
		}

		// Set custom response modifier
		proxy.ModifyResponse = func(resp *http.Response) error {
			// Log response status
			sp.logger.Infof("Proxied response from %s with status code: %d", serviceName, resp.StatusCode)

			// Read and modify response body if needed
			if resp.StatusCode >= http.StatusBadRequest {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}

				// Close original body
				resp.Body.Close()

				// Create new body with the read bytes
				resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Log error response
				sp.logger.Warnf("Error response from %s: %s", serviceName, string(bodyBytes))
			}

			return nil
		}

		// Serve the request
		proxy.ServeHTTP(c.Writer, c.Request)

		// Abort Gin's request handling since the proxy has already written the response
		c.Abort()
	}
}

// SetupRoutes sets up routes for all services
func (sp *ServiceProxy) SetupRoutes(router *gin.Engine) {
	// Set up routes for each service
	for service := range sp.config.FowardServiceUrl {
		// Create route group for the service
		group := router.Group("/" + service)

		// Add proxy handler
		group.Any("", sp.ForwardRequest(service))

		sp.logger.Infof("Registered routes for service: %s", service)
	}
}
