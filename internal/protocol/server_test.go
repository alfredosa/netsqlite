package protocol_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alfredosa/netsqlite/internal/protocol"
	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	const addr = ":3451"
	tempDir, err := os.MkdirTemp("", "netsqlite-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	var protErr error
	go func() error {
		err = protocol.Start(ctx, addr, tempDir)
		return protErr
	}()

	time.Sleep(time.Millisecond * 10) // WUHUUUUU timing solves it, appaerently its so fast

	c, err := protocol.NewClient(ctx, addr)
	assert.NoError(t, err, "can't create a client")

	err = c.Ping(ctx)
	assert.Error(t, err, "it pinged, but it should not be able to ping :O")

	err = c.Authenticate("wowdb.db", "bad_token")
	assert.Error(t, err, "should not be able to autheticate")

	err = c.Close()
	assert.NoError(t, err, "couldnt close the connection")

	c, err = protocol.NewClient(ctx, addr)
	assert.NoError(t, err, "can't create a client")

	err = c.Authenticate("wowdb.db", "SUPERINSECURETOKEN")
	assert.NoError(t, err, "was not able to authenticate")

	err = c.Ping(ctx)
	assert.NoError(t, err, "we were not able to ping post auth")

	cancel()

	time.Sleep(time.Millisecond * 10)
	assert.NoError(t, err, "failed to start protocol server :(")
	assert.DirExists(t, tempDir, "no data dir found")
	t.Log("dir exists appaerently")

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Log(err)
	}

	for _, e := range entries {
		t.Log("temp dir entry:", e.Name())
	}
}
