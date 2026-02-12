//internal/correlator/velocity/rule.go
package velocity

import "time"

type Rule struct {
	ID        string
	Name      string
	Window    time.Duration
	Threshold int // score threshold
}

var DefaultRules = []Rule{
	{
		ID:        "VEL_WARN_SPAM",
		Name:      "High Frequency Warning Alerts",
		Window:    5 * time.Minute,
		Threshold: 6000, // 100 * 60
	},
}