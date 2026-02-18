//internal/action/engine.go
package action

import (
	"context"
	"fmt"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/logger"
)

type Engine struct {
	sinks []Sink
	log   logger.Logger
}

func NewEngine(log logger.Logger, sinks ...Sink) *Engine {
	return &Engine{
		sinks: sinks,
		log:   log,
	}
}

func (e *Engine) Dispatch(ctx context.Context, incident domain.Incident) {
	for _, s := range e.sinks {
		if err := s.Send(ctx, incident); err != nil {
			e.log.Error("Sink failed to send incident", err, map[string]interface{}{
				"sink_type":    fmt.Sprintf("%T", s),
				"container_id": incident.ContainerID,
				"pod_name":     incident.PodName,
				"namespace":    incident.Namespace,
				"incident_id":  incident.IncidentID,
			})
		}
	}
}