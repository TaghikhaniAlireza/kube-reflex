//internal/model/alert.go
package model

import "time"

// Alert represents a finalized security finding emitted by the correlator.
type Alert struct {
	AlertID    string     `json:"alert_id"`
	Type       AlertType  `json:"type"`
	Severity   Severity   `json:"severity"`
	Confidence float64    `json:"confidence"`

	// Score captures the numeric risk/pressure score associated with this alert.
	// For velocity alerts this is the velocity score; for other engines it may be zero.
	Score int `json:"score"`

	Chain    AlertChain   `json:"chain"`
	Entity   AlertEntity  `json:"entity"`
	Timeline []AlertEvent `json:"timeline"`

	Source     AlertSource `json:"source"`
	Timestamps AlertTime   `json:"timestamps"`
}

/* -------------------- ENUMS -------------------- */

type AlertType string

const (
	AlertTypeMitreChainCompleted AlertType = "mitre_chain_completed"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

/* -------------------- CHAIN -------------------- */

type AlertChain struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Tactics   []string `json:"tactics"`
	Duration  int64    `json:"duration_seconds"`
}

/* -------------------- ENTITY -------------------- */

type AlertEntity struct {
	Type      string `json:"type"` // container, pod, node, user, ...
	ID        string `json:"id"`
	Namespace string `json:"namespace,omitempty"`
	Image     string `json:"image,omitempty"`
}

/* -------------------- TIMELINE -------------------- */

type AlertEvent struct {
	Tactic     string    `json:"tactic"`
	Technique  string    `json:"technique,omitempty"`
	Rule       string    `json:"rule,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

/* -------------------- SOURCE -------------------- */

type AlertSource struct {
	Engine  string `json:"engine"`
	Module  string `json:"module"`
	Version string `json:"version"`
}

/* -------------------- TIME -------------------- */

type AlertTime struct {
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}
