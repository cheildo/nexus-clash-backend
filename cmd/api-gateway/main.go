package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/viper"

	"github.com/cheildo/nexus-clash-backend/internal/apigateway"
	"github.com/cheildo/nexus-clash-backend/internal/auth"
)

func main() {
	// --- Configuration Loading ---
	viper.SetConfigName("api-gateway")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs/development")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read configuration file", "error", err)
		os.Exit(1)
	}

	// --- gRPC Client Initialization ---
	grpcClients, err := apigateway.NewGRPClients(viper.GetString("services.auth_service_addr"))
	if err != nil {
		slog.Error("Failed to initialize gRPC clients", "error", err)
		os.Exit(1)
	}

	// --- HTTP Router and Middleware Setup ---
	r := chi.NewRouter()

	// Middleware stack provides robust defaults:
	// - Recovers from panics
	// - Logs requests
	// - Sets timeouts on requests to prevent hanging
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Chi's structured logger
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// --- Route Definitions ---
	// We instantiate our handlers, injecting the gRPC clients they need.
	authHandler := auth.NewHTTPHandler(grpcClients.Auth)

	// We group routes under a `/api/v1` prefix for versioning.
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authHandler.HandleRegister)
		r.Post("/auth/login", authHandler.HandleLogin)
	})

	slog.Info("All routes initialized.")

	// --- HTTP Server Initialization and Graceful Shutdown ---
	httpPort := viper.GetString("http_server.port")
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: r,
	}

	go func() {
		slog.Info("API Gateway starting...", "port", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Could not start server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down API Gateway server...")

	// Create a context with a timeout to allow ongoing requests to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown:", "error", err)
	}

	slog.Info("API Gateway server stopped.")
}
