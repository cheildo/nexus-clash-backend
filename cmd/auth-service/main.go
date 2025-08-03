package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// Internal packages
	"github.com/cheildo/nexus-clash-backend/internal/auth"
	"github.com/cheildo/nexus-clash-backend/internal/pkg/database"

	// Proto-generated code
	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

func main() {
	// --- Configuration Loading using Viper ---
	// Viper is a popular library for handling application configuration from files, env vars, etc.
	viper.SetConfigName("auth-service") // Name of config file (without extension)
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs/development") // Path to look for the config file in
	viper.AutomaticEnv()                         // Read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read configuration file", "error", err)
		os.Exit(1)
	}

	// --- Database Connection ---
	dbConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		viper.GetString("database.host"),
		viper.GetString("database.port"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.db_name"),
		viper.GetString("database.ssl_mode"),
	)

	db, err := database.NewPostgresDB(dbConnStr)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Database connection successful.")

	// --- Dependency Injection ---
	repo := auth.NewRepository(db)

	// Create the service config by pulling values from Viper.
	svcConfig := auth.Config{
		JWTSecret:     viper.GetString("jwt.secret_key"),
		TokenDuration: viper.GetDuration("jwt.token_duration_minutes") * time.Minute,
	}

	svc := auth.NewService(repo, svcConfig)
	grpcHandler := auth.NewGRPCHandler(svc)

	// --- gRPC Server Initialization ---
	grpcPort := viper.GetString("grpc_server.port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		slog.Error("Failed to listen on gRPC port", "port", grpcPort, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	nexusclashv1.RegisterAuthServiceServer(grpcServer, grpcHandler)
	// Enable gRPC reflection. This is useful for tools like grpcurl to query the server.
	reflection.Register(grpcServer)

	// --- Graceful Shutdown ---
	// This is a critical part of a production service. It allows the server to finish
	// processing current requests before shutting down when it receives a signal.
	go func() {
		slog.Info("Auth gRPC server listening", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed to serve", "error", err)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	slog.Info("gRPC server shut down gracefully.")
}
