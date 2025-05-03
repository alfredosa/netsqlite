package proto_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	proto "github.com/alfredosa/netsqlite/internal/grpc"

	_ "github.com/alfredosa/netsqlite/pkg/drivers"
)

func prepServer(ctx context.Context) (string, string) {
	token := "123"
	addr := ":3451"
	dir := "data"

	go proto.Start(ctx, map[string]bool{token: true}, addr, dir)

	return addr, token

}

func Test_Ping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	addr, token := prepServer(ctx)

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
