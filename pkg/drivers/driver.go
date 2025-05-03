package drivers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

const DriverName = "netsqlite"

// Config holds parsed DSN info.
type Config struct {
	Addr   string // Host:Port of the gRPC server
	DBName string // Database identifier passed to server
	Token  string // Auth token

	// UseTLS is not yet implemented
	UseTLS   bool   // TODO: Flag for enabling TLS (requires more config)
	RawQuery string // Original query params if needed
}

// SQLDriver implements driver.DriverContext.
type SQLDriver struct{}

var _ driver.DriverContext = &SQLDriver{}

func init() {
	sql.Register(DriverName, &SQLDriver{})
}

// ParseDSN parses the netsqlite DSN string.
// Format: netsqlite://[host]/[token]?database=[dbname]&tls=[bool]
// Where tls is optional
func ParseDSN(dsn string) (*Config, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN format: %w", err)
	}

	if u.Scheme != DriverName {
		return nil, fmt.Errorf("invalid scheme: expected '%s', got '%s'", DriverName, u.Scheme)
	}

	// Check if this might be the old format (dbname:host:port)
	if u.User != nil && u.User.Username() != "" {
		return nil, fmt.Errorf("invalid DSN format: found username in URL authority section. " +
			"The correct format is: netsqlite://host:port/token?database=dbname")
	}

	addr := u.Host
	if addr == "" || !strings.Contains(addr, ":") {
		return nil, fmt.Errorf("gRPC server address (host:port) missing or invalid in DSN host part")
	}

	token := strings.TrimPrefix(u.Path, "/")
	if token == "" {
		return nil, fmt.Errorf("authentication token missing in DSN path")
	}

	dbName := u.Query().Get("database")
	if dbName == "" {
		return nil, fmt.Errorf("database name missing in DSN (use ?database=name)")
	}

	useTLS := false
	if u.Query().Get("tls") == "true" {
		useTLS = true
	}

	return &Config{
		Addr:     addr,
		DBName:   dbName,
		Token:    token,
		UseTLS:   useTLS,
		RawQuery: u.RawQuery,
	}, nil
}

// OpenConnector parses DSN and returns a connector.
func (d *SQLDriver) OpenConnector(dsn string) (driver.Connector, error) {
	cfg, err := ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("netsqlite: parsing DSN failed: %w", err)
	}

	conn := &SQLConnector{
		driver: d,
		config: cfg,
	}

	println("made it out with conn details")
	spew.Dump(conn)
	return conn, nil
}

// Open provides compatibility for older sql package use.
func (d *SQLDriver) Open(dsn string) (driver.Conn, error) {
	connector, err := d.OpenConnector(dsn)
	if err != nil {
		return nil, err
	}
	println("attempting to connect")
	return connector.Connect(context.Background())
}
