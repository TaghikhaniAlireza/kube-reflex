//internal/correlator/fsm/state.go
package fsm

type State struct {
	Step       int
	LastTactic string
	LastSeen   int64
	StartedAt  int64
}
