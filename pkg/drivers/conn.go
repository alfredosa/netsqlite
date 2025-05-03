package drivers

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"

	pb "github.com/alfredosa/netsqlite/proto/netsqlite/v1"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// SQLConn represents an active gRPC connection.
type SQLConn struct {
	grpcConn *grpc.ClientConn
	client   pb.NetsqliteServiceClient
	dbName   string
	closed   bool
}

// Compile-time interface checks
var _ driver.Conn = &SQLConn{}
var _ driver.Pinger = &SQLConn{}
var _ driver.ExecerContext = &SQLConn{}
var _ driver.QueryerContext = &SQLConn{}

// TODO: var _ driver.ConnPrepareContext = &SQLConn{}
// TODO: var _ driver.ConnBeginTx = &SQLConn{}

// Helper to convert driver args to proto args
func driverNamedValueToProtoValue(args []driver.NamedValue) ([]*structpb.Value, error) {
	protoArgs := make([]*structpb.Value, len(args))
	var err error
	for i, arg := range args {
		protoArgs[i], err = structpb.NewValue(arg.Value)
		if err != nil {
			return nil, fmt.Errorf("netsqlite: unsupported arg type %T at index %d: %w", arg.Value, i, err)
		}
	}
	return protoArgs, nil
}

// Ping verifies the connection via gRPC Ping RPC.
func (c *SQLConn) Ping(ctx context.Context) error {
	if c.closed || c.client == nil {
		return driver.ErrBadConn
	}
	fmt.Println("Driver: SQLConn.Ping executing gRPC Ping")
	req := &pb.PingRequest{DatabaseName: c.dbName}
	_, err := c.client.Ping(ctx, req)
	if err != nil {
		fmt.Printf("Driver: gRPC Ping failed: %v\n", err)
		return driver.ErrBadConn
	}
	return nil
}

// ExecContext executes non-query statements via gRPC Exec RPC.
func (c *SQLConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.closed || c.client == nil {
		return nil, driver.ErrBadConn
	}
	fmt.Printf("Driver: ExecContext via gRPC: %s\n", query)

	protoArgs, err := driverNamedValueToProtoValue(args)
	if err != nil {
		return nil, err
	}

	req := &pb.ExecRequest{
		DatabaseName: c.dbName,
		Sql:          query,
		Args:         protoArgs,
		// TODO: Add TransactionID if applicable
	}

	resp, err := c.client.Exec(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("netsqlite: gRPC Exec failed: %w", err)
	}

	return &SQLResult{
		rowsAffected: resp.RowsAffected,
		lastInsertId: resp.LastInsertId,
	}, nil
}

// QueryContext executes queries via gRPC Query RPC stream.
func (c *SQLConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.closed || c.client == nil {
		return nil, driver.ErrBadConn
	}
	fmt.Printf("Driver: QueryContext via gRPC: %s\n", query)

	protoArgs, err := driverNamedValueToProtoValue(args)
	if err != nil {
		return nil, err
	}

	req := &pb.QueryRequest{
		DatabaseName: c.dbName,
		Sql:          query,
		Args:         protoArgs,
		// TODO: Add TransactionID if applicable
	}

	stream, err := c.client.Query(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("netsqlite: gRPC Query failed: %w", err)
	}

	// Receive first message (must be columns or EOF)
	firstResp, err := stream.Recv()
	if err != nil {
		if err == io.EOF { // No rows returned
			return &SQLRows{closed: true, columns: []string{}}, nil
		}
		return nil, fmt.Errorf("netsqlite: failed receiving columns: %w", err)
	}

	colsResult := firstResp.GetColumns()
	if colsResult == nil {
		return nil, errors.New("netsqlite: protocol error - expected Columns first")
	}

	return &SQLRows{
		stream:  stream,
		columns: colsResult.Names,
		closed:  false,
	}, nil
}

// Close terminates the gRPC connection.
func (c *SQLConn) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	fmt.Println("Driver: Closing gRPC connection.")
	if c.grpcConn != nil {
		err := c.grpcConn.Close()
		c.grpcConn = nil
		c.client = nil
		return err
	}
	return nil
}

// --- Stubs for Future Implementation ---
// TODO: Let's see how it goes. and implement them if it is necessary.

func (c *SQLConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return nil, errors.New("netsqlite: PrepareContext not implemented")
}

func (c *SQLConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return nil, errors.New("netsqlite: BeginTx not implemented")
}

func (c *SQLConn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}
func (c *SQLConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
