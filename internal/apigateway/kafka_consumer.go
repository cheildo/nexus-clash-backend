package apigateway

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// MatchFoundEvent defines the structure of the message we expect from Kafka.
type MatchFoundEvent struct {
	MatchID   string   `json:"matchID"`
	PlayerIDs []string `json:"playerIDs"`
}

// MatchmakingConsumer listens for matchmaking events from Kafka.
type MatchmakingConsumer struct {
	reader *kafka.Reader
	cm     *ConnectionManager
}

func NewMatchmakingConsumer(reader *kafka.Reader, cm *ConnectionManager) *MatchmakingConsumer {
	return &MatchmakingConsumer{
		reader: reader,
		cm:     cm,
	}
}

// Run starts the consumer loop. It should be run in a goroutine.
func (mc *MatchmakingConsumer) Run(ctx context.Context) {
	slog.Info("Kafka consumer loop started")
	for {
		// The ReadMessage call blocks until a message is available or an error occurs.
		msg, err := mc.reader.ReadMessage(ctx)
		if err != nil {
			// Check if the context was cancelled, indicating a graceful shutdown.
			if ctx.Err() != nil {
				slog.Info("Kafka consumer context cancelled. Shutting down.")
				break
			}
			slog.Error("Error reading from Kafka", "error", err)
			continue // Continue to the next message on error
		}

		slog.Info("Received match_found event from Kafka", "key", string(msg.Key))

		var event MatchFoundEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			slog.Error("Failed to unmarshal Kafka message", "error", err)
			continue
		}

		// Notify each player in the match.
		for _, playerID := range event.PlayerIDs {
			conn, ok := mc.cm.Get(playerID)
			if !ok {
				slog.Warn("Could not find active WebSocket for player in match", "playerID", playerID)
				continue
			}

			// Send the message to the client over their WebSocket.
			// The message format is up to you; JSON is a good choice.
			notification := map[string]interface{}{
				"type":    "MATCH_FOUND",
				"matchID": event.MatchID,
				// In a real game, you would include the game server IP and port here.
			}

			if err := conn.WriteJSON(notification); err != nil {
				slog.Warn("Failed to send MATCH_FOUND notification to client", "playerID", playerID, "error", err)
			} else {
				slog.Info("Successfully sent MATCH_FOUND notification", "playerID", playerID)
			}
		}
	}
	mc.reader.Close()
	slog.Info("Kafka consumer stopped.")
}
