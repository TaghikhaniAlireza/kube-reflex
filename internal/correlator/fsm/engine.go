// internal/correlator/fsm/engine.go
package fsm

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type Store interface {
	Create(ctx context.Context, containerID, chainID, tactic string, ttl time.Duration) error
	Get(ctx context.Context, containerID, chainID string) (*State, error)
	Promote(ctx context.Context, containerID, chainID, tactic string, newStep int, ttl time.Duration) error
	Delete(ctx context.Context, containerID, chainID string) error
}

type Engine struct {
	store Store
}

func NewEngine(store Store) *Engine {
	return &Engine{store: store}
}

// Process handles state transitions.
// Input: rules.Chain (from main.go)
// Output: *model.Alert (for alert sink)
func (e *Engine) Process(ctx context.Context, containerID, tacticID string, chain *model.Chain) (*model.Alert, error) {

	// 1. Fetch current state
	state, err := e.store.Get(ctx, containerID, chain.ID)
	if err != nil {
		return nil, err
	}

	// --------------------------------------------------------
	// SCENARIO A: No state exists yet
	// --------------------------------------------------------
	if state == nil {
		// If this tactic is the START of the chain, create new state.
		if tacticID == chain.Sequence[0] {
			log.Printf("🟢 FSM START | Chain=%s | Tactic=%s", chain.ID, tacticID)

			ttl := parseDuration(chain.MaxDuration, 10*time.Minute)
			err := e.store.Create(ctx, containerID, chain.ID, tacticID, ttl)
			return nil, err
		}
		return nil, nil
	}

	// --------------------------------------------------------
	// SCENARIO B: State exists
	// --------------------------------------------------------
	nextStepIndex := state.Step + 1

	// Safety check
	if nextStepIndex >= len(chain.Sequence) {
		return nil, nil
	}

	expectedTactic := chain.Sequence[nextStepIndex]

	if tacticID == expectedTactic {
		// MATCH!

		// Is this the FINAL step?
		if nextStepIndex == len(chain.Sequence)-1 {
			log.Printf("🚨 CHAIN COMPLETED | Chain=%s | Severity=%s", chain.ID, chain.Severity)

			// 1. Delete state (chain finished)
			if err := e.store.Delete(ctx, containerID, chain.ID); err != nil {
				log.Printf("failed to cleanup finished chain: %v", err)
			}

			startedAt := time.Unix(state.StartedAt, 0)
			completedAt := time.Now()

			// 2. Return Alert using model.Alert structure
			return &model.Alert{
				AlertID:    uuid.New().String(),
				Type:       model.AlertTypeMitreChainCompleted,
				Severity:   model.Severity(chain.Severity), // Casting string to model.Severity
				Confidence: 0.9,

				Chain: model.AlertChain{
					ID:       chain.ID,
					Name:     chain.Description, // Using Description as Name usually works better
					Tactics:  chain.Sequence,
					Duration: int64(completedAt.Sub(startedAt).Seconds()),
				},

				Entity: model.AlertEntity{
					Type: "container",
					ID:   containerID,
				},

				Timeline: []model.AlertEvent{
					{
						Tactic:    tacticID,
						Timestamp: completedAt,
					},
				},

				Source: model.AlertSource{
					Engine:  "kube-reflex",
					Module:  "fsm",
					Version: "1.0.0",
				},

				Timestamps: model.AlertTime{
					StartedAt:   startedAt,
					CompletedAt: completedAt,
				},
			}, nil
		}

		// Not final step -> PROMOTE
		log.Printf("🔼 FSM PROMOTE | Chain=%s | Step=%d | Tactic=%s", chain.ID, nextStepIndex, tacticID)

		ttl := parseDuration(chain.MaxDuration, 10*time.Minute)
		err := e.store.Promote(ctx, containerID, chain.ID, tacticID, nextStepIndex, ttl)
		return nil, err
	}

	// --------------------------------------------------------
	// SCENARIO C: Mismatch / Reset
	// --------------------------------------------------------
	if tacticID == chain.Sequence[0] {
		log.Printf("🔄 FSM RESET | Chain=%s | Restarting sequence", chain.ID)

		ttl := parseDuration(chain.MaxDuration, 10*time.Minute)
		err := e.store.Create(ctx, containerID, chain.ID, tacticID, ttl)
		return nil, err
	}

	return nil, nil
}

func parseDuration(d string, fallback time.Duration) time.Duration {
	val, err := time.ParseDuration(d)
	if err != nil {
		return fallback
	}
	return val
}