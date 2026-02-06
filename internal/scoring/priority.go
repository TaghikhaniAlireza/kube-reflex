//internal/scoring/priority.go
package scoring

import "strings"

func ScoreFromPriority(priority string) int {
	switch strings.ToLower(priority) {
	case "info":
		return 1
	case "notice":
		return 2
	case "warning":
		return 5
	case "critical":
		return 10
	default:
		return 0
	}
}