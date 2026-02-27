// Package transport provides the transport interface for Claude SDK.
//
// This package exports the Transport interface that allows custom transport
// implementations for communication with Claude. The default implementation
// uses a subprocess to run the Claude CLI, but custom transports can be
// created for testing, remote connections, or other use cases.
//
// Example custom transport:
//
//	type MyTransport struct {
//	    // ... fields
//	}
//
//	func (t *MyTransport) Connect(ctx context.Context) error { ... }
//	func (t *MyTransport) Close(ctx context.Context) error { ... }
//	func (t *MyTransport) Write(ctx context.Context, data string) error { ... }
//	func (t *MyTransport) ReadMessages(ctx context.Context) <-chan map[string]interface{} { ... }
//	func (t *MyTransport) EndInput(ctx context.Context) error { ... }
//	func (t *MyTransport) IsReady() bool { ... }
package transport

import (
	"context"
)

// Transport defines the interface for bidirectional communication with Claude.
//
// This interface abstracts the communication layer, allowing for different
// implementations (subprocess, WebSocket, HTTP, etc.) while providing a
// consistent API for the SDK.
//
// All implementations must be safe for concurrent use by multiple goroutines.
// The Write and EndInput methods must be synchronized to prevent race conditions.
//
// Lifecycle:
//  1. Call Connect() to establish the connection
//  2. Call Write() to send messages, ReadMessages() to receive
//  3. Call EndInput() when done sending (for half-close)
//  4. Call Close() to release resources
type Transport interface {
	// Connect establishes the connection.
	// For subprocess transports, this starts the process.
	// For network transports, this establishes the connection.
	Connect(ctx context.Context) error

	// Close closes the transport connection and releases all resources.
	// This method should be idempotent - calling it multiple times should
	// not return an error.
	Close(ctx context.Context) error

	// Write writes raw data to the transport.
	// The data should be a complete JSON message followed by a newline.
	// This method must be safe for concurrent use.
	Write(ctx context.Context, data string) error

	// ReadMessages returns a channel that yields parsed JSON messages.
	// The channel is closed when the transport is closed or an error occurs.
	// Messages are parsed from the raw output and returned as map[string]interface{}.
	ReadMessages(ctx context.Context) <-chan map[string]interface{}

	// EndInput closes the input stream (half-close).
	// This signals that no more data will be written.
	// For subprocess transports, this closes stdin.
	EndInput(ctx context.Context) error

	// IsReady returns whether the transport is ready for communication.
	// Returns true after successful Connect() and before Close().
	IsReady() bool
}

// TransportWithErrorHandling extends Transport with error handling capabilities.
type TransportWithErrorHandling interface {
	Transport

	// GetExitError returns the error that caused the transport to exit.
	// Returns nil if the transport hasn't exited or exited cleanly.
	GetExitError() error
}

// TransportWithStderr extends Transport with stderr capture capabilities.
type TransportWithStderr interface {
	Transport

	// GetStderrChan returns a channel for receiving stderr output.
	// This is useful for debugging and logging.
	GetStderrChan() <-chan string
}
