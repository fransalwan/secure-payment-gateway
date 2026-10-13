package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/config"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("Redis server not reachable, idempotency engine will use in-memory fallback", "err", err)
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	slog.Info("Redis connection established successfully", "addr", cfg.RedisAddr)
	return client, nil
}

