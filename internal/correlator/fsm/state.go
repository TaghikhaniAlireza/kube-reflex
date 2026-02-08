// internal/correlator/fsm/state.go
package fsm

type State struct {
	Step       int    `json:"step"`
	LastTactic string `json:"last_tactic"`
	LastSeen   int64  `json:"last_seen"`
	StartedAt  int64  `json:"started_at"`
}