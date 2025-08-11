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
	Auth          nexusclashv1.AuthServiceClient
	PlayerProfile nexusclashv1.PlayerProfileServiceClient // Added PlayerProfile client
}

// NewGRPClients creates and returns gRPC clients for all backend services.
func NewGRPClients(authServiceAddr, profileServiceAddr string) (*Clients, error) {
	// --- Connect to Auth Service ---
	authConn, err := grpc.Dial(
		authServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("Failed to connect to auth service", "address", authServiceAddr, "error", err)
		return nil, err
	}
	slog.Info("Successfully connected to auth gRPC service", "address", authServiceAddr)
	authClient := nexusclashv1.NewAuthServiceClient(authConn)

	// --- Connect to Player Profile Service ---
	profileConn, err := grpc.Dial(
		profileServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		slog.Error("Failed to connect to player profile service", "address", profileServiceAddr, "error", err)
		return nil, err
	}
	slog.Info("Successfully connected to player profile gRPC service", "address", profileServiceAddr)
	profileClient := nexusclashv1.NewPlayerProfileServiceClient(profileConn)

	return &Clients{
		Auth:          authClient,
		PlayerProfile: profileClient, // Added the new client to the struct
	}, nil
}
