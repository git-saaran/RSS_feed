package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rss_feed/config"
	"rss_feed/internal/feed"
	"rss_feed/internal/handlers"
	"rss_feed/pkg/logger"

	"github.com/gorilla/mux"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(log *logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			log.Info("Request: %s %s", r.Method, r.RequestURI)

			next.ServeHTTP(w, r)

			duration := time.Since(start)
			log.Info("Completed %s in %v", r.RequestURI, duration)
		})
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(log *logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Recovered from panic: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	// Initialize configuration
	cfg := config.LoadConfig()

	// Initialize logger
	log := logger.NewLogger(cfg.LogLevel)

	// Initialize feed manager
	feedManager := feed.NewFeedManager(cfg, log)

	// Initialize handlers
	handler := handlers.NewHandlers(feedManager, log)

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/", handler.HomeHandler).Methods("GET")
	r.HandleFunc("/api/health", handler.HealthHandler).Methods("GET")
	r.HandleFunc("/api/news", handler.NewsHandler).Methods("GET")
	r.HandleFunc("/api/feeds", handler.FeedsHandler).Methods("GET")
	r.HandleFunc("/api/feeds/refresh", handler.RefreshHandler).Methods("POST")
	r.HandleFunc("/api/stats", handler.StatsHandler).Methods("GET")

	// Add middleware
	r.Use(LoggingMiddleware(log))
	r.Use(CORSMiddleware)
	r.Use(RecoveryMiddleware(log))

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ServerTimeout,
		WriteTimeout: cfg.ServerTimeout,
		IdleTimeout:  time.Second * 60,
	}

	// Start background services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go feedManager.Start(ctx)

	// Start server in goroutine
	go func() {
		log.Info("Starting RSS Feed Dashboard on %s", cfg.Port)
		log.Info("Available at http://localhost%s", cfg.Port)
		log.Info("Features: Real-time updates, sentiment analysis, enhanced error handling")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Cancel background services
	cancel()

	// Shutdown server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited")
}
