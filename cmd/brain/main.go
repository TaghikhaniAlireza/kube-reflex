// cmd/brain/main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	"github.com/TaghikhaniAlireza/kube-reflex/internal/model"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
	redisinfra "github.com/TaghikhaniAlireza/kube-reflex/internal/redis"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/scoring"
)

const debugLogPath = "c:\\Users\\CERT-01\\Documents\\kube-reflex\\.cursor\\debug.log"

func debugLog(location, message, hypothesisId string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	payload := map[string]interface{}{
		"location": location, "message": message, "hypothesisId": hypothesisId,
		"timestamp": time.Now().UnixMilli(), "data": data,
	}
	line, _ := json.Marshal(payload)
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	f.Write(append(line, '\n'))
	f.Close()
}

func main() {
	log.Println("[brain] starting kube-reflex brain")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	aggregationWindow := 10 * time.Second
	judge := decision.NewJudge(k8sClient)
	decisionEngine := decision.NewEngine(
		decision.Config{AggregationWindow: aggregationWindow},
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

	// Adaptor: model.Alert → decision.Signal using a channel bridge.
	alertCh := make(chan model.Alert, 128)

	go func() {
		for alert := range alertCh {
			sig := decision.Signal{
				ContainerID: alert.Entity.ID,
				Source:      decision.SourceVelocity,
				Score:       alert.Score,
				Timestamp:   alert.Timestamps.CompletedAt,
				Details:     nil,
			}
			select {
			case decisionInput <- sig:
			default:
				// #region agent log
				debugLog("main.go:velocityBridge", "decision channel full, dropping velocity signal", "H2", map[string]interface{}{"containerId": alert.Entity.ID})
				// #endregion
			}
		}
	}()

	velocityDetector := velocity.NewDetector(redisClient)
	velocityEngine := velocity.NewEngine(velocityDetector, alertCh)

	// ---------------- Falco Webhook (event channel) ----------------
	eventsCh := make(chan falco.Event, 256)
	webhookHandler := falco.NewWebhookHandler(eventsCh)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventsCh:
				if !ok {
					return
				}
				score := scoring.ScoreFromPriority(event.Priority)
				if score == 0 {
					continue
				}
				identity := parser.ExtractIdentity(event.OutputFields)
				if identity.ContainerID == "" {
					log.Println("[brain] skip event without container_id")
					continue
				}
				behavior, err := mapper.Map(event.Tags)
				if err != nil {
					log.Printf("[brain] mapper skip rule=%s err=%v", event.Rule, err)
					continue
				}
				velocityEngine.Process(ctx, identity.ContainerID, behavior.BehaviorID, event.Time)
				_ = redisRepo.UpdateContainerState(ctx, identity, score, 15*time.Minute)
				for _, chain := range chainRegistry.All() {
					detected, err := fsmEngine.Process(ctx, identity.ContainerID, behavior.TacticID, chain)
					if err != nil || detected == nil {
						continue
					}
					sig := decision.Signal{
						ContainerID: identity.ContainerID,
						Source:      decision.SourceFSM,
						Score:      90,
						Timestamp:  event.Time,
						Details: map[string]string{
							"chain_name": chain.ID,
							"category":   behavior.TacticID,
						},
					}
					select {
					case decisionInput <- sig:
					default:
						// #region agent log
						debugLog("main.go:fsmDecisionSend", "decision channel full, dropping FSM signal", "H2", map[string]interface{}{"containerId": identity.ContainerID, "chain": chain.ID})
						// #endregion
					}
				}
			}
		}
	}()

	// ---------------- HTTP Server (goroutine) ----------------
	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/alerts", webhookHandler)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("[brain] webhook listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[brain] http server: %v", err)
		}
	}()

	// ---------------- Graceful shutdown ----------------
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("[brain] shutdown signal received, draining...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[brain] http shutdown: %v", err)
	}

	// Allow decision engine timers to flush buffered signals
	time.Sleep(aggregationWindow + 2*time.Second)
	log.Println("[brain] stopped")
}