package playerprofile

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

type GRPCHandler struct {
	nexusclashv1.UnimplementedPlayerProfileServiceServer
	svc Service
}

func NewGRPCHandler(svc Service) *GRPCHandler {
	return &GRPCHandler{svc: svc}
}

func (h *GRPCHandler) CreateProfile(ctx context.Context, req *nexusclashv1.CreateProfileRequest) (*nexusclashv1.CreateProfileResponse, error) {
	slog.Info("gRPC CreateProfile request received", "userID", req.GetUserId().GetValue())

	profile, err := h.svc.CreateProfile(ctx, req)
	if err != nil {
		if errors.Is(err, ErrUsernameNotAvailable) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		// Add other specific error mappings here
		return nil, status.Error(codes.Internal, "failed to create profile")
	}

	return &nexusclashv1.CreateProfileResponse{Profile: profile}, nil
}

func (h *GRPCHandler) GetProfile(ctx context.Context, req *nexusclashv1.GetProfileRequest) (*nexusclashv1.GetProfileResponse, error) {
	slog.Info("gRPC GetProfile request received", "userID", req.GetUserId().GetValue())

	profile, err := h.svc.GetProfile(ctx, req)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to get profile")
	}

	return &nexusclashv1.GetProfileResponse{Profile: profile}, nil
}
