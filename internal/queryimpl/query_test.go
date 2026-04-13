// Package query provides internal query implementation for the Claude Agent SDK.
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

// mockTransport implements the Transport interface for testing.
type mockTransport struct {
	writeCalls     []string
	writeErr       error
	endInputCalled bool
	endInputErr    error
	closeCalled    bool
	closeErr       error
	messages       chan map[string]interface{}
	writeBlock     chan struct{} // Used to block writes for testing
	mu             sync.Mutex
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		messages:   make(chan map[string]interface{}, 100),
		writeBlock: make(chan struct{}),
	}
}

func (m *mockTransport) Write(ctx context.Context, data string) error {
	m.mu.Lock()
	m.writeCalls = append(m.writeCalls, data)
	m.mu.Unlock()
	if m.writeErr != nil {
		return m.writeErr
	}
	return nil
}

func (m *mockTransport) EndInput(ctx context.Context) error {
	m.mu.Lock()
	m.endInputCalled = true
	m.mu.Unlock()
	if m.endInputErr != nil {
		return m.endInputErr
	}
	return nil
}

func (m *mockTransport) ReadMessages(ctx context.Context) <-chan map[string]interface{} {
	return m.messages
}

func (m *mockTransport) Close(ctx context.Context) error {
	m.mu.Lock()
	m.closeCalled = true
	m.mu.Unlock()
	close(m.messages)
	if m.closeErr != nil {
		return m.closeErr
	}
	return nil
}

func (m *mockTransport) getWriteCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.writeCalls))
	copy(result, m.writeCalls)
	return result
}

func (m *mockTransport) wasEndInputCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.endInputCalled
}

func (m *mockTransport) wasCloseCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

func (m *mockTransport) sendMessage(msg map[string]interface{}) {
	m.messages <- msg
}

// mockMcpServer implements the McpServer interface for testing.
type mockMcpServer struct {
	name      string
	version   string
	err       error
	responses map[string]map[string]interface{}
}

func newMockMcpServer(name, version string) *mockMcpServer {
	return &mockMcpServer{
		name:      name,
		version:   version,
		responses: make(map[string]map[string]interface{}),
	}
}

func (m *mockMcpServer) Name() string {
	return m.name
}

func (m *mockMcpServer) Version() string {
	return m.version
}

func (m *mockMcpServer) HandleRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	if resp, ok := m.responses[method]; ok {
		return resp, nil
	}
	return map[string]interface{}{"method": method, "params": params}, nil
}

// TestNewQuery tests that NewQuery correctly initializes the Query struct.
func TestNewQuery(t *testing.T) {
	tests := []struct {
		name              string
		transport         Transport
		isStreamingMode   bool
		canUseTool        CanUseToolCallback
		hooks             map[string][]HookMatcher
		sdkMcpServers     map[string]McpServer
		initializeTimeout time.Duration
		agents            map[string]map[string]interface{}
		wantTimeout       time.Duration
	}{
		{
			name:              "basic initialization with defaults",
			transport:         newMockTransport(),
			isStreamingMode:   false,
			initializeTimeout: 0,
			wantTimeout:       60 * time.Second, // default timeout
		},
		{
			name:              "streaming mode enabled",
			transport:         newMockTransport(),
			isStreamingMode:   true,
			initializeTimeout: 0,
			wantTimeout:       60 * time.Second,
		},
		{
			name:              "custom initialize timeout",
			transport:         newMockTransport(),
			isStreamingMode:   true,
			initializeTimeout: 30 * time.Second,
			wantTimeout:       30 * time.Second,
		},
		{
			name:            "with permission callback",
			transport:       newMockTransport(),
			isStreamingMode: true,
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			},
			initializeTimeout: 45 * time.Second,
			wantTimeout:       45 * time.Second,
		},
		{
			name:            "with hooks",
			transport:       newMockTransport(),
			isStreamingMode: true,
			hooks: map[string][]HookMatcher{
				"PreToolUse": {
					{Matcher: "Bash", Hooks: []HookCallbackFunc{}},
				},
			},
			initializeTimeout: 50 * time.Second,
			wantTimeout:       50 * time.Second,
		},
		{
			name:            "with MCP servers",
			transport:       newMockTransport(),
			isStreamingMode: true,
			sdkMcpServers: map[string]McpServer{
				"test-server": newMockMcpServer("test-server", "1.0.0"),
			},
			initializeTimeout: 55 * time.Second,
			wantTimeout:       55 * time.Second,
		},
		{
			name:            "with agents",
			transport:       newMockTransport(),
			isStreamingMode: true,
			agents: map[string]map[string]interface{}{
				"my-agent": {
					"description": "Test agent",
					"prompt":      "You are a test agent",
				},
			},
			initializeTimeout: 40 * time.Second,
			wantTimeout:       40 * time.Second,
		},
		{
			name:            "all fields populated",
			transport:       newMockTransport(),
			isStreamingMode: true,
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				return nil, nil
			},
			hooks:             map[string][]HookMatcher{"PostToolUse": {{Matcher: "Write"}}},
			sdkMcpServers:     map[string]McpServer{"server": newMockMcpServer("server", "1.0")},
			initializeTimeout: 25 * time.Second,
			agents:            map[string]map[string]interface{}{"agent": {"prompt": "test"}},
			wantTimeout:       25 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQuery(
				tt.transport,
				tt.isStreamingMode,
				tt.canUseTool,
				tt.hooks,
				tt.sdkMcpServers,
				tt.initializeTimeout,
				tt.agents,
				nil,
			)

			if q == nil {
				t.Fatal("NewQuery returned nil")
			}

			if q.transport != tt.transport {
				t.Error("transport not set correctly")
			}

			if q.isStreamingMode != tt.isStreamingMode {
				t.Errorf("isStreamingMode = %v, want %v", q.isStreamingMode, tt.isStreamingMode)
			}

			if q.canUseTool == nil && tt.canUseTool != nil {
				t.Error("canUseTool callback not set")
			}

			if q.initializeTimeout != tt.wantTimeout {
				t.Errorf("initializeTimeout = %v, want %v", q.initializeTimeout, tt.wantTimeout)
			}

			if q.pendingControlResponses == nil {
				t.Error("pendingControlResponses not initialized")
			}

			if q.pendingControlResults == nil {
				t.Error("pendingControlResults not initialized")
			}

			if q.hookCallbacks == nil {
				t.Error("hookCallbacks not initialized")
			}

			if q.messageSend == nil {
				t.Error("messageSend channel not initialized")
			}

			if q.messageReceive == nil {
				t.Error("messageReceive channel not initialized")
			}

			if q.firstResultEvent == nil {
				t.Error("firstResultEvent channel not initialized")
			}

			if q.streamCloseTimeout == 0 {
				t.Error("streamCloseTimeout not set")
			}

			// Check that initialized and closed are false
			if q.initialized {
				t.Error("query should not be initialized initially")
			}

			if q.closed {
				t.Error("query should not be closed initially")
			}
		})
	}
}

