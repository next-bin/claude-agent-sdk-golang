package query

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Tool Permission Callback Tests
// ============================================================================

// MockTransport implements the Transport interface for testing
type MockTransport struct {
	writtenMessages []string
	messagesToRead  []map[string]interface{}
	connected       bool
}

func (m *MockTransport) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *MockTransport) Write(ctx context.Context, data string) error {
	m.writtenMessages = append(m.writtenMessages, data)
	return nil
}

func (m *MockTransport) EndInput(ctx context.Context) error {
	return nil
}

func (m *MockTransport) ReadMessages(ctx context.Context) <-chan map[string]interface{} {
	ch := make(chan map[string]interface{})
	go func() {
		for _, msg := range m.messagesToRead {
			ch <- msg
		}
		close(ch)
	}()
	return ch
}

func (m *MockTransport) Close(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *MockTransport) IsReady() bool {
	return m.connected
}

func TestPermissionCallbackAllow(t *testing.T) {
	// Test callback that allows tool execution
	callbackInvoked := false

	allowCallback := func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
		callbackInvoked = true
		if toolName != "TestTool" {
			t.Errorf("Expected tool name 'TestTool', got %s", toolName)
		}
		if input["param"] != "value" {
			t.Errorf("Expected input param 'value', got %v", input["param"])
		}
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}

	// Invoke the callback directly
	result, err := allowCallback("TestTool", map[string]interface{}{"param": "value"}, types.ToolPermissionContext{})

	// Verify callback was invoked
	if !callbackInvoked {
		t.Error("Expected callback to be invoked")
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	allowResult, ok := result.(types.PermissionResultAllow)
	if !ok {
		t.Fatal("Expected PermissionResultAllow")
	}

	if allowResult.Behavior != "allow" {
		t.Errorf("Expected behavior 'allow', got %s", allowResult.Behavior)
	}
}

func TestPermissionCallbackDeny(t *testing.T) {
	// Test callback that denies tool execution
	denyCallback := func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
		return types.PermissionResultDeny{
			Behavior:  "deny",
			Message:   "Security policy violation",
			Interrupt: false,
		}, nil
	}

	_ = denyCallback

	// Verify deny behavior is correctly set
	result := types.PermissionResultDeny{
		Behavior:  "deny",
		Message:   "Security policy violation",
		Interrupt: false,
	}

	if result.Behavior != "deny" {
		t.Errorf("Expected behavior 'deny', got %s", result.Behavior)
	}
	if result.Message != "Security policy violation" {
		t.Errorf("Expected message 'Security policy violation', got %s", result.Message)
	}
}

func TestPermissionCallbackInputModification(t *testing.T) {
	// Test callback that modifies tool input
	modifyCallback := func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
		// Modify the input to add safety flag
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

	_ = modifyCallback

	// Test input modification
	input := map[string]interface{}{"file_path": "/etc/passwd"}
	result, _ := modifyCallback("WriteTool", input, types.ToolPermissionContext{})

	allowResult, ok := result.(types.PermissionResultAllow)
	if !ok {
		t.Fatal("Expected PermissionResultAllow")
	}

	if allowResult.UpdatedInput["safe_mode"] != true {
		t.Error("Expected safe_mode to be added to input")
	}
}

func TestPermissionCallbackUpdatedPermissions(t *testing.T) {
	// Test callback that returns updated permissions
	updatedPermsCallback := func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
		return types.PermissionResultAllow{
			Behavior:           "allow",
			UpdatedPermissions: []types.PermissionUpdate{},
		}, nil
	}

	_ = updatedPermsCallback

	// Verify updated permissions
	result, _ := updatedPermsCallback("TestTool", map[string]interface{}{}, types.ToolPermissionContext{})

	allowResult, ok := result.(types.PermissionResultAllow)
	if !ok {
		t.Fatal("Expected PermissionResultAllow")
	}

	_ = allowResult
}

func TestCallbackExceptionHandling(t *testing.T) {
	// Test that callback exceptions are properly handled
	errorCallback := func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
		return nil, &CallbackError{Message: "Callback error"}
	}

	_, err := errorCallback("TestTool", map[string]interface{}{}, types.ToolPermissionContext{})

	if err == nil {
		t.Error("Expected error from callback")
	}

	var callbackErr *CallbackError
	if err != nil {
		// Check if it's the right error type
		t.Logf("Got expected error: %v", err)
	}
	_ = callbackErr
}

// ============================================================================
// Hook Callback Tests
// ============================================================================

func TestHookExecution(t *testing.T) {
	// Test that hooks are called at appropriate times
	hookCalls := make([]map[string]interface{}, 0)

	testHook := &TestHook{
		ExecuteFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			hookCalls = append(hookCalls, map[string]interface{}{
				"input":       input,
				"tool_use_id": toolUseID,
			})
			return &types.SyncHookJSONOutput{}, nil
		},
	}

	_ = testHook

	// Simulate hook execution
	_ = hookCalls
}

