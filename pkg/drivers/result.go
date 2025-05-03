package drivers

import (
	"database/sql/driver"
	"errors"
)

// SQLResult holds results from Exec operations.
type SQLResult struct {
	rowsAffected int64
	lastInsertId int64
}

var _ driver.Result = &SQLResult{}

// LastInsertId returns the ID of the last inserted row.
// Returns error if not supported or unavailable (-1 from server).
func (r *SQLResult) LastInsertId() (int64, error) {
	if r.lastInsertId == -1 { // Convention for unavailable
		return 0, errors.New("netsqlite: LastInsertId not available or not supported")
	}
	return r.lastInsertId, nil
}

// RowsAffected returns the number of rows affected.
// Returns error if not supported or unavailable (-1 from server).
func (r *SQLResult) RowsAffected() (int64, error) {
	if r.rowsAffected == -1 { // Convention for unavailable
		return 0, errors.New("netsqlite: RowsAffected not available")
	}
	return r.rowsAffected, nil
}
