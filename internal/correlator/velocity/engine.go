//internal/correlator/velocity/engine.go
package velocity

import (
	"context"
	"log"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type Engine struct {
	detector *Detector
	rules    []Rule
	alertCh  chan<- model.Alert
}

func NewEngine(
	detector *Detector,
	alertCh chan<- model.Alert,
) *Engine {
	return &Engine{
		detector: detector,
		rules:    DefaultRules,
		alertCh:  alertCh,
	}
}

func (e *Engine) Process(
	ctx context.Context,
	containerID string,
	priority string,
	timestamp time.Time,
) {

	for _, rule := range e.rules {

		score, err := e.detector.AddAndScore(
			ctx,
			containerID,
			rule.ID,
			priority,
			timestamp,
			rule.Window,
		)
		if err != nil {
			log.Printf("[velocity] detector error: %v", err)
			continue
		}

		if score < rule.Threshold {
			return
		}

		alert := buildVelocityAlert(
			containerID,
			rule,
			score,
			timestamp,
		)

		select {
		case e.alertCh <- alert:
			log.Printf(
				"[velocity] ALERT container=%s rule=%s score=%d",
				containerID,
				rule.ID,
				score,
			)
		default:
			log.Println("[velocity] alert channel full, dropping alert")
		}
	}
}