//internal/correlator/correlator.go
package correlator

import "sync"

type Engine struct {
	mu   sync.Mutex
	fsms map[string]*FSM
}

func NewEngine() *Engine {
	return &Engine{
		fsms: make(map[string]*FSM),
	}
}

func (e *Engine) ProcessEvent(containerID, rule string) *ThreatContext {
	e.mu.Lock()
	defer e.mu.Unlock()

	fsm, ok := e.fsms[containerID]
	if !ok {
		fsm = NewFSM(containerID)
		e.fsms[containerID] = fsm
	}

	threat := fsm.Transition(rule)

	if threat != nil {
		// reset FSM after detection (v1 behavior)
		delete(e.fsms, containerID)
	}

	return threat
}