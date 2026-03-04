// Package query provides internal query implementation for the Claude Agent SDK.
//
// This package is internal and not intended for direct use by SDK consumers.
// It handles the bidirectional control protocol on top of the transport layer.
package query

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// Transport defines the interface for bidirectional communication with Claude CLI.
type Transport interface {
	// Write writes raw data to the transport.
	Write(ctx context.Context, data string) error
	// EndInput closes the input stream (stdin).
	EndInput(ctx context.Context) error
	// ReadMessages returns a channel for receiving parsed messages.
	ReadMessages(ctx context.Context) <-chan map[string]interface{}
	// Close closes the transport and releases resources.
	Close(ctx context.Context) error
}

// CanUseToolCallback is the callback function type for tool permission requests.
type CanUseToolCallback func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error)

// HookCallbackFunc is the function signature for hook callbacks.
type HookCallbackFunc func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error)

// HookMatcher represents hook matcher configuration for Go.
type HookMatcher struct {
	Matcher string
	Hooks   []HookCallbackFunc
	Timeout *float64
}

// McpServer is the interface for SDK MCP servers.
type McpServer interface {
	// Name returns the server name.
	Name() string
	// Version returns the server version.
	Version() string
	// HandleRequest handles an MCP request.
	HandleRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error)
}

// Query handles bidirectional control protocol on top of Transport.
//
// This class manages:
// - Control request/response routing
// - Hook callbacks
// - Tool permission callbacks
// - Message streaming
// - Initialization handshake
type Query struct {
	transport         Transport
	isStreamingMode   bool
	canUseTool        CanUseToolCallback
	hooks             map[string][]HookMatcher
	sdkMcpServers     map[string]McpServer
	agents            map[string]map[string]interface{}
	initializeTimeout time.Duration

	// Control protocol state
	pendingControlResponses map[string]*pendingResponse
	pendingControlResults   map[string]interface{}
	hookCallbacks           map[string]HookCallbackFunc
	nextCallbackID          int
	requestCounter          int
	mu                      sync.Mutex

	// Message stream
	messageSend    chan map[string]interface{}
	messageReceive chan map[string]interface{}

	// Lifecycle state
	initialized          bool
	closed               bool
	initializationResult map[string]interface{}
	firstResultEvent     chan struct{}
	streamCloseTimeout   time.Duration

	// Context management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// pendingResponse represents a pending control response with synchronization.
type pendingResponse struct {
	done chan struct{}
}

// NewQuery creates a new Query instance.
func NewQuery(
	transport Transport,
	isStreamingMode bool,
	canUseTool CanUseToolCallback,
	hooks map[string][]HookMatcher,
	sdkMcpServers map[string]McpServer,
	initializeTimeout time.Duration,
	agents map[string]map[string]interface{},
) *Query {
	if initializeTimeout == 0 {
		initializeTimeout = 60 * time.Second
	}

	// Get stream close timeout from environment or use default
	streamCloseTimeout := 60 * time.Second
	if timeoutStr := os.Getenv("CLAUDE_CODE_STREAM_CLOSE_TIMEOUT"); timeoutStr != "" {
		if timeoutMs, err := parseDuration(timeoutStr); err == nil {
			streamCloseTimeout = timeoutMs
		}
	}

	return &Query{
		transport:               transport,
		isStreamingMode:         isStreamingMode,
		canUseTool:              canUseTool,
		hooks:                   hooks,
		sdkMcpServers:           sdkMcpServers,
		agents:                  agents,
		initializeTimeout:       initializeTimeout,
		pendingControlResponses: make(map[string]*pendingResponse),
		pendingControlResults:   make(map[string]interface{}),
		hookCallbacks:           make(map[string]HookCallbackFunc),
		messageSend:             make(chan map[string]interface{}, 100),
		messageReceive:          make(chan map[string]interface{}, 100),
		firstResultEvent:        make(chan struct{}),
		streamCloseTimeout:      streamCloseTimeout,
	}
}

// Initialize initializes the control protocol if in streaming mode.
//
// Returns the initialization response with supported commands, or nil if not streaming.
func (q *Query) Initialize(ctx context.Context) (map[string]interface{}, error) {
	if !q.isStreamingMode {
		return nil, nil
	}

	// Build hooks configuration for initialization
	hooksConfig := make(map[string]interface{})
	if q.hooks != nil {
		for event, matchers := range q.hooks {
			if len(matchers) == 0 {
				continue
			}

			hookMatchers := make([]map[string]interface{}, 0, len(matchers))
			for _, matcher := range matchers {
				q.mu.Lock()
				callbackIDs := make([]string, 0, len(matcher.Hooks))
				for _, callback := range matcher.Hooks {
					callbackID := fmt.Sprintf("hook_%d", q.nextCallbackID)
					q.nextCallbackID++
					q.hookCallbacks[callbackID] = callback
					callbackIDs = append(callbackIDs, callbackID)
				}
				q.mu.Unlock()

				hookMatcherConfig := map[string]interface{}{
					"matcher":         matcher.Matcher,
					"hookCallbackIds": callbackIDs,
				}
				if matcher.Timeout != nil {
					hookMatcherConfig["timeout"] = *matcher.Timeout
				}
				hookMatchers = append(hookMatchers, hookMatcherConfig)
			}
			hooksConfig[event] = hookMatchers
		}
	}

	// Send initialize request
	request := map[string]interface{}{
		"subtype": "initialize",
	}
	if len(hooksConfig) > 0 {
		request["hooks"] = hooksConfig
	}
	if q.agents != nil {
		request["agents"] = q.agents
	}

	// Use longer timeout for initialize since MCP servers may take time to start
	response, err := q.sendControlRequest(ctx, request, q.initializeTimeout)
	if err != nil {
		return nil, err
	}

	q.mu.Lock()
	q.initialized = true
	q.initializationResult = response
	q.mu.Unlock()

	return response, nil
}

// Start starts reading messages from transport.
func (q *Query) Start(ctx context.Context) error {
	q.ctx, q.cancel = context.WithCancel(ctx)
	q.wg.Add(1)
	go q.readMessages()
	return nil
}

// readMessages reads messages from transport and routes them.
func (q *Query) readMessages() {
	defer q.wg.Done()

	msgChan := q.transport.ReadMessages(q.ctx)

	for {
		select {
		case <-q.ctx.Done():
			return
		case message, ok := <-msgChan:
			if !ok {
				// Channel closed
				close(q.messageSend)
				return
			}

			if q.isClosed() {
				return
			}

			msgType, _ := message["type"].(string)

			// Route control messages
			if msgType == "control_response" {
				response, ok := message["response"].(map[string]interface{})
				if !ok {
					continue
				}
				requestID, _ := response["request_id"].(string)
				if requestID == "" {
					continue
				}

				q.mu.Lock()
				if pending, exists := q.pendingControlResponses[requestID]; exists {
					subtype, _ := response["subtype"].(string)
					if subtype == "error" {
						errMsg, _ := response["error"].(string)
						if errMsg == "" {
							errMsg = "Unknown error"
						}
						q.pendingControlResults[requestID] = fmt.Errorf("%s", errMsg)
					} else {
						q.pendingControlResults[requestID] = response
					}
					close(pending.done)
				}
				q.mu.Unlock()
				continue
			}

			if msgType == "control_request" {
				// Handle incoming control requests from CLI
				q.wg.Add(1)
				go func(req map[string]interface{}) {
					defer q.wg.Done()
					q.handleControlRequest(q.ctx, req)
				}(message)
				continue
			}

			if msgType == "control_cancel_request" {
				// Handle cancel requests
				// TODO: Implement cancellation support
				continue
			}

			// Track results for proper stream closure
			if msgType == "result" {
				select {
				case <-q.firstResultEvent:
					// Already closed
				default:
					close(q.firstResultEvent)
				}
			}

			// Regular SDK messages go to the stream
			select {
			case q.messageSend <- message:
			case <-q.ctx.Done():
				return
			}
		}
	}
}

// handleControlRequest handles incoming control request from CLI.
func (q *Query) handleControlRequest(ctx context.Context, request map[string]interface{}) {
	requestID, _ := request["request_id"].(string)
	requestData, _ := request["request"].(map[string]interface{})
	if requestData == nil {
		return
	}
	subtype, _ := requestData["subtype"].(string)

	var responseData map[string]interface{}
	var err error

	switch subtype {
	case "can_use_tool":
		responseData, err = q.handleToolPermissionRequest(ctx, requestID, requestData)
	case "hook_callback":
		responseData, err = q.handleHookCallbackRequest(ctx, requestID, requestData)
	case "mcp_message":
		responseData, err = q.handleMCPMessageRequest(ctx, requestID, requestData)
	default:
		err = fmt.Errorf("unsupported control request subtype: %s", subtype)
	}

	// Send response
	var response map[string]interface{}
	if err != nil {
		response = map[string]interface{}{
			"type": "control_response",
			"response": map[string]interface{}{
				"subtype":    "error",
				"request_id": requestID,
				"error":      err.Error(),
			},
		}
	} else {
		response = map[string]interface{}{
			"type": "control_response",
			"response": map[string]interface{}{
				"subtype":    "success",
				"request_id": requestID,
				"response":   responseData,
			},
		}
	}

	data, _ := json.Marshal(response)
	if writeErr := q.transport.Write(ctx, string(data)+"\n"); writeErr != nil {
		// Log error but don't fail
		fmt.Fprintf(os.Stderr, "Error writing control response: %v\n", writeErr)
	}
}

// handleToolPermissionRequest handles tool permission requests.
func (q *Query) handleToolPermissionRequest(ctx context.Context, requestID string, requestData map[string]interface{}) (map[string]interface{}, error) {
	if q.canUseTool == nil {
		return nil, fmt.Errorf("canUseTool callback is not provided")
	}

	toolName, _ := requestData["tool_name"].(string)
	input, _ := requestData["input"].(map[string]interface{})
	if input == nil {
		input = make(map[string]interface{})
	}

	// Extract permission suggestions
	suggestions := make([]types.PermissionUpdate, 0)
	if rawSuggestions, ok := requestData["permission_suggestions"].([]interface{}); ok {
		for _, s := range rawSuggestions {
			if suggestion, ok := s.(map[string]interface{}); ok {
				// Parse permission update from suggestion
				permUpdate := parsePermissionUpdate(suggestion)
				suggestions = append(suggestions, permUpdate)
			}
		}
	}

	context := types.ToolPermissionContext{
		Signal:      nil, // TODO: Add context cancellation support
		Suggestions: suggestions,
	}

	result, err := q.canUseTool(ctx, toolName, input, context)
	if err != nil {
		return nil, err
	}

	// Convert PermissionResult to expected dict format
	switch r := result.(type) {
	case types.PermissionResultAllow:
		response := map[string]interface{}{
			"behavior":     "allow",
			"updatedInput": input,
		}
		if r.UpdatedInput != nil {
			response["updatedInput"] = r.UpdatedInput
		}
		if r.UpdatedPermissions != nil {
			updatedPerms := make([]map[string]interface{}, 0, len(r.UpdatedPermissions))
			for _, perm := range r.UpdatedPermissions {
				updatedPerms = append(updatedPerms, perm.ToDict())
			}
			response["updatedPermissions"] = updatedPerms
		}
		return response, nil
	case types.PermissionResultDeny:
		response := map[string]interface{}{
			"behavior":  "deny",
			"message":   r.Message,
			"interrupt": r.Interrupt,
		}
		return response, nil
	default:
		return nil, fmt.Errorf("tool permission callback must return PermissionResult (PermissionResultAllow or PermissionResultDeny), got %T", result)
	}
}

// handleHookCallbackRequest handles hook callback requests.
func (q *Query) handleHookCallbackRequest(ctx context.Context, requestID string, requestData map[string]interface{}) (map[string]interface{}, error) {
	callbackID, _ := requestData["callback_id"].(string)

	q.mu.Lock()
	callback, exists := q.hookCallbacks[callbackID]
	q.mu.Unlock()

	if !exists {
		return nil, fmt.Errorf("no hook callback found for ID: %s", callbackID)
	}

	input := requestData["input"]
	toolUseID, _ := requestData["tool_use_id"].(string)
	var toolUseIDPtr *string
	if toolUseID != "" {
		toolUseIDPtr = &toolUseID
	}

	hookContext := types.HookContext{
		Signal: nil, // TODO: Add abort signal support
	}

	hookOutput, err := callback(ctx, input, toolUseIDPtr, hookContext)
	if err != nil {
		return nil, err
	}

	// Convert Python-safe field names (async_, continue_) to CLI-expected names (async, continue)
	return convertHookOutputForCLI(hookOutput), nil
}

// handleMCPMessageRequest handles MCP message requests for SDK servers.
func (q *Query) handleMCPMessageRequest(ctx context.Context, requestID string, requestData map[string]interface{}) (map[string]interface{}, error) {
	serverName, _ := requestData["server_name"].(string)
	mcpMessage, _ := requestData["message"].(map[string]interface{})

	if serverName == "" || mcpMessage == nil {
		return nil, fmt.Errorf("missing server_name or message for MCP request")
	}

	mcpResponse, err := q.handleSDKMCPRequest(ctx, serverName, mcpMessage)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"mcp_response": mcpResponse,
	}, nil
}