// TestSendControlRequest tests the sendControlRequest method.
func TestSendControlRequest(t *testing.T) {
	tests := []struct {
		name          string
		isStreaming   bool
		request       map[string]interface{}
		setupMock     func(*mockTransport)
		wantErr       bool
		errContains   string
		responseDelay time.Duration
		response      map[string]interface{}
	}{
		{
			name:        "non-streaming mode returns error",
			isStreaming: false,
			request:     map[string]interface{}{"subtype": "test"},
			wantErr:     true,
			errContains: "streaming mode",
		},
		{
			name:        "write error",
			isStreaming: true,
			request:     map[string]interface{}{"subtype": "test"},
			setupMock: func(m *mockTransport) {
				m.writeErr = errors.New("write failed")
			},
			wantErr:     true,
			errContains: "failed to write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := newMockTransport()
			if tt.setupMock != nil {
				tt.setupMock(mockTransport)
			}

			q := NewQuery(
				mockTransport,
				tt.isStreaming,
				nil,
				nil,
				nil,
				30*time.Second,
				nil,
				nil,
			)

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// For streaming mode tests that need a response, we need to handle it differently
			if tt.isStreaming && tt.response != nil {
				go func() {
					time.Sleep(tt.responseDelay)
					// Simulate response
					mockTransport.sendMessage(map[string]interface{}{
						"type": "control_response",
						"response": map[string]interface{}{
							"request_id": "test-id",
							"response":   tt.response,
						},
					})
				}()
			}

			resp, err := q.sendControlRequest(ctx, tt.request, 1*time.Second)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp == nil {
					t.Error("expected response, got nil")
				}
			}
		})
	}
}

// TestSendControlRequestWithResponse tests sendControlRequest with simulated responses.
func TestSendControlRequestWithResponse(t *testing.T) {
	mockTransport := newMockTransport()

	q := NewQuery(
		mockTransport,
		true, // streaming mode
		nil,
		nil,
		nil,
		30*time.Second,
		nil,
		nil,
	)

	// Start the query to enable message processing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = q.Start(ctx)

	// Give the goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Test timeout scenario
	t.Run("timeout on no response", func(t *testing.T) {
		// Use a very short timeout
		_, err := q.sendControlRequest(context.Background(), map[string]interface{}{"subtype": "test"}, 50*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error")
		}
		if !containsSubstring(err.Error(), "timeout") {
			t.Errorf("expected timeout error, got: %v", err)
		}
	})

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := q.sendControlRequest(ctx, map[string]interface{}{"subtype": "test"}, 5*time.Second)
		if err == nil {
			t.Error("expected context canceled error")
		}
	})

	// Cleanup
	_ = q.Close(context.Background())
}

// TestHandleToolPermissionRequest tests the handleToolPermissionRequest method.
func TestHandleToolPermissionRequest(t *testing.T) {
	tests := []struct {
		name         string
		canUseTool   CanUseToolCallback
		requestData  map[string]interface{}
		wantErr      bool
		errContains  string
		wantBehavior string
		wantMessage  string
		wantInput    map[string]interface{}
	}{
		{
			name:        "no callback returns error",
			canUseTool:  nil,
			requestData: map[string]interface{}{"tool_name": "Bash"},
			wantErr:     true,
			errContains: "not provided",
		},
		{
			name: "allow permission",
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				return types.PermissionResultAllow{
					Behavior: "allow",
				}, nil
			},
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
			},
			wantErr:      false,
			wantBehavior: "allow",
		},
		{
			name: "allow permission with updated input",
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				return types.PermissionResultAllow{
					Behavior:     "allow",
					UpdatedInput: map[string]interface{}{"command": "ls -la"},
				}, nil
			},
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
			},
			wantErr:      false,
			wantBehavior: "allow",
			wantInput:    map[string]interface{}{"command": "ls -la"},
		},
		{
			name: "allow permission with updated permissions",
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				perm := types.PermissionBehaviorAllow
				return types.PermissionResultAllow{
					Behavior: "allow",
					UpdatedPermissions: []types.PermissionUpdate{
						{
							Type:        types.PermissionUpdateTypeAddRules,
							Behavior:    &perm,
							Destination: types.PermissionUpdateDestinationPtr(types.PermissionUpdateDestinationSession),
						},
					},
				}, nil
			},
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
			},
			wantErr:      false,
			wantBehavior: "allow",
		},
		{
			name: "deny permission",
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				return types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   "Operation not allowed",
					Interrupt: true,
				}, nil
			},
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "rm -rf /"},
			},
			wantErr:      false,
			wantBehavior: "deny",
			wantMessage:  "Operation not allowed",
		},
		{
			name: "callback returns error",
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				return nil, errors.New("permission check failed")
			},
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
			},
			wantErr:     true,
			errContains: "permission check failed",
		},
		{
			name: "nil input defaults to empty map",
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				if input == nil {
					return nil, errors.New("input should not be nil")
				}
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			},
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				// No input provided
			},
			wantErr:      false,
			wantBehavior: "allow",
		},
		{
			name: "permission suggestions are parsed",
			canUseTool: func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
				if len(context.Suggestions) == 0 {
					return nil, errors.New("expected suggestions")
				}
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			},
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				"input":     map[string]interface{}{},
				"permission_suggestions": []interface{}{
					map[string]interface{}{
						"type": "addRules",
						"rules": []interface{}{
							map[string]interface{}{
								"toolName":    "Bash",
								"ruleContent": "ls",
							},
						},
					},
				},
			},
			wantErr:      false,
			wantBehavior: "allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := newMockTransport()
			q := NewQuery(mockTransport, true, tt.canUseTool, nil, nil, 30*time.Second, nil, nil)

			ctx := context.Background()
			resp, err := q.handleToolPermissionRequest(ctx, "test-request-id", tt.requestData)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if resp == nil {
					t.Error("expected response, got nil")
					return
				}

				behavior, ok := resp["behavior"].(string)
				if !ok {
					t.Error("response missing behavior field")
					return
				}

				if behavior != tt.wantBehavior {
					t.Errorf("behavior = %v, want %v", behavior, tt.wantBehavior)
				}

				if tt.wantInput != nil {
					updatedInput, ok := resp["updatedInput"].(map[string]interface{})
					if !ok {
						t.Error("response missing updatedInput field")
						return
					}
					for k, v := range tt.wantInput {
						if updatedInput[k] != v {
							t.Errorf("updatedInput[%v] = %v, want %v", k, updatedInput[k], v)
						}
					}
				}

				if tt.wantMessage != "" {
					message, ok := resp["message"].(string)
					if !ok {
						t.Error("response missing message field")
						return
					}
					if message != tt.wantMessage {
						t.Errorf("message = %v, want %v", message, tt.wantMessage)
					}
				}
			}
		})
	}
}

