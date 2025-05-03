// file: internal/server/grpc_server.go (or similar)
package proto

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/alfredosa/netsqlite/internal/nsqlite"
	pb "github.com/alfredosa/netsqlite/proto/netsqlite/v1"

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

	// dbManager will manage all connections to all databases concurrently and safely
	dbManager *nsqlite.DBManager
}

// NewNetsqliteServer creates a new server instance.
func NewNetsqliteServer(tokens map[string]bool, datadir string) *netsqliteServer {
	manager := nsqlite.NewManager(datadir)

	return &netsqliteServer{
		validTokens: tokens,
		dbManager:   manager,
	}
}

// --- Service Method Implementations ---
func (s *netsqliteServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	log.Println("Received Ping request")
	pool, err := s.dbManager.AcquirePool(req.DatabaseName)
	if err != nil {
		return nil, err
	}

	db, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	if err := db.Value().PingContext(ctx); err != nil {
		log.Printf("Actual DB Ping failed for %s: %v", req.DatabaseName, err)
		return nil, status.Errorf(codes.Internal, "database ping failed for %s: %v", req.DatabaseName, err)
	}

	return &pb.PingResponse{Message: fmt.Sprintf("PONG for db %s", req.DatabaseName)}, nil
}

func (s *netsqliteServer) Exec(ctx context.Context, req *pb.ExecRequest) (*pb.ExecResponse, error) {
	pool, err := s.dbManager.AcquirePool(req.DatabaseName)
	if err != nil {
		return nil, err
	}

	db, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	// Convert protobuf Value args to []any
	args := make([]any, len(req.Args))
	for i, val := range req.Args {
		args[i] = val.AsInterface()
	}

	sqlResult, err := db.Value().ExecContext(ctx, req.Sql, args...)
	if err != nil {
		log.Printf("Exec failed for DB '%s': %v", req.DatabaseName, err)
		// TODO: Map SQLite errors to gRPC codes more specifically
		return nil, status.Errorf(codes.Internal, "SQL execution failed: %v", err)
	}

	// TODO: don't ignore error
	rowsAffected, _ := sqlResult.RowsAffected() // Ignore error for simplicity here
	lastInsertId, _ := sqlResult.LastInsertId() // Ignore error for simplicity here

	slog.Info("Exec successful for DB", "db", req.DatabaseName, "ra", rowsAffected, "last_insert", lastInsertId)
	return &pb.ExecResponse{
		RowsAffected: rowsAffected,
		LastInsertId: lastInsertId,
	}, nil
}

func (s *netsqliteServer) Query(req *pb.QueryRequest, stream pb.NetsqliteService_QueryServer) error {
	pool, err := s.dbManager.AcquirePool(req.DatabaseName)
	if err != nil {
		return err
	}

	db, err := pool.Acquire(stream.Context())
	if err != nil {
		return err
	}

	// Convert protobuf Value args to []any
	args := make([]any, len(req.Args))
	for i, val := range req.Args {
		args[i] = val.AsInterface()
	}

	rows, err := db.Value().QueryContext(stream.Context(), req.Sql, args...)
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
