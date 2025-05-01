package protocol

// --- Protocol Definition ---

// Request is the structure for messages sent from the client driver to the server.
type Request struct {
	Command  string // e.g., "CONNECT", "PING", "EXEC", "QUERY_PREPARE", "QUERY_FETCH", "TX_BEGIN", etc.
	Database string // For "CONNECT" command
	Token    string // For "CONNECT" command
	SQL      string // For commands involving SQL statements
	Args     []any  // Arguments for SQL statements (gob handles basic types in any{})
}
