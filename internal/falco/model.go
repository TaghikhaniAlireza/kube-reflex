//internal/falco/model.go
package falco

import "time"

type FalcoEventRaw struct {
	Time         time.Time              `json:"time"`
	Rule         string                 `json:"rule"`
	Priority     string                 `json:"priority"`
	Source       string                 `json:"source"`
	Hostname     string                 `json:"hostname"`
	Output       string                 `json:"output"`
	Tags         []string               `json:"tags"`
	OutputFields map[string]interface{} `json:"output_fields"`
}