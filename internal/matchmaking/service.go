package matchmaking

import (
	"context"
	"log/slog"
	"time"
)

// Service orchestrates the matchmaking process.
type Service struct {
	pool            Pool
	checkInterval   time.Duration
	playersPerMatch int
}

// NewService creates a new matchmaking service.
func NewService(pool Pool, checkInterval time.Duration, playersPerMatch int) *Service {
	return &Service{
		pool:            pool,
		checkInterval:   checkInterval,
		playersPerMatch: playersPerMatch,
	}
}

// Start runs the main matchmaking loop in a separate goroutine.
// It periodically checks the pool for potential matches.
func (s *Service) Start(ctx context.Context) {
	slog.Info("Matchmaking service loop started", "interval", s.checkInterval)
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Matchmaking service loop stopping.")
				return
			case <-ticker.C:
				s.findAndProcessMatches(ctx)
			}
		}
	}()
}

func (s *Service) findAndProcessMatches(ctx context.Context) {
	players, err := s.pool.FindMatch(ctx, s.playersPerMatch)
	if err != nil {
		slog.Error("Error finding match", "error", err)
		return
	}

	if players == nil {
		// No match found, which is a normal occurrence.
		return
	}

	slog.Info("Processing found match", "players", players)

	// TODO:
	// 1. Generate a unique Match ID.
	// 2. Call the Game Orchestration Service via gRPC to request a new game server instance.
	// 3. For each player in the `players` slice, find their active WebSocket connection
	//    and send them the "MATCH_FOUND" message with the connection details.
}
