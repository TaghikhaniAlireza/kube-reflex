// internal/model/chain.go
package model

// Chain represents a dangerous behavioral sequence (Tactic-level for now).
type Chain struct {
	// Unique identifier (e.g. remote_exploit_chain)
	ID string `yaml:"id"`

	// Human-readable name
	Name string `yaml:"name"`

	// Long description explaining the threat logic
	Description string `yaml:"description"`

	// Ordered list of MITRE ATT&CK Tactic IDs (TAxxxx)
	Sequence []string `yaml:"sequence"`

	// Max allowed time between first and last step (e.g. 10m, 1h)
	MaxDuration string `yaml:"max_duration"`

	// CRITICAL / HIGH / MEDIUM / LOW
	Severity string `yaml:"severity"`
}

// ChainFile is the root object of chains.yml
type ChainFile struct {
	Version int     `yaml:"version"`
	Chains  []Chain `yaml:"chains"`
}