// TestHandleToolPermissionRequestInvalidResult tests that invalid permission results return an error.
func TestHandleToolPermissionRequestInvalidResult(t *testing.T) {
	// Test callback returning invalid type
	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		// Return nil (which is invalid)
		return nil, nil
	}

	mockTransport := newMockTransport()
	q := NewQuery(mockTransport, true, canUseTool, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	_, err := q.handleToolPermissionRequest(ctx, "test-id", map[string]interface{}{"tool_name": "Bash"})
	if err == nil {
		t.Error("expected error for invalid permission result type")
	}
}

// TestHandleHookCallbackRequest tests the handleHookCallbackRequest method.
func TestHandleHookCallbackRequest(t *testing.T) {
	tests := []struct {
		name        string
		setupQuery  func(*Query)
		requestData map[string]interface{}
		wantErr     bool
		errContains string
		validate    func(t *testing.T, resp map[string]interface{})
	}{
		{
			name: "callback not found",
			setupQuery: func(q *Query) {
				// Don't register any callback
			},
			requestData: map[string]interface{}{
				"callback_id": "nonexistent",
			},
			wantErr:     true,
			errContains: "no hook callback found",
		},
		{
			name: "callback returns result",
			setupQuery: func(q *Query) {
				q.hookCallbacks["test-callback"] = func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error) {
					return map[string]interface{}{
						"decision": "allow",
					}, nil
				}
			},
			requestData: map[string]interface{}{
				"callback_id": "test-callback",
				"input":       map[string]interface{}{"test": "data"},
			},
			wantErr: false,
			validate: func(t *testing.T, resp map[string]interface{}) {
				if resp["decision"] != "allow" {
					t.Errorf("expected decision 'allow', got %v", resp["decision"])
				}
			},
		},
		{
			name: "callback with tool_use_id",
			setupQuery: func(q *Query) {
				q.hookCallbacks["test-callback"] = func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error) {
					if toolUseID == nil {
						return nil, errors.New("expected tool_use_id")
					}
					if *toolUseID != "tool-123" {
						return nil, errors.New("wrong tool_use_id")
					}
					return map[string]interface{}{"success": true}, nil
				}
			},
			requestData: map[string]interface{}{
				"callback_id": "test-callback",
				"input":       map[string]interface{}{},
				"tool_use_id": "tool-123",
			},
			wantErr: false,
			validate: func(t *testing.T, resp map[string]interface{}) {
				if resp["success"] != true {
					t.Error("expected success true")
				}
			},
		},
		{
			name: "callback returns error",
			setupQuery: func(q *Query) {
				q.hookCallbacks["error-callback"] = func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error) {
					return nil, errors.New("callback error")
				}
			},
			requestData: map[string]interface{}{
				"callback_id": "error-callback",
			},
			wantErr:     true,
			errContains: "callback error",
		},
		{
			name: "callback with empty tool_use_id",
			setupQuery: func(q *Query) {
				q.hookCallbacks["test-callback"] = func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error) {
					if toolUseID != nil {
						return nil, errors.New("tool_use_id should be nil when empty")
					}
					return map[string]interface{}{"success": true}, nil
				}
			},
			requestData: map[string]interface{}{
				"callback_id": "test-callback",
				"input":       map[string]interface{}{},
				"tool_use_id": "", // Empty string
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := newMockTransport()
			q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

			if tt.setupQuery != nil {
				tt.setupQuery(q)
			}

			ctx := context.Background()
			resp, err := q.handleHookCallbackRequest(ctx, "test-request-id", tt.requestData)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestHandleMCPMessageRequest tests the handleMCPMessageRequest method.
func TestHandleMCPMessageRequest(t *testing.T) {
	tests := []struct {
		name         string
		sdkMcpServer map[string]McpServer
		requestData  map[string]interface{}
		wantErr      bool
		errContains  string
		validate     func(t *testing.T, resp map[string]interface{})
	}{
		{
			name:        "missing server_name",
			requestData: map[string]interface{}{"message": map[string]interface{}{}},
			wantErr:     true,
			errContains: "missing server_name",
		},
		{
			name:        "missing message",
			requestData: map[string]interface{}{"server_name": "test-server"},
			wantErr:     true,
			errContains: "missing server_name or message",
		},
		{
			name:        "server not found",
			requestData: map[string]interface{}{"server_name": "nonexistent", "message": map[string]interface{}{"method": "test"}},
			wantErr:     false, // Returns error response, not Go error
			validate: func(t *testing.T, resp map[string]interface{}) {
				mcpResp, ok := resp["mcp_response"].(map[string]interface{})
				if !ok {
					t.Error("expected mcp_response in result")
					return
				}
				errObj, ok := mcpResp["error"].(map[string]interface{})
				if !ok {
					t.Error("expected error in mcp_response")
					return
				}
				code := getErrorCode(errObj["code"])
				if code != -32601 {
					t.Errorf("expected error code -32601, got %v", code)
				}
			},
		},
		{
			name: "initialize request",
			sdkMcpServer: map[string]McpServer{
				"test-server": newMockMcpServer("test-server", "1.0.0"),
			},
			requestData: map[string]interface{}{
				"server_name": "test-server",
				"message": map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      1,
					"method":  "initialize",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp map[string]interface{}) {
				mcpResp, ok := resp["mcp_response"].(map[string]interface{})
				if !ok {
					t.Error("expected mcp_response in result")
					return
				}
				result, ok := mcpResp["result"].(map[string]interface{})
				if !ok {
					t.Error("expected result in mcp_response")
					return
				}
				serverInfo, ok := result["serverInfo"].(map[string]interface{})
				if !ok {
					t.Error("expected serverInfo in result")
					return
				}
				if serverInfo["name"] != "test-server" {
					t.Errorf("expected server name 'test-server', got %v", serverInfo["name"])
				}
				if serverInfo["version"] != "1.0.0" {
					t.Errorf("expected server version '1.0.0', got %v", serverInfo["version"])
				}
			},
		},
		{
			name: "tools/list request",
			sdkMcpServer: map[string]McpServer{
				"test-server": newMockMcpServer("test-server", "1.0.0"),
			},
			requestData: map[string]interface{}{
				"server_name": "test-server",
				"message": map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      2,
					"method":  "tools/list",
					"params":  map[string]interface{}{},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp map[string]interface{}) {
				mcpResp, ok := resp["mcp_response"].(map[string]interface{})
				if !ok {
					t.Error("expected mcp_response in result")
					return
				}
				result, ok := mcpResp["result"].(map[string]interface{})
				if !ok {
					t.Error("expected result in mcp_response")
					return
				}
				if result["method"] != "tools/list" {
					t.Errorf("expected method 'tools/list', got %v", result["method"])
				}
			},
		},
		{
			name: "tools/call request with params",
			sdkMcpServer: map[string]McpServer{
				"test-server": newMockMcpServer("test-server", "1.0.0"),
			},
			requestData: map[string]interface{}{
				"server_name": "test-server",
				"message": map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      3,
					"method":  "tools/call",
					"params": map[string]interface{}{
						"name": "test-tool",
						"arguments": map[string]interface{}{
							"arg1": "value1",
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp map[string]interface{}) {
				mcpResp, ok := resp["mcp_response"].(map[string]interface{})
				if !ok {
					t.Error("expected mcp_response in result")
					return
				}
				result, ok := mcpResp["result"].(map[string]interface{})
				if !ok {
					t.Error("expected result in mcp_response")
					return
				}
				params, ok := result["params"].(map[string]interface{})
				if !ok {
					t.Error("expected params in result")
					return
				}
				if params["name"] != "test-tool" {
					t.Errorf("expected tool name 'test-tool', got %v", params["name"])
				}
			},
		},
		{
			name: "server returns error",
			sdkMcpServer: map[string]McpServer{
				"error-server": func() McpServer {
					s := newMockMcpServer("error-server", "1.0.0")
					s.err = errors.New("tool execution failed")
					return s
				}(),
			},
			requestData: map[string]interface{}{
				"server_name": "error-server",
				"message": map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      4,
					"method":  "tools/call",
				},
			},
			wantErr: false, // Returns error in JSON-RPC response, not Go error
			validate: func(t *testing.T, resp map[string]interface{}) {
				mcpResp, ok := resp["mcp_response"].(map[string]interface{})
				if !ok {
					t.Error("expected mcp_response in result")
					return
				}
				errObj, ok := mcpResp["error"].(map[string]interface{})
				if !ok {
					t.Error("expected error in mcp_response")
					return
				}
				code := getErrorCode(errObj["code"])
				if code != -32603 {
					t.Errorf("expected error code -32603, got %v", code)
				}
				if errObj["message"] != "tool execution failed" {
					t.Errorf("expected error message 'tool execution failed', got %v", errObj["message"])
				}
			},
		},
		{
			name: "notifications/initialized request",
			sdkMcpServer: map[string]McpServer{
				"test-server": newMockMcpServer("test-server", "1.0.0"),
			},
			requestData: map[string]interface{}{
				"server_name": "test-server",
				"message": map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "notifications/initialized",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, resp map[string]interface{}) {
				mcpResp, ok := resp["mcp_response"].(map[string]interface{})
				if !ok {
					t.Error("expected mcp_response in result")
					return
				}
				if mcpResp["jsonrpc"] != "2.0" {
					t.Errorf("expected jsonrpc '2.0', got %v", mcpResp["jsonrpc"])
				}
			},
		},
		{
			name: "unknown method returns error",
			sdkMcpServer: map[string]McpServer{
				"test-server": newMockMcpServer("test-server", "1.0.0"),
			},
			requestData: map[string]interface{}{
				"server_name": "test-server",
				"message": map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      5,
					"method":  "unknown/method",
				},
			},
			wantErr: false, // Returns error in JSON-RPC response
			validate: func(t *testing.T, resp map[string]interface{}) {
				mcpResp, ok := resp["mcp_response"].(map[string]interface{})
				if !ok {
					t.Error("expected mcp_response in result")
					return
				}
				errObj, ok := mcpResp["error"].(map[string]interface{})
				if !ok {
					t.Error("expected error in mcp_response")
					return
				}
				code := getErrorCode(errObj["code"])
				if code != -32601 {
					t.Errorf("expected error code -32601, got %v", code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := newMockTransport()
			q := NewQuery(mockTransport, true, nil, nil, tt.sdkMcpServer, 30*time.Second, nil, nil)

			ctx := context.Background()
			resp, err := q.handleMCPMessageRequest(ctx, "test-request-id", tt.requestData)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

// TestConvertHookOutputForCLI tests the convertHookOutputForCLI function.
func TestConvertHookOutputForCLI(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name:     "no conversion needed",
			input:    map[string]interface{}{"decision": "allow", "message": "test"},
			expected: map[string]interface{}{"decision": "allow", "message": "test"},
		},
		{
			name:     "convert async_ to async",
			input:    map[string]interface{}{"async_": true, "message": "test"},
			expected: map[string]interface{}{"async": true, "message": "test"},
		},
		{
			name:     "convert Async_ to async",
			input:    map[string]interface{}{"Async_": true, "message": "test"},
			expected: map[string]interface{}{"async": true, "message": "test"},
		},
		{
			name:     "convert continue_ to continue",
			input:    map[string]interface{}{"continue_": false, "decision": "block"},
			expected: map[string]interface{}{"continue": false, "decision": "block"},
		},
		{
			name:     "convert Continue_ to continue",
			input:    map[string]interface{}{"Continue_": false, "decision": "block"},
			expected: map[string]interface{}{"continue": false, "decision": "block"},
		},
		{
			name:     "convert both async_ and continue_",
			input:    map[string]interface{}{"async_": true, "continue_": false, "reason": "timeout"},
			expected: map[string]interface{}{"async": true, "continue": false, "reason": "timeout"},
		},
		{
			name:     "nested values preserved",
			input:    map[string]interface{}{"async_": true, "nested": map[string]interface{}{"key": "value"}},
			expected: map[string]interface{}{"async": true, "nested": map[string]interface{}{"key": "value"}},
		},
		{
			name:     "array values preserved",
			input:    map[string]interface{}{"async_": true, "items": []interface{}{1, 2, 3}},
			expected: map[string]interface{}{"async": true, "items": []interface{}{1, 2, 3}},
		},
		{
			name:     "mixed case keys not converted",
			input:    map[string]interface{}{"ASYNC_": true, "Continue": false},
			expected: map[string]interface{}{"ASYNC_": true, "Continue": false},
		},
		{
			name: "complex hook output",
			input: map[string]interface{}{
				"async_":            true,
				"continue_":         true,
				"suppressOutput":    true,
				"stopReason":        "user_request",
				"decision":          "allow",
				"additionalContext": "extra info",
			},
			expected: map[string]interface{}{
				"async":             true,
				"continue":          true,
				"suppressOutput":    true,
				"stopReason":        "user_request",
				"decision":          "allow",
				"additionalContext": "extra info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHookOutputForCLI(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("result has %d keys, expected %d", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				resultV, ok := result[k]
				if !ok {
					t.Errorf("missing key %q in result", k)
					continue
				}

				if !deepEqual(v, resultV) {
					t.Errorf("result[%q] = %v, expected %v", k, resultV, v)
				}
			}
		})
	}
}

// TestHandleControlRequest tests the handleControlRequest method.
func TestHandleControlRequest(t *testing.T) {
	tests := []struct {
		name        string
		setupQuery  func(*Query)
		request     map[string]interface{}
		wantErrType string // "error" response subtype, empty for success
		validate    func(t *testing.T, writes []string)
	}{
		{
			name: "can_use_tool success",
			setupQuery: func(q *Query) {
				q.canUseTool = func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
					return types.PermissionResultAllow{Behavior: "allow"}, nil
				}
			},
			request: map[string]interface{}{
				"request_id": "req-123",
				"request": map[string]interface{}{
					"subtype":   "can_use_tool",
					"tool_name": "Bash",
					"input":     map[string]interface{}{"command": "ls"},
				},
			},
			validate: func(t *testing.T, writes []string) {
				if len(writes) != 1 {
					t.Errorf("expected 1 write, got %d", len(writes))
					return
				}
				var resp map[string]interface{}
				if err := json.Unmarshal([]byte(writes[0]), &resp); err != nil {
					t.Errorf("failed to parse response: %v", err)
					return
				}
				if resp["type"] != "control_response" {
					t.Errorf("expected type 'control_response', got %v", resp["type"])
				}
				response := resp["response"].(map[string]interface{})
				if response["subtype"] != "success" {
					t.Errorf("expected subtype 'success', got %v", response["subtype"])
				}
			},
		},
		{
			name:    "unsupported subtype returns error",
			request: map[string]interface{}{"request_id": "req-456", "request": map[string]interface{}{"subtype": "unknown"}},
			validate: func(t *testing.T, writes []string) {
				if len(writes) != 1 {
					t.Errorf("expected 1 write, got %d", len(writes))
					return
				}
				var resp map[string]interface{}
				if err := json.Unmarshal([]byte(writes[0]), &resp); err != nil {
					t.Errorf("failed to parse response: %v", err)
					return
				}
				response := resp["response"].(map[string]interface{})
				if response["subtype"] != "error" {
					t.Errorf("expected subtype 'error', got %v", response["subtype"])
				}
				if !containsSubstring(response["error"].(string), "unsupported control request subtype") {
					t.Errorf("expected 'unsupported' error, got %v", response["error"])
				}
			},
		},
		{
			name:    "missing request data returns early",
			request: map[string]interface{}{"request_id": "req-789"},
			validate: func(t *testing.T, writes []string) {
				if len(writes) != 0 {
					t.Errorf("expected no writes for missing request data, got %d", len(writes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := newMockTransport()
			q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

			if tt.setupQuery != nil {
				tt.setupQuery(q)
			}

			ctx := context.Background()
			q.handleControlRequest(ctx, tt.request)

			writes := mockTransport.getWriteCalls()
			if tt.validate != nil {
				tt.validate(t, writes)
			}
		})
	}
}

// TestParsePermissionUpdate tests the parsePermissionUpdate function.
func TestParsePermissionUpdate(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected types.PermissionUpdate
	}{
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: types.PermissionUpdate{},
		},
		{
			name: "type only",
			input: map[string]interface{}{
				"type": "addRules",
			},
			expected: types.PermissionUpdate{
				Type: types.PermissionUpdateTypeAddRules,
			},
		},
		{
			name: "type with destination",
			input: map[string]interface{}{
				"type":        "setMode",
				"destination": "session",
			},
			expected: types.PermissionUpdate{
				Type:        types.PermissionUpdateTypeSetMode,
				Destination: types.PermissionUpdateDestinationPtr(types.PermissionUpdateDestinationSession),
			},
		},
		{
			name: "full permission update",
			input: map[string]interface{}{
				"type":        "addRules",
				"destination": "userSettings",
				"behavior":    "allow",
				"rules": []interface{}{
					map[string]interface{}{
						"toolName":    "Bash",
						"ruleContent": "ls",
					},
				},
			},
			expected: types.PermissionUpdate{
				Type:        types.PermissionUpdateTypeAddRules,
				Destination: types.PermissionUpdateDestinationPtr(types.PermissionUpdateDestinationUserSettings),
				Behavior:    types.PermissionBehaviorPtr(types.PermissionBehaviorAllow),
				Rules: []types.PermissionRuleValue{
					{ToolName: "Bash", RuleContent: strPtr("ls")},
				},
			},
		},
		{
			name: "with mode",
			input: map[string]interface{}{
				"type": "setMode",
				"mode": "plan",
			},
			expected: types.PermissionUpdate{
				Type: types.PermissionUpdateTypeSetMode,
				Mode: types.PermissionModePtr(types.PermissionModePlan),
			},
		},
		{
			name: "with directories",
			input: map[string]interface{}{
				"type":        "addDirectories",
				"directories": []interface{}{"/path/one", "/path/two"},
			},
			expected: types.PermissionUpdate{
				Type:        types.PermissionUpdateTypeAddDirectories,
				Directories: []string{"/path/one", "/path/two"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePermissionUpdate(tt.input)

			if result.Type != tt.expected.Type {
				t.Errorf("Type = %v, expected %v", result.Type, tt.expected.Type)
			}

			if !ptrEqual(result.Destination, tt.expected.Destination) {
				t.Errorf("Destination = %v, expected %v", result.Destination, tt.expected.Destination)
			}

			if !ptrEqual(result.Behavior, tt.expected.Behavior) {
				t.Errorf("Behavior = %v, expected %v", result.Behavior, tt.expected.Behavior)
			}

			if !ptrEqual(result.Mode, tt.expected.Mode) {
				t.Errorf("Mode = %v, expected %v", result.Mode, tt.expected.Mode)
			}

			if len(result.Rules) != len(tt.expected.Rules) {
				t.Errorf("Rules length = %d, expected %d", len(result.Rules), len(tt.expected.Rules))
			} else {
				for i, r := range result.Rules {
					if r.ToolName != tt.expected.Rules[i].ToolName {
						t.Errorf("Rules[%d].ToolName = %v, expected %v", i, r.ToolName, tt.expected.Rules[i].ToolName)
					}
					if !strPtrEqual(r.RuleContent, tt.expected.Rules[i].RuleContent) {
						t.Errorf("Rules[%d].RuleContent = %v, expected %v", i, r.RuleContent, tt.expected.Rules[i].RuleContent)
					}
				}
			}

			if len(result.Directories) != len(tt.expected.Directories) {
				t.Errorf("Directories length = %d, expected %d", len(result.Directories), len(tt.expected.Directories))
			} else {
				for i, d := range result.Directories {
					if d != tt.expected.Directories[i] {
						t.Errorf("Directories[%d] = %v, expected %v", i, d, tt.expected.Directories[i])
					}
				}
			}
		})
	}
}

// TestParseDuration tests the parseDuration function.
func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"1000", 1000 * time.Millisecond, false},
		{"500", 500 * time.Millisecond, false},
		{"0", 0, false},
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("result = %v, expected %v", result, tt.expected)
				}
			}
		})
	}
}

