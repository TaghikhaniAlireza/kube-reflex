//internal/action/postgres_sink.go
package action

import (
	"context"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/domain"
)

type PostgresSink struct {
	repo *db.AlertRepository
}

func NewPostgresSink(r *db.AlertRepository) *PostgresSink {
	return &PostgresSink{repo: r}
}

func (p *PostgresSink) Send(ctx context.Context, inc domain.Incident) error {
	return p.repo.SaveIncident(ctx, inc)
}