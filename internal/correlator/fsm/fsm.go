//internal/correlator/fsm.go
package correlator

import "time"

type FSM struct {
	ContainerID string
	State       State
	Evidence    []string
	FirstSeen   time.Time
	LastSeen    time.Time
}

func NewFSM(containerID string) *FSM {
	now := time.Now()
	return &FSM{
		ContainerID: containerID,
		State:       StateIdle,
		FirstSeen:   now,
		LastSeen:    now,
	}
}

func (f *FSM) Transition(rule string) *ThreatContext {
	f.LastSeen = time.Now()
	f.Evidence = append(f.Evidence, rule)

	switch f.State {

	case StateIdle:
		if rule == "Terminal shell in container" {
			f.State = StateShellSpawned
		}

	case StateShellSpawned:
		if rule == "Write below binary dir" {
			f.State = StateThreatDetected
			return f.buildThreat("Container Shell Escape")
		}
	}

	return nil
}

func (f *FSM) buildThreat(threatType string) *ThreatContext {
	return &ThreatContext{
		ContainerID: f.ContainerID,
		ThreatType:  threatType,
		Confidence:  0.85,
		Evidence:    f.Evidence,
		FirstSeen:   f.FirstSeen,
		LastSeen:    f.LastSeen,
	}
}