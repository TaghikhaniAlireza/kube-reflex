//internal/correlator/types.go
package correlator

import "time"

type State string

const (
	StateIdle           State = "IDLE"
	StateExecSeen       State = "EXEC_SEEN"
	StateShellSpawned   State = "SHELL_SPAWNED"
	StateThreatDetected State = "THREAT_DETECTED"
)

type ThreatContext struct {
	ContainerID string
	ThreatType  string
	Confidence  float64
	Evidence    []string
	FirstSeen   time.Time
	LastSeen    time.Time
}