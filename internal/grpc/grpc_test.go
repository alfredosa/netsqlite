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

func Test_Exec(t *testing.T) {
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

	_, err = conn.Exec(`
			CREATE TABLE IF NOT EXISTS grpc_test_exec
				(emp_id varchar(255) PRIMARY KEY,
				emp_name varchar(255),
				dept varchar(255),
				duration varchar(255))
`)
	if err != nil {
		t.Fatal("Expected Ping to fail with no server, but it succeeded")
	}

	// Insert a record
	_, err = conn.Exec(`
		INSERT INTO grpc_test_exec
		(emp_id, emp_name, dept, duration)
		VALUES (?, ?, ?, ?)
	`, "emp123", "John Doe", "Engineering", "5 years")

	assert.NoError(t, err, "cant insert into grpc_test_exec")

	// Read the record
	rows, err := conn.Query("SELECT * FROM grpc_test_exec WHERE emp_id = ?", "emp123")
	assert.NoError(t, err)

	defer rows.Close()

	// Verify record exists and contains expected data
	if !rows.Next() {
		t.Fatal("Expected record not found")
	}

	cols, err := rows.Columns()
	assert.NoError(t, err)

	t.Log("columns", cols)

	var empID, empName, dept, duration string
	err = rows.Scan(&empID, &empName, &dept, &duration)
	if err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}

	// Verify values
	if empID != "emp123" || empName != "John Doe" || dept != "Engineering" || duration != "5 years" {
		t.Fatalf("Retrieved record doesn't match inserted data: got (%s, %s, %s, %s)",
			empID, empName, dept, duration)
	}

	// wait a bit for the server to die
	defer time.Sleep(time.Millisecond * 10)
}
