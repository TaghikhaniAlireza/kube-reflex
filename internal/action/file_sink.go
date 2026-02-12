// internal/action/file_sink.go
package action

import (
	"context"
	"fmt"
)

type StdoutSink struct{}

func NewStdoutSink() *StdoutSink {
	return &StdoutSink{}
}

func (s *StdoutSink) Send(ctx context.Context, incident Incident) error {
	fmt.Printf("[STDOUT SINK] %+v\n", incident)
	return nil
}