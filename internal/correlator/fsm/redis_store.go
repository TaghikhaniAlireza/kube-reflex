// internal/correlator/fsm/redis_store.go
package fsm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) key(containerID, chainID string) string {
	return fmt.Sprintf("fsm:%s:%s", containerID, chainID)
}

func (s *RedisStore) Create(ctx context.Context, containerID, chainID, tactic string, ttl time.Duration) error {
	// Uses the State struct defined in state.go (same package)
	state := State{
		Step:       0,
		LastTactic: tactic,
		LastSeen:   time.Now().Unix(),
		StartedAt:  time.Now().Unix(),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	key := s.key(containerID, chainID)

	pipe := s.client.Pipeline()
	pipe.Set(ctx, key, data, 0)
	pipe.Expire(ctx, key, ttl)
	_, err = pipe.Exec(ctx)

	return err
}

func (s *RedisStore) Get(ctx context.Context, containerID, chainID string) (*State, error) {
	key := s.key(containerID, chainID)
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

const promoteMaxRetries = 100

func (s *RedisStore) Promote(ctx context.Context, containerID, chainID, tactic string, newStep int, ttl time.Duration) error {
	key := s.key(containerID, chainID)

	txf := func(tx *redis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err == redis.Nil {
			return fmt.Errorf("state not found for promote")
		}
		if err != nil {
			return err
		}

		var state State
		if err := json.Unmarshal([]byte(val), &state); err != nil {
			return err
		}

		state.Step = newStep
		state.LastTactic = tactic
		state.LastSeen = time.Now().Unix()

		data, err := json.Marshal(state)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, data, 0)
			pipe.Expire(ctx, key, ttl)
			return nil
		})
		return err
	}

	for i := 0; i < promoteMaxRetries; i++ {
		err := s.client.Watch(ctx, txf, key)
		if err == nil {
			return nil
		}
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return err
	}
	return fmt.Errorf("promote: max retries (%d) exceeded", promoteMaxRetries)
}

func (s *RedisStore) Delete(ctx context.Context, containerID, chainID string) error {
	return s.client.Del(ctx, s.key(containerID, chainID)).Err()
}