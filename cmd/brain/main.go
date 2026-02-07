// cmd/brain/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/taxonomy" // New: Import Mapper package
	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/falco"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
	redisinfra "github.com/TaghikhaniAlireza/kube-reflex/internal/redis"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/scoring"
)

func main() {
	log.Println("starting kube-reflex brain...")

	ctx := context.Background()

	// ------------------------------------------------------------------
	// 1. Run DB migrations
	// ------------------------------------------------------------------
	db.RunMigrations()

	// ------------------------------------------------------------------
	// 2. Postgres connection
	// ------------------------------------------------------------------
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	falcoRepo := db.NewFalcoRepository(pool)

	// ------------------------------------------------------------------
	// 3. Redis connection
	// ------------------------------------------------------------------
	redisClient, err := redisinfra.NewRedisClient()
	if err != nil {
		log.Fatal(err)
	}
	redisRepo := redisinfra.NewRepository(redisClient)

	// ------------------------------------------------------------------
	// 4. Mapper initialization
	// ------------------------------------------------------------------
	// Assuming behaviors.yml is located in a standard configs/ path relative to the runtime.
	mapperInstance, err := taxonomy.NewMapper("internal/correlator/taxonomy/behaviors.yml")
	if err != nil {
		log.Fatalf("failed to initialize taxonomy mapper: %v", err)
	}

	// ------------------------------------------------------------------
	// 5. Falco ingest + Brain hook
	// ------------------------------------------------------------------
	err = falco.IngestFromFile(
		ctx,
		"/app/falco_sample_log.txt",
		falcoRepo,
		func(event falco.Event) {

			// ----------------------------------------------------------
			// 5.1 Static scoring (priority-based)
			// ----------------------------------------------------------
			score := scoring.ScoreFromPriority(event.Priority)
			if score == 0 {
				return
			}

			// ----------------------------------------------------------
			// 5.2 Identity extraction
			// ----------------------------------------------------------
			identity := parser.ExtractIdentity(event.OutputFields)

			if identity.ContainerID == "" {
				log.Println("skip event without container.id")
				return
			}
            
            // ----------------------------------------------------------
			// 5.3 Behavior Mapping (The new core logic)
			// ----------------------------------------------------------
            
            // IMPORTANT: Assuming event.Tags is the source of raw Falco tags []string
            mappedBehavior, err := mapperInstance.Map(event.Tags) 
            if err != nil {
                // If no valid MITRE ID is found, we log it and skip further correlation steps.
                log.Printf("Mapper skip for rule %s: %v", event.Rule, err)
                return 
            }
            
            // For now, log the mapped behavior to confirm the mapper is working correctly
            log.Printf(
                "MAPPED container=%s behavior=%s tactic=%s tags=%v", 
                identity.ContainerID, 
                mappedBehavior.BehaviorID, 
                mappedBehavior.TacticName, 
                mappedBehavior.ContextTags,
            )

			// ----------------------------------------------------------
			// 5.4 Update container snapshot (state)
			// ----------------------------------------------------------
			if err := redisRepo.UpdateContainerState(
				ctx,
				identity,
				score,
				15*time.Minute,
			); err != nil {
				log.Printf("redis UpdateContainerState error: %v", err)
				return
			}

			// ----------------------------------------------------------
			// 5.5 Add behavioral data (Phase 1 core)
			// ----------------------------------------------------------

			// Event type (later: enum / normalized type)
			eventType := event.Rule

			// 1) Temporal stream (ZSET)
			if err := redisRepo.AddEvent(
				ctx,
				identity.ContainerID,
				eventType,
				event.Time,
			); err != nil {
				log.Printf("redis AddEvent error: %v", err)
			}

			// 2) Frequency counter (velocity)
			if err := redisRepo.IncrementFrequency(
				ctx,
				identity.ContainerID,
				eventType,
				time.Minute,
			); err != nil {
				log.Printf("redis IncrementFrequency error: %v", err)
			}

			// ----------------------------------------------------------
			// 5.6 Debug log
			// ----------------------------------------------------------
			log.Printf(
				"INGEST SUCCESS container=%s rule=%s score=%d",
				identity.ContainerID,
				event.Rule,
				score,
			)
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("brain finished ingesting falco events")
}