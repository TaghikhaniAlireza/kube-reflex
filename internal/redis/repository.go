//internal/redis/repository.go
package redis

import (
	"context"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
)

type Repository interface {
	UpdateContainerState(
		ctx context.Context,
		identity parser.Identity,
		scoreDelta int,
		ttl time.Duration,
	) error
}