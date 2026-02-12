//internal/domain/incident.go
package domain

import "time"

type Incident struct {
	IncidentID  string
	ContainerID string
	PodName     string
	Namespace   string
	RiskScore   int
	Severity    string
	Categories  []string
	SignalCount int
	Findings    []string
	DetectedAt  time.Time
}