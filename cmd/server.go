package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// StartHTTPServer starts the HTTP server with graceful shutdown support
func StartHTTPServer(router *gin.Engine) error {
	// Configure server
	addr := fmt.Sprintf("%s:%s", appConfig.Server.Host, appConfig.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create shutdown channel with buffer
	shutdownChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		pkgLogger.Infof("Starting server on %s", addr)

		var err error
		if appConfig.Server.TLS.Enable {
			err = srv.ListenAndServeTLS(appConfig.Server.TLS.CertFile, appConfig.Server.TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			shutdownChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Listen for interrupt signal or shutdown channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal or an error
	select {
	case <-quit:
		pkgLogger.Info("Shutdown signal received")
	case err := <-shutdownChan:
		pkgLogger.Errorf("Server error: %v", err)
		return err
	}

	// Create shutdown timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	pkgLogger.Info("Shutting down server gracefully...")
	if err := srv.Shutdown(ctx); err != nil {
		pkgLogger.Errorf("Forced shutdown: %v", err)
		return err
	}

	pkgLogger.Info("Server stopped successfully")
	return nil
}
