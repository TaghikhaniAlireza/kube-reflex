//internal/correlator/fsm/alert_builder.go
package fsm

import (
	"time"

	"github.com/google/uuid"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

// BuildAlert constructs a finalized alert from FSM state and chain definition.
// Pure function: no IO, no Redis, no logging.
func BuildAlert(
	containerID string,
	chain *model.Chain,
	state *State,
	timeline []model.AlertEvent,
) *model.Alert {

	completedAt := time.Now()

	startedAt := time.Unix(state.StartedAt, 0)
	duration := int64(completedAt.Sub(startedAt).Seconds())

	return &model.Alert{
		AlertID: uuid.NewString(),
		Type:    model.AlertTypeMitreChainCompleted,

		Severity:   deriveSeverity(chain),
		Confidence: deriveConfidence(chain, timeline),

		Chain: model.AlertChain{
			ID:       chain.ID,
			Name:     chain.Name,
			Tactics:  chain.Sequence,
			Duration: duration,
		},

		Entity: model.AlertEntity{
			Type: "container",
			ID:   containerID,
		},

		Timeline: timeline,

		Source: model.AlertSource{
			Engine:  "kube-reflex",
			Module:  "fsm-correlator",
			Version: "0.4.0",
		},

		Timestamps: model.AlertTime{
			StartedAt:   startedAt,
			CompletedAt: completedAt,
		},
	}
}

func deriveSeverity(chain *model.Chain) model.Severity {
	switch {
	case len(chain.Sequence) >= 4:
		return model.SeverityCritical
	case len(chain.Sequence) == 3:
		return model.SeverityHigh
	default:
		return model.SeverityMedium
	}
}

func deriveConfidence(chain *model.Chain, timeline []model.AlertEvent) float64 {
	base := 0.6
	stepBonus := float64(len(timeline)) * 0.1

	conf := base + stepBonus
	if conf > 0.95 {
		return 0.95
	}
	return conf
}