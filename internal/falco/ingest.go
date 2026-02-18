//internal/falco/ingest.go
package falco

import (
	"context"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/logger"
)

type Repository interface {
	Insert(ctx context.Context, event db.FalcoEventDB) error
}

func IngestFromFile(
	ctx context.Context,
	path string,
	repo Repository,
	onEvent func(Event),
	log logger.Logger,
) error {

	events := make(chan FalcoEventRaw)

	go func() {
		defer close(events)
		if err := ReadFromFile(path, events); err != nil {
			log.Error("Falco file reader failed", err, map[string]interface{}{
				"path": path,
			})
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Info("Falco ingest stopped by context", nil)
			return ctx.Err()

		case raw, ok := <-events:
			if !ok {
				log.Info("Falco finished ingesting events", nil)
				return nil
			}

			event := DecodeEvent(raw)

			dbEvent := db.FalcoEventDB{
				Time:         event.Time,
				Rule:         event.Rule,
				Priority:     event.Priority,
				Source:       event.Source,
				Hostname:     event.Hostname,
				Output:       event.Output,
				Tags:         event.Tags,
				OutputFields: event.OutputFields,
			}

			// DB
			if repo != nil {
				if err := repo.Insert(ctx, dbEvent); err != nil {
					log.Error("Falco DB insert failed", err, map[string]interface{}{
						"rule": event.Rule, "path": path,
					})
				}
			}

			// ---- callback
			if onEvent != nil {
				onEvent(event)
			}
		}
	}
}