// cmd/brain/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/rules"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/taxonomy"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/falco"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
	redisinfra "github.com/TaghikhaniAlireza/kube-reflex/internal/redis"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/scoring"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/fsm"
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
	// 4. Load Chain Definitions (NEW – Step 2 result)
	// ------------------------------------------------------------------
	chainLoader := rules.NewChainLoader("internal/correlator/rules/chains.yml")

	chainFile, err := chainLoader.Load()
	if err != nil {
		log.Fatalf("failed to load chains.yml: %v", err)
	}

	chainRegistry := rules.NewChainRegistry(chainFile)

	log.Printf("loaded %d chains", len(chainRegistry.All()))

	// ------------------------------------------------------------------
	// 5. Mapper initialization
	// ------------------------------------------------------------------
	mapperInstance, err := taxonomy.NewMapper(
		"internal/correlator/taxonomy/behaviors.yml",
	)
	if err != nil {
		log.Fatalf("failed to initialize taxonomy mapper: %v", err)
	}

	// ------------------------------------------------------------------
	// 6. Falco ingest + Brain hook
	// ------------------------------------------------------------------
	err = falco.IngestFromFile(
		ctx,
		"/app/falco_sample_log.txt",
		falcoRepo,
		func(event falco.Event) {

			// ----------------------------------------------------------
			// 6.1 Static scoring
			// ----------------------------------------------------------
			score := scoring.ScoreFromPriority(event.Priority)
			if score == 0 {
				return
			}

			// ----------------------------------------------------------
			// 6.2 Identity extraction
			// ----------------------------------------------------------
			identity := parser.ExtractIdentity(event.OutputFields)
			if identity.ContainerID == "" {
				log.Println("skip event without container.id")
				return
			}

			// ----------------------------------------------------------
			// 6.3 Behavior Mapping
			// ----------------------------------------------------------
			mappedBehavior, err := mapperInstance.Map(event.Tags)
			if err != nil {
				log.Printf("mapper skip rule=%s err=%v", event.Rule, err)
				return
			}

			log.Printf(
				"MAPPED container=%s behavior=%s tactic=%s",
				identity.ContainerID,
				mappedBehavior.BehaviorID,
				mappedBehavior.TacticID,
			)

			// ----------------------------------------------------------
			// 6.4 Update container snapshot
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
			// 6.5 FSM entry point (Phase 1 – logging only)
			// ----------------------------------------------------------
			fsmStore := fsm.NewStore(redisClient)
			fsmEngine := fsm.NewEngine(fsmStore)
				
			chains := chainRegistry.GetStartingWith(mappedBehavior.TacticID)
			for _, chain := range chains {
				log.Printf(
					"FSM CANDIDATE container=%s chain=%s next=%s",
					identity.ContainerID,
					chain.ID,
					chain.Sequence[0],
				)
				fsmEngine.Process(
					ctx,
					identity.ContainerID,
					mappedBehavior.TacticID,
					chain,
				)
			}

			// ----------------------------------------------------------
			// 6.6 Temporal + frequency tracking
			// ----------------------------------------------------------
			eventType := event.Rule

			_ = redisRepo.AddEvent(
				ctx,
				identity.ContainerID,
				eventType,
				event.Time,
			)

			_ = redisRepo.IncrementFrequency(
				ctx,
				identity.ContainerID,
				eventType,
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