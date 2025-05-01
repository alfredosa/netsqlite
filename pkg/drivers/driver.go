package drivers

import (
	"context" // Import context
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strings"
)

const DriverName = "netsqlite"

// Ensure SQLDriver implements driver.DriverContext at compile time.
var _ driver.DriverContext = &SQLDriver{} // Changed from driver.Driver

type SQLDriver struct{}

func init() {
	sql.Register(DriverName, &SQLDriver{})
}

// OpenConnector parses the DSN and returns a Connector.
// It does NOT establish the connection; Connector.Connect does that.
// DSN format: netsqlite://db:host/token
func (d *SQLDriver) OpenConnector(dsn string) (driver.Connector, error) { // Renamed and changed return type
	cfg, err := ParseDSN(dsn)
	if err != nil {
		// It's often better to wrap the error for context
		return nil, fmt.Errorf("netsqlite: parsing DSN failed: %w", err)
	}

	// Return the SQLConnector struct, which implements driver.Connector
	return &SQLConnector{
		driver: d, // Reference to the driver
		config: cfg,
	}, nil
}

// Open is included for compatibility with older Go versions or code
// that might expect it. It simply calls OpenConnector and then Connect.
// Note: sql package prefers OpenConnector if available.
func (d *SQLDriver) Open(dsn string) (driver.Conn, error) {
	// This implementation follows the older pattern if needed,
	// but directly establishes the connection here.
	connector, err := d.OpenConnector(dsn)
	if err != nil {
		return nil, err
	}
	return connector.Connect(context.Background()) // Connect immediately
}

// Config holds the parsed DSN information.
type Config struct {
	// RawDSN string // Optional: Store the original DSN if needed
	Host   string // e.g., "db:host" from the DSN
	DBName string // e.g., "db" extracted from Host
	Addr   string // e.g., "host" extracted from Host (or just Host if no split)
	Token  string // e.g., "token" from the DSN path
	// Add other options derived from DSN query parameters if needed
}

// ParseDSN parses the netsqlite DSN string.
// Example format: netsqlite://db:host/token
func ParseDSN(dsn string) (*Config, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid URL scheme: %w", err)
	}

	if u.Scheme != DriverName {
		return nil, fmt.Errorf("invalid scheme: expected %s, got %s", DriverName, u.Scheme)
	}

	cfg := &Config{
		Host: u.Host, // "db:host"
	}

	parts := strings.SplitN(u.Host, ":", 2)
	if len(parts) == 2 {
		cfg.DBName = parts[0]
		cfg.Addr = parts[1]
	} else {
		return nil, fmt.Errorf("invalid host format: expected 'dbname:hostname', got '%s'", u.Host)
	}

	cfg.Token = strings.TrimPrefix(u.Path, "/")
	if cfg.Token == "" {
		return nil, fmt.Errorf("missing token in DSN path")
	}

	// TODO: Parse query parameters from u.Query() if you add any

	return cfg, nil
}
