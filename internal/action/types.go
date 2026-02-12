//internal/action/types.go
package action

import (
	"context"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
)

type Sink interface {
	Send(ctx context.Context, incident domain.Incident) error
}