// Package middleware_test tests the transport middleware functionality.
package transport_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang/transport"
)

// mockTransport is a mock transport for testing.
type mockTransport struct {
	writeChan chan string
	readChan  chan map[string]interface{}
	ready     bool
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		writeChan: make(chan string, 10),
		readChan:  make(chan map[string]interface{}, 10),
		ready:     true,
	}
}

func (t *mockTransport) Connect(ctx context.Context) error { return nil }
func (t *mockTransport) Close(ctx context.Context) error   { return nil }
func (t *mockTransport) Write(ctx context.Context, data string) error {
	t.writeChan <- data
	return nil
}
func (t *mockTransport) ReadMessages(ctx context.Context) <-chan map[string]interface{} {
	return t.readChan
}
func (t *mockTransport) EndInput(ctx context.Context) error { return nil }
func (t *mockTransport) IsReady() bool                      { return t.ready }

func TestMiddlewareTransport_Write(t *testing.T) {
	mock := newMockTransport()

	// Create middleware that modifies data
	modifyMiddleware := &testModifyMiddleware{prefix: "modified:"}

	middlewareTransport := transport.NewMiddlewareTransport(mock, modifyMiddleware)

	ctx := context.Background()
	err := middlewareTransport.Write(ctx, "original")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check that data was modified
	select {
	case data := <-mock.writeChan:
		if data != "modified:original" {
			t.Errorf("Expected modified:original, got %s", data)
		}
	default:
		t.Error("No data received")
	}
}

func TestMiddlewareTransport_Read(t *testing.T) {
	mock := newMockTransport()

	// Create middleware that filters messages
	filterMiddleware := &testFilterMiddleware{filterType: "filtered"}

	middlewareTransport := transport.NewMiddlewareTransport(mock, filterMiddleware)

	ctx := context.Background()
	msgChan := middlewareTransport.ReadMessages(ctx)

	// Send message through mock AFTER ReadMessages is called
	mock.readChan <- map[string]interface{}{"type": "test", "value": 1}

	// Should receive the message (not filtered)
	select {
	case msg := <-msgChan:
		if msg["type"] != "test" {
			t.Errorf("Expected type test, got %v", msg["type"])
		}
	case <-ctx.Done():
		t.Error("Context done")
	}

	// Send filtered message
	mock.readChan <- map[string]interface{}{"type": "filtered", "value": 2}

	// Should not receive filtered message - use short timeout
	select {
	case msg := <-msgChan:
		t.Errorf("Should not receive filtered message, got %v", msg)
	default:
		// Expected - message was filtered or not yet processed
	}

	// Close mock readChan to end the goroutine
	close(mock.readChan)
}

func TestLoggingMiddleware(t *testing.T) {
	writeLog := ""
	readLog := ""

	loggingMiddleware := transport.NewLoggingMiddleware(
		func(ctx context.Context, data string) {
			writeLog = data
		},
		func(ctx context.Context, msg map[string]interface{}) {
			if typ, ok := msg["type"].(string); ok {
				readLog = typ
			}
		},
	)

	ctx := context.Background()

	// Test InterceptWrite
	data, err := loggingMiddleware.InterceptWrite(ctx, "test-data")
	if err != nil {
		t.Fatalf("InterceptWrite failed: %v", err)
	}
	if writeLog != "test-data" {
		t.Errorf("Expected writeLog=test-data, got %s", writeLog)
	}
	if data != "test-data" {
		t.Errorf("Expected data=test-data, got %s", data)
	}

	// Test InterceptRead
	msg := map[string]interface{}{"type": "test-msg"}
	result, err := loggingMiddleware.InterceptRead(ctx, msg)
	if err != nil {
		t.Fatalf("InterceptRead failed: %v", err)
	}
	if readLog != "test-msg" {
		t.Errorf("Expected readLog=test-msg, got %s", readLog)
	}
	if result["type"] != "test-msg" {
		t.Errorf("Expected type=test-msg, got %v", result["type"])
	}
}

func TestMetricsMiddleware(t *testing.T) {
	metricsMiddleware := transport.NewMetricsMiddleware()

	ctx := context.Background()

	// Test InterceptWrite
	for i := 0; i < 3; i++ {
		_, err := metricsMiddleware.InterceptWrite(ctx, fmt.Sprintf("data-%d", i))
		if err != nil {
			t.Fatalf("InterceptWrite failed: %v", err)
		}
	}

	if metricsMiddleware.GetWriteCount() != 3 {
		t.Errorf("Expected writeCount=3, got %d", metricsMiddleware.GetWriteCount())
	}

	// Test InterceptRead
	for i := 0; i < 5; i++ {
		_, err := metricsMiddleware.InterceptRead(ctx, map[string]interface{}{"value": i})
		if err != nil {
			t.Fatalf("InterceptRead failed: %v", err)
		}
	}

	if metricsMiddleware.GetReadCount() != 5 {
		t.Errorf("Expected readCount=5, got %d", metricsMiddleware.GetReadCount())
	}
}

// testModifyMiddleware modifies write data.
type testModifyMiddleware struct {
	prefix string
}

func (m *testModifyMiddleware) InterceptWrite(ctx context.Context, data string) (string, error) {
	return m.prefix + data, nil
}

func (m *testModifyMiddleware) InterceptRead(ctx context.Context, msg map[string]interface{}) (map[string]interface{}, error) {
	return msg, nil
}

// testFilterMiddleware filters messages by type.
type testFilterMiddleware struct {
	filterType string
}

func (m *testFilterMiddleware) InterceptWrite(ctx context.Context, data string) (string, error) {
	return data, nil
}

func (m *testFilterMiddleware) InterceptRead(ctx context.Context, msg map[string]interface{}) (map[string]interface{}, error) {
	if msg["type"] == m.filterType {
		return nil, nil // Filter out
	}
	return msg, nil
}
