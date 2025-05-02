// file: internal/server/auth_interceptor.go
package proto

import (
	"context"
	"log"
	"strings" // For splitting metadata if needed

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Define keys for metadata
const (
	AuthTokenHeader = "authorization"   // Standard-ish header, e.g., "Bearer yourtoken"
	DatabaseHeader  = "x-database-name" // Custom header for the target DB
)

// AuthInterceptor provides gRPC interceptors for authentication.
type AuthInterceptor struct {
	validTokens map[string]bool // Replace with your actual token validation logic
}

// NewAuthInterceptor creates a new interceptor.
func NewAuthInterceptor(validTokens map[string]bool) *AuthInterceptor {
	return &AuthInterceptor{
		validTokens: validTokens,
	}
}

// authenticate performs the actual validation.
func (a *AuthInterceptor) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	// 1. Extract and validate Token
	authHeaders := md.Get(AuthTokenHeader)
	if len(authHeaders) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization token is not provided")
	}
	// Assuming "Bearer <token>" format
	token := strings.TrimPrefix(authHeaders[0], "Bearer ")
	if token == "" || !a.validTokens[token] { // Use your actual validation logic
		log.Printf("Auth failed: Invalid token received '%s'", token)
		return "", status.Error(codes.Unauthenticated, "invalid authorization token")
	}

	// 2. Extract Database Name
	dbNames := md.Get(DatabaseHeader)
	if len(dbNames) == 0 {
		return "", status.Error(codes.InvalidArgument, "x-database-name header is not provided")
	}
	dbName := dbNames[0]
	if dbName == "" {
		return "", status.Error(codes.InvalidArgument, "x-database-name header cannot be empty")
	}

	// TODO: Add any further checks here if needed (e.g., does this token
	// have permission for this specific database?)

	// Authentication successful
	log.Printf("Auth successful for DB: %s (Token: %s...)", dbName, token[:min(len(token), 4)]) // Log truncated token
	return dbName, nil                                                                          // Return validated DB name
}

// Unary returns a server interceptor function for unary RPCs
func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for specific methods if needed (e.g., a health check)
		// if info.FullMethod == "/your.service.v1.YourService/HealthCheck" {
		//     return handler(ctx, req)
		// }

		log.Printf("--> Unary Interceptor: %s", info.FullMethod)
		_, err := a.authenticate(ctx)
		if err != nil {
			return nil, err // Authentication failed
		}

		// Authentication successful, proceed with the handler
		return handler(ctx, req)
	}
}

// Stream returns a server interceptor function for streaming RPCs
func (a *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Printf("--> Stream Interceptor: %s", info.FullMethod)
		_, err := a.authenticate(stream.Context()) // Auth check uses the stream's context
		if err != nil {
			return err // Authentication failed
		}

		// Authentication successful, proceed with the handler
		return handler(srv, stream)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
