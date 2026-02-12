//internal/falco/ingest.go
package falco

import (
	"context"
	"log"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
)

type Repository interface {
	Insert(ctx context.Context, event db.FalcoEventDB) error
}

func IngestFromFile(
	ctx context.Context,
	path string,
	repo Repository,
	onEvent func(Event),
) error {

	events := make(chan FalcoEventRaw)

	go func() {
		defer close(events)
		if err := ReadFromFile(path, events); err != nil {
			log.Println("[falco] reader error:", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("[falco] ingest stopped by context")
			return ctx.Err()

		case raw, ok := <-events:
			if !ok {
				log.Println("[falco] finished ingesting events")
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
					log.Println("[falco] db insert error:", err)
				}
			}

			// ---- callback
			if onEvent != nil {
				onEvent(event)
			}
		}
	}
}