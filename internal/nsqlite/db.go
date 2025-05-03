package nsqlite

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/jackc/puddle/v2"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DBManager struct {
	dbHandles map[string]*puddle.Pool[*sql.DB]
	dbMutex   sync.RWMutex
	datadir   string
}

func NewManager(datadir string) *DBManager {
	// Make sure that the datadir exists
	err := os.MkdirAll(datadir, os.ModePerm)
	if err != nil {
		log.Fatal("Unable to create dir", err)
	}

	return &DBManager{
		dbHandles: make(map[string]*puddle.Pool[*sql.DB]),
		datadir:   datadir,
	}
}

// --- Helper to get/open DB handle ---
func (s *DBManager) AcquirePool(dbName string) (*puddle.Pool[*sql.DB], error) {
	if dbName == "" {
		return nil, status.Error(codes.InvalidArgument, "database_name is required")
	}

	dbpath := filepath.Join(s.datadir, dbName)

	s.dbMutex.RLock()
	dbpool, exists := s.dbHandles[dbpath]
	s.dbMutex.RUnlock()

	if exists {
		return dbpool, nil
	}

	// DB not open, try opening it (use WAL mode, etc.)
	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	// Double-check if another goroutine opened it while waiting for lock
	dbpool, exists = s.dbHandles[dbpath]
	if exists {
		return dbpool, nil
	}

	// This is going to be a poooool
	newDb, err := NewPool(context.Background(), dbpath)
	if err != nil {
		slog.Error("Fatal Pool Creation", "error", err)
		return nil, status.Error(codes.Internal, "failed to created a pool")
	}

	s.dbHandles[dbpath] = newDb
	return newDb, nil
}
