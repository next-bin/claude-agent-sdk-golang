// Package client provides the Claude SDK client for interactive sessions.
package client

import (
	"context"
	"testing"

	"github.com/unitsvc/claude-agent-sdk-golang/internal/query"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// =============================================================================
// Test New and NewWithOptions
// =============================================================================

func TestNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("New() returned nil client")
	}
	if client.options == nil {
		t.Error("New() client has nil options, expected non-nil default options")
	}
}

func TestNewWithOptions_NilOptions(t *testing.T) {
	client := NewWithOptions(nil)
	if client == nil {
		t.Fatal("NewWithOptions(nil) returned nil client")
	}
	if client.options == nil {
		t.Error("NewWithOptions(nil) client has nil options, expected non-nil default options")
	}
}

func TestNewWithOptions_WithCustomOptions(t *testing.T) {
	cliPath := "/custom/path"
	opts := &types.ClaudeAgentOptions{
		CLIPath: cliPath,
	}
	client := NewWithOptions(opts)
	if client == nil {
		t.Fatal("NewWithOptions returned nil client")
	}
	if client.options == nil {
		t.Fatal("client options is nil")
	}
	if client.options.CLIPath != cliPath {
		t.Errorf("expected CLIPath %q, got %v", cliPath, client.options.CLIPath)
	}
}

func TestNewWithOptions_WithMultipleOptions(t *testing.T) {
	cliPath := "/usr/local/bin/claude"
	permissionMode := types.PermissionModeAcceptEdits
	opts := &types.ClaudeAgentOptions{
		CLIPath:        cliPath,
		PermissionMode: &permissionMode,
	}
	client := NewWithOptions(opts)
	if client == nil {
		t.Fatal("NewWithOptions returned nil client")
	}
	if client.options.CLIPath != cliPath {
		t.Errorf("expected CLIPath %q, got %v", cliPath, client.options.CLIPath)
	}
	if client.options.PermissionMode == nil || *client.options.PermissionMode != permissionMode {
		t.Errorf("expected PermissionMode %q, got %v", permissionMode, client.options.PermissionMode)
	}
}

func TestOptions(t *testing.T) {
	cliPath := "/test/path"
	opts := &types.ClaudeAgentOptions{
		CLIPath: cliPath,
	}
	client := NewWithOptions(opts)

	returnedOpts := client.Options()
	if returnedOpts == nil {
		t.Fatal("Options() returned nil")
	}
	if returnedOpts.CLIPath != cliPath {
		t.Errorf("expected CLIPath %q, got %v", cliPath, returnedOpts.CLIPath)
	}
}

// =============================================================================
// Test convertHooksToInternalFormat
// =============================================================================

// mockHookCallback is a mock implementation of types.HookCallback for testing
type mockHookCallback struct {
	executeFunc func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error)
}

func (m *mockHookCallback) Execute(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(input, toolUseID, context)
	}
	return types.SyncHookJSONOutput{}, nil
}

func TestConvertHooksToInternalFormat_EmptyHooks(t *testing.T) {
	client := &Client{}
	hooks := map[types.HookEvent][]types.HookMatcher{}
	result := client.convertHooksToInternalFormat(hooks)

	if result == nil {
		t.Error("expected non-nil result for empty hooks")
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestConvertHooksToInternalFormat_NilHooks(t *testing.T) {
	client := &Client{}
	result := client.convertHooksToInternalFormat(nil)

	if result == nil {
		t.Error("expected non-nil result for nil hooks")
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestConvertHooksToInternalFormat_SingleHook(t *testing.T) {
	client := &Client{}
	callback := &mockHookCallback{}
	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []types.HookCallback{callback},
			},
		},
	}

	result := client.convertHooksToInternalFormat(hooks)

	if len(result) != 1 {
		t.Fatalf("expected 1 event type, got %d", len(result))
	}

	matchers, ok := result["PreToolUse"]
	if !ok {
		t.Fatal("expected PreToolUse key in result")
	}
	if len(matchers) != 1 {
		t.Fatalf("expected 1 matcher, got %d", len(matchers))
	}
	if matchers[0].Matcher != "Bash" {
		t.Errorf("expected matcher 'Bash', got %q", matchers[0].Matcher)
	}
	if len(matchers[0].Hooks) != 1 {
		t.Errorf("expected 1 hook callback, got %d", len(matchers[0].Hooks))
	}
}

func TestConvertHooksToInternalFormat_MultipleHooks(t *testing.T) {
	client := &Client{}
	callback1 := &mockHookCallback{}
	callback2 := &mockHookCallback{}

	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []types.HookCallback{callback1},
			},
			{
				Matcher: "Write",
				Hooks:   []types.HookCallback{callback2},
			},
		},
		types.HookEventPostToolUse: {
			{
				Matcher: "Edit",
				Hooks:   []types.HookCallback{callback1, callback2},
			},
		},
	}

	result := client.convertHooksToInternalFormat(hooks)

	if len(result) != 2 {
		t.Fatalf("expected 2 event types, got %d", len(result))
	}

	preMatchers, ok := result["PreToolUse"]
	if !ok {
		t.Fatal("expected PreToolUse key in result")
	}
	if len(preMatchers) != 2 {
		t.Errorf("expected 2 matchers for PreToolUse, got %d", len(preMatchers))
	}

	postMatchers, ok := result["PostToolUse"]
	if !ok {
		t.Fatal("expected PostToolUse key in result")
	}
	if len(postMatchers) != 1 {
		t.Errorf("expected 1 matcher for PostToolUse, got %d", len(postMatchers))
	}
	if len(postMatchers[0].Hooks) != 2 {
		t.Errorf("expected 2 hook callbacks, got %d", len(postMatchers[0].Hooks))
	}
}

