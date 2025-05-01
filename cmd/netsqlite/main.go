package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/alfredosa/netsqlite/internal/protocol"
)

func usage() {
	slog.Info("Usage:")
	slog.Info("     -p: port")
	slog.Info("     <data dir>")
}

// --- Global Variables / Config ---
var (
	listenAddr = flag.String("addr", ":3541", "Address and port to listen on")
	dbPath     = flag.String("data", "data/", "Path to the SQLite database file")
	// In a real app, manage tokens securely (e.g., config file, env vars, secrets management)
)

func main() {
	flag.Parse()

	log.Printf("Starting netsqlite server on %s", *listenAddr)
	log.Printf("Using database file: %s", *dbPath)

	// --- Graceful Shutdown Handling ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err := protocol.Start(ctx, *listenAddr, *dbPath)
	if err != nil {
		log.Fatal(err)
	}
}
