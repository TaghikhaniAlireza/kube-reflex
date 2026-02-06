//cmd/brain/main.go
package main

import (
	"log"

	"github.com/TaghikhaniAlireza/kube-reflex/internal/db"
)

func main() {
	log.Println("starting kube-reflex brain...")

	db.RunMigrations()

	log.Println("brain started successfully")
}