// handleSDKMCPRequest handles an MCP request for an SDK server.
func (q *Query) handleSDKMCPRequest(ctx context.Context, serverName string, message map[string]interface{}) (map[string]interface{}, error) {
	server, exists := q.sdkMcpServers[serverName]
	if !exists {
		msgID, _ := message["id"]
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      msgID,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": fmt.Sprintf("Server '%s' not found", serverName),
			},
		}, nil
	}

	method, _ := message["method"].(string)
	params, _ := message["params"].(map[string]interface{})
	if params == nil {
		params = make(map[string]interface{})
	}
	msgID := message["id"]

	// Handle MCP protocol methods
	switch method {
	case "initialize":
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      msgID,
			"result": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    server.Name(),
					"version": server.Version(),
				},
			},
		}, nil
	case "tools/list", "tools/call":
		// Delegate to server implementation
		result, err := server.HandleRequest(ctx, method, params)
		if err != nil {
			return map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      msgID,
				"error": map[string]interface{}{
					"code":    -32603,
					"message": err.Error(),
				},
			}, nil
		}
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      msgID,
			"result":  result,
		}, nil
	case "notifications/initialized":
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"result":  map[string]interface{}{},
		}, nil
	default:
		return map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      msgID,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": fmt.Sprintf("Method '%s' not found", method),
			},
		}, nil
	}
}

