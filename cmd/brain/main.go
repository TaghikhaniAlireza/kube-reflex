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
	"github.com/TaghikhaniAlireza/kube-reflex/internal/parser"
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

	// 4. ingest with brain hook
	err = falco.IngestFromFile(
		ctx,
		"/app/falco_sample_log.txt",
		falcoRepo,
		func(event falco.Event) {

			score := scoring.ScoreFromPriority(event.Priority)
			if score == 0 {
				return
			}

			identity := parser.ExtractIdentity(event.OutputFields)

			if identity.ContainerID == "" {
				log.Println("skip event without container.id")
				return
			}

			err := redisRepo.UpdateContainerState(
				ctx,
				identity,
				score,
				10*time.Minute,
			)
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