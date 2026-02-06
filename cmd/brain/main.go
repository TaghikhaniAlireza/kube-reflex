//cmd/brain/main.go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/falco"
	redisinfra "github.com/TaghikhaniAlireza/kube-reflex/internal/redis"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/scoring"
)

func main() {
	log.Println("starting kube-reflex brain...")

	ctx := context.Background()

	// 1. migrations
	db.RunMigrations()

	// 2. database
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

	// 3. redis
	redisClient, err := redisinfra.NewRedisClient()
	if err != nil {
		log.Fatal(err)
	}
	redisRepo := redisinfra.NewRepository(redisClient)

	// 4. ingest with scoring hook
	err = falco.IngestFromFile(
		ctx,
		"/app/falco_sample_log.txt",
		falcoRepo,
		func(event falco.Event) {
			log.Printf(
				"HOOK CALLED ✅ rule=%q priority=%q",
				event.Rule,
				event.Priority,
			)
			score := scoring.ScoreFromPriority(event.Priority)
			if score == 0 {
				return
			}

			key := "risk:global"
			_, err := redisRepo.IncrementScore(ctx, key, score, 10*time.Minute)
			if err != nil {
				log.Printf("redis error: %v", err)
			}
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("brain finished ingesting falco events")
}
