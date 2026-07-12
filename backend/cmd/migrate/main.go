package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/rozy/backend/internal/platform/db"
)

func main() {
	_ = godotenv.Load()
	down := flag.Bool("down", false, "roll back one migration")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	if *down {
		if err := db.RunMigrationsDown(databaseURL); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		fmt.Println("migration rolled back")
		return
	}

	if err := db.RunMigrations(databaseURL); err != nil {
		log.Fatalf("migrate up: %v", err)
	}
	fmt.Println("migrations up to date")
}