// TestClose tests the Close method.
func TestClose(t *testing.T) {
	mockTransport := newMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

	// Start the query
	ctx := context.Background()
	_ = q.Start(ctx)

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Close
	err := q.Close(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !mockTransport.wasCloseCalled() {
		t.Error("transport Close was not called")
	}

	// Calling Close again should be safe
	err = q.Close(ctx)
	if err != nil {
		t.Errorf("unexpected error on second Close: %v", err)
	}
}

// TestIsClosed tests the isClosed method.
func TestIsClosed(t *testing.T) {
	mockTransport := newMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

	if q.isClosed() {
		t.Error("query should not be closed initially")
	}

	_ = q.Close(context.Background())

	if !q.isClosed() {
		t.Error("query should be closed after Close")
	}
}

// TestGetInitializationResult tests the GetInitializationResult method.
func TestGetInitializationResult(t *testing.T) {
	mockTransport := newMockTransport()
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

	// Initially nil
	if result := q.GetInitializationResult(); result != nil {
		t.Error("expected nil initialization result initially")
	}

	// Set initialization result
	q.mu.Lock()
	q.initializationResult = map[string]interface{}{"test": "data"}
	q.mu.Unlock()

	result := q.GetInitializationResult()
	if result == nil {
		t.Error("expected non-nil initialization result")
		return
	}
	if result["test"] != "data" {
		t.Errorf("expected result['test'] = 'data', got %v", result["test"])
	}
}

// Helper functions

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getErrorCode extracts an error code from either int or float64 representation
func getErrorCode(code interface{}) int {
	switch v := code.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

func deepEqual(a, b interface{}) bool {
	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !deepEqual(v, bv[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for i, v := range av {
			if !deepEqual(v, bv[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func strPtr(s string) *string {
	return &s
}

func strPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrEqual[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// ============================================================================
// Streaming Tests - Message Channel Behavior
// ============================================================================

// TestStreamingMessageChannel tests message channel behavior during streaming.
func TestStreamingMessageChannel(t *testing.T) {
	t.Run("messages flow through channel", func(t *testing.T) {
		mockTransport := newMockTransport()
		q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

		ctx := context.Background()
		if err := q.Start(ctx); err != nil {
			t.Fatalf("failed to start: %v", err)
		}

		// Send messages to transport
		messages := []map[string]interface{}{
			{"type": "assistant", "message": map[string]interface{}{"content": "Hello"}},
			{"type": "user", "message": map[string]interface{}{"content": "Hi"}},
			{"type": "result", "subtype": "success"},
		}

		for _, msg := range messages {
			mockTransport.sendMessage(msg)
		}

		// Receive messages
		receivedCount := 0
		timeout := time.After(2 * time.Second)
		msgChan := q.ReceiveMessages(ctx)

		for receivedCount < 3 {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					t.Fatal("channel closed unexpectedly")
				}
				receivedCount++
				msgType, _ := msg["type"].(string)
				t.Logf("Received message type: %s", msgType)
			case <-timeout:
				t.Fatalf("timeout waiting for messages, received %d", receivedCount)
			}
		}

		_ = q.Close(ctx)
	})

	t.Run("channel closes on end message", func(t *testing.T) {
		mockTransport := newMockTransport()
		q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

		ctx := context.Background()
		_ = q.Start(ctx)

		// Send end message
		mockTransport.sendMessage(map[string]interface{}{"type": "end"})

		msgChan := q.ReceiveMessages(ctx)
		_, ok := <-msgChan
		if ok {
			t.Error("expected channel to be closed after end message")
		}

		_ = q.Close(ctx)
	})

	t.Run("channel handles error message", func(t *testing.T) {
		mockTransport := newMockTransport()
		q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

		ctx := context.Background()
		_ = q.Start(ctx)

		// Send error message
		mockTransport.sendMessage(map[string]interface{}{
			"type":  "error",
			"error": "test error",
		})

		msgChan := q.ReceiveMessages(ctx)

		select {
		case msg, ok := <-msgChan:
			if !ok {
				t.Fatal("channel closed without delivering error")
			}
			if msg["type"] != "error" {
				t.Errorf("expected error message, got %v", msg)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for error message")
		}

		_ = q.Close(ctx)
	})
}

// TestConcurrentStreaming tests concurrent access to streaming.
func TestConcurrentStreaming(t *testing.T) {
	t.Run("concurrent send and receive", func(t *testing.T) {
		mockTransport := newMockTransport()
		q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

		ctx := context.Background()
		_ = q.Start(ctx)

		var wg sync.WaitGroup
		sentCount := 0
		receivedCount := 0

		// Start multiple goroutines sending messages
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					mockTransport.sendMessage(map[string]interface{}{
						"type":    "assistant",
						"id":      id,
						"message": j,
					})
					sentCount++
				}
			}(i)
		}

		// Receive messages in separate goroutine
		done := make(chan struct{})
		go func() {
			msgChan := q.ReceiveMessages(ctx)
			for {
				select {
				case _, ok := <-msgChan:
					if !ok {
						return
					}
					receivedCount++
				case <-ctx.Done():
					return
				case <-done:
					return
				}
			}
		}()

		wg.Wait()

		// Give time for messages to be received
		time.Sleep(100 * time.Millisecond)
		close(done)

		_ = q.Close(ctx)

		t.Logf("Sent %d, received %d messages", sentCount, receivedCount)
	})

	t.Run("concurrent initialize and close", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			mockTransport := newMockTransport()
			q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

			var wg sync.WaitGroup
			wg.Add(2)

			var initErr, closeErr error

			// Use a context with short timeout to prevent indefinite blocking
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)

			go func() {
				defer wg.Done()
				// Initialize may fail if close happens first
				_, initErr = q.Initialize(ctx)
			}()

			go func() {
				defer wg.Done()
				time.Sleep(time.Duration(i) * time.Millisecond)
				closeErr = q.Close(context.Background())
			}()

			wg.Wait()
			cancel()
			// One of them may error due to timing, but should not panic
			t.Logf("Init err: %v, Close err: %v", initErr, closeErr)
		}
	})
}

// TestStreamInputChannelClosure tests handling of input stream closure.
func TestStreamInputChannelClosure(t *testing.T) {
	t.Run("closes transport on stream end", func(t *testing.T) {
		mockTransport := newMockTransport()
		q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

		ctx := context.Background()
		_ = q.Start(ctx)

		// Create input stream that closes immediately
		inputChan := make(chan map[string]interface{})
		close(inputChan)

		err := q.StreamInput(ctx, inputChan)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// EndInput should be called
		if !mockTransport.wasEndInputCalled() {
			t.Error("expected EndInput to be called")
		}

		_ = q.Close(ctx)
	})

	t.Run("waits for first result with hooks", func(t *testing.T) {
		mockTransport := newMockTransport()

		// Create hooks to trigger bidirectional mode
		hooks := map[string][]HookMatcher{
			"PreToolUse": {{Matcher: "Bash", Hooks: []HookCallbackFunc{func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error) {
				return map[string]interface{}{}, nil
			}}}},
		}

		q := NewQuery(mockTransport, true, nil, hooks, nil, 30*time.Second, nil, nil)

		ctx := context.Background()
		_ = q.Start(ctx)

		// Create input stream
		inputChan := make(chan map[string]interface{})

		var streamErr error
		done := make(chan struct{})

		go func() {
			streamErr = q.StreamInput(ctx, inputChan)
			close(done)
		}()

		// Send result message to unblock
		time.Sleep(50 * time.Millisecond)
		mockTransport.sendMessage(map[string]interface{}{"type": "result", "subtype": "success"})

		// Close input
		close(inputChan)

		select {
		case <-done:
			// Stream finished
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for stream to finish")
		}

		if streamErr != nil {
			t.Errorf("unexpected error: %v", streamErr)
		}

		_ = q.Close(ctx)
	})
}

// TestReceiveMessagesChannelClosure tests ReceiveMessages channel closure.
func TestReceiveMessagesChannelClosure(t *testing.T) {
	t.Run("channel closes on context cancel", func(t *testing.T) {
		mockTransport := newMockTransport()
		q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

		ctx, cancel := context.WithCancel(context.Background())
		_ = q.Start(ctx)

		msgChan := q.ReceiveMessages(ctx)

		// Cancel context
		cancel()

		// Channel should close
		select {
		case _, ok := <-msgChan:
			if ok {
				t.Error("expected channel to close on context cancel")
			}
		case <-time.After(1 * time.Second):
			t.Error("timeout waiting for channel to close")
		}
	})
}

// ============================================================================
// Tool Permission Callback Tests - Comprehensive Coverage
// ============================================================================

// TestToolPermissionCallbackWithInputModification tests modifying tool input.
func TestToolPermissionCallbackWithInputModification(t *testing.T) {
	mockTransport := newMockTransport()

	var callbackInvoked bool
	callback := func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		callbackInvoked = true

		// Verify tool name
		if toolName != "WriteTool" {
			t.Errorf("expected tool name 'WriteTool', got %q", toolName)
		}

		// Modify input
		modifiedInput := make(map[string]interface{})
		for k, v := range input {
			modifiedInput[k] = v
		}
		modifiedInput["safe_mode"] = true

		return types.PermissionResultAllow{
			Behavior:     "allow",
			UpdatedInput: modifiedInput,
		}, nil
	}

	q := NewQuery(mockTransport, true, callback, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	request := map[string]interface{}{
		"request_id": "req-modify",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "WriteTool",
			"input":     map[string]interface{}{"file_path": "/etc/passwd"},
		},
	}

	q.handleControlRequest(ctx, request)

	if !callbackInvoked {
		t.Error("callback was not invoked")
	}

	writes := mockTransport.getWriteCalls()
	if len(writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(writes))
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	response := resp["response"].(map[string]interface{})
	data := response["response"].(map[string]interface{})

	if data["behavior"] != "allow" {
		t.Errorf("expected behavior 'allow', got %v", data["behavior"])
	}

	updatedInput := data["updatedInput"].(map[string]interface{})
	if updatedInput["safe_mode"] != true {
		t.Errorf("expected safe_mode to be true, got %v", updatedInput["safe_mode"])
	}
}

