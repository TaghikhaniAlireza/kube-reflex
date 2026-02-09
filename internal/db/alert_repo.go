//internal/db/alert_repo.go
package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

type AlertRepository struct {
	pool *pgxpool.Pool
}

func NewAlertRepository(pool *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{pool: pool}
}

// Save persists the alert model into the postgres database.
func (r *AlertRepository) Save(ctx context.Context, alert *model.Alert) error {
	// Marshal the full struct to JSONB to preserve complex fields (Timeline, Source, etc.)
	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert payload: %w", err)
	}

	query := `
		INSERT INTO alerts (
			alert_id, 
			container_id, 
			chain_id, 
			severity, 
			type, 
			occurred_at, 
			payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.pool.Exec(ctx, query,
		alert.AlertID,
		alert.Entity.ID,
		alert.Chain.ID,
		string(alert.Severity),
		string(alert.Type),
		alert.Timestamps.CompletedAt,
		payload,
	)

	if err != nil {
		return fmt.Errorf("failed to insert alert into postgres: %w", err)
	}

	return nil
}