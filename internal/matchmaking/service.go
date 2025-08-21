package matchmaking

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// MatchFoundEvent is the payload we will send to Kafka.
type MatchFoundEvent struct {
	MatchID   string   `json:"matchID"`
	PlayerIDs []string `json:"playerIDs"`
}

// Service orchestrates the matchmaking process.
type Service struct {
	pool            Pool
	checkInterval   time.Duration
	playersPerMatch int
	producer        *kafka.Writer // Added Kafka producer
}

// NewService creates a new matchmaking service.
func NewService(pool Pool, producer *kafka.Writer, checkInterval time.Duration, playersPerMatch int) *Service {
	return &Service{
		pool:            pool,
		producer:        producer,
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
		return
	}

	slog.Info("Processing found match", "players", players)

	// 1. Generate a unique Match ID.
	matchID := uuid.New().String()

	// 2. Create the event payload.
	event := MatchFoundEvent{
		MatchID:   matchID,
		PlayerIDs: players,
	}

	// 3. Marshal the event to JSON.
	eventBytes, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal MatchFoundEvent", "error", err)
		// Ideally, we'd add the players back to the pool here.
		return
	}

	// 4. Publish the event to Kafka.
	err = s.producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(matchID), // Use matchID as the key for partitioning.
		Value: eventBytes,
	})
	if err != nil {
		slog.Error("Failed to write message to Kafka", "error", err)
		// Critical error: handle this with retries or an alert system.
	} else {
		slog.Info("MatchFoundEvent published to Kafka", "matchID", matchID)
	}

	// The TODO is now complete! This service's job is done for this match.
}