// TestToolPermissionCallbackWithUpdatedPermissions tests permission updates.
func TestToolPermissionCallbackWithUpdatedPermissions(t *testing.T) {
	mockTransport := newMockTransport()

	updatedPerms := []types.PermissionUpdate{
		{
			Type:        types.PermissionUpdateTypeAddRules,
			Destination: types.PermissionUpdateDestinationPtr(types.PermissionUpdateDestinationSession),
			Rules: []types.PermissionRuleValue{
				{ToolName: "Bash", RuleContent: strPtr("ls")},
			},
		},
	}

	callback := func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		return types.PermissionResultAllow{
			Behavior:           "allow",
			UpdatedPermissions: updatedPerms,
		}, nil
	}

	q := NewQuery(mockTransport, true, callback, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	request := map[string]interface{}{
		"request_id": "req-perms",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "ls"},
		},
	}

	q.handleControlRequest(ctx, request)

	writes := mockTransport.getWriteCalls()
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	response := resp["response"].(map[string]interface{})
	data := response["response"].(map[string]interface{})

	updatedPermsResp, ok := data["updatedPermissions"].([]interface{})
	if !ok {
		t.Fatal("expected updatedPermissions in response")
	}
	if len(updatedPermsResp) != 1 {
		t.Errorf("expected 1 permission update, got %d", len(updatedPermsResp))
	}
}

