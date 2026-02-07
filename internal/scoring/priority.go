//internal/scoring/priority.go
package scoring

import "strings"

func ScoreFromPriority(priority string) int {
	switch strings.ToLower(priority) {
	case "info":
		return 15
	case "notice":
		return 35
	case "warning":
		return 60
	case "critical":
		return 100
	default:
		return 5
	}
}