//internal/redis/repository.go
package redis

import (
	"context"
	"time"
)

type Repository interface {
	IncrementScore(ctx context.Context, key string, delta int, ttl time.Duration) (int64, error)
	GetScore(ctx context.Context, key string) (int64, error)
	ResetScore(ctx context.Context, key string) error
}