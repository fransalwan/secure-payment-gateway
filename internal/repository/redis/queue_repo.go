package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/fransalwan/secure-payment-gateway/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	redisPendingListKey = "spg:oms:queue:pending"
	redisDelayedZSetKey = "spg:oms:queue:delayed"
	redisJobHashKey     = "spg:oms:queue:jobs"
)

// QueueRepository implements a BullMQ-inspired Asynchronous Job Queue backed by Redis
// Lists & Sorted Sets (with an automatic thread-safe in-memory fallback when Redis is offline).
type QueueRepository struct {
	client       *redis.Client
	mu           sync.Mutex
	jobs         map[string]*domain.QueueJob
	pendingQueue []string
}

func NewQueueRepository(client *redis.Client) *QueueRepository {
	return &QueueRepository{
		client:       client,
		jobs:         make(map[string]*domain.QueueJob),
		pendingQueue: make([]string, 0),
	}
}

func (q *QueueRepository) BackendName() string {
	if q.client != nil {
		return "redis-streams-zset"
	}
	return "in-memory-hybrid-fallback"
}

func cloneJob(j *domain.QueueJob) *domain.QueueJob {
	if j == nil {
		return nil
	}
	cp := *j
	if j.NextRetryAt != nil {
		t := *j.NextRetryAt
		cp.NextRetryAt = &t
	}
	if len(j.Payload) > 0 {
		cp.Payload = append(json.RawMessage(nil), j.Payload...)
	}
	return &cp
}

func (q *QueueRepository) Enqueue(ctx context.Context, job *domain.QueueJob) error {
	now := time.Now().UTC()
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 4
	}
	job.Status = domain.JobStatusPending
	job.CreatedAt = now
	job.UpdatedAt = now

	q.mu.Lock()
	q.jobs[job.ID] = cloneJob(job)
	q.pendingQueue = append(q.pendingQueue, job.ID)
	q.mu.Unlock()

	if q.client != nil {
		raw, err := json.Marshal(job)
		if err == nil {
			pipe := q.client.Pipeline()
			pipe.HSet(ctx, redisJobHashKey, job.ID, raw)
			pipe.LPush(ctx, redisPendingListKey, job.ID)
			_, _ = pipe.Exec(ctx)
		}
	}
	return nil
}

