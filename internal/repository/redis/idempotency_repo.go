package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type LockStatus string

const (
	LockAcquired   LockStatus = "ACQUIRED"
	LockInProgress LockStatus = "IN_PROGRESS"
	LockCompleted  LockStatus = "COMPLETED"
)

const (
	statusProcessing = "PROCESSING"
)

type IdempotencyRepository interface {
	LockOrGet(ctx context.Context, key string, ttl time.Duration) (LockStatus, []byte, error)
	SaveCompleted(ctx context.Context, key string, response []byte, ttl time.Duration) error
	Release(ctx context.Context, key string) error
}

type RedisIdempotencyRepository struct {
	client    *redis.Client
	mu        sync.RWMutex
	memoryMap map[string]memoryEntry
}

type memoryEntry struct {
	value     []byte
	status    LockStatus
	expiresAt time.Time
}

func NewIdempotencyRepository(client *redis.Client) IdempotencyRepository {
	return &RedisIdempotencyRepository{
		client:    client,
		memoryMap: make(map[string]memoryEntry),
	}
}

func (r *RedisIdempotencyRepository) LockOrGet(ctx context.Context, key string, ttl time.Duration) (LockStatus, []byte, error) {
	redisKey := fmt.Sprintf("idempotency:%s", key)

	if r.client != nil {
		success, err := r.client.SetNX(ctx, redisKey, statusProcessing, ttl).Result()
		if err != nil {
			return "", nil, fmt.Errorf("redis error during SetNX: %w", err)
		}

		if success {
			return LockAcquired, nil, nil
		}

		val, err := r.client.Get(ctx, redisKey).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return LockAcquired, nil, nil
			}
			return "", nil, fmt.Errorf("redis error during Get: %w", err)
		}

		if string(val) == statusProcessing {
			return LockInProgress, nil, nil
		}

		return LockCompleted, val, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry, exists := r.memoryMap[key]
	if exists && entry.expiresAt.After(now) {
		if entry.status == LockInProgress {
			return LockInProgress, nil, nil
		}
		return LockCompleted, entry.value, nil
	}

	r.memoryMap[key] = memoryEntry{
		status:    LockInProgress,
		expiresAt: now.Add(ttl),
	}
	return LockAcquired, nil, nil
}

func (r *RedisIdempotencyRepository) SaveCompleted(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	redisKey := fmt.Sprintf("idempotency:%s", key)

	if r.client != nil {
		return r.client.Set(ctx, redisKey, response, ttl).Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.memoryMap[key] = memoryEntry{
		value:     response,
		status:    LockCompleted,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (r *RedisIdempotencyRepository) Release(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("idempotency:%s", key)

	if r.client != nil {
		return r.client.Del(ctx, redisKey).Err()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.memoryMap, key)
	return nil
}

