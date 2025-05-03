package drivers

import (
	"context"
	"fmt"
)

// Keys matching server interceptor expectations (lowercase convention)
// NOTE: IS THIS A MAINTAINANCE Nightmare?
// TODO: Reconsider one place?
const (
	clientAuthTokenHeader = "authorization"
	clientDatabaseHeader  = "x-database-name"
)

// staticCredentials implements credentials.PerRPCCredentials
type staticCredentials struct {
	Token        string
	DatabaseName string
	// RequireTLS is not yet fully implemented, and this can't be properly used right now
	RequireTLS bool
}

// GetRequestMetadata attaches auth token and db name to each RPC.
func (c *staticCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		clientAuthTokenHeader: fmt.Sprintf("Bearer %s", c.Token),
		clientDatabaseHeader:  c.DatabaseName,
	}, nil
}

// RequireTransportSecurity dictates if TLS is mandatory.
func (c *staticCredentials) RequireTransportSecurity() bool {
	return c.RequireTLS
}
