// internal/model/falco_event.go
package model

import "time"

type FalcoEvent struct {
	Time         time.Time              `json:"time"`
	Rule         string                 `json:"rule"`
	Priority     string                 `json:"priority"`
	Source       string                 `json:"source"`
	Hostname     string                 `json:"hostname"`
	Tags         []string               `json:"tags"`
	Output       string                 `json:"output"`
	OutputFields map[string]interface{} `json:"output_fields"`
}