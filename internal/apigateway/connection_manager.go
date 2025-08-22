package apigateway

import (
	"sync"

	"github.com/gorilla/websocket"
)

// ConnectionManager safely stores and retrieves active WebSocket connections.
type ConnectionManager struct {
	connections sync.Map // A thread-safe map: map[playerID]*websocket.Conn
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{}
}

func (cm *ConnectionManager) Add(playerID string, conn *websocket.Conn) {
	cm.connections.Store(playerID, conn)
}

func (cm *ConnectionManager) Remove(playerID string) {
	cm.connections.Delete(playerID)
}

func (cm *ConnectionManager) Get(playerID string) (*websocket.Conn, bool) {
	conn, ok := cm.connections.Load(playerID)
	if !ok {
		return nil, false
	}
	return conn.(*websocket.Conn), true
}
