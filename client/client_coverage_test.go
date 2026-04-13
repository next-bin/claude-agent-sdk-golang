// Package client provides comprehensive unit tests for client package.
package client

import (
	"context"
	"testing"
	"time"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Client Creation Tests
// ============================================================================

func TestNewCoverage(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
	defer c.Close()

	opts := c.Options()
	if opts == nil {
		t.Fatal("Options() returned nil")
	}
}

func TestNewWithOptionsCoverage(t *testing.T) {
	opts := &types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	}
	c := NewWithOptions(opts)
	if c == nil {
		t.Fatal("NewWithOptions returned nil")
	}
	defer c.Close()

	retOpts := c.Options()
	if retOpts.Model == nil || *retOpts.Model != types.ModelSonnet {
		t.Errorf("Options().Model mismatch")
	}
}

func TestNewWithOptionsNilCoverage(t *testing.T) {
	c := NewWithOptions(nil)
	if c == nil {
		t.Fatal("NewWithOptions(nil) returned nil")
	}
	defer c.Close()

	opts := c.Options()
	if opts == nil {
		t.Fatal("Options() returned nil")
	}
}

func TestOptionsCoverage(t *testing.T) {
	systemPrompt := "test prompt"
	opts := &types.ClaudeAgentOptions{
		SystemPrompt: systemPrompt,
	}
	c := NewWithOptions(opts)
	defer c.Close()

	retOpts := c.Options()
	if retOpts.SystemPrompt != systemPrompt {
		t.Errorf("Options().SystemPrompt = %s, want %s", retOpts.SystemPrompt, systemPrompt)
	}
}

// ============================================================================
// Connect Error Tests
// ============================================================================

func TestConnectTwiceCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	// First connect returns nil
	err := c.Connect(context.Background())
	if err != nil {
		t.Logf("First connect error (expected in test env): %v", err)
	}

	// Second connect should also return nil (already connected)
	err2 := c.Connect(context.Background())
	if err2 != nil {
		t.Errorf("Second connect should return nil, got: %v", err2)
	}
}

func TestConnectCanUseToolWithStringPromptCoverage(t *testing.T) {
	canUseTool := func(toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}

	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model:      types.String(types.ModelSonnet),
		CanUseTool: canUseTool,
	})
	defer c.Close()

	// Connect with string prompt should fail
	err := c.Connect(context.Background(), "test prompt")
	if err == nil {
		t.Error("Connect with string prompt and CanUseTool should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestConnectCanUseToolWithPermissionPromptToolNameCoverage(t *testing.T) {
	canUseTool := func(toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}
	permPromptTool := "test-tool"

	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model:                    types.String(types.ModelSonnet),
		CanUseTool:               canUseTool,
		PermissionPromptToolName: &permPromptTool,
	})
	defer c.Close()

	// Connect should fail with mutually exclusive options
	err := c.Connect(context.Background())
	if err == nil {
		t.Error("Connect with CanUseTool and PermissionPromptToolName should fail")
	}
	t.Logf("Expected error: %v", err)
}

// ============================================================================
// Hook Parsing Tests
// ============================================================================

func TestParsePreToolUseHookInputCoverage(t *testing.T) {
	m := map[string]interface{}{
		"tool_name":       "Bash",
		"tool_input":      map[string]interface{}{"command": "ls"},
		"tool_use_id":     "abc123",
		"session_id":      "sess1",
		"transcript_path": "/path/to/transcript",
		"cwd":             "/home/user",
		"permission_mode": "acceptEdits",
		"agent_id":        "agent1",
		"agent_type":      "general-purpose",
	}

	input := parsePreToolUseHookInput(m)

	if input.ToolName != "Bash" {
		t.Errorf("ToolName = %s, want Bash", input.ToolName)
	}
	if input.ToolInput["command"] != "ls" {
		t.Errorf("ToolInput mismatch")
	}
	if input.ToolUseID != "abc123" {
		t.Errorf("ToolUseID = %s, want abc123", input.ToolUseID)
	}
	if input.SessionID != "sess1" {
		t.Errorf("SessionID = %s, want sess1", input.SessionID)
	}
	if input.TranscriptPath != "/path/to/transcript" {
		t.Errorf("TranscriptPath mismatch")
	}
	if input.CWD != "/home/user" {
		t.Errorf("CWD mismatch")
	}
	if input.PermissionMode == nil || *input.PermissionMode != "acceptEdits" {
		t.Errorf("PermissionMode mismatch")
	}
	if input.AgentID == nil || *input.AgentID != "agent1" {
		t.Errorf("AgentID mismatch")
	}
	if input.AgentType == nil || *input.AgentType != "general-purpose" {
		t.Errorf("AgentType mismatch")
	}
}

