// Package query provides internal query implementation for the Claude Agent SDK.
//
// This test file provides comprehensive coverage for control methods.
package queryimpl

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Mock Transport for Control Method Tests
// ============================================================================

// controlMockTransport implements the Transport interface for testing control methods.
type controlMockTransport struct {
	writeCalls     []string
	writeErr       error
	endInputCalled bool
	endInputErr    error
	closeCalled    bool
	closeErr       error
	messages       chan map[string]interface{}
	mu             sync.Mutex
}

func newControlMockTransport() *controlMockTransport {
	return &controlMockTransport{
		messages: make(chan map[string]interface{}, 100),
	}
}

func (m *controlMockTransport) Write(ctx context.Context, data string) error {
	m.mu.Lock()
	m.writeCalls = append(m.writeCalls, data)
	m.mu.Unlock()
	if m.writeErr != nil {
		return m.writeErr
	}
	return nil
}

func (m *controlMockTransport) EndInput(ctx context.Context) error {
	m.mu.Lock()
	m.endInputCalled = true
	m.mu.Unlock()
	if m.endInputErr != nil {
		return m.endInputErr
	}
	return nil
}

func (m *controlMockTransport) ReadMessages(ctx context.Context) <-chan map[string]interface{} {
	return m.messages
}

func (m *controlMockTransport) Close(ctx context.Context) error {
	m.mu.Lock()
	m.closeCalled = true
	m.mu.Unlock()
	close(m.messages)
	if m.closeErr != nil {
		return m.closeErr
	}
	return nil
}

func (m *controlMockTransport) getWriteCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.writeCalls))
	copy(result, m.writeCalls)
	return result
}

func (m *controlMockTransport) sendMessage(msg map[string]interface{}) {
	m.messages <- msg
}

func (m *controlMockTransport) closeMessages() {
	close(m.messages)
}

// ============================================================================
// Helper Functions
// ============================================================================

// extractRequestID extracts the request_id from a written control request.
func extractRequestID(writeData string) string {
	var req map[string]interface{}
	if err := json.Unmarshal([]byte(writeData), &req); err != nil {
		return ""
	}
	reqID, ok := req["request_id"].(string)
	if !ok {
		return ""
	}
	return reqID
}

// sendSuccessResponse sends a success response matching the given request ID.
func sendSuccessResponse(mock *controlMockTransport, requestID string, responseData map[string]interface{}) {
	mock.sendMessage(map[string]interface{}{
		"type": "control_response",
		"response": map[string]interface{}{
			"subtype":    "success",
			"request_id": requestID,
			"response":   responseData,
		},
	})
}

// sendErrorResponse sends an error response matching the given request ID.
func sendErrorResponse(mock *controlMockTransport, requestID string, errMsg string) {
	mock.sendMessage(map[string]interface{}{
		"type": "control_response",
		"response": map[string]interface{}{
			"subtype":    "error",
			"request_id": requestID,
			"error":      errMsg,
		},
	})
}

// ============================================================================
// GetMCPStatus Tests
// ============================================================================

func TestGetMCPStatus_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_, err := q.GetMCPStatus(ctx)

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestGetMCPStatus_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("write failed")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_, err := q.GetMCPStatus(ctx)

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestGetMCPStatus_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _ = q.GetMCPStatus(ctx)

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	if request["type"] != "control_request" {
		t.Errorf("expected type 'control_request', got %v", request["type"])
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "mcp_status" {
		t.Errorf("expected subtype 'mcp_status', got %v", reqData["subtype"])
	}
}

