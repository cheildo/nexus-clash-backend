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
	"github.com/cheildo/nexus-clash-backend/internal/matchmaking" // Import matchmaking
	"github.com/cheildo/nexus-clash-backend/internal/pkg/redis"   // Import redis
	"github.com/cheildo/nexus-clash-backend/internal/playerprofile"
)

func main() {
	// --- Configuration Loading ---
	// ... (no changes here) ...
	viper.SetConfigName("api-gateway")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs/development")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read configuration file", "error", err)
		os.Exit(1)
	}

	// --- Redis Connection for Matchmaking Pool ---
	redisCfg := redis.Config{
		Addr:     viper.GetString("redis.addr"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	}
	rdb, err := redis.NewClient(redisCfg)
	if err != nil {
		slog.Error("Failed to connect to Redis for matchmaking", "error", err)
		os.Exit(1)
	}
	slog.Info("API Gateway Redis connection successful.")

	// --- gRPC Client Initialization ---
	// ... (no changes here) ...
	grpcClients, err := apigateway.NewGRPClients(
		viper.GetString("services.auth_service_addr"),
		viper.GetString("services.player_profile_service_addr"),
	)
	if err != nil {
		slog.Error("Failed to initialize gRPC clients", "error", err)
		os.Exit(1)
	}

	// --- HTTP Router and Middleware Setup ---
	r := chi.NewRouter()
	// ... (middleware setup remains the same) ...
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// --- Route Definitions ---
	// Instantiate the matchmaking pool, which will be shared with the WebSocket handler.
	matchmakingPool := matchmaking.NewPool(rdb, viper.GetString("matchmaking.pool_key"))

	// Instantiate all our HTTP handlers.
	authHandler := auth.NewHTTPHandler(grpcClients.Auth)
	profileHandler := playerprofile.NewHTTPHandler(grpcClients.PlayerProfile)
	matchmakingHandler := matchmaking.NewWebsocketHandler(matchmakingPool) // Create the new WebSocket handler

	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes
		r.Post("/auth/register", authHandler.HandleRegister)
		r.Post("/auth/login", authHandler.HandleLogin)

		// Player Profile routes
		r.Get("/profiles/{userID}", profileHandler.HandleGetProfile)

		// Matchmaking WebSocket route
		// Use .Handle() for WebSocket handlers as it supports the GET request used for the upgrade.
		r.Handle("/matchmaking/find", matchmakingHandler)
	})

	slog.Info("All routes initialized.")

	// --- HTTP Server Initialization and Graceful Shutdown ---
	// ... (no changes here, the rest of the file is the same) ...
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down API Gateway server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown:", "error", err)
	}

	slog.Info("API Gateway server stopped.")
}
