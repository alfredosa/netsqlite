package nsqlite_test

import (
	"context"
	"os"
	"testing"

	"github.com/alfredosa/netsqlite/internal/nsqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBManager_AcquirePool(t *testing.T) {
	// Create a temporary directory
	dir, err := os.MkdirTemp("", "netsqlite-*")
	require.NoError(t, err, "Failed to create temp directory")

	// Ensure the directory has correct permissions
	err = os.Chmod(dir, 0755)
	require.NoError(t, err, "Failed to set directory permissions")

	// Initialize the manager
	s := nsqlite.NewManager(dir)

	// Use a simple filename - not a path
	dbFileName := "wowdb.db"

	// Get the pool
	got, err := s.AcquirePool(dbFileName)
	require.NoError(t, err, "Failed to acquire pool")
	require.NotNil(t, got, "Pool should not be nil")

	// Acquire a connection from the pool
	db, err := got.Acquire(context.Background())
	require.NoError(t, err, "Failed to acquire connection")
	require.NotNil(t, db, "Connection should not be nil")

	// Ping the database
	err = db.Value().Ping()
	assert.NoError(t, err, "Ping should not fail")

	// Release the connection
	db.Release()

	// Close the pool
	got.Close()
}
