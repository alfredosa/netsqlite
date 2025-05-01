package drivers

import (
	"database/sql/driver"
	"errors"
)

// Ensure SQLResult implements driver.Result at compile time.
var _ driver.Result = &SQLResult{}

// SQLResult holds the result of an Exec operation.
type SQLResult struct {
	lastInsertId int64
	rowsAffected int64
	// Add error field if needed for lazy error reporting, though usually errors are returned directly from ExecContext
}

// LastInsertId returns the database's auto-generated ID
// after, for example, an INSERT into a table with primary
// key.
func (r *SQLResult) LastInsertId() (int64, error) {
	// TODO: Return the actual LastInsertId if available and supported.
	// If not supported, return an error.
	// return r.lastInsertId, nil
	return 0, errors.New("netsqlite: LastInsertId not supported or available") // Placeholder
}

// RowsAffected returns the number of rows affected by the
// query.
func (r *SQLResult) RowsAffected() (int64, error) {
	// TODO: Return the actual RowsAffected count if available.
	// If not available, potentially return 0 or an error depending on expected behavior.
	return r.rowsAffected, nil
}