func TestConvertHooksToInternalFormat_WithTimeout(t *testing.T) {
	client := &Client{}
	timeout := 30.0
	callback := &mockHookCallback{}

	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []types.HookCallback{callback},
				Timeout: &timeout,
			},
		},
	}

	result := client.convertHooksToInternalFormat(hooks)

	matchers := result["PreToolUse"]
	if len(matchers) != 1 {
		t.Fatalf("expected 1 matcher, got %d", len(matchers))
	}
	if matchers[0].Timeout == nil || *matchers[0].Timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, matchers[0].Timeout)
	}
}

func TestConvertHooksToInternalFormat_CallbackExecution(t *testing.T) {
	client := &Client{}
	executed := false
	callback := &mockHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			executed = true
			return types.SyncHookJSONOutput{}, nil
		},
	}

	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []types.HookCallback{callback},
			},
		},
	}

	result := client.convertHooksToInternalFormat(hooks)

	// Execute the internal callback wrapper
	if len(result["PreToolUse"]) > 0 && len(result["PreToolUse"][0].Hooks) > 0 {
		internalCallback := result["PreToolUse"][0].Hooks[0]
		input := map[string]interface{}{
			"hook_event_name": "PreToolUse",
			"tool_name":       "Bash",
		}
		_, err := internalCallback(context.Background(), input, nil, types.HookContext{})
		if err != nil {
			t.Errorf("unexpected error executing callback: %v", err)
		}
		if !executed {
			t.Error("expected callback to be executed")
		}
	} else {
		t.Error("expected internal callback to be present")
	}
}

func TestConvertHooksToInternalFormat_AllHookEvents(t *testing.T) {
	client := &Client{}
	callback := &mockHookCallback{}

	testCases := []struct {
		event types.HookEvent
		name  string
	}{
		{types.HookEventPreToolUse, "PreToolUse"},
		{types.HookEventPostToolUse, "PostToolUse"},
		{types.HookEventPostToolUseFailure, "PostToolUseFailure"},
		{types.HookEventUserPromptSubmit, "UserPromptSubmit"},
		{types.HookEventStop, "Stop"},
		{types.HookEventSubagentStop, "SubagentStop"},
		{types.HookEventPreCompact, "PreCompact"},
		{types.HookEventNotification, "Notification"},
		{types.HookEventSubagentStart, "SubagentStart"},
		{types.HookEventPermissionRequest, "PermissionRequest"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hooks := map[types.HookEvent][]types.HookMatcher{
				tc.event: {
					{
						Matcher: "TestMatcher",
						Hooks:   []types.HookCallback{callback},
					},
				},
			}

			result := client.convertHooksToInternalFormat(hooks)

			if len(result) != 1 {
				t.Errorf("expected 1 event type, got %d", len(result))
			}
			if _, ok := result[tc.name]; !ok {
				t.Errorf("expected key %q in result", tc.name)
			}
		})
	}
}

// =============================================================================
// Test convertToHookInput
// =============================================================================

