package nsqlite

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jackc/puddle/v2"
)

func CreateOrOpen(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		return nil, err
	}

	// Verify WAL mode is enabled
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		slog.Error("Failed to query PRAGMA Journal mode", "error", err)
		return nil, err
	}
	slog.Info("Database journal mode", "mode", journalMode, "database", path)

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS _test_wal (id INTEGER PRIMARY KEY);")
	if err != nil {
		slog.Error("Failed to query PRAGMA Journal mode", "error", err)
		return nil, err
	}

	slog.Info("Created database successfully", "database", path)
	return db, nil
}

func NewPool(ctx context.Context, dbPath string) (*puddle.Pool[*sql.DB], error) {
	constructor := func(context.Context) (*sql.DB, error) {

		return CreateOrOpen(dbPath)
	}
	destructor := func(value *sql.DB) {
		if err := value.Close(); err != nil {
			slog.Error("Failed to close db", "error", err)
		}
	}

	// TODO: Make it configurable
	return puddle.NewPool(&puddle.Config[*sql.DB]{
		Constructor: constructor,
		Destructor:  destructor,
		MaxSize:     int32(5),
	})

}
