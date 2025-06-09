package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize configuration
	config := LoadConfig()

	// Initialize logger
	logger := NewLogger(config.LogLevel)

	// Initialize feed manager
	feedManager := NewFeedManager(config, logger)

	// Initialize WebSocket manager
	wsManager := NewWebSocketManager(logger)

	// Initialize handlers
	handlers := NewHandlers(feedManager, wsManager, logger)

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/", handlers.HomeHandler).Methods("GET")
	r.HandleFunc("/ws", handlers.WebSocketHandler).Methods("GET")
	r.HandleFunc("/api/health", handlers.HealthHandler).Methods("GET")
	r.HandleFunc("/api/news", handlers.NewsHandler).Methods("GET")
	r.HandleFunc("/api/feeds", handlers.FeedsHandler).Methods("GET")
	r.HandleFunc("/api/feeds/refresh", handlers.RefreshHandler).Methods("POST")
	r.HandleFunc("/api/stats", handlers.StatsHandler).Methods("GET")

	// Add middleware
	r.Use(LoggingMiddleware(logger))
	r.Use(CORSMiddleware)
	r.Use(RecoveryMiddleware(logger))

	// Create HTTP server
	srv := &http.Server{
		Addr:         config.Port,
		Handler:      r,
		ReadTimeout:  config.ServerTimeout,
		WriteTimeout: config.ServerTimeout,
		IdleTimeout:  time.Second * 60,
	}

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go feedManager.Start(ctx)
	go wsManager.Start(ctx)

	// Start server in goroutine
	go func() {
		logger.Info("Starting Enhanced Financial News Dashboard on %s", config.Port)
		logger.Info("Features: WebSocket updates, sentiment analysis, enhanced error handling")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Cancel background services
	cancel()

	// Shutdown server with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
