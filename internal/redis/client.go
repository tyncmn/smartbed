// Package redisclient provides a Redis client instance.
package redisclient

import (
	"context"
	"fmt"

	"smartbed/internal/config"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// New creates and pings a Redis client.
func New(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	log.Info().Msg("Redis connected")
	return rdb, nil
}
