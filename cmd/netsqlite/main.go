package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	grpchandler "github.com/alfredosa/netsqlite/internal/grpc"
	pb "github.com/alfredosa/netsqlite/proto/netsqlite/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	listenAddr = flag.String("addr", ":3541", "Address and port to listen on for gRPC")
	datadir    = flag.String("dir", "data", "Data directory for all databases")
	// TODO: Add flags or env vars for loading tokens securely
)

func main() {
	flag.Parse()
	log.Printf("Starting netsqlite gRPC server on %s", *listenAddr)

	Start()
}

func Start() {
	lis, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// --- Load Tokens (TODO: Replace with secure method) ---
	validTokens := map[string]bool{
		"SUPERSECRETTOKEN": true,
		"ANOTHERVALIDONE":  true,
	}
	log.Printf("Loaded %d valid token(s) (INSECURELY!)", len(validTokens))

	authInterceptor := grpchandler.NewAuthInterceptor(validTokens)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(authInterceptor.Unary()),
		grpc.ChainStreamInterceptor(authInterceptor.Stream()),
	)

	netsqliteSrv := grpchandler.NewNetsqliteServer(validTokens, *datadir)
	pb.RegisterNetsqliteServiceServer(grpcServer, netsqliteSrv)

	reflection.Register(grpcServer)
	log.Println("gRPC reflection service registered.")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("gRPC server listening at %v", lis.Addr())
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("Failed to serve gRPC: %v", err)
		} else if err == grpc.ErrServerStopped {
			log.Println("gRPC server stopped serving.")
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received. Attempting graceful shutdown...")

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
		log.Println("gRPC server gracefully stopped.")
	case <-time.After(15 * time.Second): // Increased timeout
		log.Println("Graceful shutdown timed out. Forcing stop.")
		grpcServer.Stop()
	}

	// TODO: Close all dbs gracefully
	// netsqliteSrv.CloseAllDBs()
	log.Println("Server shut down.")
}
