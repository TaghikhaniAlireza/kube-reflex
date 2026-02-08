//internal/correlator/fsm/redis_store.go
package fsm

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *redis.Client
}

func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func key(containerID, chainID string) string {
	return "fsm:" + containerID + ":" + chainID
}

func (s *Store) Get(ctx context.Context, containerID, chainID string) (*State, error) {
	res, err := s.rdb.HGetAll(ctx, key(containerID, chainID)).Result()
	if err != nil || len(res) == 0 {
		return nil, nil
	}

	step, _ := strconv.Atoi(res["step"])
	lastSeen, _ := strconv.ParseInt(res["last_seen"], 10, 64)
	startedAt, _ := strconv.ParseInt(res["started_at"], 10, 64)

	return &State{
		Step:       step,
		LastTactic: res["last_tactic"],
		LastSeen:   lastSeen,
		StartedAt:  startedAt,
	}, nil
}

func (s *Store) Create(
	ctx context.Context,
	containerID, chainID, tactic string,
	ttl time.Duration,
) error {
	now := time.Now().Unix()
	return s.rdb.HSet(ctx, key(containerID, chainID), map[string]interface{}{
		"step":        0,
		"last_tactic": tactic,
		"last_seen":   now,
		"started_at":  now,
	}).Err()
}

func (s *Store) Promote(
	ctx context.Context,
	containerID, chainID, tactic string,
	step int,
	ttl time.Duration,
) error {
	now := time.Now().Unix()
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key(containerID, chainID), map[string]interface{}{
		"step":        step,
		"last_tactic": tactic,
		"last_seen":   now,
	})
	pipe.Expire(ctx, key(containerID, chainID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) Delete(ctx context.Context, containerID, chainID string) error {
	return s.rdb.Del(ctx, key(containerID, chainID)).Err()
}