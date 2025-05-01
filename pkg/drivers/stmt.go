package drivers

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
)

// Ensure SQLStmt implements driver.Stmt and related interfaces at compile time.
var _ driver.Stmt = &SQLStmt{}
var _ driver.StmtExecContext = &SQLStmt{}
var _ driver.StmtQueryContext = &SQLStmt{}

// SQLStmt represents a prepared statement.
type SQLStmt struct {
	conn  *SQLConn
	query string
	// serverHandle interface{} // Handle/ID returned by server upon Prepare
	// numInput     int         // Number of placeholders, if known
	closed bool
}

// Close closes the statement.
func (s *SQLStmt) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	// TODO: Inform the server to release resources associated with this statement handle (if applicable).
	fmt.Printf("Statement Close: %s\n", s.query)
	// err := s.conn.protoClient.CloseStatement(context.Background(), s.serverHandle) // Example
	// return err
	return nil
}

// NumInput returns the number of placeholder parameters.
// If unknown, return -1.
func (s *SQLStmt) NumInput() int {
	// TODO: Determine the number of placeholders if possible (e.g., from server response during Prepare).
	// If your protocol/server doesn't provide this, you might need to parse the query string
	// or just return -1.
	// return s.numInput
	return -1 // Placeholder
}

// ExecContext executes a prepared statement query that does not return rows.
func (s *SQLStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if s.closed {
		return nil, errors.New("netsqlite: statement closed")
	}
	// TODO: Implement execution of prepared statement.
	// 1. Send the statement handle/ID and arguments (args) to the server.
	// 2. Receive result (LastInsertId, RowsAffected).
	// 3. Handle errors.
	fmt.Printf("Statement ExecContext: %s, Args: %v\n", s.query, args)

	// This might delegate to a connection method or use the protoClient directly
	// result, err := s.conn.protoClient.ExecPrepared(ctx, s.serverHandle, args) // Example
	// if err != nil {
	//     return nil, err // Convert error
	// }
	// return result, nil

	// Minimal placeholder delegating to connection's exec method (less efficient if server supports true prepare)
	return s.conn.execContext(ctx, s.query, args)
}

// QueryContext executes a prepared statement query that returns rows.
func (s *SQLStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if s.closed {
		return nil, errors.New("netsqlite: statement closed")
	}
	// TODO: Implement query execution of prepared statement.
	// 1. Send the statement handle/ID and arguments (args) to the server.
	// 2. Receive column information and row data stream/iterator.
	// 3. Wrap the results in your SQLRows implementation.
	// 4. Handle errors.
	fmt.Printf("Statement QueryContext: %s, Args: %v\n", s.query, args)

	// This might delegate to a connection method or use the protoClient directly
	// serverRows, err := s.conn.protoClient.QueryPrepared(ctx, s.serverHandle, args) // Example
	// if err != nil {
	//     return nil, err // Convert error
	// }
	// return NewSQLRows(s.conn, serverRows), nil

	// Minimal placeholder delegating to connection's query method (less efficient if server supports true prepare)
	return s.conn.queryContext(ctx, s.query, args)
}

// --- Deprecated Methods (Provide for compatibility) ---

// Exec executes a prepared statement query that does not return rows.
func (s *SQLStmt) Exec(args []driver.Value) (driver.Result, error) {
	// Convert driver.Value to driver.NamedValue for ExecContext
	namedArgs := make([]driver.NamedValue, len(args))
	for i, v := range args {
		namedArgs[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return s.ExecContext(context.Background(), namedArgs)
}

// Query executes a prepared statement query that returns rows.
func (s *SQLStmt) Query(args []driver.Value) (driver.Rows, error) {
	// Convert driver.Value to driver.NamedValue for QueryContext
	namedArgs := make([]driver.NamedValue, len(args))
	for i, v := range args {
		namedArgs[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return s.QueryContext(context.Background(), namedArgs)
}