// sendControlRequest sends a control request to CLI and waits for response.
func (q *Query) sendControlRequest(ctx context.Context, request map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	if !q.isStreamingMode {
		return nil, fmt.Errorf("control requests require streaming mode")
	}

	// Generate unique request ID
	q.mu.Lock()
	q.requestCounter++
	counter := q.requestCounter
	q.mu.Unlock()

	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	requestID := fmt.Sprintf("req_%d_%s", counter, hex.EncodeToString(randomBytes))

	// Create channel for response
	done := make(chan struct{})
	q.mu.Lock()
	q.pendingControlResponses[requestID] = &pendingResponse{done: done}
	q.mu.Unlock()

	// Cleanup on exit
	defer func() {
		q.mu.Lock()
		delete(q.pendingControlResponses, requestID)
		delete(q.pendingControlResults, requestID)
		q.mu.Unlock()
	}()

	// Build and send request
	controlRequest := map[string]interface{}{
		"type":       "control_request",
		"request_id": requestID,
		"request":    request,
	}

	data, err := json.Marshal(controlRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal control request: %w", err)
	}

	if err := q.transport.Write(ctx, string(data)+"\n"); err != nil {
		return nil, fmt.Errorf("failed to write control request: %w", err)
	}

	// Wait for response with timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("control request timeout: %s", request["subtype"])
	case <-done:
		q.mu.Lock()
		result := q.pendingControlResults[requestID]
		q.mu.Unlock()

		if err, ok := result.(error); ok {
			return nil, err
		}

		if response, ok := result.(map[string]interface{}); ok {
			responseData, _ := response["response"].(map[string]interface{})
			return responseData, nil
		}
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}
}

