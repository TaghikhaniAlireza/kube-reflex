//internal/decision/judge.go
package decision

import (
	"context"
	"fmt"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/k8s"
)

type Judge struct {
	k8sResolver k8s.Resolver
}

func NewJudge(resolver k8s.Resolver) *Judge {
	return &Judge{
		k8sResolver: resolver,
	}
}

func (j *Judge) Evaluate(
	ctx context.Context,
	containerID string,
	signals []Signal,
) (*domain.Incident, error) {

	if len(signals) == 0 {
		return nil, nil
	}

	// --- K8s Enrichment ---
	podName := "unknown"
	namespace := "unknown"

	if j.k8sResolver != nil {
		if info, err := j.k8sResolver.Resolve(containerID); err == nil {
			podName = info.Pod
			namespace = info.Namespace
		}
	}

	// --- Risk Calculation ---
	score, findings, categories := j.calculateRisk(signals, namespace)

	// Noise suppression threshold
	if score < 20 {
		return nil, nil
	}

	return &domain.Incident{
		IncidentID:  fmt.Sprintf("inc-%s-%d", containerID, time.Now().UnixNano()),
		ContainerID: containerID,
		PodName:     podName,
		Namespace:   namespace,
		RiskScore:   score,
		Severity:    j.determineSeverity(score),
		Categories:  categories,
		SignalCount: len(signals),
		Findings:    findings,
		DetectedAt:  time.Now(),
	}, nil
}

func (j *Judge) calculateRisk(
	signals []Signal,
	namespace string,
) (int, []string, []string) {

	total := 0
	maxSingle := 0
	fsmHits := 0
	velocityHits := 0

	categorySet := make(map[string]struct{})
	var findings []string

	for _, s := range signals {

		if s.Score > maxSingle {
			maxSingle = s.Score
		}

		if cat, ok := s.Details["category"]; ok {
			categorySet[cat] = struct{}{}
		}

		switch s.Source {

		case SourceFSM:
			fsmHits++
			total += s.Score * 2

			if name, ok := s.Details["chain_name"]; ok {
				findings = append(findings,
					fmt.Sprintf("Attack Chain Detected: %s", name))
			}

		case SourceVelocity:
			velocityHits++
			total += s.Score / 2
		}
	}

	// Correlation boost
	if fsmHits > 0 && velocityHits > 10 {
		total += 50
		findings = append(findings,
			"High confidence: Chain + Volume anomaly")
	}

	if total < maxSingle {
		total = maxSingle
	}

	// Context awareness
	if namespace == "kube-system" || namespace == "monitoring" {
		total = int(float64(total) * 1.5)
		findings = append(findings,
			"Sensitive namespace activity")
	}

	var categories []string
	for c := range categorySet {
		categories = append(categories, c)
	}

	return total, findings, categories
}

func (j *Judge) determineSeverity(score int) string {
	switch {
	case score >= 100:
		return "CRITICAL"
	case score >= 70:
		return "HIGH"
	case score >= 40:
		return "MEDIUM"
	default:
		return "LOW"
	}
}