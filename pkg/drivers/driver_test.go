package drivers_test

import (
	"database/sql"

	_ "github.com/alfredosa/netsqlite/pkg/drivers"

	"github.com/alfredosa/netsqlite/pkg/drivers"
	"testing"
)

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		dsn     string
		want    *drivers.Config
		wantErr bool
	}{
		{
			name:    "itworks",
			dsn:     "netsqlite://0.0.0.0:8080/token123?database=database1",
			want:    &drivers.Config{DBName: "database1", Addr: "0.0.0.0:8080", Token: "token123", UseTLS: false},
			wantErr: false,
		},
		{
			name:    "with_tls",
			dsn:     "netsqlite://0.0.0.0:8080/token123?database=database1&tls=true",
			want:    &drivers.Config{DBName: "database1", Addr: "0.0.0.0:8080", Token: "token123", UseTLS: true},
			wantErr: false,
		},
		{
			name:    "should_not_parse",
			dsn:     "netsqlite://test:localhost:8080/123",
			want:    &drivers.Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := drivers.ParseDSN(tt.dsn)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ParseDSN() failed: %v %s", gotErr, tt.dsn)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ParseDSN() succeeded unexpectedly")
			}

			if tt.want.Addr != got.Addr || tt.want.DBName != got.DBName {
				t.Fatal("It's not what it was supposed to be")
			}
		})
	}
}

// Main flow of driver, should fail since no available server is running
func Test_E2E(t *testing.T) {
	dns := "netsqlite://localhost:8080/123?database=test"
	conn, err := sql.Open("netsqlite", dns)
	if err != nil {
		t.Fatalf("Failed to open connection: %v", err)
	}
	defer conn.Close()

	err = conn.Ping()
	if err == nil {
		t.Fatal("Expected Ping to fail with no server, but it succeeded")
	}
	t.Logf("Got expected error: %v", err)
}

func Test_E2EFailure(t *testing.T) {
	dns := "netsqlite://test:localhost:8080/123"
	conn, err := sql.Open("netsqlite", dns)
	if err != nil {
		t.FailNow()
	}
	err = conn.Ping()
	if err != nil {
		t.Fail()
	}
	t.Fail()
}
