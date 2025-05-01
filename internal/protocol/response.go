package protocol

// Response is the structure for messages sent from the server back to the client.
type Response struct {
	Error        string   // Non-empty if an error occurred
	Result       any      // General purpose result (e.g., "PONG", LastInsertID, RowsAffected)
	Columns      []string // For query results (sent once)
	Row          []any    // For query results (sent per row, nil when done)
	RowsAffected int64    // Specific field for EXEC results
	LastInsertId int64    // Specific field for EXEC results
}
