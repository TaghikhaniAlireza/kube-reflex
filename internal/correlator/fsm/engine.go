//internal/correlator/fsm/engine.go
package fsm

import (
	"context"
	"log"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type Engine struct {
	store *Store
}

func NewEngine(store *Store) *Engine {
	return &Engine{store: store}
}

func (e *Engine) Process(
	ctx context.Context,
	containerID string,
	tactic string,
	chain *model.Chain,
) {
	ttl, _ := time.ParseDuration(chain.MaxDuration)
	state, _ := e.store.Get(ctx, containerID, chain.ID)

	// No state yet
	if state == nil {
		if tactic == chain.Sequence[0] {
			_ = e.store.Create(ctx, containerID, chain.ID, tactic, ttl)
			_ = e.store.rdb.Expire(ctx,
				"fsm:"+containerID+":"+chain.ID,
				ttl,
			)
			log.Printf("FSM START container=%s chain=%s step=0",
				containerID, chain.ID)
		}
		return
	}

	nextStep := state.Step + 1
	if nextStep >= len(chain.Sequence) {
		return
	}

	if tactic != chain.Sequence[nextStep] {
		return
	}

	// Promote
	if nextStep == len(chain.Sequence)-1 {
		log.Printf(
			"CHAIN COMPLETED container=%s chain=%s severity=%s",
			containerID, chain.ID, chain.Severity,
		)
		_ = e.store.Delete(ctx, containerID, chain.ID)
		return
	}

	_ = e.store.Promote(
		ctx,
		containerID,
		chain.ID,
		tactic,
		nextStep,
		ttl,
	)

	log.Printf("FSM PROMOTE container=%s chain=%s step=%d",
		containerID, chain.ID, nextStep)
}