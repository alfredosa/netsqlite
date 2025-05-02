// file: internal/server/grpc_server.go (or similar)
package proto

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"sync" // For managing DB handles potentially

	// Import generated protobuf code (adjust path based on your module)
	pb "github.com/alfredosa/netsqlite/internal/generated/proto/v1"
	"github.com/alfredosa/netsqlite/internal/nsqlite"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// netsqliteServer implements the NetsqliteServiceServer interface.
type netsqliteServer struct {
	pb.UnimplementedNetsqliteServiceServer // Recommended for forward compatibility

	// --- Server State ---
	// TODO: VERY basic token check - REPLACE with secure validation
	validTokens map[string]bool

	// Map to hold open database connections (simple approach)
	// Key: database_name from request
	// Value: *sql.DB handle
	// Needs mutex for concurrent access
	dbManager *DBManager
}

type DBManager struct {
	dbHandles map[string]*sql.DB
	dbMutex   sync.RWMutex
	datadir   string
}

func NewManager(datadir string) *DBManager {
	return &DBManager{
		dbHandles: make(map[string]*sql.DB),
		datadir:   datadir,
	}
}

// NewNetsqliteServer creates a new server instance.
func NewNetsqliteServer(tokens map[string]bool, datadir string) *netsqliteServer {
	manager := NewManager(datadir)

	return &netsqliteServer{
		validTokens: tokens,
		dbManager:   manager,
	}
}

// --- Helper to get/open DB handle ---
func (s *DBManager) getDB(dbName string) (*sql.DB, error) {
	if dbName == "" {
		return nil, status.Error(codes.InvalidArgument, "database_name is required")
	}

	dbpath := filepath.Join(s.datadir, dbName)

	s.dbMutex.RLock()
	db, exists := s.dbHandles[dbpath]
	s.dbMutex.RUnlock()

	if exists {
		return db, nil
	}

	// DB not open, try opening it (use WAL mode, etc.)
	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()

	// Double-check if another goroutine opened it while waiting for lock
	db, exists = s.dbHandles[dbpath]
	if exists {
		return db, nil
	}

	newDb := nsqlite.CreateOrOpen(dbpath)

	s.dbHandles[dbpath] = newDb
	return newDb, nil
}

// // CloseAllDBs cleans up database handles on shutdown.
// func (s *netsqliteServer) CloseAllDBs() {
// 	s.dbMutex.Lock()
// 	defer s.dbMutex.Unlock()
// 	log.Println("Closing all database handles...")
// 	for name, db := range s.dbHandles {
// 		log.Printf("Closing DB: %s", name)
// 		db.Close()
// 	}
// 	s.dbHandles = make(map[string]*sql.DB) // Clear the map
// }

// --- Service Method Implementations ---
func (s *netsqliteServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	log.Println("Received Ping request")
	db, err := s.dbManager.getDB(req.DatabaseName)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		log.Printf("Actual DB Ping failed for %s: %v", req.DatabaseName, err)
		return nil, status.Errorf(codes.Internal, "database ping failed for %s: %v", req.DatabaseName, err)
	}

	return &pb.PingResponse{Message: fmt.Sprintf("PONG for db %s", req.DatabaseName)}, nil
}

func (s *netsqliteServer) Exec(ctx context.Context, req *pb.ExecRequest) (*pb.ExecResponse, error) {
	log.Printf("Received Exec request for DB: %s, SQL: %s", req.DatabaseName, req.Sql)

	db, err := s.dbManager.getDB(req.DatabaseName)
	if err != nil {
		return nil, err
	}

	// Convert protobuf Value args to []any
	args := make([]any, len(req.Args))
	for i, val := range req.Args {
		args[i] = val.AsInterface() // Convert protobuf Value to Go type
	}

	sqlResult, err := db.ExecContext(ctx, req.Sql, args...)
	if err != nil {
		log.Printf("Exec failed for DB '%s': %v", req.DatabaseName, err)
		// TODO: Map SQLite errors to gRPC codes more specifically
		return nil, status.Errorf(codes.Internal, "SQL execution failed: %v", err)
	}

	// TODO: don't ignore error
	rowsAffected, _ := sqlResult.RowsAffected() // Ignore error for simplicity here
	lastInsertId, _ := sqlResult.LastInsertId() // Ignore error for simplicity here

	log.Printf("Exec successful for DB '%s': RowsAffected=%d, LastInsertId=%d", req.DatabaseName, rowsAffected, lastInsertId)
	return &pb.ExecResponse{
		RowsAffected: rowsAffected,
		LastInsertId: lastInsertId,
	}, nil
}

