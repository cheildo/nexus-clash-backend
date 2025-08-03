package auth

import (
	"context"
	"errors"
	"log/slog"

	//"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

// GRPCHandler implements the generated AuthServiceServer interface.
type GRPCHandler struct {
	// UnimplementedAuthServiceServer is embedded for forward compatibility.
	// It ensures that if new methods are added to the .proto file, our server won't fail to compile.
	nexusclashv1.UnimplementedAuthServiceServer
	svc Service
}

func NewGRPCHandler(svc Service) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

// Register handles the incoming gRPC request for user registration.
func (h *GRPCHandler) Register(ctx context.Context, req *nexusclashv1.RegisterRequest) (*nexusclashv1.RegisterResponse, error) {
	slog.Info("gRPC Register request received", "email", req.GetEmail(), "username", req.GetUsername())

	// Call the business logic service to perform the registration.
	userID, err := h.svc.Register(ctx, req.GetEmail(), req.GetUsername(), req.GetPassword())
	if err != nil {
		// Map our internal errors to appropriate gRPC status codes.
		if errors.Is(err, ErrEmailOrUserExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		// For any other error, return a generic Internal error to avoid leaking implementation details.
		return nil, status.Error(codes.Internal, "an unexpected error occurred")
	}

	// On success, return the response containing the new user's ID.
	return &nexusclashv1.RegisterResponse{UserId: userID}, nil
}

// Login handles the incoming gRPC request for user login.
func (h *GRPCHandler) Login(ctx context.Context, req *nexusclashv1.LoginRequest) (*nexusclashv1.LoginResponse, error) {
	slog.Info("gRPC Login request received", "email", req.GetEmail())

	// Call the business logic service to perform the login.
	token, err := h.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		// Map our internal errors to appropriate gRPC status codes.
		if errors.Is(err, ErrUserNotFound) {
			return nil, status.Error(codes.NotFound, "invalid credentials")
		}
		// Return a generic Internal error for other failures.
		return nil, status.Error(codes.Internal, "an unexpected error occurred")
	}

	// On success, return the response containing the session token.
	return &nexusclashv1.LoginResponse{SessionToken: token}, nil
}
