// cmd/brain/main.go
// cmd/brain/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/alert"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/fsm"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/rules"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/taxonomy"
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
	// 1. DB migrations
	// ------------------------------------------------------------------
	db.RunMigrations()

	// ------------------------------------------------------------------
	// 2. PostgreSQL
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
	// 3. Redis
	// ------------------------------------------------------------------
	redisClient, err := redisinfra.NewRedisClient()
	if err != nil {
		log.Fatal(err)
	}
	redisRepo := redisinfra.NewRepository(redisClient)

	// ------------------------------------------------------------------
	// 4. Load MITRE Chains
	// ------------------------------------------------------------------
	chainLoader := rules.NewChainLoader(
		"internal/correlator/rules/chains.yml",
	)

	chainFile, err := chainLoader.Load()
	if err != nil {
		log.Fatalf("failed to load chains.yml: %v", err)
	}

	chainRegistry := rules.NewChainRegistry(chainFile)
	log.Printf("loaded %d MITRE chains", len(chainRegistry.All()))

	// ------------------------------------------------------------------
	// 5. Behavior Mapper
	// ------------------------------------------------------------------
	mapper, err := taxonomy.NewMapper(
		"internal/correlator/taxonomy/behaviors.yml",
	)
	if err != nil {
		log.Fatalf("failed to initialize taxonomy mapper: %v", err)
	}

	// ------------------------------------------------------------------
	// 6. FSM Engine (ONE instance)
	// ------------------------------------------------------------------
	fsmStore := fsm.NewStore(redisClient)
	fsmEngine := fsm.NewEngine(fsmStore)

	// ------------------------------------------------------------------
	// 7. Alert Sink
	// ------------------------------------------------------------------
	alertSink, err := alert.NewFileSink("/app/alerts.log")
	if err != nil {
		log.Fatalf("failed to create alert sink: %v", err)
	}

	// ------------------------------------------------------------------
	// 8. Falco Ingest Loop
	// ------------------------------------------------------------------
	err = falco.IngestFromFile(
		ctx,
		"/app/falco_sample_log.txt",
		falcoRepo,
		func(event falco.Event) {

			// ----------------------------------------------------------
			// 8.1 Static scoring
			// ----------------------------------------------------------
			score := scoring.ScoreFromPriority(event.Priority)
			if score == 0 {
				return
			}

			// ----------------------------------------------------------
			// 8.2 Identity extraction
			// ----------------------------------------------------------
			identity := parser.ExtractIdentity(event.OutputFields)
			if identity.ContainerID == "" {
				log.Println("skip event without container.id")
				return
			}

			// ----------------------------------------------------------
			// 8.3 Behavior mapping
			// ----------------------------------------------------------
			behavior, err := mapper.Map(event.Tags)
			if err != nil {
				log.Printf("mapper skip rule=%s err=%v", event.Rule, err)
				return
			}

			log.Printf(
				"MAPPED container=%s behavior=%s tactic=%s",
				identity.ContainerID,
				behavior.BehaviorID,
				behavior.TacticID,
			)

			// ----------------------------------------------------------
			// 8.4 Update container snapshot
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
			// 8.5 FSM Correlation + Alert Emission
			// ----------------------------------------------------------
			chains := chainRegistry.GetStartingWith(behavior.TacticID)
			for _, chain := range chains {

				alertObj, err := fsmEngine.Process(
					ctx,
					identity.ContainerID,
					behavior.TacticID,
					chain,
				)
				if err != nil {
					log.Printf("FSM error chain=%s err=%v", chain.ID, err)
					continue
				}

				if alertObj != nil {
					log.Printf(
						"ALERT id=%s severity=%s chain=%s container=%s",
						alertObj.AlertID,
						alertObj.Severity,
						alertObj.Chain.ID,
						identity.ContainerID,
					)

					if err := alertSink.Emit(ctx, alertObj); err != nil {
						log.Printf("alert sink error: %v", err)
					}
				}
			}

			// ----------------------------------------------------------
			// 8.6 Temporal & frequency tracking
			// ----------------------------------------------------------
			_ = redisRepo.AddEvent(
				ctx,
				identity.ContainerID,
				event.Rule,
				event.Time,
			)

			_ = redisRepo.IncrementFrequency(
				ctx,
				identity.ContainerID,
				event.Rule,
				time.Minute,
			)

			log.Printf(
				"INGEST OK container=%s rule=%s score=%d",
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