func TestParsePostToolUseHookInputCoverage(t *testing.T) {
	m := map[string]interface{}{
		"tool_name":     "Read",
		"tool_input":    map[string]interface{}{"file_path": "/test"},
		"tool_response": "file content",
		"tool_use_id":   "xyz789",
		"agent_id":      "agent2",
		"agent_type":    "plan",
	}

	input := parsePostToolUseHookInput(m)

	if input.ToolName != "Read" {
		t.Errorf("ToolName = %s, want Read", input.ToolName)
	}
	if input.ToolResponse != "file content" {
		t.Errorf("ToolResponse mismatch")
	}
	if input.AgentID == nil || *input.AgentID != "agent2" {
		t.Errorf("AgentID mismatch")
	}
	if input.AgentType == nil || *input.AgentType != "plan" {
		t.Errorf("AgentType mismatch")
	}
}

func TestParsePostToolUseFailureHookInputCoverage(t *testing.T) {
	m := map[string]interface{}{
		"tool_name":   "Write",
		"tool_input":  map[string]interface{}{"file_path": "/test"},
		"tool_use_id": "fail123",
		"error":       "permission denied",
		"agent_id":    "agent3",
		"agent_type":  "test-runner",
	}

	input := parsePostToolUseFailureHookInput(m)

	if input.ToolName != "Write" {
		t.Errorf("ToolName = %s, want Write", input.ToolName)
	}
	if input.Error != "permission denied" {
		t.Errorf("Error = %s, want permission denied", input.Error)
	}
	if input.AgentID == nil || *input.AgentID != "agent3" {
		t.Errorf("AgentID mismatch")
	}
}

func TestParseUserPromptSubmitHookInputCoverage(t *testing.T) {
	m := map[string]interface{}{
		"prompt": "Hello Claude",
	}

	input := parseUserPromptSubmitHookInput(m)

	if input.Prompt != "Hello Claude" {
		t.Errorf("Prompt = %s, want Hello Claude", input.Prompt)
	}
}

func TestParsePermissionRequestHookInputCoverage(t *testing.T) {
	m := map[string]interface{}{
		"tool_name":              "Bash",
		"tool_input":             map[string]interface{}{"command": "rm -rf"},
		"permission_suggestions": []interface{}{"allow", "deny"},
		"agent_id":               "agent4",
		"agent_type":             "general",
	}

	input := parsePermissionRequestHookInput(m)

	if input.ToolName != "Bash" {
		t.Errorf("ToolName = %s, want Bash", input.ToolName)
	}
	if len(input.PermissionSuggestions) != 2 {
		t.Errorf("PermissionSuggestions count = %d, want 2", len(input.PermissionSuggestions))
	}
	if input.AgentID == nil || *input.AgentID != "agent4" {
		t.Errorf("AgentID mismatch")
	}
}

// ============================================================================
// Hook Output Conversion Tests
// ============================================================================

func TestHookOutputToMapSyncCoverage(t *testing.T) {
	continueVal := true
	suppressOutput := false
	stopReason := "user_stop"
	decision := "approve"
	systemMsg := "approved"
	reason := "tool is safe"

	output := types.SyncHookJSONOutput{
		Continue_:      &continueVal,
		SuppressOutput: &suppressOutput,
		StopReason:     &stopReason,
		Decision:       &decision,
		SystemMessage:  &systemMsg,
		Reason:         &reason,
	}

	result := hookOutputToMap(output)

	if result["continue"] != true {
		t.Errorf("continue = %v, want true", result["continue"])
	}
	if result["suppressOutput"] != false {
		t.Errorf("suppressOutput = %v, want false", result["suppressOutput"])
	}
	if result["stopReason"] != "user_stop" {
		t.Errorf("stopReason mismatch")
	}
	if result["decision"] != "approve" {
		t.Errorf("decision mismatch")
	}
	if result["systemMessage"] != "approved" {
		t.Errorf("systemMessage mismatch")
	}
	if result["reason"] != "tool is safe" {
		t.Errorf("reason mismatch")
	}
}

