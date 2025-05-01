package protocol

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"
)

func Start(ctx context.Context, addr, dir string) error {
	// make sure the dir exists and is created
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		slog.Error("failed to create server data path", "error", err)
		return err
	}

	s := &State{
		validTokens: map[string]bool{"SUPERINSECURETOKEN": true},
		// TODO: Parse properly all dbs :🏴‍☠
		dbs:     make(map[string]*sql.DB, 0),
		datadir: dir,
	}

	// --- TCP Listener Setup ---
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
		return err
	}
	defer listener.Close()
	log.Printf("Server listening...")

	var wg sync.WaitGroup

	// --- Connection Acceptance Loop ---
	go func() {
		<-ctx.Done()
		log.Println("Shutdown signal received. Stopping listener...")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Println("Listener closed as part of shutdown.")
			default:
				log.Printf("Failed to accept connection: %v", err)
			}
			break // Exit loop if listener closed or error
		}

		wg.Add(1) // Increment counter for active connection
		log.Printf("Accepted connection from: %s", conn.RemoteAddr())

		go s.handleConnection(ctx, conn, &wg)
	}

	// --- Wait for Shutdown ---
	log.Println("Waiting for active connections to finish...")
	shutdownComplete := make(chan struct{})
	go func() {
		wg.Wait() // Wait for all handleConnection goroutines to finish
		close(shutdownComplete)
	}()

	select {
	case <-shutdownComplete:
		log.Println("All connections closed.")
	case <-time.After(10 * time.Second): // Adjust timeout as needed
		log.Println("Shutdown timed out waiting for connections.")
	}

	log.Println("Server shut down gracefully.")
	return nil
}