// TestToolPermissionCallbackDenyWithInterrupt tests deny with interrupt.
func TestToolPermissionCallbackDenyWithInterrupt(t *testing.T) {
	mockTransport := newMockTransport()

	callback := func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		return types.PermissionResultDeny{
			Behavior:  "deny",
			Message:   "Security policy violation",
			Interrupt: true,
		}, nil
	}

	q := NewQuery(mockTransport, true, callback, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	request := map[string]interface{}{
		"request_id": "req-deny",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "DangerousTool",
			"input":     map[string]interface{}{"command": "rm -rf /"},
		},
	}

	q.handleControlRequest(ctx, request)

	writes := mockTransport.getWriteCalls()
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	response := resp["response"].(map[string]interface{})
	data := response["response"].(map[string]interface{})

	if data["behavior"] != "deny" {
		t.Errorf("expected behavior 'deny', got %v", data["behavior"])
	}
	if data["message"] != "Security policy violation" {
		t.Errorf("expected message 'Security policy violation', got %v", data["message"])
	}
	if data["interrupt"] != true {
		t.Errorf("expected interrupt to be true, got %v", data["interrupt"])
	}
}

// TestToolPermissionCallbackExceptionHandling tests callback error handling.
func TestToolPermissionCallbackExceptionHandling(t *testing.T) {
	mockTransport := newMockTransport()

	callback := func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		return nil, errors.New("callback error")
	}

	q := NewQuery(mockTransport, true, callback, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	request := map[string]interface{}{
		"request_id": "req-error",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "TestTool",
			"input":     map[string]interface{}{},
		},
	}

	q.handleControlRequest(ctx, request)

	writes := mockTransport.getWriteCalls()
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	response := resp["response"].(map[string]interface{})
	if response["subtype"] != "error" {
		t.Errorf("expected subtype 'error', got %v", response["subtype"])
	}
	if !containsSubstring(response["error"].(string), "callback error") {
		t.Errorf("expected error to contain 'callback error', got %v", response["error"])
	}
}

