//internal/decision/signal.go
package decision

import "time"

type Source string

const (
	SourceFSM      Source = "fsm"
	SourceVelocity Source = "velocity"
)

type Signal struct {
	ContainerID string
	Source      Source
	Score       int
	Details     map[string]string
	Timestamp   time.Time
}