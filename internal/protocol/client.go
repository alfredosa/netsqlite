package protocol

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"time" // Added for dial timeout
)

// Client handles communication with the netsqlite server
type Client struct {
	conn          net.Conn
	encoder       *gob.Encoder
	decoder       *gob.Decoder
	authenticated bool
}

// NewClient establishes a connection to the server at the given address
// and returns a Client ready for communication.
func NewClient(ctx context.Context, addr string) (*Client, error) {
	var d net.Dialer
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // Adjust timeout as needed
	defer cancel()

	conn, err := d.DialContext(connectCtx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("netsqlite: failed to dial %s: %w", addr, err)
	}

	// Connection successful, set up gob streams
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)

	client := &Client{
		conn:          conn,
		encoder:       encoder,
		decoder:       decoder,
		authenticated: false,
	}

	return client, nil
}

// Authenticate sends the CONNECT command and verifies the response.
func (c *Client) Authenticate(dbName, token string) error {
	if c.authenticated {
		return nil
	}

	req := Request{
		Command:  "CONNECT",
		Database: dbName,
		Token:    token,
	}

	fmt.Printf("Client: Sending CONNECT request (DB: %s)\n", dbName)
	if err := c.encoder.Encode(&req); err != nil {
		c.Close() // Close connection on send error
		return fmt.Errorf("netsqlite: failed to send connect request: %w", err)
	}

	var resp Response
	fmt.Println("Client: Waiting for CONNECT response...")
	if err := c.decoder.Decode(&resp); err != nil {
		c.Close()
		return fmt.Errorf("netsqlite: failed to receive connect response: %w", err)
	}

	if resp.Error != "" {
		c.Close()
		return fmt.Errorf("netsqlite: server authentication failed: %s", resp.Error)
	}

	fmt.Println("Client: CONNECT successful.")
	c.authenticated = true
	return nil // Success
}

// Ping sends a PING command and checks the response.
func (c *Client) Ping(ctx context.Context) error {
	// TODO: Implement context deadline/cancellation for the request/response round trip
	req := Request{Command: "PING"}

	fmt.Println("Client: Sending PING request")
	if err := c.encoder.Encode(&req); err != nil {
		// Don't necessarily close connection on ping failure, maybe recoverable
		return fmt.Errorf("netsqlite: failed to send ping request: %w", err)
	}

	var resp Response
	fmt.Println("Client: Waiting for PING response...")
	if err := c.decoder.Decode(&resp); err != nil {
		return fmt.Errorf("netsqlite: failed to receive ping response: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("netsqlite: ping failed: %s", resp.Error)
	}

	// Optionally check resp.Result
	if resultStr, ok := resp.Result.(string); !ok || resultStr != "PONG" {
		return fmt.Errorf("netsqlite: unexpected ping response result: %v", resp.Result)
	}

	fmt.Println("Client: PING successful (Pong received).")
	return nil
}

// Close terminates the connection with the server.
func (c *Client) Close() error {
	if c.conn != nil {
		fmt.Println("Client: Closing connection")
		err := c.conn.Close()
		c.conn = nil // Prevent double close
		return err
	}
	return nil // Already closed
}

// --- Other methods (Exec, Query, etc.) would go here ---
// Example placeholder:
func (c *Client) Exec(ctx context.Context, sql string, args []interface{}) (*Response, error) {
	req := Request{
		Command: "EXEC",
		SQL:     sql,
		Args:    args,
	}
	fmt.Printf("Client: Sending EXEC request: %s | Args: %v\n", sql, args)
	if err := c.encoder.Encode(&req); err != nil {
		return nil, fmt.Errorf("netsqlite: failed to send exec request: %w", err)
	}

	var resp Response
	fmt.Println("Client: Waiting for EXEC response...")
	if err := c.decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("netsqlite: failed to receive exec response: %w", err)
	}

	// The caller will check resp.Error
	return &resp, nil
}

// --- Placeholder structures (already defined in your protocol package) ---
// type Request struct { ... }
// type Response struct { ... }
