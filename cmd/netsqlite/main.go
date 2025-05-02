package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/alfredosa/netsqlite/internal/generated/proto/v1"
	grpchandler "github.com/alfredosa/netsqlite/internal/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NOTE: Never used do we want one?
func usage() {
	slog.Info("Usage:")
	slog.Info("     -p: port")
	slog.Info("     <data dir>")
}

// --- Global Variables / Config ---
var (
	listenAddr = flag.String("addr", ":3541", "Address and port to listen on")
	dbPath     = flag.String("data", "data/", "Path to the SQLite database file")
	// TODO: In a real app, manage tokens securely (e.g., config file, env vars, secrets management)
)

func main() {
	flag.Parse()
	log.Printf("Starting netsqlite gRPC server on %s", *listenAddr)

	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// --- Create Auth Interceptor ---
	// TODO: Load valid tokens securely
	validTokens := map[string]bool{"SUPERSECRETTOKEN": true} // Replace
	authInterceptor := grpchandler.NewAuthInterceptor(validTokens)

	// --- Create gRPC Server with Interceptors ---
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor( // Chain multiple unary interceptors if needed
			authInterceptor.Unary(),
			// TODO: OTEL
			// Add other interceptors like logging, metrics here
		),
		grpc.ChainStreamInterceptor( // Chain stream interceptors
			authInterceptor.Stream(),
		),
	)

	// --- Create and Register Your Service Implementation ---
	// TODO: Remove tokens like this, very bad no like🏴‍☠
	netsqliteSrv := grpchandler.NewNetsqliteServer(validTokens, *dbPath)
	pb.RegisterNetsqliteServiceServer(grpcServer, netsqliteSrv)

	reflection.Register(grpcServer)
	log.Println("gRPC reflection service registered.")

	// --- Graceful Shutdown Handling (same as before) ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() { // Start server (same as before)
		log.Printf("gRPC server listening at %v", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	<-ctx.Done() // Wait for signal (same as before)
	log.Println("Shutdown signal received. Attempting graceful shutdown...")

	stopped := make(chan struct{}) // Graceful stop logic (same as before)
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
		log.Println("gRPC server gracefully stopped.")
	case <-time.After(10 * time.Second):
		log.Println("Graceful shutdown timed out. Forcing stop.")
		grpcServer.Stop()
	}

	// netsqliteSrv.CloseAllDBs() // Cleanup (same as before)
	log.Println("Server shut down.")
}
