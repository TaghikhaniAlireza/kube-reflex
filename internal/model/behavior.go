//internal/model/behavior.go
package model

// MappedBehavior is the normalized, enriched output of the Mapper.
// This structure is FSM-ready and contains only correlation-relevant data.
type MappedBehavior struct {
	BehaviorID     string   // e.g. T1595.002 (most specific MITRE ID)
	TacticID       string   // e.g. TA0043
	TacticName     string   // e.g. Reconnaissance
	SeverityWeight int      // e.g. 3
	ContextTags    []string // Non-MITRE Falco tags (container, shell, network, ...)
}

// Taxonomy represents the root of behaviors.yml
type Taxonomy struct {
	Version   int      `yaml:"version"`
	Behaviors []Tactic `yaml:"behaviors"`
}

// Tactic represents a MITRE ATT&CK Tactic (TAxxxx)
type Tactic struct {
	ID             string      `yaml:"id"`              // reconnaissance
	MitreID        string      `yaml:"mitre"`           // TA0043
	Name           string      `yaml:"name"`            // Reconnaissance
	Description    string      `yaml:"description"`     // Optional, for documentation
	SeverityWeight int         `yaml:"severity_weight"` // Used by FSM confidence logic
	Techniques     []Technique `yaml:"techniques"`
}

// Technique represents a MITRE ATT&CK Technique (Txxxx)
type Technique struct {
	ID            string         `yaml:"id"`   // T1595
	Name          string         `yaml:"name"` // Active Scanning
	SubTechniques []SubTechnique `yaml:"sub_techniques"`
}

// SubTechnique represents a MITRE ATT&CK Sub-technique (Txxxx.yyy)
// YAML intentionally keeps this minimal.
type SubTechnique struct {
	ID string `yaml:"id"` // T1595.002
}