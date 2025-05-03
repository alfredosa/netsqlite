package drivers

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	pb "github.com/alfredosa/netsqlite/proto/netsqlite/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // Use insecure for now
	// "google.golang.org/grpc/credentials" // Needed for TLS
)

// SQLConnector creates connections.
type SQLConnector struct {
	driver *SQLDriver
	config *Config
}

var _ driver.Connector = &SQLConnector{}

// Connect dials the gRPC server and performs an initial ping.
func (c *SQLConnector) Connect(ctx context.Context) (driver.Conn, error) {
	creds := &staticCredentials{
		Token:        c.config.Token,
		DatabaseName: c.config.DBName,
		RequireTLS:   c.config.UseTLS,
	}

	var opts []grpc.DialOption
	if creds.RequireTLS {
		// TODO: Implement secure TLS credential loading based on Config
		// Use credentials.NewClientTLSFromCert(...)
		return nil, fmt.Errorf("netsqlite: TLS configured via DSN but client TLS loading not implemented")
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, grpc.WithPerRPCCredentials(creds))

	grpcConn, err := grpc.NewClient(c.config.Addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("netsqlite: failed to dial gRPC server %s: %w", c.config.Addr, err)
	}

	grpcClient := pb.NewNetsqliteServiceClient(grpcConn)

	// Create SQLConn wrapper BEFORE pinging
	sqlConn := &SQLConn{
		grpcConn: grpcConn,
		client:   grpcClient,
		dbName:   c.config.DBName,
		closed:   false,
	}

	// Ping using the connection context to verify auth/connectivity
	pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
	defer pingCancel()
	if err := sqlConn.Ping(pingCtx); err != nil {
		sqlConn.Close()
		return nil, fmt.Errorf("netsqlite: initial gRPC ping failed (check server logs for auth/db errors): %w", err)
	}

	return sqlConn, nil
}

// Driver returns the parent driver.
func (c *SQLConnector) Driver() driver.Driver {
	return c.driver
}
