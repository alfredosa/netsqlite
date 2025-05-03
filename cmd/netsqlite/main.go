package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	proto "github.com/alfredosa/netsqlite/internal/grpc"
)

var (
	listenAddr = flag.String("addr", ":3541", "Address and port to listen on for gRPC")
	datadir    = flag.String("dir", "data", "Data directory for all databases")
	// TODO: Add flags or env vars for loading tokens securely
)

func main() {
	flag.Parse()
	log.Printf("Starting netsqlite gRPC server on %s, dir %s", *listenAddr, *datadir)

	// --- Load Tokens (TODO: Replace with secure method) ---
	validTokens := map[string]bool{
		"SUPERSECRETTOKEN": true,
		"ANOTHERVALIDONE":  true,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	proto.Start(ctx, validTokens, *listenAddr, *datadir)
}
