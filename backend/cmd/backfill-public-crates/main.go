package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := os.Getenv("SQLITE_PATH")
	if dbPath == "" {
		dbPath = "/data/db/cratedrop.sqlite" // Default path inside container
	}

	fmt.Printf("Opening database at %s...\n", dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Update all playlists to be public by default, EXCEPT the 'unsorted' one (which is default)
	// We assume 'unsorted' is the one with is_default = TRUE.
	// If there are other default playlists, they will also be kept private, which is safe.
	query := `
		UPDATE playlists 
		SET is_public = TRUE 
		WHERE is_default = FALSE
	`

	fmt.Println("Executing backfill query...")
	result, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to execute update: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Backfill complete. Updated %d playlists to be public.\n", rowsAffected)
}