func TestGetMCPStatus_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Channel to receive response
	var respErr error
	done := make(chan struct{})

	go func() {
		_, respErr = q.GetMCPStatus(ctx)
		close(done)
	}()

	// Wait for request to be sent, then extract request_id and send response
	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{
			"mcpServers": []interface{}{
				map[string]interface{}{"name": "server1", "status": "connected"},
			},
		})
	}

	select {
	case <-done:
		if respErr != nil {
			t.Errorf("unexpected error: %v", respErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

func TestGetMCPStatus_ErrorResponse(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		_, err = q.GetMCPStatus(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendErrorResponse(mockTransport, reqID, "MCP server error")
	}

	select {
	case <-done:
		if err == nil {
			t.Error("expected error from CLI")
		}
		if !containsSubstring(err.Error(), "MCP server error") {
			t.Errorf("error should contain 'MCP server error', got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// GetContextUsage Tests
// ============================================================================

func TestGetContextUsage_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_, err := q.GetContextUsage(ctx)

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestGetContextUsage_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("write failed")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_, err := q.GetContextUsage(ctx)

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestGetContextUsage_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _ = q.GetContextUsage(ctx)

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "get_context_usage" {
		t.Errorf("expected subtype 'get_context_usage', got %v", reqData["subtype"])
	}
}

func TestGetContextUsage_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var resp types.ContextUsageResponse
	var err error
	done := make(chan struct{})

	go func() {
		resp, err = q.GetContextUsage(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{
			"totalTokens":          5000,
			"maxTokens":            100000,
			"model":                "claude-3-sonnet",
			"isAutoCompactEnabled": true,
			"categories": []interface{}{
				map[string]interface{}{"name": "system", "tokens": 1000},
			},
		})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp.TotalTokens != 5000 {
			t.Errorf("TotalTokens = %d, want 5000", resp.TotalTokens)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

func TestGetContextUsage_ErrorResponse(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		_, err = q.GetContextUsage(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendErrorResponse(mockTransport, reqID, "context unavailable")
	}

	select {
	case <-done:
		if err == nil {
			t.Error("expected error from CLI")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// TestGetContextUsage_WithComplexData tests parsing of complex context usage data.
func TestGetContextUsage_WithComplexData(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var resp types.ContextUsageResponse
	var err error
	done := make(chan struct{})

	go func() {
		resp, err = q.GetContextUsage(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		// Send response with nested complex data
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{
			"totalTokens":          10000,
			"maxTokens":            200000,
			"percentage":           5.0,
			"model":                "claude-3-opus",
			"isAutoCompactEnabled": true,
			"categories": []interface{}{
				map[string]interface{}{
					"name":   "messages",
					"tokens": 5000,
					"color":  "#FF5733",
				},
			},
			"memoryFiles": []interface{}{
				map[string]interface{}{
					"path": "/memory/file1.txt",
					"size": 1024,
				},
			},
			"mcpTools": []interface{}{
				map[string]interface{}{
					"name":    "tool1",
					"enabled": true,
				},
			},
			"gridRows": []interface{}{
				[]interface{}{map[string]interface{}{"col1": "val1"}},
			},
		})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp.TotalTokens != 10000 {
			t.Errorf("TotalTokens = %d, want 10000", resp.TotalTokens)
		}
		if len(resp.Categories) != 1 {
			t.Errorf("Categories count = %d, want 1", len(resp.Categories))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// Interrupt Tests
// ============================================================================

func TestInterrupt_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.Interrupt(ctx)

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestInterrupt_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("connection closed")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.Interrupt(ctx)

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestInterrupt_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.Interrupt(ctx)

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "interrupt" {
		t.Errorf("expected subtype 'interrupt', got %v", reqData["subtype"])
	}
}

func TestInterrupt_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.Interrupt(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// SetPermissionMode Tests
// ============================================================================

func TestSetPermissionMode_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.SetPermissionMode(ctx, "plan")

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestSetPermissionMode_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("write failed")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.SetPermissionMode(ctx, "acceptEdits")

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestSetPermissionMode_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.SetPermissionMode(ctx, "plan")

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "set_permission_mode" {
		t.Errorf("expected subtype 'set_permission_mode', got %v", reqData["subtype"])
	}
	if reqData["mode"] != "plan" {
		t.Errorf("expected mode 'plan', got %v", reqData["mode"])
	}
}

func TestSetPermissionMode_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.SetPermissionMode(ctx, "acceptEdits")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

func TestSetPermissionMode_WithDifferentModes(t *testing.T) {
	modes := []string{"plan", "acceptEdits", "auto", "bypassPermissions", ""}

	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			mockTransport := newControlMockTransport()
			q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			_ = q.SetPermissionMode(ctx, mode)

			writes := mockTransport.getWriteCalls()
			if len(writes) != 1 {
				t.Fatalf("expected 1 write, got %d", len(writes))
			}

			var request map[string]interface{}
			if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
				t.Fatalf("failed to parse request JSON: %v", err)
			}

			reqData, ok := request["request"].(map[string]interface{})
			if !ok {
				t.Fatal("expected 'request' field to be a map")
			}
			if reqData["mode"] != mode {
				t.Errorf("expected mode '%s', got %v", mode, reqData["mode"])
			}
		})
	}
}

// ============================================================================
// SetModel Tests
// ============================================================================

func TestSetModel_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.SetModel(ctx, "claude-sonnet")

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestSetModel_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("connection lost")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.SetModel(ctx, "claude-sonnet")

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestSetModel_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.SetModel(ctx, "claude-opus-4-20250514")

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "set_model" {
		t.Errorf("expected subtype 'set_model', got %v", reqData["subtype"])
	}
	if reqData["model"] != "claude-opus-4-20250514" {
		t.Errorf("expected model 'claude-opus-4-20250514', got %v", reqData["model"])
	}
}

func TestSetModel_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.SetModel(ctx, "claude-sonnet")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// RewindFiles Tests
// ============================================================================

func TestRewindFiles_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.RewindFiles(ctx, "msg-123")

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestRewindFiles_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("write failed")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.RewindFiles(ctx, "msg-123")

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestRewindFiles_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.RewindFiles(ctx, "user-msg-456")

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "rewind_files" {
		t.Errorf("expected subtype 'rewind_files', got %v", reqData["subtype"])
	}
	if reqData["user_message_id"] != "user-msg-456" {
		t.Errorf("expected user_message_id 'user-msg-456', got %v", reqData["user_message_id"])
	}
}

func TestRewindFiles_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.RewindFiles(ctx, "msg-123")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

func TestRewindFiles_ErrorResponse(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.RewindFiles(ctx, "invalid-id")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendErrorResponse(mockTransport, reqID, "checkpoint not found")
	}

	select {
	case <-done:
		if err == nil {
			t.Error("expected error from CLI")
		}
		if !containsSubstring(err.Error(), "checkpoint not found") {
			t.Errorf("error should contain 'checkpoint not found', got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// ReconnectMCPServer Tests
// ============================================================================

func TestReconnectMCPServer_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.ReconnectMCPServer(ctx, "filesystem")

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestReconnectMCPServer_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("connection error")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.ReconnectMCPServer(ctx, "filesystem")

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestReconnectMCPServer_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.ReconnectMCPServer(ctx, "my-server")

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "mcp_reconnect" {
		t.Errorf("expected subtype 'mcp_reconnect', got %v", reqData["subtype"])
	}
	if reqData["serverName"] != "my-server" {
		t.Errorf("expected serverName 'my-server', got %v", reqData["serverName"])
	}
}

func TestReconnectMCPServer_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.ReconnectMCPServer(ctx, "filesystem")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

func TestReconnectMCPServer_ErrorResponse(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.ReconnectMCPServer(ctx, "nonexistent")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendErrorResponse(mockTransport, reqID, "server not found")
	}

	select {
	case <-done:
		if err == nil {
			t.Error("expected error from CLI")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// ToggleMCPServer Tests
// ============================================================================

func TestToggleMCPServer_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.ToggleMCPServer(ctx, "filesystem", true)

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestToggleMCPServer_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("write error")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.ToggleMCPServer(ctx, "filesystem", true)

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestToggleMCPServer_RequestFormat_Enable(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.ToggleMCPServer(ctx, "test-server", true)

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "mcp_toggle" {
		t.Errorf("expected subtype 'mcp_toggle', got %v", reqData["subtype"])
	}
	if reqData["serverName"] != "test-server" {
		t.Errorf("expected serverName 'test-server', got %v", reqData["serverName"])
	}
	if reqData["enabled"] != true {
		t.Errorf("expected enabled true, got %v", reqData["enabled"])
	}
}

func TestToggleMCPServer_RequestFormat_Disable(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.ToggleMCPServer(ctx, "disabled-server", false)

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["enabled"] != false {
		t.Errorf("expected enabled false, got %v", reqData["enabled"])
	}
}

func TestToggleMCPServer_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.ToggleMCPServer(ctx, "filesystem", true)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

func TestToggleMCPServer_ErrorResponse(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.ToggleMCPServer(ctx, "nonexistent", true)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendErrorResponse(mockTransport, reqID, "server not found")
	}

	select {
	case <-done:
		if err == nil {
			t.Error("expected error from CLI")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// StopTask Tests
// ============================================================================

func TestStopTask_NonStreamingMode(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, false, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.StopTask(ctx, "task-123")

	if err == nil {
		t.Error("expected error for non-streaming mode")
	}
	if !containsSubstring(err.Error(), "streaming mode") {
		t.Errorf("error should contain 'streaming mode', got: %v", err)
	}
}

func TestStopTask_WriteError(t *testing.T) {
	mockTransport := newControlMockTransport()
	mockTransport.writeErr = errors.New("connection lost")

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	err := q.StopTask(ctx, "task-123")

	if err == nil {
		t.Error("expected error for write failure")
	}
	if !containsSubstring(err.Error(), "failed to write") {
		t.Errorf("error should contain 'failed to write', got: %v", err)
	}
}

func TestStopTask_RequestFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.StopTask(ctx, "task-abc-123")

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &request); err != nil {
		t.Fatalf("failed to parse request JSON: %v", err)
	}

	reqData, ok := request["request"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'request' field to be a map")
	}

	if reqData["subtype"] != "stop_task" {
		t.Errorf("expected subtype 'stop_task', got %v", reqData["subtype"])
	}
	if reqData["task_id"] != "task-abc-123" {
		t.Errorf("expected task_id 'task-abc-123', got %v", reqData["task_id"])
	}
}

func TestStopTask_Success(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.StopTask(ctx, "task-123")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendSuccessResponse(mockTransport, reqID, map[string]interface{}{})
	}

	select {
	case <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

func TestStopTask_ErrorResponse(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx := context.Background()
	_ = q.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	var err error
	done := make(chan struct{})

	go func() {
		err = q.StopTask(ctx, "invalid-task")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	writes := mockTransport.getWriteCalls()
	if len(writes) > 0 {
		reqID := extractRequestID(writes[len(writes)-1])
		sendErrorResponse(mockTransport, reqID, "task not found")
	}

	select {
	case <-done:
		if err == nil {
			t.Error("expected error from CLI")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}

	_ = q.Close(ctx)
}

// ============================================================================
// Context Cancellation Tests
// ============================================================================

func TestControlMethods_CancelledContext(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	// Pre-cancel the context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("GetMCPStatus", func(t *testing.T) {
		_, err := q.GetMCPStatus(ctx)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("GetContextUsage", func(t *testing.T) {
		_, err := q.GetContextUsage(ctx)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("Interrupt", func(t *testing.T) {
		err := q.Interrupt(ctx)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("SetPermissionMode", func(t *testing.T) {
		err := q.SetPermissionMode(ctx, "plan")
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("SetModel", func(t *testing.T) {
		err := q.SetModel(ctx, "claude-sonnet")
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("RewindFiles", func(t *testing.T) {
		err := q.RewindFiles(ctx, "msg-123")
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("ReconnectMCPServer", func(t *testing.T) {
		err := q.ReconnectMCPServer(ctx, "server")
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("ToggleMCPServer", func(t *testing.T) {
		err := q.ToggleMCPServer(ctx, "server", true)
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})

	t.Run("StopTask", func(t *testing.T) {
		err := q.StopTask(ctx, "task-123")
		if err == nil {
			t.Error("expected error with cancelled context")
		}
	})
}

// ============================================================================
// Request ID Uniqueness Tests
// ============================================================================

func TestControlRequestIDUniqueness(t *testing.T) {
	requestIDs := make(map[string]bool)

	for i := 0; i < 10; i++ {
		// Use a new mockTransport for each iteration to avoid accumulating writes
		mockTransport := newControlMockTransport()
		q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		_ = q.Interrupt(ctx)
		cancel()

		writes := mockTransport.getWriteCalls()
		for _, write := range writes {
			reqID := extractRequestID(write)
			if reqID != "" {
				if requestIDs[reqID] {
					t.Errorf("duplicate request_id: %s", reqID)
				}
				requestIDs[reqID] = true
			}
		}
	}

	if len(requestIDs) < 5 {
		t.Errorf("expected at least 5 unique request IDs, got %d", len(requestIDs))
	}
}

// ============================================================================
// Request Format Validation Tests
// ============================================================================

func TestControlRequestWriteFormat(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = q.Interrupt(ctx)

	writes := mockTransport.getWriteCalls()
	for _, write := range writes {
		// Verify request ends with newline
		if len(write) > 0 && write[len(write)-1] != '\n' {
			t.Errorf("request should end with newline: %q", write)
		}

		// Verify valid JSON
		var req map[string]interface{}
		if err := json.Unmarshal([]byte(write), &req); err != nil {
			t.Errorf("invalid JSON in request: %v, data: %q", err, write)
		}

		// Verify required fields
		if req["type"] != "control_request" {
			t.Errorf("expected type 'control_request', got %v", req["type"])
		}
		if req["request_id"] == "" {
			t.Error("expected non-empty request_id")
		}
	}
}

// ============================================================================
// Empty Parameter Tests
// ============================================================================

func TestControlMethods_EmptyParameters(t *testing.T) {
	mockTransport := newControlMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	t.Run("SetPermissionMode empty mode", func(t *testing.T) {
		_ = q.SetPermissionMode(ctx, "")
		writes := mockTransport.getWriteCalls()
		if len(writes) > 0 {
			var req map[string]interface{}
			if err := json.Unmarshal([]byte(writes[len(writes)-1]), &req); err == nil {
				reqData := req["request"].(map[string]interface{})
				if reqData["mode"] != "" {
					t.Errorf("expected mode '', got %v", reqData["mode"])
				}
			}
		}
	})

	t.Run("SetModel empty model", func(t *testing.T) {
		_ = q.SetModel(ctx, "")
		writes := mockTransport.getWriteCalls()
		if len(writes) > 0 {
			var req map[string]interface{}
			if err := json.Unmarshal([]byte(writes[len(writes)-1]), &req); err == nil {
				reqData := req["request"].(map[string]interface{})
				if reqData["model"] != "" {
					t.Errorf("expected model '', got %v", reqData["model"])
				}
			}
		}
	})

	t.Run("RewindFiles empty message ID", func(t *testing.T) {
		_ = q.RewindFiles(ctx, "")
		writes := mockTransport.getWriteCalls()
		if len(writes) > 0 {
			var req map[string]interface{}
			if err := json.Unmarshal([]byte(writes[len(writes)-1]), &req); err == nil {
				reqData := req["request"].(map[string]interface{})
				if reqData["user_message_id"] != "" {
					t.Errorf("expected user_message_id '', got %v", reqData["user_message_id"])
				}
			}
		}
	})

	t.Run("ReconnectMCPServer empty server name", func(t *testing.T) {
		_ = q.ReconnectMCPServer(ctx, "")
		writes := mockTransport.getWriteCalls()
		if len(writes) > 0 {
			var req map[string]interface{}
			if err := json.Unmarshal([]byte(writes[len(writes)-1]), &req); err == nil {
				reqData := req["request"].(map[string]interface{})
				if reqData["serverName"] != "" {
					t.Errorf("expected serverName '', got %v", reqData["serverName"])
				}
			}
		}
	})

	t.Run("ToggleMCPServer empty server name", func(t *testing.T) {
		_ = q.ToggleMCPServer(ctx, "", true)
		writes := mockTransport.getWriteCalls()
		if len(writes) > 0 {
			var req map[string]interface{}
			if err := json.Unmarshal([]byte(writes[len(writes)-1]), &req); err == nil {
				reqData := req["request"].(map[string]interface{})
				if reqData["serverName"] != "" {
					t.Errorf("expected serverName '', got %v", reqData["serverName"])
				}
			}
		}
	})

	t.Run("StopTask empty task ID", func(t *testing.T) {
		_ = q.StopTask(ctx, "")
		writes := mockTransport.getWriteCalls()
		if len(writes) > 0 {
			var req map[string]interface{}
			if err := json.Unmarshal([]byte(writes[len(writes)-1]), &req); err == nil {
				reqData := req["request"].(map[string]interface{})
				if reqData["task_id"] != "" {
					t.Errorf("expected task_id '', got %v", reqData["task_id"])
				}
			}
		}
	})
}
