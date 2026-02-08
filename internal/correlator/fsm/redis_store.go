// internal/correlator/fsm/redis_store.go
package fsm

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
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
	k := key(containerID, chainID)
	log.Printf("REDIS GET | Key=%s", k)
	res, err := s.rdb.HGetAll(ctx, k).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}

	step, _ := strconv.Atoi(res["step"])
	lastSeen, _ := strconv.ParseInt(res["last_seen"], 10, 64)
	startedAt, _ := strconv.ParseInt(res["started_at"], 10, 64)
	log.Printf("✅ REDIS FOUND | Key=%s | Step=%d | LastTactic=%s", k, step, res["last_tactic"])
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
	k := key(containerID, chainID)
	now := time.Now().Unix()
	
	log.Printf("REDIS CREATE | Key=%s | TTL=%v", k, ttl)

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, k, map[string]interface{}{
		"step":        0,
		"last_tactic": tactic,
		"last_seen":   now,
		"started_at":  now,
	})
	pipe.Expire(ctx, k, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) Promote(
	ctx context.Context,
	containerID, chainID, tactic string,
	step int,
	ttl time.Duration,
) error {
	k := key(containerID, chainID)
	now := time.Now().Unix()
	
	log.Printf("REDIS PROMOTE | Key=%s | NewStep=%d", k, step)

	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, k, map[string]interface{}{
		"step":        step,
		"last_tactic": tactic,
		"last_seen":   now,
	})
	pipe.Expire(ctx, k, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) Delete(ctx context.Context, containerID, chainID string) error {
	return s.rdb.Del(ctx, key(containerID, chainID)).Err()
}

func (s *Store) GetTimeline(
	ctx context.Context,
	containerID, chainID string,
) ([]model.AlertEvent, error) {
	return []model.AlertEvent{}, nil
}