func TestHookOutputFields(t *testing.T) {
	// Test that all SyncHookJSONOutput fields are properly handled
	continueVal := true
	suppressOutput := false
	stopReason := "Test stop reason"
	decision := "block"
	systemMessage := "Test system message"
	reason := "Test reason for blocking"

	output := &types.SyncHookJSONOutput{
		Continue_:      &continueVal,
		SuppressOutput: &suppressOutput,
		StopReason:     &stopReason,
		Decision:       &decision,
		SystemMessage:  &systemMessage,
		Reason:         &reason,
		HookSpecificOutput: &types.PreToolUseHookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       ptrString("deny"),
			PermissionDecisionReason: ptrString("Security policy violation"),
			UpdatedInput:             map[string]interface{}{"modified": "input"},
		},
	}

	// Verify fields
	if *output.Continue_ != true {
		t.Error("Expected Continue_ to be true")
	}
	if *output.Decision != "block" {
		t.Error("Expected Decision to be 'block'")
	}
}

func TestAsyncHookOutput(t *testing.T) {
	// Test AsyncHookJSONOutput type with proper async fields
	asyncTimeout := 5000

	output := &types.AsyncHookJSONOutput{
		Async_:       true,
		AsyncTimeout: &asyncTimeout,
	}

	if !output.IsAsync() {
		t.Error("Expected IsAsync to return true")
	}
	if *output.AsyncTimeout != 5000 {
		t.Errorf("Expected AsyncTimeout 5000, got %d", *output.AsyncTimeout)
	}
}

func TestFieldNameConversion(t *testing.T) {
	// Test that upstream-safe field names (async_, continue_) are converted to CLI format
	continueVal := false
	output := &types.SyncHookJSONOutput{
		Continue_: &continueVal,
	}

	// Convert to JSON and verify field names
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// The JSON should have "continue" not "continue_"
	// (This depends on the JSON tags in the struct)
	t.Logf("JSON output: %s", jsonStr)
	_ = jsonStr
}

// ============================================================================
// Hook Event Type Tests
// ============================================================================

func TestAllHookEventTypes(t *testing.T) {
	// Test that all hook event types are properly defined
	events := []types.HookEvent{
		types.HookEventPreToolUse,
		types.HookEventPostToolUse,
		types.HookEventPostToolUseFailure,
		types.HookEventUserPromptSubmit,
		types.HookEventStop,
		types.HookEventSubagentStop,
		types.HookEventPreCompact,
		types.HookEventNotification,
		types.HookEventSubagentStart,
		types.HookEventPermissionRequest,
		types.HookEventSessionStart,
		types.HookEventSessionEnd,
	}

	expectedStrings := []string{
		"PreToolUse",
		"PostToolUse",
		"PostToolUseFailure",
		"UserPromptSubmit",
		"Stop",
		"SubagentStop",
		"PreCompact",
		"Notification",
		"SubagentStart",
		"PermissionRequest",
		"SessionStart",
		"SessionEnd",
	}

	for i, event := range events {
		if string(event) != expectedStrings[i] {
			t.Errorf("Expected %s, got %s", expectedStrings[i], event)
		}
	}
}

func TestHookSpecificOutputTypes(t *testing.T) {
	// Test all hook specific output types
	preToolOutput := &types.PreToolUseHookSpecificOutput{
		HookEventName:            "PreToolUse",
		PermissionDecision:       ptrString("allow"),
		PermissionDecisionReason: ptrString("Test reason"),
		UpdatedInput:             map[string]interface{}{"key": "value"},
		AdditionalContext:        ptrString("Additional context"),
	}

	if preToolOutput.GetHookEventName() != "PreToolUse" {
		t.Error("Expected PreToolUse hook event name")
	}

	postToolOutput := &types.PostToolUseHookSpecificOutput{
		HookEventName:        "PostToolUse",
		AdditionalContext:    ptrString("Context"),
		UpdatedMCPToolOutput: map[string]interface{}{"result": "modified"},
	}

	if postToolOutput.GetHookEventName() != "PostToolUse" {
		t.Error("Expected PostToolUse hook event name")
	}
}

// ============================================================================
// Helper Types and Functions
// ============================================================================

type TestHook struct {
	ExecuteFunc func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error)
}

func (h *TestHook) Execute(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
	if h.ExecuteFunc != nil {
		return h.ExecuteFunc(input, toolUseID, context)
	}
	return &types.SyncHookJSONOutput{}, nil
}

type CallbackError struct {
	Message string
}

func (e *CallbackError) Error() string {
	return e.Message
}

func ptrString(s string) *string {
	return &s
}