// TestToolPermissionCallbackWithSuggestions tests handling permission suggestions.
func TestToolPermissionCallbackWithSuggestions(t *testing.T) {
	mockTransport := newMockTransport()

	var receivedSuggestions []types.PermissionUpdate
	callback := func(ctx context.Context, toolName string, input map[string]interface{}, permContext types.ToolPermissionContext) (types.PermissionResult, error) {
		receivedSuggestions = permContext.Suggestions
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}

	q := NewQuery(mockTransport, true, callback, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	request := map[string]interface{}{
		"request_id": "req-suggestions",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "ls"},
			"permission_suggestions": []interface{}{
				map[string]interface{}{
					"type":  "addRules",
					"rules": []interface{}{map[string]interface{}{"toolName": "Bash"}},
				},
			},
		},
	}

	q.handleControlRequest(ctx, request)

	if len(receivedSuggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(receivedSuggestions))
	}
	if receivedSuggestions[0].Type != types.PermissionUpdateTypeAddRules {
		t.Errorf("expected type 'addRules', got %v", receivedSuggestions[0].Type)
	}
}

// TestToolPermissionCallbackMissing tests missing callback.
func TestToolPermissionCallbackMissing(t *testing.T) {
	mockTransport := newMockTransport()

	// No callback provided
	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	request := map[string]interface{}{
		"request_id": "req-no-callback",
		"request": map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "TestTool",
			"input":     map[string]interface{}{},
		},
	}

	q.handleControlRequest(ctx, request)

	writes := mockTransport.getWriteCalls()
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(writes[0]), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	response := resp["response"].(map[string]interface{})
	if response["subtype"] != "error" {
		t.Errorf("expected error response, got %v", response["subtype"])
	}
}

