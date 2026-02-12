//internal/action/engine.go
package action

import (
	"context"
	"log"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
)

type Engine struct {
	sinks []Sink
}

func NewEngine(sinks ...Sink) *Engine {
	return &Engine{
		sinks: sinks,
	}
}

func (e *Engine) Dispatch(ctx context.Context, incident domain.Incident) {
	for _, s := range e.sinks {
		if err := s.Send(ctx, incident); err != nil {
			log.Printf("[action] sink error: %v", err)
		}
	}
}