package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"net/http"
	_ "net/http/pprof"

	// Internal packages
	"github.com/cheildo/nexus-clash-backend/internal/pkg/database"
	"github.com/cheildo/nexus-clash-backend/internal/playerprofile"

	// Proto-generated code
	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

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
	viper.SetConfigName("player-profile-service")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs/development")
	viper.AutomaticEnv()

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
	repo := playerprofile.NewRepository(db)
	svc := playerprofile.NewService(repo)
	grpcHandler := playerprofile.NewGRPCHandler(svc)

	// --- gRPC Server Initialization ---
	grpcPort := viper.GetString("grpc_server.port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		slog.Error("Failed to listen on gRPC port", "port", grpcPort, "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	nexusclashv1.RegisterPlayerProfileServiceServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	// --- Start Diagnostics Server ---
	diagnosticsPort := viper.GetString("diagnostics.port")
	if diagnosticsPort != "" {
		startDiagnosticsServer(diagnosticsPort)
	}

	// --- Graceful Shutdown ---
	go func() {
		slog.Info("PlayerProfile gRPC server listening", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed to serve", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	slog.Info("PlayerProfile gRPC server shut down gracefully.")
}
