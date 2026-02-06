//cmd/brain/main.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
	"github.com/TaghikhaniAlireza/kube-reflex/internal/falco"
)

func main() {
	log.Println("starting kube-reflex brain...")

	// 1. run migrations
	db.RunMigrations()

	// 2. connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// 3. repository
	falcoRepo := db.NewFalcoRepository(pool)

	// 4. ingest falco log file
	err = falco.IngestFromFile(
		context.Background(),
		"/app/falco_sample_log.txt",
		falcoRepo,
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("brain finished ingesting falco events")
}