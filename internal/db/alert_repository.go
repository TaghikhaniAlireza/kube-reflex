//internal/db/alert_repository.go
package db

import (
	"context"
	"time"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
)

func (r *AlertRepository) SaveIncident(ctx context.Context, inc domain.Incident) error {
	query := `
	INSERT INTO incidents (
		incident_id,
		container_id,
		pod_name,
		namespace,
		risk_score,
		severity,
		signal_count,
		detected_at
	)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	ON CONFLICT (incident_id) DO NOTHING
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		inc.IncidentID,
		inc.ContainerID,
		inc.PodName,
		inc.Namespace,
		inc.RiskScore,
		inc.Severity,
		inc.SignalCount,
		inc.DetectedAt,
	)
	return err
}