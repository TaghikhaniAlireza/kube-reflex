//internal/db/falco_repo.go
package db

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FalcoRepository struct {
	db *pgxpool.Pool
}

type FalcoEventDB struct {
	Time         interface{}
	Rule         string
	Priority     string
	Source       string
	Hostname     string
	Output       string
	Tags         []string
	OutputFields map[string]interface{}
}

func NewFalcoRepository(db *pgxpool.Pool) *FalcoRepository {
	return &FalcoRepository{db: db}
}

func (r *FalcoRepository) Insert(ctx context.Context, e FalcoEventDB) error {
	outputFields, _ := json.Marshal(e.OutputFields)

	_, err := r.db.Exec(ctx, `
		INSERT INTO falco_events (
			time, rule, priority, source,
			hostname, tags, output, output_fields
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`,
		e.Time,
		e.Rule,
		e.Priority,
		e.Source,
		e.Hostname,
		e.Tags,
		e.Output,
		outputFields,
	)

	return err
}