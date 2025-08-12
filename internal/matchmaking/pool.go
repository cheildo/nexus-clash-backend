package matchmaking

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Pool represents the matchmaking pool stored in Redis.
type Pool interface {
	AddPlayer(ctx context.Context, playerID string) error
	RemovePlayer(ctx context.Context, playerID string) error
	FindMatch(ctx context.Context, requiredPlayers int) ([]string, error)
}

type redisPool struct {
	rdb     *redis.Client
	poolKey string
}

func NewPool(rdb *redis.Client, poolKey string) Pool {
	return &redisPool{
		rdb:     rdb,
		poolKey: poolKey,
	}
}

// AddPlayer adds a player to the matchmaking pool (a Redis Sorted Set).
// The score is the timestamp, so we can find players who have waited the longest.
func (p *redisPool) AddPlayer(ctx context.Context, playerID string) error {
	score := float64(time.Now().Unix())
	_, err := p.rdb.ZAdd(ctx, p.poolKey, redis.Z{Score: score, Member: playerID}).Result()
	if err != nil {
		slog.Error("Failed to add player to Redis pool", "playerID", playerID, "error", err)
		return err
	}
	slog.Info("Player added to matchmaking pool", "playerID", playerID)
	return nil
}

// RemovePlayer removes a player from the matchmaking pool. This is used when they cancel or a match is found.
func (p *redisPool) RemovePlayer(ctx context.Context, playerID string) error {
	_, err := p.rdb.ZRem(ctx, p.poolKey, playerID).Result()
	if err != nil {
		slog.Error("Failed to remove player from Redis pool", "playerID", playerID, "error", err)
	}
	slog.Info("Player removed from matchmaking pool", "playerID", playerID)
	return err
}

// FindMatch attempts to find enough players to form a match.
func (p *redisPool) FindMatch(ctx context.Context, requiredPlayers int) ([]string, error) {
	// ZCard gets the total number of players in the pool.
	count, err := p.rdb.ZCard(ctx, p.poolKey).Result()
	if err != nil {
		return nil, err
	}

	// If we don't have enough players, there's no match.
	if count < int64(requiredPlayers) {
		return nil, nil // Not an error, just no match found yet.
	}

	// ZRange gets a range of members from the sorted set. We get the first `requiredPlayers` members,
	// which are the ones who have been waiting the longest (lowest score/timestamp).
	playerIDs, err := p.rdb.ZRange(ctx, p.poolKey, 0, int64(requiredPlayers-1)).Result()
	if err != nil {
		return nil, err
	}

	// Important: Once we've identified the players for a match, we must remove them from the pool
	// to prevent them from being matched into another game simultaneously.
	// ZRem is variadic, so we can remove multiple members at once.
	// We convert our slice of strings to a slice of interface{} for the ZRem function.
	members := make([]interface{}, len(playerIDs))
	for i, v := range playerIDs {
		members[i] = v
	}
	if _, err := p.rdb.ZRem(ctx, p.poolKey, members...).Result(); err != nil {
		// If this fails, we should ideally add the players back to not lose them.
		// For now, we'll log the error.
		slog.Error("CRITICAL: Failed to remove matched players from pool", "error", err)
		return nil, err
	}

	slog.Info("Match found!", "player_count", len(playerIDs), "players", playerIDs)
	return playerIDs, nil
}