func TestConvertToHookInput_NilInput(t *testing.T) {
	result := convertToHookInput(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestConvertToHookInput_InvalidInput(t *testing.T) {
	result := convertToHookInput("not a map")
	if result != nil {
		t.Errorf("expected nil for non-map input, got %v", result)
	}
}

func TestConvertToHookInput_PreToolUse(t *testing.T) {
	input := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input": map[string]interface{}{
			"command": "ls -la",
		},
	}

	result := convertToHookInput(input)

	preInput, ok := result.(types.PreToolUseHookInput)
	if !ok {
		t.Fatalf("expected PreToolUseHookInput, got %T", result)
	}
	if preInput.HookEventName != "PreToolUse" {
		t.Errorf("expected HookEventName 'PreToolUse', got %q", preInput.HookEventName)
	}
	if preInput.ToolName != "Bash" {
		t.Errorf("expected ToolName 'Bash', got %q", preInput.ToolName)
	}
	if preInput.ToolInput == nil {
		t.Error("expected ToolInput to be set")
	}
}

func TestConvertToHookInput_PostToolUse(t *testing.T) {
	toolResponse := map[string]interface{}{"output": "file contents"}
	input := map[string]interface{}{
		"hook_event_name": "PostToolUse",
		"tool_name":       "Read",
		"tool_input": map[string]interface{}{
			"file_path": "/test/file.go",
		},
		"tool_response": toolResponse,
		"tool_use_id":   "tool-456",
	}

	result := convertToHookInput(input)

	postInput, ok := result.(types.PostToolUseHookInput)
	if !ok {
		t.Fatalf("expected PostToolUseHookInput, got %T", result)
	}
	if postInput.HookEventName != "PostToolUse" {
		t.Errorf("expected HookEventName 'PostToolUse', got %q", postInput.HookEventName)
	}
	if postInput.ToolName != "Read" {
		t.Errorf("expected ToolName 'Read', got %q", postInput.ToolName)
	}
	if postInput.ToolResponse == nil {
		t.Error("expected ToolResponse to be set")
	}
}

func TestConvertToHookInput_PostToolUseFailure(t *testing.T) {
	input := map[string]interface{}{
		"hook_event_name": "PostToolUseFailure",
		"tool_name":       "Bash",
		"tool_input": map[string]interface{}{
			"command": "exit 1",
		},
		"error": "command failed with exit code 1",
	}

	result := convertToHookInput(input)

	failureInput, ok := result.(types.PostToolUseFailureHookInput)
	if !ok {
		t.Fatalf("expected PostToolUseFailureHookInput, got %T", result)
	}
	if failureInput.HookEventName != "PostToolUseFailure" {
		t.Errorf("expected HookEventName 'PostToolUseFailure', got %q", failureInput.HookEventName)
	}
	if failureInput.Error != "command failed with exit code 1" {
		t.Errorf("expected Error 'command failed with exit code 1', got %q", failureInput.Error)
	}
}

func TestConvertToHookInput_UserPromptSubmit(t *testing.T) {
	input := map[string]interface{}{
		"hook_event_name": "UserPromptSubmit",
		"prompt":          "Write a hello world program",
	}

	result := convertToHookInput(input)

	submitInput, ok := result.(types.UserPromptSubmitHookInput)
	if !ok {
		t.Fatalf("expected UserPromptSubmitHookInput, got %T", result)
	}
	if submitInput.HookEventName != "UserPromptSubmit" {
		t.Errorf("expected HookEventName 'UserPromptSubmit', got %q", submitInput.HookEventName)
	}
	if submitInput.Prompt != "Write a hello world program" {
		t.Errorf("expected Prompt 'Write a hello world program', got %q", submitInput.Prompt)
	}
}

func TestConvertToHookInput_UnknownEvent(t *testing.T) {
	input := map[string]interface{}{
		"hook_event_name": "UnknownEvent",
		"custom_field":    "custom_value",
	}

	result := convertToHookInput(input)

	// Should return genericHookInput for unknown events
	if result == nil {
		t.Fatal("expected non-nil result for unknown event")
	}
	if result.GetHookEventName() != "UnknownEvent" {
		t.Errorf("expected HookEventName 'UnknownEvent', got %q", result.GetHookEventName())
	}
}

func TestConvertToHookInput_MissingEventName(t *testing.T) {
	input := map[string]interface{}{
		"tool_name": "Bash",
	}

	result := convertToHookInput(input)

	// Should return genericHookInput with empty event name
	if result == nil {
		t.Fatal("expected non-nil result for missing event name")
	}
	if result.GetHookEventName() != "" {
		t.Errorf("expected empty HookEventName, got %q", result.GetHookEventName())
	}
}

