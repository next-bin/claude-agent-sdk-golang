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

// ============================================================================
// Middleware Support
// ============================================================================

// TransportMiddleware defines an interface for intercepting transport operations.
// Middleware can be used for logging, debugging, metrics collection, or
// transforming messages before they are sent/received.
//
// Middleware is applied in order: the first middleware wraps the second, etc.
// For Write operations, middleware is called from outer to inner.
// For Read operations, middleware is called from inner to outer.
type TransportMiddleware interface {
	// InterceptWrite intercepts data being written to the transport.
	// The middleware can modify the data or return an error to block the write.
	// Return the (possibly modified) data to continue the write chain.
	InterceptWrite(ctx context.Context, data string) (string, error)

	// InterceptRead intercepts messages being read from the transport.
	// The middleware can modify the message or return nil to filter it out.
	// Return the (possibly modified) message to continue the read chain.
	InterceptRead(ctx context.Context, msg map[string]interface{}) (map[string]interface{}, error)
}

// MiddlewareTransport wraps a Transport with one or more middleware.
// Middleware is applied in order: first middleware is outermost.
type MiddlewareTransport struct {
	base       Transport
	middleware []TransportMiddleware
}

// NewMiddlewareTransport creates a new transport with middleware applied.
// Middleware is applied in order: first middleware is outermost for writes.
func NewMiddlewareTransport(base Transport, middleware ...TransportMiddleware) *MiddlewareTransport {
	return &MiddlewareTransport{
		base:       base,
		middleware: middleware,
	}
}

// Connect establishes the connection through the base transport.
func (t *MiddlewareTransport) Connect(ctx context.Context) error {
	return t.base.Connect(ctx)
}

// Close closes the transport through the base transport.
func (t *MiddlewareTransport) Close(ctx context.Context) error {
	return t.base.Close(ctx)
}

// Write writes data through the middleware chain.
// Middleware is applied from first to last (outer to inner).
func (t *MiddlewareTransport) Write(ctx context.Context, data string) error {
	// Apply middleware in order (outer to inner)
	for _, m := range t.middleware {
		modifiedData, err := m.InterceptWrite(ctx, data)
		if err != nil {
			return err
		}
		data = modifiedData
	}
	return t.base.Write(ctx, data)
}

// ReadMessages returns a channel with middleware applied to each message.
func (t *MiddlewareTransport) ReadMessages(ctx context.Context) <-chan map[string]interface{} {
	// Create output channel
	output := make(chan map[string]interface{}, 100)

	// Start goroutine to apply middleware to incoming messages
	go func() {
		defer close(output)

		for msg := range t.base.ReadMessages(ctx) {
			// Apply middleware in reverse order (inner to outer)
			result := msg
			for i := len(t.middleware) - 1; i >= 0; i-- {
				modifiedResult, err := t.middleware[i].InterceptRead(ctx, result)
				if err != nil {
					// Error in middleware, skip message
					result = nil
					break
				}
				result = modifiedResult
				if result == nil {
					// Message filtered out
					break
				}
			}

			if result != nil {
				select {
				case output <- result:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return output
}

// EndInput closes the input stream through the base transport.
func (t *MiddlewareTransport) EndInput(ctx context.Context) error {
	return t.base.EndInput(ctx)
}

// IsReady returns whether the base transport is ready.
func (t *MiddlewareTransport) IsReady() bool {
	return t.base.IsReady()
}

// ============================================================================
// Common Middleware Implementations
// ============================================================================

// LoggingMiddleware logs all write and read operations.
type LoggingMiddleware struct {
	logWrite func(ctx context.Context, data string)
	logRead  func(ctx context.Context, msg map[string]interface{})
}

// NewLoggingMiddleware creates a middleware that logs operations.
func NewLoggingMiddleware(logWrite func(ctx context.Context, data string), logRead func(ctx context.Context, msg map[string]interface{})) *LoggingMiddleware {
	return &LoggingMiddleware{
		logWrite: func(ctx context.Context, data string) {
			if logWrite != nil {
				logWrite(ctx, data)
			}
		},
		logRead: func(ctx context.Context, msg map[string]interface{}) {
			if logRead != nil {
				logRead(ctx, msg)
			}
		},
	}
}

// InterceptWrite logs write operations.
func (m *LoggingMiddleware) InterceptWrite(ctx context.Context, data string) (string, error) {
	if m.logWrite != nil {
		m.logWrite(ctx, data)
	}
	return data, nil
}

// InterceptRead logs read operations.
func (m *LoggingMiddleware) InterceptRead(ctx context.Context, msg map[string]interface{}) (map[string]interface{}, error) {
	if m.logRead != nil {
		m.logRead(ctx, msg)
	}
	return msg, nil
}

// MetricsMiddleware collects metrics on transport operations.
type MetricsMiddleware struct {
	writeCount int64
	readCount  int64
}

// NewMetricsMiddleware creates a middleware that collects operation counts.
func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{}
}

// InterceptWrite increments write counter.
func (m *MetricsMiddleware) InterceptWrite(ctx context.Context, data string) (string, error) {
	m.writeCount++
	return data, nil
}

// InterceptRead increments read counter.
func (m *MetricsMiddleware) InterceptRead(ctx context.Context, msg map[string]interface{}) (map[string]interface{}, error) {
	m.readCount++
	return msg, nil
}

// GetWriteCount returns the number of write operations.
func (m *MetricsMiddleware) GetWriteCount() int64 {
	return m.writeCount
}

// GetReadCount returns the number of read operations.
func (m *MetricsMiddleware) GetReadCount() int64 {
	return m.readCount
}
