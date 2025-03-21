package main

import (
	"api-gateway-service-ms/config"
	"api-gateway-service-ms/internal/controller"
	"api-gateway-service-ms/internal/middleware"
	"api-gateway-service-ms/internal/pkg/cache"
	"api-gateway-service-ms/internal/pkg/logger"
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	pkgLogger *logger.Logger
	pkgCache  *cache.Cache
	appConfig *config.Config = &config.Config{}
)

func init() {
	// init configuration
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	if err := config.LoadConfig(workDir+"/config", appConfig); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// init logger
	loggerConfig := logger.LoggerConfig{
		Env:         "development",
		Level:       logrus.InfoLevel,
		ServiceName: "api-gateway",
		EnableJSON:  false,
		Fields: map[string]interface{}{
			"version": "1.0.0",
		},
	}
	pkgLogger = logger.SetupLogger(loggerConfig)

	// init cache client
	pkgCache = cache.NewCacheClient(pkgLogger, appConfig)
	if err := pkgCache.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping cache: %v", err)
	}
}

func main() {
	logger.SetConfig(appConfig.Env)
	if appConfig.Env == "development" {
		logger.SetLevel(logrus.DebugLevel)
	} else if appConfig.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// init the router
	router := gin.New()

	// init the middleware
	loggerMiddleware := middleware.NewLoggerMiddleware(pkgLogger)
	authMiddleware := middleware.NewAuthMiddleware(appConfig, pkgLogger)
	idempotencyMiddleware := middleware.NewIdempotencyMiddleware(pkgCache, pkgLogger)
	rateLimiterMiddleware := middleware.NewRateLimiterMiddleware(pkgCache, pkgLogger, appConfig)
	middleware := middleware.NewMiddleware(
		rateLimiterMiddleware,
		loggerMiddleware,
		authMiddleware,
		idempotencyMiddleware,
	)

	// init the controller
	healthController := controller.NewHealthController(appConfig, pkgCache, pkgLogger)

	// Register the middleware
	router.Use(middleware.Logger())
	router.Use(middleware.RateLimiter())
	router.Use(middleware.Idempotency())

	// regi the routes
	healthRouter := router.Group("/health")
	healthRouter.GET("", healthController.CheckHealth)

	// register the proxy
	// proxy := proxy.NewServiceProxy(appConfig, pkgLogger)
	// proxy.SetupRoutes(router)

	// Start the server
	if err := StartHTTPServer(router); err != nil {
		pkgLogger.Fatalf("Failed to start the server: %v", err)
	}
}