// =============================================================================
// Test hookOutputToMap
// =============================================================================

func TestHookOutputToMap_AsyncOutput(t *testing.T) {
	asyncTimeout := 5000
	output := types.AsyncHookJSONOutput{
		Async_:       true,
		AsyncTimeout: &asyncTimeout,
	}

	result := hookOutputToMap(output)

	if result["async"] != true {
		t.Errorf("expected async to be true, got %v", result["async"])
	}
	if result["asyncTimeout"] != asyncTimeout {
		t.Errorf("expected asyncTimeout %d, got %v", asyncTimeout, result["asyncTimeout"])
	}
}

func TestHookOutputToMap_AsyncOutput_NilTimeout(t *testing.T) {
	output := types.AsyncHookJSONOutput{
		Async_:       true,
		AsyncTimeout: nil,
	}

	result := hookOutputToMap(output)

	if result["async"] != true {
		t.Errorf("expected async to be true, got %v", result["async"])
	}
	if _, ok := result["asyncTimeout"]; ok {
		t.Error("expected asyncTimeout to not be present")
	}
}

func TestHookOutputToMap_SyncOutput_AllFields(t *testing.T) {
	continueVal := true
	suppressOutput := false
	stopReason := "completed"
	decision := "allow"
	systemMessage := "Test message"
	reason := "Test reason"
	specificOutput := types.PreToolUseHookSpecificOutput{
		HookEventName:      "PreToolUse",
		PermissionDecision: &decision,
	}

	output := types.SyncHookJSONOutput{
		Continue_:          &continueVal,
		SuppressOutput:     &suppressOutput,
		StopReason:         &stopReason,
		Decision:           &decision,
		SystemMessage:      &systemMessage,
		Reason:             &reason,
		HookSpecificOutput: specificOutput,
	}

	result := hookOutputToMap(output)

	if result["continue"] != true {
		t.Errorf("expected continue to be true, got %v", result["continue"])
	}
	if result["suppressOutput"] != false {
		t.Errorf("expected suppressOutput to be false, got %v", result["suppressOutput"])
	}
	if result["stopReason"] != "completed" {
		t.Errorf("expected stopReason 'completed', got %v", result["stopReason"])
	}
	if result["decision"] != "allow" {
		t.Errorf("expected decision 'allow', got %v", result["decision"])
	}
	if result["systemMessage"] != "Test message" {
		t.Errorf("expected systemMessage 'Test message', got %v", result["systemMessage"])
	}
	if result["reason"] != "Test reason" {
		t.Errorf("expected reason 'Test reason', got %v", result["reason"])
	}
	if _, ok := result["hookSpecificOutput"]; !ok {
		t.Error("expected hookSpecificOutput to be present")
	}
}

func TestHookOutputToMap_SyncOutput_NilFields(t *testing.T) {
	output := types.SyncHookJSONOutput{}

	result := hookOutputToMap(output)

	if len(result) != 0 {
		t.Errorf("expected empty result for output with all nil fields, got %v", result)
	}
}

func TestHookOutputToMap_SyncOutput_ContinueOnly(t *testing.T) {
	continueVal := true
	output := types.SyncHookJSONOutput{
		Continue_: &continueVal,
	}

	result := hookOutputToMap(output)

	if result["continue"] != true {
		t.Errorf("expected continue to be true, got %v", result["continue"])
	}
	if len(result) != 1 {
		t.Errorf("expected 1 field in result, got %d", len(result))
	}
}

func TestHookOutputToMap_NilOutput(t *testing.T) {
	result := hookOutputToMap(nil)

	if result == nil {
		t.Error("expected non-nil result for nil output")
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for nil output, got %v", result)
	}
}

// =============================================================================
// Test hookSpecificOutputToMap
// =============================================================================