// GetMCPStatus gets current MCP server connection status.
func (q *Query) GetMCPStatus(ctx context.Context) (map[string]interface{}, error) {
	return q.sendControlRequest(ctx, map[string]interface{}{"subtype": "mcp_status"}, 60*time.Second)
}

// Interrupt sends an interrupt control request.
func (q *Query) Interrupt(ctx context.Context) error {
	_, err := q.sendControlRequest(ctx, map[string]interface{}{"subtype": "interrupt"}, 60*time.Second)
	return err
}

// SetPermissionMode changes the permission mode.
func (q *Query) SetPermissionMode(ctx context.Context, mode string) error {
	_, err := q.sendControlRequest(ctx, map[string]interface{}{
		"subtype": "set_permission_mode",
		"mode":    mode,
	}, 60*time.Second)
	return err
}

// SetModel changes the AI model.
func (q *Query) SetModel(ctx context.Context, model string) error {
	_, err := q.sendControlRequest(ctx, map[string]interface{}{
		"subtype": "set_model",
		"model":   model,
	}, 60*time.Second)
	return err
}

// RewindFiles rewinds tracked files to their state at a specific user message.
//
// Requires file checkpointing to be enabled via the enable_file_checkpointing option.
func (q *Query) RewindFiles(ctx context.Context, userMessageID string) error {
	_, err := q.sendControlRequest(ctx, map[string]interface{}{
		"subtype":         "rewind_files",
		"user_message_id": userMessageID,
	}, 60*time.Second)
	return err
}

