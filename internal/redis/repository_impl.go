// internal/redis/repository_impl.go
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisRepository struct {
	client *redis.Client
}

func NewRepository(client *redis.Client) Repository {
	return &redisRepository{client: client}
}

func (r *redisRepository) IncrementScore(
	ctx context.Context,
	key string,
	delta int,
	ttl time.Duration,
) (int64, error) {

	pipe := r.client.TxPipeline()

	incr := pipe.IncrBy(ctx, key, int64(delta))
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

func (r *redisRepository) GetScore(
	ctx context.Context,
	key string,
) (int64, error) {

	val, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (r *redisRepository) ResetScore(
	ctx context.Context,
	key string,
) error {
	return r.client.Del(ctx, key).Err()
}