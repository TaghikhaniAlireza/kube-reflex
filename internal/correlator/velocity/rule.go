//internal/correlator/velocity/rule.go
package velocity

import "time"

type Rule struct {
	ID        string
	Name      string
	Window    time.Duration
}

var DefaultRules = []Rule{
	{
		ID:        "VEL_WARN_SPAM",
		Name:      "High Frequency Warning Alerts",
		Window:    5 * time.Minute,
	},
}