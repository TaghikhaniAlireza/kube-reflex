//internal/correlator/velocity/detector.go
package velocity

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/logger"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/scoring"
)

type Detector struct {
	redis *redis.Client
	log   logger.Logger
}

func NewDetector(redis *redis.Client, log logger.Logger) *Detector {
	return &Detector{redis: redis, log: log}
}

func (d *Detector) AddAndScore(
	ctx context.Context,
	containerID string,
	ruleID string,
	priority string,
	ts time.Time,
	window time.Duration,
) (int, error) {

	key := fmt.Sprintf("velocity:%s:%s", containerID, ruleID)
	now := ts.Unix()
	weight := scoring.ScoreFromPriority(priority)

	member := fmt.Sprintf("%d:%d", now, weight)

	pipe := d.redis.TxPipeline()

	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: member,
	})

	pipe.ZRemRangeByScore(
		ctx,
		key,
		"-inf",
		strconv.FormatInt(now-int64(window.Seconds()), 10),
	)

	eventsCmd := pipe.ZRange(ctx, key, 0, -1)
	pipe.Expire(ctx, key, window*2)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	totalScore := 0
	for _, e := range eventsCmd.Val() {
		_, w, _ := strings.Cut(e, ":")
		v, err := strconv.Atoi(w)
		if err != nil {
			d.log.Warn("Failed to parse velocity score from Redis member", map[string]interface{}{
				"error": err.Error(), "member": e, "container_id": containerID, "rule_id": ruleID,
			})
			continue
		}
		totalScore += v
	}

	return totalScore, nil
}