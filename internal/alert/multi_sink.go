//internal/alert/multi_sink.go
package alert

import (
	"context"
	"fmt"
	"strings"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

// MultiSink broadcasts an alert to multiple underlying sinks.
type MultiSink struct {
	sinks []Sink
}

func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

func (ms *MultiSink) Emit(ctx context.Context, alert *model.Alert) error {
	var errors []string

	for _, s := range ms.sinks {
		if err := s.Emit(ctx, alert); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multisink errors: %s", strings.Join(errors, "; "))
	}
	return nil
}