// TestFirstResultEventSetOnEarlyExit tests that firstResultEvent is set when
// readMessages exits early (e.g., due to context cancellation), matching
// upstream SDK's finally block behavior.
func TestFirstResultEventSetOnEarlyExit(t *testing.T) {
	mockTransport := newMockTransport()

	// Create hooks to trigger bidirectional mode
	hooks := map[string][]HookMatcher{
		"PreToolUse": {{Matcher: "Bash", Hooks: []HookCallbackFunc{func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		}}}},
	}

	q := NewQuery(mockTransport, true, nil, hooks, nil, 30*time.Second, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	_ = q.Start(ctx)

	// Create input stream
	inputChan := make(chan map[string]interface{})

	var streamErr error
	done := make(chan struct{})

	go func() {
		streamErr = q.StreamInput(ctx, inputChan)
		close(done)
	}()

	// Cancel context before sending any result message
	// This simulates early exit
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Close input
	close(inputChan)

	select {
	case <-done:
		// Stream finished - this means firstResultEvent was set by the defer
		// in readMessages, allowing finishInputStream to proceed
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stream to finish - firstResultEvent may not have been set on early exit")
	}

	if streamErr != nil && streamErr != context.Canceled {
		t.Errorf("unexpected error: %v", streamErr)
	}

	_ = q.Close(context.Background())
}

// TestFirstResultEventNotDoubleClose tests that closing firstResultEvent twice
// doesn't cause a panic.
func TestFirstResultEventNotDoubleClose(t *testing.T) {
	mockTransport := newMockTransport()

	q := NewQuery(mockTransport, true, nil, nil, nil, 30*time.Second, nil, nil)

	ctx := context.Background()
	_ = q.Start(ctx)

	// Send a result message first (this closes firstResultEvent)
	mockTransport.sendMessage(map[string]interface{}{"type": "result", "subtype": "success"})
	time.Sleep(50 * time.Millisecond)

	// Now close the query - this should trigger the defer in readMessages
	// which tries to close firstResultEvent again, but should handle it gracefully
	err := q.Close(ctx)
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}
