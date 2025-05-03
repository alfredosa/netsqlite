package drivers

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"io"

	pb "github.com/alfredosa/netsqlite/proto/netsqlite/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// interface check, just in case hehe
var _ driver.Rows = &SQLRows{}

// SQLRows iterates over gRPC query stream results.
type SQLRows struct {
	stream  pb.NetsqliteService_QueryClient
	columns []string
	closed  bool
}

// Columns returns column names.
func (r *SQLRows) Columns() []string {
	return r.columns
}

// Close marks the iterator as closed.
func (r *SQLRows) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.stream = nil // Allow GC
	fmt.Println("Driver: SQLRows closed.")
	return nil
}

// Next fetches the next row from the gRPC stream.
func (r *SQLRows) Next(dest []driver.Value) error {
	if r.closed || r.stream == nil {
		return io.EOF
	}

	resp, err := r.stream.Recv()
	if err != nil {
		r.closed = true
		if err == io.EOF {
			fmt.Println("Driver: SQLRows received EOF.")
			return io.EOF
		}
		if status.Code(err) == codes.Canceled {
			fmt.Println("Driver: SQLRows stream context canceled.")
			return driver.ErrBadConn
		}
		fmt.Printf("Driver: SQLRows stream Recv error: %v\n", err)
		return fmt.Errorf("netsqlite: receiving row data failed: %w", err)
	}

	rowData := resp.GetRow()
	if rowData == nil {
		r.closed = true
		return errors.New("netsqlite: protocol error - expected Row data")
	}

	if len(rowData.Values) != len(r.columns) {
		r.closed = true
		return fmt.Errorf("netsqlite: column count mismatch (expected %d, got %d)", len(r.columns), len(rowData.Values))
	}

	// Convert proto values to driver values
	for i, pv := range rowData.Values {
		dest[i] = pv.AsInterface()
	}

	return nil
}