func TestHookSpecificOutputToMap_PreToolUseOutput(t *testing.T) {
	decision := "allow"
	reason := "User approved"
	updatedInput := map[string]interface{}{"command": "ls -la"}
	additionalContext := "Additional info"

	output := types.PreToolUseHookSpecificOutput{
		HookEventName:            "PreToolUse",
		PermissionDecision:       &decision,
		PermissionDecisionReason: &reason,
		UpdatedInput:             updatedInput,
		AdditionalContext:        &additionalContext,
	}

	result := hookSpecificOutputToMap(output)

	if result["permissionDecision"] != "allow" {
		t.Errorf("expected permissionDecision 'allow', got %v", result["permissionDecision"])
	}
	if result["permissionDecisionReason"] != "User approved" {
		t.Errorf("expected permissionDecisionReason 'User approved', got %v", result["permissionDecisionReason"])
	}
	if result["updatedInput"] == nil {
		t.Error("expected updatedInput to be present")
	}
	if result["additionalContext"] != "Additional info" {
		t.Errorf("expected additionalContext 'Additional info', got %v", result["additionalContext"])
	}
}

func TestHookSpecificOutputToMap_PreToolUseOutput_NilFields(t *testing.T) {
	output := types.PreToolUseHookSpecificOutput{
		HookEventName: "PreToolUse",
	}

	result := hookSpecificOutputToMap(output)

	if len(result) != 0 {
		t.Errorf("expected empty result for output with nil fields, got %v", result)
	}
}

func TestHookSpecificOutputToMap_PostToolUseOutput(t *testing.T) {
	additionalContext := "Context info"
	updatedOutput := map[string]interface{}{"result": "modified"}

	output := types.PostToolUseHookSpecificOutput{
		HookEventName:        "PostToolUse",
		AdditionalContext:    &additionalContext,
		UpdatedMCPToolOutput: updatedOutput,
	}

	result := hookSpecificOutputToMap(output)

	if result["additionalContext"] != "Context info" {
		t.Errorf("expected additionalContext 'Context info', got %v", result["additionalContext"])
	}
	if result["updatedMCPToolOutput"] == nil {
		t.Error("expected updatedMCPToolOutput to be present")
	}
}

func TestHookSpecificOutputToMap_PostToolUseFailureOutput(t *testing.T) {
	additionalContext := "Error context"

	output := types.PostToolUseFailureHookSpecificOutput{
		HookEventName:     "PostToolUseFailure",
		AdditionalContext: &additionalContext,
	}

	result := hookSpecificOutputToMap(output)

	if result["additionalContext"] != "Error context" {
		t.Errorf("expected additionalContext 'Error context', got %v", result["additionalContext"])
	}
}

func TestHookSpecificOutputToMap_UserPromptSubmitOutput(t *testing.T) {
	additionalContext := "User context"

	output := types.UserPromptSubmitHookSpecificOutput{
		HookEventName:     "UserPromptSubmit",
		AdditionalContext: &additionalContext,
	}

	result := hookSpecificOutputToMap(output)

	if result["additionalContext"] != "User context" {
		t.Errorf("expected additionalContext 'User context', got %v", result["additionalContext"])
	}
}

func TestHookSpecificOutputToMap_NilOutput(t *testing.T) {
	result := hookSpecificOutputToMap(nil)

	if result == nil {
		t.Error("expected non-nil result for nil output")
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for nil output, got %v", result)
	}
}

