// internal/correlator/rules/chain_registry.go
package rules

import (
	"sync"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

// ChainRegistry is an in-memory indexed store for chains
// optimized for FSM usage.
type ChainRegistry struct {
	mu sync.RWMutex

	// chain_id -> chain
	byID map[string]*model.Chain

	// first_tactic -> chains starting with that tactic
	byFirstTactic map[string][]*model.Chain

	// flat list (debug, iteration, metrics)
	allChains []*model.Chain
}

// NewChainRegistry builds indexes from a loaded ChainFile
func NewChainRegistry(file *model.ChainFile) *ChainRegistry {
	r := &ChainRegistry{
		byID:          make(map[string]*model.Chain),
		byFirstTactic: make(map[string][]*model.Chain),
		allChains:     make([]*model.Chain, 0, len(file.Chains)),
	}

	for i := range file.Chains {
		chain := &file.Chains[i]

		r.byID[chain.ID] = chain

		firstTactic := chain.Sequence[0]
		r.byFirstTactic[firstTactic] = append(
			r.byFirstTactic[firstTactic],
			chain,
		)

		r.allChains = append(r.allChains, chain)
	}

	return r
}

// GetByID returns a chain by its ID
func (r *ChainRegistry) GetByID(id string) (*model.Chain, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.byID[id]
	return c, ok
}

// GetStartingWith returns all chains that start with a given tactic
func (r *ChainRegistry) GetStartingWith(tactic string) []*model.Chain {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.byFirstTactic[tactic]
}

// All returns all registered chains
func (r *ChainRegistry) All() []*model.Chain {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.allChains
}