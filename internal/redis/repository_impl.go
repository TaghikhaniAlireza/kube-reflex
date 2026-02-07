// internal/redis/repository_impl.go
package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
)

type redisRepository struct {
	client *redis.Client
}

func NewRepository(client *redis.Client) Repository {
	return &redisRepository{
		client: client,
	}
}

func (r *redisRepository) UpdateContainerState(
	ctx context.Context,
	identity parser.Identity,
	scoreDelta int,
	ttl time.Duration,
) error {

	key := "container:" + identity.ContainerID

	// ---- increment risk score ----
	if err := r.client.HIncrBy(ctx, key, "risk_score", int64(scoreDelta)).Err(); err != nil {
		return err
	}

	// ---- metadata ----
	fields := map[string]interface{}{
		"namespace":      identity.Namespace,
		"pod":            identity.PodName,
		"container_name": identity.ContainerName,
		"image_repo":     identity.ImageRepo,
		"image_tag":      identity.ImageTag,
		"user_name":      identity.UserName,
		"user_uid":       identity.UserUID,
		"last_seen":      strconv.FormatInt(identity.LastSeen.Unix(), 10),
	}

	if err := r.client.HSet(ctx, key, fields).Err(); err != nil {
		return err
	}

	// ---- runtime behavior ----
	if identity.ProcExePath != "" {
		r.client.SAdd(ctx, key+":procs", identity.ProcExePath)
	}

	if identity.ProcCmdline != "" {
		r.client.SAdd(ctx, key+":cmdlines", identity.ProcCmdline)
	}

	// ---- ttl ----
	r.client.Expire(ctx, key, ttl)
	r.client.Expire(ctx, key+":procs", ttl)
	r.client.Expire(ctx, key+":cmdlines", ttl)

	return nil
}

func (r *redisRepository) AddEvent(
	ctx context.Context,
	containerID string,
	eventType string,
	ts time.Time,
) error {

	key := "container:" + containerID + ":events"
	score := float64(ts.Unix())

	err := r.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: eventType,
	}).Err()
	if err != nil {
		return err
	}

	// TTL event stream
	r.client.Expire(ctx, key, 10*time.Minute)

	return nil
}

func (r *redisRepository) IncrementFrequency(
	ctx context.Context,
	containerID string,
	eventType string,
	ttl time.Duration,
) error {

	key := "container:" + containerID + ":counters:" + eventType

	if err := r.client.Incr(ctx, key).Err(); err != nil {
		return err
	}

	r.client.Expire(ctx, key, ttl)
	return nil
}