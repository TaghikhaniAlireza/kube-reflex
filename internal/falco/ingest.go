//internal/falco/ingest.go
package falco

import (
	"context"
	"log"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
)

func IngestFromFile(
	ctx context.Context,
	path string,
	repo *db.FalcoRepository,
) error {

	events := make(chan FalcoEventRaw)

	go func() {
		defer close(events)
		if err := ReadFromFile(path, events); err != nil {
			log.Println("falco reader error:", err)
		}
	}()

	for e := range events {
		dbEvent := db.FalcoEventDB{
			Time:         e.Time,
			Rule:         e.Rule,
			Priority:     e.Priority,
			Source:       e.Source,
			Hostname:     e.Hostname,
			Output:       e.Output,
			Tags:         e.Tags,
			OutputFields: e.OutputFields,
		}

		if err := repo.Insert(ctx, dbEvent); err != nil {
			log.Println("db insert error:", err)
		}
	}

	return nil
}