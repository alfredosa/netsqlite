package drivers

import (
	"database/sql/driver"
	"fmt"
)

// Ensure SQLTx implements driver.Tx at compile time.
var _ driver.Tx = &SQLTx{}

// SQLTx represents an active database transaction.
type SQLTx struct {
	conn *SQLConn
	// You might need a flag to ensure Commit/Rollback is called only once
	// done bool
}

// Commit commits the transaction.
func (tx *SQLTx) Commit() error {
	if tx.conn == nil || tx.conn.closed {
		return driver.ErrBadConn
	}
	// if tx.done { return errors.New("netsqlite: transaction already committed or rolled back") }

	// TODO: Send "COMMIT" command (or equivalent) to the server.
	fmt.Println("Transaction Commit")

	// err := tx.conn.protoClient.Commit(context.Background()) // Example
	// if err != nil {
	//    // Should we automatically rollback on commit error? Depends on server behavior.
	//    // tx.Rollback() // Maybe?
	//    return err // Convert server error
	// }

	// tx.done = true
	// tx.conn.inTx = false // Update connection state
	return nil
}

// Rollback aborts the transaction.
func (tx *SQLTx) Rollback() error {
	if tx.conn == nil || tx.conn.closed {
		// Standard library often ignores rollback errors on closed connections
		return nil // Or return driver.ErrBadConn if preferred
	}
	// if tx.done { return errors.New("netsqlite: transaction already committed or rolled back") }

	// TODO: Send "ROLLBACK" command (or equivalent) to the server.
	fmt.Println("Transaction Rollback")

	// err := tx.conn.protoClient.Rollback(context.Background()) // Example
	// if err != nil {
	//     // Log error? Standard library often ignores rollback errors.
	//     // return err // Convert server error
	// }

	// tx.done = true
	// tx.conn.inTx = false // Update connection state
	return nil
}
