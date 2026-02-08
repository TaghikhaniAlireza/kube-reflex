//internal/alert/sink.go
package alert

import (
	"context"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type Sink interface {
	Emit(ctx context.Context, alert *model.Alert) error
}