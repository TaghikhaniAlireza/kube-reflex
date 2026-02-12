// internal/action/file_sink.go
package action

import (
	"context"
	"fmt"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
)

type StdoutSink struct{}

func NewStdoutSink() *StdoutSink {
	return &StdoutSink{}
}

func (s *StdoutSink) Send(ctx context.Context, incident domain.Incident) error {
	fmt.Printf("[STDOUT SINK] %+v\n", incident)
	return nil
}