//internal/parser/identity.go
package parser

import "time"

type Identity struct {
	// ---- Identity ----
	ContainerID   string
	ContainerName string

	ImageRepo string
	ImageTag  string

	Namespace string
	PodName   string

	// ---- Runtime Context ----
	ProcCmdline string
	ProcExePath string

	UserName string
	UserUID  string

	// ---- State ----
	RiskScore int
	LastSeen  time.Time
}

// helper map[string]interface{}
func safeGetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func ExtractIdentity(fields map[string]interface{}) Identity {
	return Identity{
		ContainerID:   safeGetString(fields, "container.id"),
		ContainerName: safeGetString(fields, "container.name"),

		ImageRepo: safeGetString(fields, "container.image.repository"),
		ImageTag:  safeGetString(fields, "container.image.tag"),

		Namespace: safeGetString(fields, "k8s.ns.name"),
		PodName:   safeGetString(fields, "k8s.pod.name"),

		ProcCmdline: safeGetString(fields, "proc.cmdline"),
		ProcExePath: safeGetString(fields, "proc.exepath"),

		UserName: safeGetString(fields, "user.name"),
		UserUID:  safeGetString(fields, "user.uid"),

		LastSeen: time.Now().UTC(),
	}
}