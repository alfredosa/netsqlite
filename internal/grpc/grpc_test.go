package proto_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	proto "github.com/alfredosa/netsqlite/internal/grpc"
	"github.com/stretchr/testify/assert"

	_ "github.com/alfredosa/netsqlite/pkg/drivers"
)

func prepServer(ctx context.Context, t *testing.T) (string, string, string) {
	t.Helper()
	token := "123"
	addr := ":3451"
	dir, err := os.MkdirTemp("", "netsqlite-*")
	assert.NoError(t, err)

	go proto.Start(ctx, map[string]bool{token: true}, addr, dir)

	return addr, token, dir

}

func Test_Ping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	addr, token, dir := prepServer(ctx, t)
	defer os.RemoveAll(dir)

	dns := fmt.Sprintf("netsqlite://%s/%s?database=%s", addr, token, "testdb")
	conn, err := sql.Open("netsqlite", dns)
	if err != nil {
		t.Fatalf("Failed to open connection: %v", err)
	}
	defer conn.Close()

	err = conn.Ping()
	if err != nil {
		t.Fatal("Expected Ping to fail with no server, but it succeeded")
	}

	// wait a bit for the server to die
	defer time.Sleep(time.Millisecond * 10)
}
