package orchestration

// MatchFoundEvent is the data structure for incoming events.
type MatchFoundEvent struct {
	MatchID   string   `json:"matchID"`
	PlayerIDs []string `json:"playerIDs"`
}

// GameServerReadyEvent is the payload for our outgoing events.
type GameServerReadyEvent struct {
	MatchID    string   `json:"matchID"`
	PlayerIDs  []string `json:"playerIDs"`
	ServerAddr string   `json:"serverAddr"` // The crucial address of the game server.
	ServerPort string   `json:"serverPort"`
}
