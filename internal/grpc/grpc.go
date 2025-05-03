package proto

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/alfredosa/netsqlite/proto/netsqlite/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Start(ctx context.Context, validTokens map[string]bool, addr, dir string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Loaded %d valid token(s) (INSECURELY!)", len(validTokens))

	authInterceptor := NewAuthInterceptor(validTokens)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(authInterceptor.Unary()),
		grpc.ChainStreamInterceptor(authInterceptor.Stream()),
	)

	netsqliteSrv := NewNetsqliteServer(validTokens, dir)
	pb.RegisterNetsqliteServiceServer(grpcServer, netsqliteSrv)

	// for reflection and grpcurl
	reflection.Register(grpcServer)
	log.Println("gRPC reflection service registered.")

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
