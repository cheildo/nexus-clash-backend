package playerprofile

import (
	"context"
	"errors"

	nexusclashv1 "github.com/cheildo/nexus-clash-backend/api/proto/nexusclash/v1"
)

// Service defines the business logic for player profiles.
type Service interface {
	CreateProfile(ctx context.Context, req *nexusclashv1.CreateProfileRequest) (*nexusclashv1.Profile, error)
	GetProfile(ctx context.Context, req *nexusclashv1.GetProfileRequest) (*nexusclashv1.Profile, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateProfile(ctx context.Context, req *nexusclashv1.CreateProfileRequest) (*nexusclashv1.Profile, error) {
	// Input validation
	if req.GetUserId() == nil || req.GetUserId().GetValue() == "" {
		return nil, errors.New("user_id is required")
	}
	if req.GetUsername() == "" {
		return nil, errors.New("username is required")
	}

	return s.repo.CreateProfile(ctx, req.GetUserId().GetValue(), req.GetUsername())
}

func (s *service) GetProfile(ctx context.Context, req *nexusclashv1.GetProfileRequest) (*nexusclashv1.Profile, error) {
	if req.GetUserId() == nil || req.GetUserId().GetValue() == "" {
		return nil, errors.New("user_id is required")
	}

	return s.repo.GetProfile(ctx, req.GetUserId().GetValue())
}
