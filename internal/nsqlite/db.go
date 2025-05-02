package nsqlite

import (
	"database/sql"
	"log"
	"log/slog"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

func CreateOrOpen(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Verify WAL mode is enabled
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		log.Fatalf("Failed to query journal mode: %v", err)
	}
	slog.Info("Database journal mode", "mode", journalMode, "database", path)

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS _test_wal (id INTEGER PRIMARY KEY);")
	if err != nil {
		log.Fatalf("Failed to create test table: %v", err)
	}

	slog.Info("Created database successfully", "database", path)
	return db
}
