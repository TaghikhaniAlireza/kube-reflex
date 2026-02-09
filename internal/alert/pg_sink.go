//internal/alert/pg_sink.go
package alert

import (
	"context"
	
	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

// PostgresSink implements the Sink interface for PostgreSQL storage.
type PostgresSink struct {
	repo *db.AlertRepository
}

func NewPostgresSink(repo *db.AlertRepository) *PostgresSink {
	return &PostgresSink{repo: repo}
}

func (s *PostgresSink) Emit(ctx context.Context, alert *model.Alert) error {
	return s.repo.Save(ctx, alert)
}