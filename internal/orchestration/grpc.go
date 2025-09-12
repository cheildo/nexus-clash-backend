package orchestration

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

type GRPCHandler struct {
	nexusclashv1.UnimplementedGameOrchestrationServiceServer
	listener *Listener
}

func NewGRPCHandler(listener *Listener) *GRPCHandler {
	return &GRPCHandler{listener: listener}
}

func (h *GRPCHandler) GetStatus(ctx context.Context, req *emptypb.Empty) (*nexusclashv1.StatusResponse, error) {
	return &nexusclashv1.StatusResponse{
		Status:         "OK",
		RunningServers: h.listener.GetRunningServers(),
	}, nil
}
