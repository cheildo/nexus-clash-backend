package matchmaking

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// upgrader is used to upgrade an HTTP connection to a persistent WebSocket connection.
var upgrader = websocket.Upgrader{
	// Allow connections from any origin (for development).
	// In production, you should restrict this to your game client's origin.
	CheckOrigin: func(r *http.Request) bool { return true },
	// Specify read/write buffer sizes.
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebsocketHandler handles the WebSocket connection for matchmaking.
type WebsocketHandler struct {
	pool Pool
}

func NewWebsocketHandler(pool Pool) *WebsocketHandler {
	return &WebsocketHandler{pool: pool}
}

// ServeHTTP is the entry point for an HTTP request. It upgrades the connection and handles it.
func (h *WebsocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// For this to be a real-world project, we must authenticate the user here.
	// A production implementation would use a middleware to validate the JWT passed
	// in a header (e.g., `Authorization: Bearer <token>`) or query parameter.
	// For now, we will simulate this by getting the player ID from a query parameter.
	playerID := r.URL.Query().Get("playerID")
	if playerID == "" {
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection to WebSocket", "error", err)
		return
	}
	slog.Info("WebSocket connection established", "playerID", playerID)

	// Once the connection is established, add the player to the matchmaking pool.
	if err := h.pool.AddPlayer(r.Context(), playerID); err != nil {
		slog.Error("Failed to add player to pool", "playerID", playerID, "error", err)
		conn.Close()
		return
	}

	// The `handleConnection` method runs for the lifetime of the connection.
	h.handleConnection(conn, playerID)
}

// handleConnection manages a single WebSocket connection.
func (h *WebsocketHandler) handleConnection(conn *websocket.Conn, playerID string) {
	// The defer statement is crucial. It ensures that when the connection is closed for any reason
	// (client disconnects, error, etc.), we clean up by removing the player from the pool.
	defer func() {
		slog.Info("Closing WebSocket connection and removing player from pool", "playerID", playerID)
		h.pool.RemovePlayer(context.Background(), playerID)
		conn.Close()
	}()

	// Set a deadline for reading the next message from the client.
	// If no message is received (e.g., a ping), the connection is considered dead.
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// This is the "read pump". It's an infinite loop that waits for messages from the client.
	// We don't expect many messages, but this loop is necessary to detect when the client closes the connection.
	// When `conn.ReadMessage()` returns an error, it signifies the connection is broken, and the loop will exit.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket connection closed unexpectedly", "playerID", playerID, "error", err)
			}
			break // Exit the loop on any error, triggering the defer.
		}
		// In a real game, you might handle incoming messages here, e.g., "cancel_matchmaking".
	}
}