func TestHookOutputToMapAsyncCoverage(t *testing.T) {
	async := true
	asyncTimeout := 5000

	output := types.AsyncHookJSONOutput{
		Async_:       async,
		AsyncTimeout: &asyncTimeout,
	}

	result := hookOutputToMap(output)

	if result["async"] != true {
		t.Errorf("async = %v, want true", result["async"])
	}
	if result["asyncTimeout"] != 5000 {
		t.Errorf("asyncTimeout = %v, want 5000", result["asyncTimeout"])
	}
}

func TestHookSpecificOutputToMapPreToolUseCoverage(t *testing.T) {
	permDecision := "allow"
	permReason := "safe tool"
	updatedInput := map[string]interface{}{"modified": true}
	additionalCtx := "context info"

	output := types.PreToolUseHookSpecificOutput{
		PermissionDecision:       &permDecision,
		PermissionDecisionReason: &permReason,
		UpdatedInput:             updatedInput,
		AdditionalContext:        &additionalCtx,
	}

	result := hookSpecificOutputToMap(output)

	if result["permissionDecision"] != "allow" {
		t.Errorf("permissionDecision mismatch")
	}
	if result["permissionDecisionReason"] != "safe tool" {
		t.Errorf("permissionDecisionReason mismatch")
	}
	if result["updatedInput"] == nil {
		t.Errorf("updatedInput should not be nil")
	}
	if result["additionalContext"] != "context info" {
		t.Errorf("additionalContext mismatch")
	}
}

func TestHookSpecificOutputToMapPostToolUseCoverage(t *testing.T) {
	additionalCtx := "post context"
	updatedMCP := map[string]interface{}{"result": "modified"}

	output := types.PostToolUseHookSpecificOutput{
		AdditionalContext:    &additionalCtx,
		UpdatedMCPToolOutput: updatedMCP,
	}

	result := hookSpecificOutputToMap(output)

	if result["additionalContext"] != "post context" {
		t.Errorf("additionalContext mismatch")
	}
	if result["updatedMCPToolOutput"] == nil {
		t.Errorf("updatedMCPToolOutput should not be nil")
	}
}

func TestHookSpecificOutputToMapPostToolUseFailureCoverage(t *testing.T) {
	additionalCtx := "failure context"

	output := types.PostToolUseFailureHookSpecificOutput{
		AdditionalContext: &additionalCtx,
	}

	result := hookSpecificOutputToMap(output)

	if result["additionalContext"] != "failure context" {
		t.Errorf("additionalContext mismatch")
	}
}

func TestHookSpecificOutputToMapUserPromptSubmitCoverage(t *testing.T) {
	additionalCtx := "submit context"

	output := types.UserPromptSubmitHookSpecificOutput{
		AdditionalContext: &additionalCtx,
	}

	result := hookSpecificOutputToMap(output)

	if result["additionalContext"] != "submit context" {
		t.Errorf("additionalContext mismatch")
	}
}

// ============================================================================
// Not Connected Error Tests
// ============================================================================

func TestQueryNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.Query(context.Background(), "test")
	if err == nil {
		t.Error("Query without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestInterruptNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.Interrupt(context.Background())
	if err == nil {
		t.Error("Interrupt without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestSetPermissionModeNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.SetPermissionMode(context.Background(), "acceptEdits")
	if err == nil {
		t.Error("SetPermissionMode without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestSetModelNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.SetModel(context.Background(), "claude-opus-4-6")
	if err == nil {
		t.Error("SetModel without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestRewindFilesNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.RewindFiles(context.Background(), "msg-uuid-123")
	if err == nil {
		t.Error("RewindFiles without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestReconnectMCPServerNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.ReconnectMCPServer(context.Background(), "test-server")
	if err == nil {
		t.Error("ReconnectMCPServer without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestToggleMCPServerNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.ToggleMCPServer(context.Background(), "test-server", true)
	if err == nil {
		t.Error("ToggleMCPServer without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestStopTaskNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	err := c.StopTask(context.Background(), "task-123")
	if err == nil {
		t.Error("StopTask without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestGetMCPStatusNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	_, err := c.GetMCPStatus(context.Background())
	if err == nil {
		t.Error("GetMCPStatus without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestGetContextUsageNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	_, err := c.GetContextUsage(context.Background())
	if err == nil {
		t.Error("GetContextUsage without Connect should fail")
	}
	t.Logf("Expected error: %v", err)
}

func TestGetServerInfoNilCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c.Close()

	info := c.GetServerInfo()
	if info != nil {
		t.Errorf("GetServerInfo without Connect should return nil, got %v", info)
	}
}

// ============================================================================
// Disconnect Tests
// ============================================================================

func TestDisconnectNotConnectedCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})

	err := c.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect without Connect should return nil, got: %v", err)
	}
}

func TestCloseCoverage(t *testing.T) {
	c := NewWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})

	err := c.Close()
	if err != nil {
		t.Errorf("Close should return nil, got: %v", err)
	}

	// Close again (should be idempotent)
	err = c.Close()
	if err != nil {
		t.Errorf("Second Close should return nil, got: %v", err)
	}
}

// ============================================================================
// ReceiveResponse Tests
// ============================================================================

func TestReceiveResponseChannelBehaviorCoverage(t *testing.T) {
	c := NewWithOptions(nil)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// ReceiveResponse should return a channel
	ch := c.ReceiveResponse(ctx)
	if ch == nil {
		t.Fatal("ReceiveResponse returned nil channel")
	}

	// Channel should close after context timeout (no messages without Connect)
	select {
	case msg, ok := <-ch:
		if ok {
			t.Errorf("Expected channel to close, got message: %v", msg)
		}
	case <-ctx.Done():
		// Expected - context done
	}
}

// ============================================================================
// ReceiveMessages Tests
// ============================================================================

func TestReceiveMessagesWithoutConnectCoverage(t *testing.T) {
	c := NewWithOptions(nil)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// ReceiveMessages should return a channel even without Connect
	ch := c.ReceiveMessages(ctx)
	if ch == nil {
		t.Fatal("ReceiveMessages returned nil channel")
	}

	// Channel should close (no query running)
	select {
	case msg, ok := <-ch:
		if ok {
			t.Logf("Received message: %v", msg)
		}
	case <-ctx.Done():
		// Expected - context done or channel closed
	}
}

func TestReceiveMessagesCachedCoverage(t *testing.T) {
	c := NewWithOptions(nil)
	defer c.Close()

	ctx := context.Background()

	// First call creates channel
	ch1 := c.ReceiveMessages(ctx)
	if ch1 == nil {
		t.Fatal("First ReceiveMessages returned nil")
	}

	// Second call returns same cached channel
	ch2 := c.ReceiveMessages(ctx)
	if ch2 == nil {
		t.Fatal("Second ReceiveMessages returned nil")
	}

	// They should be the same channel
	if ch1 != ch2 {
		t.Error("ReceiveMessages should return cached channel")
	}
}

// ============================================================================
// GenericHookInput Tests
// ============================================================================

func TestGenericHookInputGetHookEventNameCoverage(t *testing.T) {
	input := &genericHookInput{
		hookEventName: "CustomEvent",
		data:          map[string]interface{}{"test": "value"},
	}

	if input.GetHookEventName() != "CustomEvent" {
		t.Errorf("GetHookEventName = %s, want CustomEvent", input.GetHookEventName())
	}
}

// ============================================================================
// ConvertToHookInput Tests
// ============================================================================

func TestConvertToHookInputPreToolUseCoverage(t *testing.T) {
	m := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input":      map[string]interface{}{"command": "test"},
	}

	input := convertToHookInput(m)

	if input == nil {
		t.Fatal("convertToHookInput returned nil")
	}
	if input.GetHookEventName() != "PreToolUse" {
		t.Errorf("GetHookEventName = %s, want PreToolUse", input.GetHookEventName())
	}
}

func TestConvertToHookInputPostToolUseCoverage(t *testing.T) {
	m := map[string]interface{}{
		"hook_event_name": "PostToolUse",
		"tool_name":       "Read",
		"tool_response":   "content",
	}

	input := convertToHookInput(m)

	if input == nil {
		t.Fatal("convertToHookInput returned nil")
	}
	if input.GetHookEventName() != "PostToolUse" {
		t.Errorf("GetHookEventName = %s, want PostToolUse", input.GetHookEventName())
	}
}

func TestConvertToHookInputUnknownEventCoverage(t *testing.T) {
	m := map[string]interface{}{
		"hook_event_name": "UnknownEvent",
		"test":            "value",
	}

	input := convertToHookInput(m)

	if input == nil {
		t.Fatal("convertToHookInput returned nil for unknown event")
	}
	if input.GetHookEventName() != "UnknownEvent" {
		t.Errorf("GetHookEventName = %s, want UnknownEvent", input.GetHookEventName())
	}
}

func TestConvertToHookInputNonMapCoverage(t *testing.T) {
	input := convertToHookInput("not a map")

	if input != nil {
		t.Errorf("convertToHookInput for non-map should return nil, got %T", input)
	}
}