func (s *netsqliteServer) Query(req *pb.QueryRequest, stream pb.NetsqliteService_QueryServer) error {
	log.Printf("Received Query request for DB: %s, SQL: %s", req.DatabaseName, req.Sql)

	db, err := s.dbManager.getDB(req.DatabaseName)
	if err != nil {
		return err // getDB returns gRPC status error
	}

	// Convert protobuf Value args to []any
	args := make([]any, len(req.Args))
	for i, val := range req.Args {
		args[i] = val.AsInterface()
	}

	rows, err := db.QueryContext(stream.Context(), req.Sql, args...)
	if err != nil {
		log.Printf("Query failed for DB '%s': %v", req.DatabaseName, err)
		return status.Errorf(codes.Internal, "SQL query failed: %v", err)
	}
	defer rows.Close() // Ensure rows are closed

	// 1. Get column names and send first message
	columns, err := rows.Columns()
	if err != nil {
		log.Printf("Failed to get columns for DB '%s': %v", req.DatabaseName, err)
		return status.Errorf(codes.Internal, "failed to get columns: %v", err)
	}

	columnResp := &pb.QueryResponse{
		Result: &pb.QueryResponse_Columns{
			Columns: &pb.Columns{Names: columns},
		},
	}
	if err := stream.Send(columnResp); err != nil {
		log.Printf("Failed to send columns to client: %v", err)
		return status.Errorf(codes.Internal, "failed to send columns: %v", err)
	}
	log.Printf("Sent columns for query on DB '%s': %v", req.DatabaseName, columns)

	// 2. Stream rows
	// Prepare a slice of pointers for Scan (needed for handling NULLs properly)
	colCount := len(columns)
	values := make([]any, colCount)
	scanArgs := make([]any, colCount)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			log.Printf("Failed to scan row for DB '%s': %v", req.DatabaseName, err)
			return status.Errorf(codes.Internal, "failed to scan row: %v", err)
		}

		// Convert scanned Go values to protobuf Values
		protoValues := make([]*structpb.Value, colCount)
		for i, v := range values {
			// structpb.NewValue handles basic types (int64, float64, bool, string, []byte, nil)
			// It might need help with time.Time or other complex types depending on driver
			protoValues[i], err = structpb.NewValue(v)
			if err != nil {
				log.Printf("Failed to convert value to protobuf Value: %v", err)
				return status.Errorf(codes.Internal, "failed to convert value: %v", err)
			}
		}

		rowResp := &pb.QueryResponse{
			Result: &pb.QueryResponse_Row{
				Row: &pb.Row{Values: protoValues},
			},
		}

		// Send the row
		if err := stream.Send(rowResp); err != nil {
			log.Printf("Failed to send row to client: %v", err)
			// This usually means the client disconnected
			return status.Errorf(codes.Internal, "failed to send row: %v", err)
		}
		// Check context cancellation frequently during long streams
		select {
		case <-stream.Context().Done():
			log.Printf("Client disconnected during query stream for DB '%s'", req.DatabaseName)
			return status.Error(codes.Canceled, "client disconnected")
		default:
		}
	} // end rows.Next() loop

	// Check for errors after iterating
	if err := rows.Err(); err != nil {
		log.Printf("Error during row iteration for DB '%s': %v", req.DatabaseName, err)
		return status.Errorf(codes.Internal, "row iteration error: %v", err)
	}

	log.Printf("Finished streaming query results for DB '%s'", req.DatabaseName)
	return nil // Indicates successful end of stream
}
