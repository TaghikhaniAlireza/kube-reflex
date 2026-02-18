//internal/redis/client.go
package redis

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	opts := &redis.Options{
		Addr: addr,
		DB:   0,
	}
	if pw := os.Getenv("REDIS_PASSWORD"); pw != "" {
		opts.Password = pw
	}

	rdb := redis.NewClient(opts)

	// Health check
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return rdb, nil
}