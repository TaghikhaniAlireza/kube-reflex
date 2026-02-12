// internal/correlator/velocity/alert_factory.go
package velocity

import (
	"time"

	"github.com/google/uuid"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

func buildVelocityAlert(
	containerID string,
	rule Rule,
	score int,
	ts time.Time,
) model.Alert {

	now := time.Now()

	return model.Alert{
		AlertID:    uuid.NewString(),
		Type:       model.AlertType("velocity_score_exceeded"),
		Severity:   model.SeverityHigh,
		Confidence: 0.7,

		Entity: model.AlertEntity{
			Type: "container",
			ID:   containerID,
		},

		// Velocity has no chain
		Chain: model.AlertChain{
			ID:       "velocity",
			Name:     "Velocity / Pressure Detection",
			Tactics:  nil,
			Duration: int64(rule.Window.Seconds()),
		},

		// Timeline is the correct place
		Timeline: []model.AlertEvent{
			{
				Rule:      rule.ID,
				Timestamp: ts,
			},
		},

		Source: model.AlertSource{
			Engine:  "kube-reflex",
			Module:  "velocity",
			Version: "0.4.0",
		},

		Timestamps: model.AlertTime{
			StartedAt:   ts.Add(-rule.Window),
			CompletedAt: now,
		},
	}
}
