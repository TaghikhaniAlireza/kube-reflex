//internal/falco/event.go
package falco

import "time"

// Event is the normalized Falco event used inside the brain
type Event struct {
	Time         time.Time
	Rule         string
	Priority     string
	Source       string
	Hostname     string
	Output       string
	Tags         []string
	OutputFields map[string]interface{}
}