func TestHookSpecificOutputToMap_AllTypes(t *testing.T) {
	testCases := []struct {
		name   string
		output types.HookSpecificOutput
	}{
		{
			name: "PreToolUseHookSpecificOutput",
			output: types.PreToolUseHookSpecificOutput{
				HookEventName: "PreToolUse",
			},
		},
		{
			name: "PostToolUseHookSpecificOutput",
			output: types.PostToolUseHookSpecificOutput{
				HookEventName: "PostToolUse",
			},
		},
		{
			name: "PostToolUseFailureHookSpecificOutput",
			output: types.PostToolUseFailureHookSpecificOutput{
				HookEventName: "PostToolUseFailure",
			},
		},
		{
			name: "UserPromptSubmitHookSpecificOutput",
			output: types.UserPromptSubmitHookSpecificOutput{
				HookEventName: "UserPromptSubmit",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hookSpecificOutputToMap(tc.output)
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// =============================================================================
// Test parsePreToolUseHookInput
// =============================================================================

func TestParsePreToolUseHookInput(t *testing.T) {
	input := map[string]interface{}{
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": "ls -la",
		},
	}

	result := parsePreToolUseHookInput(input)

	if result.HookEventName != "PreToolUse" {
		t.Errorf("expected HookEventName 'PreToolUse', got %q", result.HookEventName)
	}
	if result.ToolName != "Bash" {
		t.Errorf("expected ToolName 'Bash', got %q", result.ToolName)
	}
	if result.ToolInput == nil {
		t.Error("expected ToolInput to be set")
	}
}

func TestParsePreToolUseHookInput_Empty(t *testing.T) {
	input := map[string]interface{}{}

	result := parsePreToolUseHookInput(input)

	if result.HookEventName != "PreToolUse" {
		t.Errorf("expected HookEventName 'PreToolUse', got %q", result.HookEventName)
	}
	if result.ToolName != "" {
		t.Errorf("expected empty ToolName, got %q", result.ToolName)
	}
}

// =============================================================================
// Test parsePostToolUseHookInput
// =============================================================================

func TestParsePostToolUseHookInput(t *testing.T) {
	toolResponse := map[string]interface{}{"output": "result"}
	input := map[string]interface{}{
		"tool_name":     "Read",
		"tool_input":    map[string]interface{}{"file_path": "/test"},
		"tool_response": toolResponse,
		"tool_use_id":   "tool-456",
	}

	result := parsePostToolUseHookInput(input)

	if result.HookEventName != "PostToolUse" {
		t.Errorf("expected HookEventName 'PostToolUse', got %q", result.HookEventName)
	}
	if result.ToolName != "Read" {
		t.Errorf("expected ToolName 'Read', got %q", result.ToolName)
	}
	if result.ToolResponse == nil {
		t.Error("expected ToolResponse to be set")
	}
}

// =============================================================================
// Test parsePostToolUseFailureHookInput
// =============================================================================

func TestParsePostToolUseFailureHookInput(t *testing.T) {
	input := map[string]interface{}{
		"tool_name":  "Bash",
		"tool_input": map[string]interface{}{"command": "exit 1"},
		"error":      "command failed",
	}

	result := parsePostToolUseFailureHookInput(input)

	if result.HookEventName != "PostToolUseFailure" {
		t.Errorf("expected HookEventName 'PostToolUseFailure', got %q", result.HookEventName)
	}
	if result.Error != "command failed" {
		t.Errorf("expected Error 'command failed', got %q", result.Error)
	}
}

// =============================================================================
// Test parseUserPromptSubmitHookInput
// =============================================================================

func TestParseUserPromptSubmitHookInput(t *testing.T) {
	input := map[string]interface{}{
		"prompt": "Write a function",
	}

	result := parseUserPromptSubmitHookInput(input)

	if result.HookEventName != "UserPromptSubmit" {
		t.Errorf("expected HookEventName 'UserPromptSubmit', got %q", result.HookEventName)
	}
	if result.Prompt != "Write a function" {
		t.Errorf("expected Prompt 'Write a function', got %q", result.Prompt)
	}
}

// =============================================================================
// Test genericHookInput
// =============================================================================

func TestGenericHookInput(t *testing.T) {
	data := map[string]interface{}{
		"hook_event_name": "CustomEvent",
		"custom_field":    "custom_value",
	}

	input := &genericHookInput{
		hookEventName: "CustomEvent",
		data:          data,
	}

	if input.GetHookEventName() != "CustomEvent" {
		t.Errorf("expected HookEventName 'CustomEvent', got %q", input.GetHookEventName())
	}
}

// =============================================================================
// Integration tests for hook conversion chain
// =============================================================================

func TestHookConversionChain(t *testing.T) {
	// This test verifies the full conversion chain from types.HookMatcher to
	// query.HookMatcher through convertHooksToInternalFormat

	client := &Client{}
	expectedOutput := types.SyncHookJSONOutput{
		Continue_: boolPtr(true),
	}

	callback := &mockHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			return expectedOutput, nil
		},
	}

	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []types.HookCallback{callback},
			},
		},
	}

	result := client.convertHooksToInternalFormat(hooks)

	// Verify the internal format
	if len(result) != 1 {
		t.Fatalf("expected 1 event type, got %d", len(result))
	}

	matchers := result["PreToolUse"]
	if len(matchers) != 1 {
		t.Fatalf("expected 1 matcher, got %d", len(matchers))
	}

	// Execute the internal callback and verify result
	internalCallback := matchers[0].Hooks[0]
	input := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input":      map[string]interface{}{"command": "ls"},
	}

	output, err := internalCallback(context.Background(), input, nil, types.HookContext{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output["continue"] != true {
		t.Errorf("expected continue to be true, got %v", output["continue"])
	}
}

// =============================================================================
// Helper functions
// =============================================================================

func boolPtr(b bool) *bool {
	return &b
}

// Ensure query.HookMatcher interface is satisfied
var _ = []query.HookMatcher{}