// ReconnectMCPServer reconnects a disconnected or failed MCP server.
func (q *Query) ReconnectMCPServer(ctx context.Context, serverName string) error {
	_, err := q.sendControlRequest(ctx, map[string]interface{}{
		"subtype":    "mcp_reconnect",
		"serverName": serverName,
	}, 60*time.Second)
	return err
}

// ToggleMCPServer enables or disables an MCP server.
func (q *Query) ToggleMCPServer(ctx context.Context, serverName string, enabled bool) error {
	_, err := q.sendControlRequest(ctx, map[string]interface{}{
		"subtype":    "mcp_toggle",
		"serverName": serverName,
		"enabled":    enabled,
	}, 60*time.Second)
	return err
}

// StopTask stops a running task.
func (q *Query) StopTask(ctx context.Context, taskID string) error {
	_, err := q.sendControlRequest(ctx, map[string]interface{}{
		"subtype": "stop_task",
		"task_id": taskID,
	}, 60*time.Second)
	return err
}

// StreamInput streams input messages to transport.
//
// If SDK MCP servers or hooks are present, waits for the first result
// before closing stdin to allow bidirectional control protocol communication.
func (q *Query) StreamInput(ctx context.Context, stream <-chan map[string]interface{}) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message, ok := <-stream:
			if !ok {
				// Stream closed, handle cleanup
				return q.finishInputStream(ctx)
			}

			if q.isClosed() {
				return nil
			}

			data, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			if err := q.transport.Write(ctx, string(data)+"\n"); err != nil {
				return fmt.Errorf("failed to write message: %w", err)
			}
		}
	}
}

