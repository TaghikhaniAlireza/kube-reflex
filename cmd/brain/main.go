// cmd/brain/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/action"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/fsm"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/rules"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/taxonomy"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/correlator/velocity"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/decision"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/falco"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/k8s"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
	redisinfra "github.com/TaghikhaniAlireza/kube-reflex/internal/redis"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/scoring"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
)

func main() {

	log.Println("[brain] starting kube-reflex brain")
	ctx := context.Background()

	// ---------------- Database ----------------
	db.RunMigrations()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("[brain] DATABASE_URL not set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("[brain] postgres error: %v", err)
	}
	defer pool.Close()

	alertRepo := db.NewAlertRepository(pool)

	// ---------------- Redis ----------------
	redisClient, err := redisinfra.NewRedisClient()
	if err != nil {
		log.Fatalf("[brain] redis error: %v", err)
	}

	redisRepo := redisinfra.NewRepository(redisClient)

	// ---------------- K8s ----------------
	k8sClient, err := k8s.NewK8sClient()
	if err != nil {
		log.Fatalf("[brain] k8s init error: %v", err)
	}
	defer k8sClient.Close()

	// ---------------- Action Layer ----------------
	fileSink := action.NewStdoutSink() // ساده‌تر برای تست
	pgSink := action.NewPostgresSink(alertRepo)
	actionEngine := action.NewEngine(fileSink, pgSink)

	// ---------------- Decision Layer ----------------
	judge := decision.NewJudge(k8sClient)
	decisionEngine := decision.NewEngine(
		decision.Config{
			AggregationWindow: 10 * time.Second,
		},
		judge,
		actionEngine,
	)
	decisionEngine.Start(ctx)
	decisionInput := decisionEngine.InputChannel()

	// ---------------- Chains ----------------
	chainLoader := rules.NewChainLoader("internal/correlator/rules/chains.yml")
	chainsFile, err := chainLoader.Load()
	if err != nil {
		log.Fatalf("[brain] chain load error: %v", err)
	}
	chainRegistry := rules.NewChainRegistry(chainsFile)

	// ---------------- Mapper ----------------
	mapper, err := taxonomy.NewMapper("internal/correlator/taxonomy/behaviors.yml")
	if err != nil {
		log.Fatalf("[brain] taxonomy error: %v", err)
	}

	// ---------------- FSM ----------------
	fsmStore := fsm.NewStore(redisClient)
	fsmEngine := fsm.NewEngine(fsmStore)

	// ---------------- Velocity ----------------

	// Adaptor: model.Alert → decision.Signal
	alertToSignal := func(alert model.Alert) {
		decisionInput <- decision.Signal{
			ID:          alert.ID,
			ContainerID: alert.ContainerID,
			Source:      decision.SourceVelocity,
			Score:       alert.Score,
			Timestamp:   alert.Timestamp,
			Details:     alert.Details,
		}
	}

	velocityDetector := velocity.NewDetector(redisClient)
	velocityEngine := velocity.NewEngine(velocityDetector, alertToSignal)

	// ---------------- Falco Ingest ----------------
	err = falco.IngestFromFile(ctx, "/app/falco_sample_log.txt", nil, func(event falco.Event) {

		score := scoring.ScoreFromPriority(event.Priority)
		if score == 0 {
			return
		}

		identity := parser.ExtractIdentity(event.OutputFields)
		if identity.ContainerID == "" {
			log.Println("[brain] skip event without container_id")
			return
		}

		behavior, err := mapper.Map(event.Tags)
		if err != nil {
			log.Printf("[brain] mapper skip rule=%s err=%v", event.Rule, err)
			return
		}

		// ---- Velocity
		velocityEngine.Process(ctx, identity.ContainerID, behavior.BehaviorID, event.Time)

		// ---- Redis Snapshot
		_ = redisRepo.UpdateContainerState(ctx, identity, score, 15*time.Minute)

		// ---- FSM
		for _, chain := range chainRegistry.All() {
			detected, err := fsmEngine.Process(ctx, identity.ContainerID, behavior.TacticID, chain)
			if err != nil || detected == nil {
				continue
			}

			// Produce Decision Signal
			signal := decision.Signal{
				ContainerID: identity.ContainerID,
				Source:      decision.SourceFSM,
				Score:       90,
				Timestamp:   event.Time,
				Details: map[string]string{
					"chain_name": chain.ID,
					"category":   behavior.TacticID,
				},
			}

			decisionInput <- signal
		}
	})

	if err != nil {
		log.Fatalf("[brain] ingest error: %v", err)
	}

	log.Println("[brain] finished ingesting falco events")
}