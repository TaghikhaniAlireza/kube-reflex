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
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/velocity"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/falco"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
	redisinfra "github.com/TaghikhaniAlireza/kube-reflex/internal/redis"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/scoring"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

func main() {
	log.Println("[brain] starting kube-reflex brain")

	ctx := context.Background()

	// ------------------------------------------------------------------
	// 1. Database migrations
	// ------------------------------------------------------------------
	db.RunMigrations()

	// ------------------------------------------------------------------
	// 2. PostgreSQL
	// ------------------------------------------------------------------
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("[brain] DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("[brain] failed to connect postgres: %v", err)
	}
	defer pool.Close()

	falcoRepo := db.NewFalcoRepository(pool)
	alertRepo := db.NewAlertRepository(pool)

	// ------------------------------------------------------------------
	// 3. Redis
	// ------------------------------------------------------------------
	redisClient, err := redisinfra.NewRedisClient()
	if err != nil {
		log.Fatalf("[brain] redis init error: %v", err)
	}

	redisRepo := redisinfra.NewRepository(redisClient)

	// ------------------------------------------------------------------
	// 4. Alert Sinks (Composite)
	// ------------------------------------------------------------------
	fileSink, err := alert.NewFileSink("/app/alerts.log")
	if err != nil {
		log.Fatalf("[brain] file sink error: %v", err)
	}

	pgSink := alert.NewPostgresSink(alertRepo)
	alertSink := alert.NewMultiSink(fileSink, pgSink)

	// ------------------------------------------------------------------
	// 5. Velocity Alert Channel ✅ FIX
	// ------------------------------------------------------------------
	alertCh := make(chan model.Alert, 100)

	go func() {
		for a := range alertCh {
			alertCopy := a // ✅ avoid pointer aliasing
			if err := alertSink.Emit(ctx, &alertCopy); err != nil {
				log.Printf("[brain] velocity alert emit error: %v", err)
			}
		}
	}()

	// ------------------------------------------------------------------
	// 6. Load MITRE Chains
	// ------------------------------------------------------------------
	chainLoader := rules.NewChainLoader(
		"internal/correlator/rules/chains.yml",
	)

	chainFile, err := chainLoader.Load()
	if err != nil {
		log.Fatalf("[brain] failed to load chains.yml: %v", err)
	}

	chainRegistry := rules.NewChainRegistry(chainFile)
	log.Printf("[brain] loaded %d MITRE chains", len(chainRegistry.All()))

	// ------------------------------------------------------------------
	// 7. Behavior Mapper
	// ------------------------------------------------------------------
	mapper, err := taxonomy.NewMapper(
		"internal/correlator/taxonomy/behaviors.yml",
	)
	if err != nil {
		log.Fatalf("[brain] taxonomy mapper error: %v", err)
	}

	// ------------------------------------------------------------------
	// 8. FSM Engine
	// ------------------------------------------------------------------
	fsmStore := fsm.NewStore(redisClient)
	fsmEngine := fsm.NewEngine(fsmStore)

	// ------------------------------------------------------------------
	// 9. Velocity Engine ✅ ALIGNED
	// ------------------------------------------------------------------
	velocityDetector := velocity.NewDetector(redisClient)
	velocityEngine := velocity.NewEngine(
		velocityDetector,
		alertCh, // ✅ model.Alert channel
	)

	// ------------------------------------------------------------------
	// 10. Falco Ingest Loop
	// ------------------------------------------------------------------
	err = falco.IngestFromFile(
		ctx,
		"/app/falco_sample_log.txt",
		falcoRepo,
		func(event falco.Event) {

			// ---- Scoring ---------------------------------------------
			score := scoring.ScoreFromPriority(event.Priority)
			if score == 0 {
				return
			}

			// ---- Identity --------------------------------------------
			identity := parser.ExtractIdentity(event.OutputFields)
			if identity.ContainerID == "" {
				log.Println("[brain] skip event without container_id")
				return
			}

			// ---- Behavior Mapping ------------------------------------
			behavior, err := mapper.Map(event.Tags)
			if err != nil {
				log.Printf("[brain] mapper skip rule=%s err=%v", event.Rule, err)
				return
			}

			log.Printf(
				"[brain] mapped container=%s behavior=%s tactic=%s",
				identity.ContainerID,
				behavior.BehaviorID,
				behavior.TacticID,
			)

			// ---- Velocity Engine (Volume / Pressure) ----------------
			velocityEngine.Process(
				ctx,
				identity.ContainerID,
				behavior.BehaviorID,
				event.Time,
			)

			// ---- Update Container Snapshot ---------------------------
			if err := redisRepo.UpdateContainerState(
				ctx,
				identity,
				score,
				15*time.Minute,
			); err != nil {
				log.Printf("[brain] redis UpdateContainerState error: %v", err)
			}

			// ---- FSM Correlation (Sequence) --------------------------
			for _, chain := range chainRegistry.All() {

				alertObj, err := fsmEngine.Process(
					ctx,
					identity.ContainerID,
					behavior.TacticID,
					chain,
				)
				if err != nil {
					log.Printf("[brain] FSM error chain=%s err=%v", chain.ID, err)
					continue
				}

				if alertObj != nil {
					log.Printf(
						"[brain] EMIT FSM ALERT id=%s severity=%s chain=%s",
						alertObj.AlertID,
						alertObj.Severity,
						alertObj.Chain.ID,
					)

					if err := alertSink.Emit(ctx, alertObj); err != nil {
						log.Printf("[brain] alert sink error: %v", err)
					}
				}
			}
		},
	)

	if err != nil {
		log.Fatalf("[brain] ingest error: %v", err)
	}

	log.Println("[brain] finished ingesting falco events")
}