package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/cheildo/nexus-clash-backend/internal/matchmaking"
	"github.com/cheildo/nexus-clash-backend/internal/pkg/kafka"
	"github.com/cheildo/nexus-clash-backend/internal/pkg/redis"
)

// Helper function to start the diagnostics server
func startDiagnosticsServer(port string) {
	go func() {
		slog.Info("Starting diagnostics server", "port", port)
		// http.DefaultServeMux already has the pprof handlers registered by the import.
		if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
			slog.Error("Diagnostics server failed to start", "error", err)
		}
	}()
}

func main() {
	// --- Configuration Loading ---
	viper.SetConfigName("matchmaking-service")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs/development")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read configuration file", "error", err)
		os.Exit(1)
	}

	// --- Redis Connection ---
	redisCfg := redis.Config{
		Addr:     viper.GetString("redis.addr"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	}
	rdb, err := redis.NewClient(redisCfg)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("Redis connection successful.")

	// --- Kafka Producer Initialization ---
	producer := kafka.NewProducer(
		viper.GetStringSlice("kafka.brokers"),
		viper.GetString("kafka.match_found_topic"),
	)
	defer producer.Close()

	// --- Dependency Injection ---
	pool := matchmaking.NewPool(rdb, viper.GetString("matchmaking.pool_key"))
	svc := matchmaking.NewService(
		pool,
		producer, // Inject the producer
		viper.GetDuration("matchmaking.check_interval_seconds")*time.Second,
		viper.GetInt("matchmaking.players_per_match"),
	)

	// --- Start Matchmaking Loop ---
	// We create a cancellable context for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	// --- gRPC Server Initialization (for future inter-service communication) ---
	grpcPort := viper.GetString("grpc_server.port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		slog.Error("Failed to listen on gRPC port", "port", grpcPort, "error", err)
		os.Exit(1)
	}
	grpcServer := grpc.NewServer()
	// nexusclashv1.RegisterMatchmakingServiceServer(grpcServer, grpcHandler) // This would be added once the proto is defined.

	go func() {
		slog.Info("Matchmaking gRPC server listening", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed to serve", "error", err)
		}
	}()

	// --- Start Diagnostics Server ---
	diagnosticsPort := viper.GetString("diagnostics.port")
	if diagnosticsPort != "" {
		startDiagnosticsServer(diagnosticsPort)
	}

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down servers...")
	cancel() // Signal the matchmaking loop to stop.
	grpcServer.GracefulStop()
	slog.Info("Servers shut down gracefully.")
}
