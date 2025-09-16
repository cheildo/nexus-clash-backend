package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/cheildo/nexus-clash-backend/internal/orchestration"
	"github.com/cheildo/nexus-clash-backend/internal/pkg/kafka"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

// Main application struct to hold dependencies.
type application struct {
	grpcServer *grpc.Server
	listener   *orchestration.Listener
}

func main() {
	// --- Configuration ---
	viper.SetConfigName("game-orchestration-service")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs/development")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read configuration file", "error", err)
		os.Exit(1)
	}

	// --- Kafka Initialization ---
	consumer := kafka.NewConsumer(
		viper.GetStringSlice("kafka.brokers"),
		viper.GetString("kafka.match_found_topic"),
		viper.GetString("kafka.consumer_group_id"),
	)
	producer := kafka.NewProducer(
		viper.GetStringSlice("kafka.brokers"),
		viper.GetString("kafka.server_ready_topic"),
	)

	// --- Dependency Injection ---
	listener := orchestration.NewListener(consumer, producer)
	grpcHandler := orchestration.NewGRPCHandler(listener)

	app := &application{
		grpcServer: grpc.NewServer(),
		listener:   listener,
	}

	// --- Start Servers ---
	ctx, cancel := context.WithCancel(context.Background())

	go app.startGRPCServer(ctx, grpcHandler, viper.GetString("grpc_server.port"))
	go app.listener.Run(ctx)

	startDiagnosticsServer(viper.GetString("diagnostics.port"))

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down servers...")
	cancel() // Signal goroutines to stop
	app.grpcServer.GracefulStop()
	slog.Info("Servers shut down gracefully.")
}

func (app *application) startGRPCServer(ctx context.Context, handler *orchestration.GRPCHandler, port string) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		slog.Error("Failed to listen on gRPC port", "port", port, "error", err)
		os.Exit(1)
	}

	nexusclashv1.RegisterGameOrchestrationServiceServer(app.grpcServer, handler)

	slog.Info("Orchestration gRPC server listening", "address", lis.Addr().String())
	if err := app.grpcServer.Serve(lis); err != nil {
		slog.Error("gRPC server failed to serve", "error", err)
	}
}

func startDiagnosticsServer(port string) {
	go func() {
		slog.Info("Starting diagnostics server", "port", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
			slog.Error("Diagnostics server failed to start", "error", err)
		}
	}()
}