// PopNextDueJob promotes delayed retry jobs whose backoff timer has elapsed and pops the next pending job.
func (q *QueueRepository) PopNextDueJob(ctx context.Context) (*domain.QueueJob, error) {
	now := time.Now().UTC()

	// Promote delayed retries in Redis if connected
	if q.client != nil {
		dueIDs, err := q.client.ZRangeByScore(ctx, redisDelayedZSetKey, &redis.ZRangeBy{
			Min: "-inf",
			Max: fmt.Sprintf("%d", now.UnixMilli()),
		}).Result()
		if err == nil && len(dueIDs) > 0 {
			for _, id := range dueIDs {
				removed, _ := q.client.ZRem(ctx, redisDelayedZSetKey, id).Result()
				if removed > 0 {
					_ = q.client.LPush(ctx, redisPendingListKey, id).Err()
				}
			}
		}
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	// Promote due retrying jobs in memory state
	for _, j := range q.jobs {
		if j.Status == domain.JobStatusRetrying && j.NextRetryAt != nil && !now.Before(*j.NextRetryAt) {
			j.Status = domain.JobStatusPending
			j.NextRetryAt = nil
			j.UpdatedAt = now
			q.pendingQueue = append(q.pendingQueue, j.ID)
		}
	}

	for len(q.pendingQueue) > 0 {
		nextID := q.pendingQueue[0]
		q.pendingQueue = q.pendingQueue[1:]

		if q.client != nil {
			_ = q.client.LRem(ctx, redisPendingListKey, 1, nextID).Err()
		}

		j, exists := q.jobs[nextID]
		if !exists || j.Status != domain.JobStatusPending {
			continue
		}

		j.Status = domain.JobStatusActive
		j.Attempts++
		j.UpdatedAt = now
		q.syncJobToRedis(ctx, j)
		return cloneJob(j), nil
	}

	return nil, nil
}

func (q *QueueRepository) MarkCompleted(ctx context.Context, jobID string) {
	now := time.Now().UTC()
	q.mu.Lock()
	if j, exists := q.jobs[jobID]; exists {
		j.Status = domain.JobStatusCompleted
		j.LastError = ""
		j.NextRetryAt = nil
		j.UpdatedAt = now
		q.syncJobToRedis(ctx, j)
	}
	q.mu.Unlock()
}

// FailOrScheduleRetry applies Exponential Backoff: baseDelay * 2^(attempts-1) -> e.g. 2s, 4s, 8s, 16s.
// Once attempts >= max_attempts (4), moves job to FAILED (Dead-Letter Queue) and returns movedToDLQ = true.
func (q *QueueRepository) FailOrScheduleRetry(
	ctx context.Context,
	jobID string,
	errReason string,
	baseDelay time.Duration,
) (movedToDLQ bool, updatedJob *domain.QueueJob) {
	now := time.Now().UTC()
	if baseDelay <= 0 {
		baseDelay = 2 * time.Second
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	j, exists := q.jobs[jobID]
	if !exists {
		return false, nil
	}

	j.LastError = errReason
	j.UpdatedAt = now

	if j.Attempts >= j.MaxAttempts {
		j.Status = domain.JobStatusFailedDLQ
		j.NextRetryAt = nil
		q.syncJobToRedis(ctx, j)
		return true, cloneJob(j)
	}

	multiplier := math.Pow(2, float64(j.Attempts-1))
	backoff := time.Duration(float64(baseDelay) * multiplier)
	retryAt := now.Add(backoff)

	j.Status = domain.JobStatusRetrying
	j.NextRetryAt = &retryAt
	q.syncJobToRedis(ctx, j)

	if q.client != nil {
		_ = q.client.ZAdd(ctx, redisDelayedZSetKey, redis.Z{
			Score:  float64(retryAt.UnixMilli()),
			Member: j.ID,
		}).Err()
	}

	return false, cloneJob(j)
}

func (q *QueueRepository) ManualRetry(ctx context.Context, jobID string) (*domain.QueueJob, error) {
	now := time.Now().UTC()
	q.mu.Lock()
	defer q.mu.Unlock()

	j, exists := q.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	j.Attempts = 0
	j.LastError = ""
	j.NextRetryAt = nil
	j.Status = domain.JobStatusPending
	j.UpdatedAt = now
	q.pendingQueue = append(q.pendingQueue, j.ID)
	q.syncJobToRedis(ctx, j)

	if q.client != nil {
		_ = q.client.LPush(ctx, redisPendingListKey, j.ID).Err()
	}

	return cloneJob(j), nil
}

func (q *QueueRepository) GetMetrics(chaosMode domain.ChaosFailMode) domain.QueueMetricsDTO {
	q.mu.Lock()
	defer q.mu.Unlock()

	var active, retrying, completed, failed int
	all := make([]*domain.QueueJob, 0, len(q.jobs))

	for _, j := range q.jobs {
		switch j.Status {
		case domain.JobStatusActive, domain.JobStatusPending:
			active++
		case domain.JobStatusRetrying:
			retrying++
		case domain.JobStatusCompleted:
			completed++
		case domain.JobStatusFailedDLQ:
			failed++
		}
		all = append(all, cloneJob(j))
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})

	if len(all) > 35 {
		all = all[:35]
	}

	return domain.QueueMetricsDTO{
		ActiveCount:    active,
		RetryingCount:  retrying,
		CompletedCount: completed,
		FailedDLQCount: failed,
		ChaosMode:      chaosMode,
		QueueBackend:   q.BackendName(),
		RecentJobs:     all,
	}
}

func (q *QueueRepository) syncJobToRedis(ctx context.Context, j *domain.QueueJob) {
	if q.client == nil {
		return
	}
	if raw, err := json.Marshal(j); err == nil {
		_ = q.client.HSet(ctx, redisJobHashKey, j.ID, raw).Err()
	}
}

