package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Config holds the configuration required to connect to Redis.
type Config struct {
	Addr     string
	Password string
	DB       int
}

// NewClient creates a new Redis client and pings it to ensure connectivity.
func NewClient(cfg Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Ping the server to check the connection.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}
