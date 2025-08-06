package apigateway

import (
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

// Clients holds the gRPC clients for all backend services.
type Clients struct {
	Auth nexusclashv1.AuthServiceClient
	// We will add more clients here, e.g., for PlayerProfileService
}

// NewGRPClients creates and returns gRPC clients for all backend services.
// It is responsible for establishing connections to the gRPC servers.
func NewGRPClients(authServiceAddr string) (*Clients, error) {
	// Create a gRPC connection to the Auth Service.
	// We use `insecure.NewCredentials()` because we are running in a trusted internal network (e.g., Docker, Kubernetes).
	// In a production environment with services across different networks, you'd use TLS credentials.
	authConn, err := grpc.Dial(
		authServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // This makes the connection synchronous; it will wait until connected.
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("Failed to connect to auth service", "address", authServiceAddr, "error", err)
		return nil, err
	}
	slog.Info("Successfully connected to auth gRPC service", "address", authServiceAddr)

	// Create a client stub for the Auth Service using the connection.
	authClient := nexusclashv1.NewAuthServiceClient(authConn)

	return &Clients{
		Auth: authClient,
	}, nil
}
