package protocol

// TODO: BIG TODO we need to use slog. log sucks

import (
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"path/filepath"
	"sync"

	db "github.com/alfredosa/netsqlite/internal/sqlite"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

type State struct {
	validTokens map[string]bool
	// TODO: I think adding some sort of actual pooling would be best for the future
	// NOTE: Think about it a bit more this is just prototyping tho
	dbs     map[string]*sql.DB
	datadir string
}

// handleConnection manages a single client connection.
func (s *State) handleConnection(ctx context.Context, conn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()    // Decrement counter when connection handling finishes
	defer conn.Close() // Ensure connection is closed eventually
	defer log.Printf("Connection closed for: %s", conn.RemoteAddr())

	var currentDb *sql.DB
	defer func() {
		if currentDb != nil {
			currentDb.Close()
			slog.Info("closed database successfully")
		}
	}()

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)

	// NOTE: Unauthenticated by default
	isAuthenticated := false

	for {
		select {
		case <-ctx.Done():
			log.Printf("Shutdown initiated, closing connection for %s", conn.RemoteAddr())
			encoder.Encode(Response{Error: "Server shutting down"})
			return
		default:
			// Continue reading
		}

		var req Request
		// NOTE: Question, does this block???
		err := decoder.Decode(&req)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				log.Printf("Client %s disconnected.", conn.RemoteAddr())
			} else if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				slog.Error("Read timeout", "remote_addr", conn.RemoteAddr(), "error", err)
				continue
			} else {
				log.Printf("Error decoding request from %s: %v", conn.RemoteAddr(), err)
			}
			return
		}

		log.Printf("Received [%s] from %s", req.Command, conn.RemoteAddr())

		// --- Initial Authentication Check ---
		if !isAuthenticated && req.Command != "CONNECT" {
			log.Printf("Unauthorized command '%s' from %s", req.Command, conn.RemoteAddr())
			if err := encoder.Encode(Response{Error: "Authentication required"}); err != nil {
				log.Printf("Error sending auth required response to %s: %v", conn.RemoteAddr(), err)
			}
			continue // Don't process further, wait for CONNECT
		}

		// --- Command Processing ---
		var resp Response
		switch req.Command {
		case "CONNECT":
			// *** VERY BASIC/INSECURE AUTH - REPLACE WITH REAL MECHANISM ***
			if token := req.Token; s.validTokens[token] {
				log.Printf("Client %s authenticated successfully.", conn.RemoteAddr())
				isAuthenticated = true
				resp = Response{} // Empty response means success
			} else {
				log.Printf("Authentication failed for %s (token: %s)", conn.RemoteAddr(), token)
				resp = Response{Error: "Invalid authentication token"}
				isAuthenticated = false // Ensure state is false
				break
			}

			if req.Database == "" {
				resp = Response{Error: "database is required."} // NOTE: Is it? Random name gen?
				break
			}

			dbpath := filepath.Join(s.datadir, req.Database)

			database := db.CreateOrOpen(dbpath)
			s.dbs[dbpath] = database
			currentDb = database

		case "PING":
			err := currentDb.Ping()
			if err != nil {
				resp = Response{Error: err.Error()}
			}

			resp = Response{Result: "PONG"}

		case "EXEC":
			if req.SQL == "" {
				resp = Response{Error: "EXEC command requires SQL"}
				break
			}
			log.Printf("Executing SQL for %s: %s | Args: %v", conn.RemoteAddr(), req.SQL, req.Args)

			// Use ExecContext for context propagation (though client driver needs to send it)
			// For now, use Background context on server-side unless protocol adds context ID
			sqlResult, err := currentDb.ExecContext(context.Background(), req.SQL, req.Args...)
			if err != nil {
				log.Printf("SQL Exec error for %s: %v", conn.RemoteAddr(), err)
				resp = Response{Error: fmt.Sprintf("Execution failed: %v", err)}
			} else {
				rowsAffected, errAff := sqlResult.RowsAffected()
				lastInsertId, errLid := sqlResult.LastInsertId()

				if errAff != nil {
					log.Printf("Error getting RowsAffected for %s: %v", conn.RemoteAddr(), errAff)
					// Depending on driver needs, maybe send partial success or specific error
					rowsAffected = -1 // Indicate error or unavailability
				}
				if errLid != nil {
					log.Printf("Error getting LastInsertId for %s: %v", conn.RemoteAddr(), errLid)
					lastInsertId = -1 // Indicate error or unavailability
				}
				resp = Response{RowsAffected: rowsAffected, LastInsertId: lastInsertId}
				log.Printf("Exec successful for %s: RowsAffected=%d, LastInsertId=%d", conn.RemoteAddr(), rowsAffected, lastInsertId)
			}

		// --- Placeholders for other commands ---
		case "QUERY_PREPARE": // Simplified Query handling for barebones
			// TODO: Implement db.QueryContext, send back columns, then handle QUERY_FETCH
			log.Printf("QUERY command received (not fully implemented) for %s: %s | Args: %v", conn.RemoteAddr(), req.SQL, req.Args)
			resp = Response{Error: "QUERY not fully implemented in this barebones server"}

		// case "QUERY_FETCH":
		//  // TODO: Read next row from activeQueryRows, serialize, send back. Send nil row on EOF.
		//  resp = Response{Error: "QUERY_FETCH not implemented"}

		case "TX_BEGIN":
			// TODO: Implement db.BeginTx, store the *sql.Tx associated with this connection
			resp = Response{Error: "Transactions not implemented"}

		// case "COMMIT":
		//  // TODO: Call Commit() on the stored *sql.Tx
		//  resp = Response{Error: "COMMIT not implemented"}

		// case "ROLLBACK":
		//  // TODO: Call Rollback() on the stored *sql.Tx
		//  resp = Response{Error: "ROLLBACK not implemented"}

		default:
			log.Printf("Unknown command '%s' from %s", req.Command, conn.RemoteAddr())
			resp = Response{Error: fmt.Sprintf("Unknown command: %s", req.Command)}
		}

		// Send the response back to the client
		if err := encoder.Encode(&resp); err != nil {
			log.Printf("Error encoding response to %s: %v", conn.RemoteAddr(), err)
			return // Assume connection is broken if we can't send response
		}
	}
}