// finishInputStream handles cleanup after input stream ends.
func (q *Query) finishInputStream(ctx context.Context) error {
	// If we have SDK MCP servers or hooks that need bidirectional communication,
	// wait for first result before closing the channel
	hasHooks := q.hooks != nil && len(q.hooks) > 0
	if len(q.sdkMcpServers) > 0 || hasHooks {
		select {
		case <-q.firstResultEvent:
			// Received first result
		case <-time.After(q.streamCloseTimeout):
			// Timed out waiting for first result
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// After all messages sent (and result received if needed), end input
	return q.transport.EndInput(ctx)
}

// ReceiveMessages returns a channel for receiving SDK messages (not control messages).
func (q *Query) ReceiveMessages(ctx context.Context) <-chan map[string]interface{} {
	output := make(chan map[string]interface{})

	go func() {
		defer close(output)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-q.messageSend:
				if !ok {
					return
				}

				// Check for special messages
				msgType, _ := msg["type"].(string)
				if msgType == "end" {
					return
				}
				if msgType == "error" {
					errMsg, _ := msg["error"].(string)
					if errMsg == "" {
						errMsg = "Unknown error"
					}
					// Send error as message, receiver can handle it
					output <- map[string]interface{}{
						"type":  "error",
						"error": errMsg,
					}
					return
				}

				output <- msg
			}
		}
	}()

	return output
}

// Close closes the query and transport.
func (q *Query) Close(ctx context.Context) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return nil
	}
	q.closed = true
	q.mu.Unlock()

	// Cancel context
	if q.cancel != nil {
		q.cancel()
	}

	// Wait for goroutines to finish
	q.wg.Wait()

	// Close transport
	return q.transport.Close(ctx)
}

// isClosed returns whether the query is closed.
func (q *Query) isClosed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.closed
}

// GetInitializationResult returns the initialization result from the CLI.
func (q *Query) GetInitializationResult() map[string]interface{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.initializationResult
}

// convertHookOutputForCLI converts Go-safe field names to CLI-expected field names.
//
// The Go SDK uses Async_ and Continue_ to avoid keyword conflicts,
// but the CLI expects async and continue. This function performs the
// necessary conversion.
func convertHookOutputForCLI(hookOutput map[string]interface{}) map[string]interface{} {
	converted := make(map[string]interface{})
	for key, value := range hookOutput {
		// Convert Go-safe names to JavaScript names
		switch key {
		case "Async_", "async_":
			converted["async"] = value
		case "Continue_", "continue_":
			converted["continue"] = value
		default:
			converted[key] = value
		}
	}
	return converted
}

// parsePermissionUpdate parses a permission update from a map.
func parsePermissionUpdate(data map[string]interface{}) types.PermissionUpdate {
	update := types.PermissionUpdate{}

	if t, ok := data["type"].(string); ok {
		update.Type = types.PermissionUpdateType(t)
	}

	if dest, ok := data["destination"].(string); ok {
		update.Destination = types.PermissionUpdateDestinationPtr(types.PermissionUpdateDestination(dest))
	}

	if rules, ok := data["rules"].([]interface{}); ok {
		for _, r := range rules {
			if ruleMap, ok := r.(map[string]interface{}); ok {
				rule := types.PermissionRuleValue{}
				if toolName, ok := ruleMap["toolName"].(string); ok {
					rule.ToolName = toolName
				}
				if ruleContent, ok := ruleMap["ruleContent"].(string); ok {
					rule.RuleContent = &ruleContent
				}
				update.Rules = append(update.Rules, rule)
			}
		}
	}

	if behavior, ok := data["behavior"].(string); ok {
		b := types.PermissionBehavior(behavior)
		update.Behavior = &b
	}

	if mode, ok := data["mode"].(string); ok {
		m := types.PermissionMode(mode)
		update.Mode = &m
	}

	if dirs, ok := data["directories"].([]interface{}); ok {
		for _, d := range dirs {
			if dir, ok := d.(string); ok {
				update.Directories = append(update.Directories, dir)
			}
		}
	}

	return update
}

// parseDuration parses a duration string (in milliseconds) to time.Duration.
func parseDuration(s string) (time.Duration, error) {
	var ms int
	if _, err := fmt.Sscanf(s, "%d", &ms); err != nil {
		return 0, err
	}
	return time.Duration(ms) * time.Millisecond, nil
}
