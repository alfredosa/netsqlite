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
			dsn:     "netsqlite://database1:0.0.0.0:8080/token123",
			want:    &drivers.Config{Host: "database1:0.0.0.0:8080", DBName: "database1", Addr: "0.0.0.0:8080", Token: "token123"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := drivers.ParseDSN(tt.dsn)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ParseDSN() failed: %v", gotErr)
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

func Test_E2E(t *testing.T) {

	dns := "netsqlite://test:localhost:8080/123"
	conn, err := sql.Open("netsqlite", dns)
	if err != nil {
		t.Fail()
	}
	err = conn.Ping()
	if err != nil {
		t.Fail()
	}

	t.Fail()
}
