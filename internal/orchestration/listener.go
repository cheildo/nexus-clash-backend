package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

// Listener is the main component that listens to Kafka and orchestrates games.
type Listener struct {
	consumer       *kafka.Reader
	producer       *kafka.Writer
	runningServers *atomic.Int64 // Safely count running servers
}

func NewListener(consumer *kafka.Reader, producer *kafka.Writer) *Listener {
	return &Listener{
		consumer:       consumer,
		producer:       producer,
		runningServers: &atomic.Int64{},
	}
}

// Run starts the Kafka consumer loop. It should be run in a goroutine.
func (l *Listener) Run(ctx context.Context) {
	slog.Info("Orchestration listener started")
	defer l.consumer.Close()
	defer l.producer.Close()

	for {
		msg, err := l.consumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break // Context cancelled, graceful shutdown.
			}
			slog.Error("Error reading from Kafka", "error", err)
			continue
		}

		var event MatchFoundEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			slog.Error("Failed to unmarshal match_found event", "error", err)
			continue
		}

		go l.provisionGameServer(ctx, event)
	}
	slog.Info("Orchestration listener stopped.")
}

// provisionGameServer simulates the process of starting a new server.
func (l *Listener) provisionGameServer(ctx context.Context, event MatchFoundEvent) {
	slog.Info("Provisioning new game server...", "matchID", event.MatchID)
	l.runningServers.Add(1)
	defer l.runningServers.Add(-1)

	// --- SIMULATION ---
	// In a real system, this is where you'd call the Docker SDK or Kubernetes API.
	// This process could take several seconds. We simulate a delay.
	time.Sleep(2 * time.Second)

	gameServerAddr := "localhost"
	gameServerPort := "7777" // Placeholder port

	slog.Info("Game server provisioned successfully", "matchID", event.MatchID, "address", fmt.Sprintf("%s:%s", gameServerAddr, gameServerPort))

	// --- PUBLISH RESULT ---
	readyEvent := GameServerReadyEvent{
		MatchID:    event.MatchID,
		PlayerIDs:  event.PlayerIDs,
		ServerAddr: gameServerAddr,
		ServerPort: gameServerPort,
	}

	eventBytes, err := json.Marshal(readyEvent)
	if err != nil {
		slog.Error("Failed to marshal game_server_ready event", "error", err)
		return
	}

	err = l.producer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.MatchID),
		Value: eventBytes,
	})

	if err != nil {
		slog.Error("Failed to publish game_server_ready event", "error", err)
	} else {
		slog.Info("Published game_server_ready event", "matchID", event.MatchID)
	}
}

// GetRunningServers provides a thread-safe way to check the count.
func (l *Listener) GetRunningServers() int64 {
	return l.runningServers.Load()
}
