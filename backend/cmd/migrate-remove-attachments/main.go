package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	confirm := flag.Bool("confirm", false, "confirm destructive migration")
	flag.Parse()

	if !*confirm {
		fmt.Println("This migration is destructive and will TRUNCATE email and DROP attachment.")
		fmt.Println("Re-run with --confirm to proceed.")
		return
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DROP TABLE IF EXISTS attachment CASCADE`); err != nil {
		log.Fatalf("Failed to drop attachment table: %v", err)
	}
	if _, err := tx.Exec(`TRUNCATE TABLE email`); err != nil {
		log.Fatalf("Failed to truncate email table: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit migration: %v", err)
	}

	log.Println("Migration complete: attachment table dropped and email table truncated")
}
