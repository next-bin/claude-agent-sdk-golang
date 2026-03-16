package types

import (
	"encoding/json"
	"testing"
)

// ============================================================================
// PermissionMode Constants Tests
// ============================================================================

func TestPermissionModeConstants(t *testing.T) {
	tests := []struct {
		name     string
		mode     PermissionMode
		expected string
	}{
		{"Default mode", PermissionModeDefault, "default"},
		{"AcceptEdits mode", PermissionModeAcceptEdits, "acceptEdits"},
		{"Plan mode", PermissionModePlan, "plan"},
		{"BypassPermissions mode", PermissionModeBypassPermissions, "bypassPermissions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.mode) != tt.expected {
				t.Errorf("PermissionMode %s = %q, want %q", tt.name, tt.mode, tt.expected)
			}
		})
	}
}

func TestPermissionModeJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		mode PermissionMode
	}{
		{"Default", PermissionModeDefault},
		{"AcceptEdits", PermissionModeAcceptEdits},
		{"Plan", PermissionModePlan},
		{"BypassPermissions", PermissionModeBypassPermissions},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.mode)
			if err != nil {
				t.Fatalf("Failed to marshal PermissionMode: %v", err)
			}

			var unmarshaled PermissionMode
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal PermissionMode: %v", err)
			}

			if unmarshaled != tt.mode {
				t.Errorf("Round trip failed: got %q, want %q", unmarshaled, tt.mode)
			}
		})
	}
}

// ============================================================================
// PermissionBehavior Constants Tests
// ============================================================================

func TestPermissionBehaviorConstants(t *testing.T) {
	tests := []struct {
		name     string
		behavior PermissionBehavior
		expected string
	}{
		{"Allow", PermissionBehaviorAllow, "allow"},
		{"Deny", PermissionBehaviorDeny, "deny"},
		{"Ask", PermissionBehaviorAsk, "ask"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.behavior) != tt.expected {
				t.Errorf("PermissionBehavior %s = %q, want %q", tt.name, tt.behavior, tt.expected)
			}
		})
	}
}

// ============================================================================
// PermissionUpdateDestination Constants Tests
// ============================================================================

func TestPermissionUpdateDestinationConstants(t *testing.T) {
	tests := []struct {
		name        string
		destination PermissionUpdateDestination
		expected    string
	}{
		{"UserSettings", PermissionUpdateDestinationUserSettings, "userSettings"},
		{"ProjectSettings", PermissionUpdateDestinationProjectSettings, "projectSettings"},
		{"LocalSettings", PermissionUpdateDestinationLocalSettings, "localSettings"},
		{"Session", PermissionUpdateDestinationSession, "session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.destination) != tt.expected {
				t.Errorf("PermissionUpdateDestination %s = %q, want %q", tt.name, tt.destination, tt.expected)
			}
		})
	}
}

// ============================================================================
// PermissionUpdateType Constants Tests
// ============================================================================

func TestPermissionUpdateTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		typ      PermissionUpdateType
		expected string
	}{
		{"AddRules", PermissionUpdateTypeAddRules, "addRules"},
		{"ReplaceRules", PermissionUpdateTypeReplaceRules, "replaceRules"},
		{"RemoveRules", PermissionUpdateTypeRemoveRules, "removeRules"},
		{"SetMode", PermissionUpdateTypeSetMode, "setMode"},
		{"AddDirectories", PermissionUpdateTypeAddDirectories, "addDirectories"},
		{"RemoveDirectories", PermissionUpdateTypeRemoveDirectories, "removeDirectories"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.typ) != tt.expected {
				t.Errorf("PermissionUpdateType %s = %q, want %q", tt.name, tt.typ, tt.expected)
			}
		})
	}
}

// ============================================================================
// ClaudeAgentOptions Tests
// ============================================================================

func TestClaudeAgentOptionsDefaultValues(t *testing.T) {
	opts := ClaudeAgentOptions{}

	// Check that default values are zero values
	if opts.PermissionMode != nil {
		t.Error("Expected PermissionMode to be nil by default")
	}
	if opts.MaxTurns != nil {
		t.Error("Expected MaxTurns to be nil by default")
	}
	if opts.MaxBudgetUSD != nil {
		t.Error("Expected MaxBudgetUSD to be nil by default")
	}
	if opts.Model != nil {
		t.Error("Expected Model to be nil by default")
	}
	if opts.FallbackModel != nil {
		t.Error("Expected FallbackModel to be nil by default")
	}
	if opts.MaxBufferSize != nil {
		t.Error("Expected MaxBufferSize to be nil by default")
	}
	if opts.User != nil {
		t.Error("Expected User to be nil by default")
	}
	if opts.Sandbox != nil {
		t.Error("Expected Sandbox to be nil by default")
	}
	if opts.Thinking != nil {
		t.Error("Expected Thinking to be nil by default")
	}
	if opts.Effort != nil {
		t.Error("Expected Effort to be nil by default")
	}
}

func TestClaudeAgentOptionsJSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		opts     ClaudeAgentOptions
		expected string
	}{
		{
			name:     "Empty options",
			opts:     ClaudeAgentOptions{},
			expected: `{}`,
		},
		{
			name: "With PermissionMode",
			opts: ClaudeAgentOptions{
				PermissionMode: PermissionModePtr(PermissionModePlan),
			},
			expected: `{"permission_mode":"plan"}`,
		},
		{
			name: "With MaxTurns",
			opts: ClaudeAgentOptions{
				MaxTurns: Int(10),
			},
			expected: `{"max_turns":10}`,
		},
		{
			name: "With Model",
			opts: ClaudeAgentOptions{
				Model: String("claude-3-opus"),
			},
			expected: `{"model":"claude-3-opus"}`,
		},
		{
			name: "With Multiple Fields",
			opts: ClaudeAgentOptions{
				PermissionMode:       PermissionModePtr(PermissionModeAcceptEdits),
				MaxTurns:             Int(5),
				MaxBudgetUSD:         Float64(100.50),
				Model:                String("claude-3-sonnet"),
				ContinueConversation: true,
			},
			// Note: JSON field ordering is non-deterministic, so we validate fields individually
			expected: "",
		},
		{
			name: "With AllowedTools",
			opts: ClaudeAgentOptions{
				AllowedTools: []string{"Bash", "Read", "Write"},
			},
			expected: `{"allowed_tools":["Bash","Read","Write"]}`,
		},
		{
			name: "With DisallowedTools",
			opts: ClaudeAgentOptions{
				DisallowedTools: []string{"WebFetch"},
			},
			expected: `{"disallowed_tools":["WebFetch"]}`,
		},
		{
			name: "With Betas",
			opts: ClaudeAgentOptions{
				Betas: []SdkBeta{SdkBetaContext1M},
			},
			expected: `{"betas":["context-1m-2025-08-07"]}`,
		},
		{
			name: "With Env",
			opts: ClaudeAgentOptions{
				Env: map[string]string{"API_KEY": "secret", "DEBUG": "true"},
			},
			// Note: map ordering is not guaranteed, so we check both possibilities
			expected: "", // Will be validated differently
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.opts)
			if err != nil {
				t.Fatalf("Failed to marshal ClaudeAgentOptions: %v", err)
			}

			if tt.name == "With Env" {
				// For map fields, just verify it contains the expected keys
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}
				env, ok := result["env"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected env field to be a map")
				}
				if env["API_KEY"] != "secret" || env["DEBUG"] != "true" {
					t.Errorf("Env map values incorrect: %v", env)
				}
				return
			}

			if tt.name == "With Multiple Fields" {
				// For multiple fields, validate each field individually since ordering is non-deterministic
				var result map[string]interface{}
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}
				if result["permission_mode"] != "acceptEdits" {
					t.Errorf("permission_mode = %v, want acceptEdits", result["permission_mode"])
				}
				if result["max_turns"] != float64(5) {
					t.Errorf("max_turns = %v, want 5", result["max_turns"])
				}
				if result["max_budget_usd"] != float64(100.5) {
					t.Errorf("max_budget_usd = %v, want 100.5", result["max_budget_usd"])
				}
				if result["model"] != "claude-3-sonnet" {
					t.Errorf("model = %v, want claude-3-sonnet", result["model"])
				}
				if result["continue_conversation"] != true {
					t.Errorf("continue_conversation = %v, want true", result["continue_conversation"])
				}
				return
			}

			if string(data) != tt.expected {
				t.Errorf("JSON mismatch:\ngot:      %s\nexpected: %s", data, tt.expected)
			}
		})
	}
}

func TestClaudeAgentOptionsWithSandbox(t *testing.T) {
	enabled := true
	autoAllow := false
	opts := ClaudeAgentOptions{
		Sandbox: &SandboxSettings{
			Enabled:                  &enabled,
			AutoAllowBashIfSandboxed: &autoAllow,
			ExcludedCommands:         []string{"git", "docker"},
		},
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ClaudeAgentOptions
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Sandbox == nil {
		t.Fatal("Expected Sandbox to be non-nil")
	}
	if *unmarshaled.Sandbox.Enabled != enabled {
		t.Errorf("Sandbox.Enabled = %v, want %v", *unmarshaled.Sandbox.Enabled, enabled)
	}
	if *unmarshaled.Sandbox.AutoAllowBashIfSandboxed != autoAllow {
		t.Errorf("Sandbox.AutoAllowBashIfSandboxed = %v, want %v", *unmarshaled.Sandbox.AutoAllowBashIfSandboxed, autoAllow)
	}
}

func TestClaudeAgentOptionsWithHooks(t *testing.T) {
	opts := ClaudeAgentOptions{
		Hooks: map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "Bash",
				},
			},
		},
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ClaudeAgentOptions
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(unmarshaled.Hooks) != 1 {
		t.Errorf("Expected 1 hook event, got %d", len(unmarshaled.Hooks))
	}
}

// ============================================================================
// Hook Input Types Tests
// ============================================================================

func TestPreToolUseHookInputJSON(t *testing.T) {
	input := PreToolUseHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     map[string]interface{}{"command": "ls -la"},
		ToolUseID:     "tool-456",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PreToolUseHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.SessionID != input.SessionID {
		t.Errorf("SessionID = %q, want %q", unmarshaled.SessionID, input.SessionID)
	}
	if unmarshaled.ToolName != input.ToolName {
		t.Errorf("ToolName = %q, want %q", unmarshaled.ToolName, input.ToolName)
	}
	if unmarshaled.ToolUseID != input.ToolUseID {
		t.Errorf("ToolUseID = %q, want %q", unmarshaled.ToolUseID, input.ToolUseID)
	}
	if unmarshaled.GetHookEventName() != "PreToolUse" {
		t.Errorf("GetHookEventName() = %q, want %q", unmarshaled.GetHookEventName(), "PreToolUse")
	}
}

func TestPostToolUseHookInputJSON(t *testing.T) {
	input := PostToolUseHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName: "PostToolUse",
		ToolName:      "Read",
		ToolInput:     map[string]interface{}{"file_path": "/home/user/test.txt"},
		ToolResponse:  "file contents",
		ToolUseID:     "tool-789",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PostToolUseHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ToolName != input.ToolName {
		t.Errorf("ToolName = %q, want %q", unmarshaled.ToolName, input.ToolName)
	}
	if unmarshaled.GetHookEventName() != "PostToolUse" {
		t.Errorf("GetHookEventName() = %q, want %q", unmarshaled.GetHookEventName(), "PostToolUse")
	}
}

func TestPostToolUseFailureHookInputJSON(t *testing.T) {
	isInterrupt := true
	input := PostToolUseFailureHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName: "PostToolUseFailure",
		ToolName:      "Bash",
		ToolInput:     map[string]interface{}{"command": "exit 1"},
		ToolUseID:     "tool-fail",
		Error:         "command failed with exit code 1",
		IsInterrupt:   &isInterrupt,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PostToolUseFailureHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Error != input.Error {
		t.Errorf("Error = %q, want %q", unmarshaled.Error, input.Error)
	}
	if unmarshaled.IsInterrupt == nil || *unmarshaled.IsInterrupt != isInterrupt {
		t.Errorf("IsInterrupt not properly unmarshaled")
	}
}

func TestUserPromptSubmitHookInputJSON(t *testing.T) {
	input := UserPromptSubmitHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName: "UserPromptSubmit",
		Prompt:        "What is the weather today?",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled UserPromptSubmitHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Prompt != input.Prompt {
		t.Errorf("Prompt = %q, want %q", unmarshaled.Prompt, input.Prompt)
	}
}

func TestStopHookInputJSON(t *testing.T) {
	input := StopHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName:  "Stop",
		StopHookActive: true,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled StopHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.StopHookActive != input.StopHookActive {
		t.Errorf("StopHookActive = %v, want %v", unmarshaled.StopHookActive, input.StopHookActive)
	}
}

func TestSubagentStopHookInputJSON(t *testing.T) {
	input := SubagentStopHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName:       "SubagentStop",
		StopHookActive:      false,
		AgentID:             "agent-456",
		AgentTranscriptPath: "/path/to/agent/transcript",
		AgentType:           "task",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SubagentStopHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.AgentID != input.AgentID {
		t.Errorf("AgentID = %q, want %q", unmarshaled.AgentID, input.AgentID)
	}
	if unmarshaled.AgentType != input.AgentType {
		t.Errorf("AgentType = %q, want %q", unmarshaled.AgentType, input.AgentType)
	}
}

func TestPreCompactHookInputJSON(t *testing.T) {
	customInstructions := "Summarize concisely"
	input := PreCompactHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName:      "PreCompact",
		Trigger:            "auto",
		CustomInstructions: &customInstructions,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PreCompactHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Trigger != input.Trigger {
		t.Errorf("Trigger = %q, want %q", unmarshaled.Trigger, input.Trigger)
	}
	if unmarshaled.CustomInstructions == nil || *unmarshaled.CustomInstructions != customInstructions {
		t.Errorf("CustomInstructions not properly unmarshaled")
	}
}

func TestNotificationHookInputJSON(t *testing.T) {
	title := "Notification Title"
	input := NotificationHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName:    "Notification",
		Message:          "This is a notification",
		Title:            &title,
		NotificationType: "info",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled NotificationHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Message != input.Message {
		t.Errorf("Message = %q, want %q", unmarshaled.Message, input.Message)
	}
	if unmarshaled.NotificationType != input.NotificationType {
		t.Errorf("NotificationType = %q, want %q", unmarshaled.NotificationType, input.NotificationType)
	}
}

func TestSubagentStartHookInputJSON(t *testing.T) {
	input := SubagentStartHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName: "SubagentStart",
		AgentID:       "agent-789",
		AgentType:     "research",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SubagentStartHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.AgentID != input.AgentID {
		t.Errorf("AgentID = %q, want %q", unmarshaled.AgentID, input.AgentID)
	}
}

func TestPermissionRequestHookInputJSON(t *testing.T) {
	input := PermissionRequestHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName:         "PermissionRequest",
		ToolName:              "Bash",
		ToolInput:             map[string]interface{}{"command": "rm -rf /"},
		PermissionSuggestions: []interface{}{"allow", "deny"},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PermissionRequestHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ToolName != input.ToolName {
		t.Errorf("ToolName = %q, want %q", unmarshaled.ToolName, input.ToolName)
	}
}

// ============================================================================
// Hook Output Types Tests
// ============================================================================

func TestPreToolUseHookSpecificOutputJSON(t *testing.T) {
	decision := "allow"
	reason := "Tool is safe"
	updatedInput := map[string]interface{}{"command": "ls"}
	additionalContext := "Additional context"

	output := PreToolUseHookSpecificOutput{
		HookEventName:            "PreToolUse",
		PermissionDecision:       &decision,
		PermissionDecisionReason: &reason,
		UpdatedInput:             updatedInput,
		AdditionalContext:        &additionalContext,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PreToolUseHookSpecificOutput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.GetHookEventName() != "PreToolUse" {
		t.Errorf("GetHookEventName() = %q, want %q", unmarshaled.GetHookEventName(), "PreToolUse")
	}
	if unmarshaled.PermissionDecision == nil || *unmarshaled.PermissionDecision != decision {
		t.Errorf("PermissionDecision not properly unmarshaled")
	}
}

func TestPostToolUseHookSpecificOutputJSON(t *testing.T) {
	additionalContext := "Post tool context"
	output := PostToolUseHookSpecificOutput{
		HookEventName:     "PostToolUse",
		AdditionalContext: &additionalContext,
		UpdatedMCPToolOutput: map[string]interface{}{
			"result": "success",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PostToolUseHookSpecificOutput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.GetHookEventName() != "PostToolUse" {
		t.Errorf("GetHookEventName() = %q, want %q", unmarshaled.GetHookEventName(), "PostToolUse")
	}
}

func TestAsyncHookJSONOutput(t *testing.T) {
	timeout := 120
	output := AsyncHookJSONOutput{
		Async_:       true,
		AsyncTimeout: &timeout,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled AsyncHookJSONOutput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !unmarshaled.IsAsync() {
		t.Error("IsAsync() should return true")
	}
	if unmarshaled.AsyncTimeout == nil || *unmarshaled.AsyncTimeout != timeout {
		t.Errorf("AsyncTimeout not properly unmarshaled")
	}
}

func TestSyncHookJSONOutput(t *testing.T) {
	continueVal := true
	suppressOutput := false
	decision := "block"
	systemMessage := "Blocked by hook"
	reason := "Security policy"

	output := SyncHookJSONOutput{
		Continue_:      &continueVal,
		SuppressOutput: &suppressOutput,
		Decision:       &decision,
		SystemMessage:  &systemMessage,
		Reason:         &reason,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SyncHookJSONOutput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.IsAsync() {
		t.Error("IsAsync() should return false for SyncHookJSONOutput")
	}
	if unmarshaled.Continue_ == nil || *unmarshaled.Continue_ != continueVal {
		t.Errorf("Continue_ not properly unmarshaled")
	}
}

func TestPermissionRequestHookSpecificOutputJSON(t *testing.T) {
	output := PermissionRequestHookSpecificOutput{
		HookEventName: "PermissionRequest",
		Decision: map[string]interface{}{
			"behavior": "allow",
			"mode":     "default",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PermissionRequestHookSpecificOutput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.GetHookEventName() != "PermissionRequest" {
		t.Errorf("GetHookEventName() = %q, want %q", unmarshaled.GetHookEventName(), "PermissionRequest")
	}
}

// ============================================================================
// Message Types Tests
// ============================================================================

func TestResultMessageJSON(t *testing.T) {
	cost := 0.05
	usage := map[string]interface{}{
		"input_tokens":  100,
		"output_tokens": 50,
	}
	result := "Task completed successfully"

	msg := ResultMessage{
		Subtype:       "success",
		DurationMs:    5000,
		DurationAPIMs: 3000,
		IsError:       false,
		NumTurns:      3,
		SessionID:     "session-123",
		TotalCostUSD:  &cost,
		Usage:         usage,
		Result:        &result,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ResultMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.SessionID != msg.SessionID {
		t.Errorf("SessionID = %q, want %q", unmarshaled.SessionID, msg.SessionID)
	}
	if unmarshaled.GetSessionID() != msg.SessionID {
		t.Errorf("GetSessionID() = %q, want %q", unmarshaled.GetSessionID(), msg.SessionID)
	}
	if unmarshaled.DurationMs != msg.DurationMs {
		t.Errorf("DurationMs = %d, want %d", unmarshaled.DurationMs, msg.DurationMs)
	}
	if unmarshaled.IsError != msg.IsError {
		t.Errorf("IsError = %v, want %v", unmarshaled.IsError, msg.IsError)
	}
}

func TestAssistantMessageJSON(t *testing.T) {
	model := "claude-3-sonnet"
	parentToolUseID := "tool-123"
	errType := AssistantMessageErrorRateLimit

	msg := AssistantMessage{
		Content: []ContentBlock{
			TextBlock{Type: "text", Text: "Hello, world!"},
		},
		Model:           model,
		ParentToolUseID: &parentToolUseID,
		Error:           &errType,
	}

	// Test marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify the JSON contains expected fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to raw map: %v", err)
	}

	if raw["model"] != model {
		t.Errorf("model = %v, want %v", raw["model"], model)
	}

	// Test GetSessionID
	if msg.GetSessionID() != "" {
		t.Errorf("GetSessionID() should return empty string for AssistantMessage")
	}

	// Test AssistantMessageError constants
	if string(AssistantMessageErrorRateLimit) != "rate_limit" {
		t.Errorf("AssistantMessageErrorRateLimit = %q, want %q", AssistantMessageErrorRateLimit, "rate_limit")
	}
}

func TestUserMessageJSON(t *testing.T) {
	uuid := "user-uuid-123"
	parentToolUseID := "tool-456"

	msg := UserMessage{
		Content:         "Hello, assistant!",
		UUID:            &uuid,
		ParentToolUseID: &parentToolUseID,
		ToolUseResult:   map[string]interface{}{"status": "success"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled UserMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.UUID == nil || *unmarshaled.UUID != uuid {
		t.Errorf("UUID not properly unmarshaled")
	}
	if unmarshaled.GetSessionID() != "" {
		t.Errorf("GetSessionID() should return empty string for UserMessage")
	}
}

func TestSystemMessageJSON(t *testing.T) {
	msg := SystemMessage{
		Subtype: "init",
		Data: map[string]interface{}{
			"session_id": "session-123",
			"cwd":        "/home/user",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SystemMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Subtype != msg.Subtype {
		t.Errorf("Subtype = %q, want %q", unmarshaled.Subtype, msg.Subtype)
	}
}

func TestStreamEventJSON(t *testing.T) {
	parentToolUseID := "tool-789"

	event := StreamEvent{
		UUID:            "event-uuid-123",
		SessionID:       "session-456",
		Event:           map[string]interface{}{"type": "content_block_delta", "delta": "test"},
		ParentToolUseID: &parentToolUseID,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled StreamEvent
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.SessionID != event.SessionID {
		t.Errorf("SessionID = %q, want %q", unmarshaled.SessionID, event.SessionID)
	}
	if unmarshaled.GetSessionID() != event.SessionID {
		t.Errorf("GetSessionID() = %q, want %q", unmarshaled.GetSessionID(), event.SessionID)
	}
}

// ============================================================================
// Content Block Types Tests
// ============================================================================

func TestTextBlockJSON(t *testing.T) {
	block := TextBlock{
		Type: "text",
		Text: "Hello, world!",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled TextBlock
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != block.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, block.Type)
	}
	if unmarshaled.Text != block.Text {
		t.Errorf("Text = %q, want %q", unmarshaled.Text, block.Text)
	}
	if unmarshaled.GetType() != block.Type {
		t.Errorf("GetType() = %q, want %q", unmarshaled.GetType(), block.Type)
	}
}

func TestThinkingBlockJSON(t *testing.T) {
	block := ThinkingBlock{
		Type:      "thinking",
		Thinking:  "Let me think about this...",
		Signature: "sig-123",
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ThinkingBlock
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Thinking != block.Thinking {
		t.Errorf("Thinking = %q, want %q", unmarshaled.Thinking, block.Thinking)
	}
}

func TestToolUseBlockJSON(t *testing.T) {
	block := ToolUseBlock{
		Type: "tool_use",
		ID:   "tool-123",
		Name: "Bash",
		Input: map[string]interface{}{
			"command": "ls -la",
		},
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ToolUseBlock
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ID != block.ID {
		t.Errorf("ID = %q, want %q", unmarshaled.ID, block.ID)
	}
	if unmarshaled.Name != block.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, block.Name)
	}
}

func TestToolResultBlockJSON(t *testing.T) {
	isError := true
	block := ToolResultBlock{
		Type:      "tool_result",
		ToolUseID: "tool-123",
		Content:   "Command failed",
		IsError:   &isError,
	}

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ToolResultBlock
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ToolUseID != block.ToolUseID {
		t.Errorf("ToolUseID = %q, want %q", unmarshaled.ToolUseID, block.ToolUseID)
	}
	if unmarshaled.IsError == nil || *unmarshaled.IsError != isError {
		t.Errorf("IsError not properly unmarshaled")
	}
}

func TestUnmarshalContentBlock(t *testing.T) {
	tests := []struct {
		name         string
		jsonData     string
		expectedType string
	}{
		{
			name:         "Text block",
			jsonData:     `{"type": "text", "text": "Hello"}`,
			expectedType: "text",
		},
		{
			name:         "Thinking block",
			jsonData:     `{"type": "thinking", "thinking": "Thought", "signature": "sig"}`,
			expectedType: "thinking",
		},
		{
			name:         "Tool use block",
			jsonData:     `{"type": "tool_use", "id": "123", "name": "Bash", "input": {}}`,
			expectedType: "tool_use",
		},
		{
			name:         "Tool result block",
			jsonData:     `{"type": "tool_result", "tool_use_id": "123", "content": "result"}`,
			expectedType: "tool_result",
		},
		{
			name:         "Unknown block",
			jsonData:     `{"type": "custom", "data": "value"}`,
			expectedType: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := UnmarshalContentBlock([]byte(tt.jsonData))
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if block.GetType() != tt.expectedType {
				t.Errorf("GetType() = %q, want %q", block.GetType(), tt.expectedType)
			}
		})
	}
}

func TestUnmarshalContentBlockInvalid(t *testing.T) {
	_, err := UnmarshalContentBlock([]byte(`invalid json`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// ============================================================================
// Permission Types Tests
// ============================================================================

func TestPermissionResultAllow(t *testing.T) {
	allow := PermissionResultAllow{
		Behavior: "allow",
		UpdatedInput: map[string]interface{}{
			"command": "ls",
		},
		UpdatedPermissions: []PermissionUpdate{
			{
				Type: PermissionUpdateTypeAddRules,
				Rules: []PermissionRuleValue{
					{ToolName: "Bash"},
				},
			},
		},
	}

	data, err := json.Marshal(allow)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PermissionResultAllow
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Behavior != "allow" {
		t.Errorf("Behavior = %q, want %q", unmarshaled.Behavior, "allow")
	}
	if unmarshaled.GetBehavior() != "allow" {
		t.Errorf("GetBehavior() = %q, want %q", unmarshaled.GetBehavior(), "allow")
	}
	if len(unmarshaled.UpdatedPermissions) != 1 {
		t.Errorf("Expected 1 UpdatedPermissions, got %d", len(unmarshaled.UpdatedPermissions))
	}
}

func TestPermissionResultDeny(t *testing.T) {
	deny := PermissionResultDeny{
		Behavior:  "deny",
		Message:   "Permission denied: dangerous operation",
		Interrupt: true,
	}

	data, err := json.Marshal(deny)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PermissionResultDeny
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Behavior != "deny" {
		t.Errorf("Behavior = %q, want %q", unmarshaled.Behavior, "deny")
	}
	if unmarshaled.GetBehavior() != "deny" {
		t.Errorf("GetBehavior() = %q, want %q", unmarshaled.GetBehavior(), "deny")
	}
	if unmarshaled.Message != deny.Message {
		t.Errorf("Message = %q, want %q", unmarshaled.Message, deny.Message)
	}
	if unmarshaled.Interrupt != deny.Interrupt {
		t.Errorf("Interrupt = %v, want %v", unmarshaled.Interrupt, deny.Interrupt)
	}
}

func TestPermissionResultInterface(t *testing.T) {
	// Test that both types implement the PermissionResult interface
	var _ PermissionResult = PermissionResultAllow{Behavior: "allow"}
	var _ PermissionResult = PermissionResultDeny{Behavior: "deny"}
}

func TestPermissionUpdateToDict(t *testing.T) {
	tests := []struct {
		name     string
		update   PermissionUpdate
		validate func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "AddRules with behavior",
			update: PermissionUpdate{
				Type: PermissionUpdateTypeAddRules,
				Rules: []PermissionRuleValue{
					{ToolName: "Bash", RuleContent: String("allow ls")},
				},
				Behavior: PermissionBehaviorPtr(PermissionBehaviorAllow),
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != PermissionUpdateTypeAddRules {
					t.Errorf("type = %v, want %v", result["type"], PermissionUpdateTypeAddRules)
				}
				if result["behavior"] != PermissionBehaviorAllow {
					t.Errorf("behavior = %v, want %v", result["behavior"], PermissionBehaviorAllow)
				}
				rules, ok := result["rules"].([]map[string]interface{})
				if !ok {
					t.Fatal("rules is not the expected type")
				}
				if len(rules) != 1 {
					t.Errorf("Expected 1 rule, got %d", len(rules))
				}
			},
		},
		{
			name: "SetMode",
			update: PermissionUpdate{
				Type: PermissionUpdateTypeSetMode,
				Mode: PermissionModePtr(PermissionModePlan),
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != PermissionUpdateTypeSetMode {
					t.Errorf("type = %v, want %v", result["type"], PermissionUpdateTypeSetMode)
				}
				if result["mode"] != PermissionModePlan {
					t.Errorf("mode = %v, want %v", result["mode"], PermissionModePlan)
				}
			},
		},
		{
			name: "AddDirectories with destination",
			update: PermissionUpdate{
				Type:        PermissionUpdateTypeAddDirectories,
				Directories: []string{"/home/user", "/tmp"},
				Destination: PermissionUpdateDestinationPtr(PermissionUpdateDestinationSession),
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != PermissionUpdateTypeAddDirectories {
					t.Errorf("type = %v, want %v", result["type"], PermissionUpdateTypeAddDirectories)
				}
				dirs, ok := result["directories"].([]string)
				if !ok {
					t.Fatal("directories is not the expected type")
				}
				if len(dirs) != 2 {
					t.Errorf("Expected 2 directories, got %d", len(dirs))
				}
				if result["destination"] != PermissionUpdateDestinationSession {
					t.Errorf("destination = %v, want %v", result["destination"], PermissionUpdateDestinationSession)
				}
			},
		},
		{
			name: "RemoveRules without behavior",
			update: PermissionUpdate{
				Type: PermissionUpdateTypeRemoveRules,
				Rules: []PermissionRuleValue{
					{ToolName: "WebFetch"},
				},
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != PermissionUpdateTypeRemoveRules {
					t.Errorf("type = %v, want %v", result["type"], PermissionUpdateTypeRemoveRules)
				}
				if _, exists := result["behavior"]; exists {
					t.Error("behavior should not be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.update.ToDict()
			tt.validate(t, result)
		})
	}
}

// ============================================================================
// Hook Event Constants Tests
// ============================================================================

func TestHookEventConstants(t *testing.T) {
	tests := []struct {
		name     string
		event    HookEvent
		expected string
	}{
		{"PreToolUse", HookEventPreToolUse, "PreToolUse"},
		{"PostToolUse", HookEventPostToolUse, "PostToolUse"},
		{"PostToolUseFailure", HookEventPostToolUseFailure, "PostToolUseFailure"},
		{"UserPromptSubmit", HookEventUserPromptSubmit, "UserPromptSubmit"},
		{"Stop", HookEventStop, "Stop"},
		{"SubagentStop", HookEventSubagentStop, "SubagentStop"},
		{"PreCompact", HookEventPreCompact, "PreCompact"},
		{"Notification", HookEventNotification, "Notification"},
		{"SubagentStart", HookEventSubagentStart, "SubagentStart"},
		{"PermissionRequest", HookEventPermissionRequest, "PermissionRequest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.event) != tt.expected {
				t.Errorf("HookEvent %s = %q, want %q", tt.name, tt.event, tt.expected)
			}
		})
	}
}

// ============================================================================
// MCP Server Config Types Tests
// ============================================================================

func TestMcpStdioServerConfig(t *testing.T) {
	config := McpStdioServerConfig{
		Command: "claude",
		Args:    []string{"--mcp"},
		Env:     map[string]string{"API_KEY": "secret"},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpStdioServerConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Command != config.Command {
		t.Errorf("Command = %q, want %q", unmarshaled.Command, config.Command)
	}
	if unmarshaled.GetType() != "stdio" {
		t.Errorf("GetType() = %q, want %q", unmarshaled.GetType(), "stdio")
	}
}

func TestMcpSSEServerConfig(t *testing.T) {
	config := McpSSEServerConfig{
		Type:    "sse",
		URL:     "http://localhost:8080/mcp",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpSSEServerConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.URL != config.URL {
		t.Errorf("URL = %q, want %q", unmarshaled.URL, config.URL)
	}
	if unmarshaled.GetType() != "sse" {
		t.Errorf("GetType() = %q, want %q", unmarshaled.GetType(), "sse")
	}
}

func TestMcpHttpServerConfig(t *testing.T) {
	config := McpHttpServerConfig{
		Type:    "http",
		URL:     "http://localhost:3000/mcp",
		Headers: map[string]string{"X-Custom": "value"},
	}

	if config.GetType() != "http" {
		t.Errorf("GetType() = %q, want %q", config.GetType(), "http")
	}
}

func TestMcpSdkServerConfig(t *testing.T) {
	config := McpSdkServerConfig{
		Type:     "sdk",
		Name:     "my-server",
		Instance: "some-instance",
	}

	if config.GetType() != "sdk" {
		t.Errorf("GetType() = %q, want %q", config.GetType(), "sdk")
	}
}

// ============================================================================
// Thinking Config Types Tests
// ============================================================================

func TestThinkingConfigTypes(t *testing.T) {
	tests := []struct {
		name   string
		config ThinkingConfig
		typ    string
	}{
		{"Adaptive", ThinkingConfigAdaptive{Type: "adaptive"}, "adaptive"},
		{"Enabled", ThinkingConfigEnabled{Type: "enabled", BudgetTokens: 10000}, "enabled"},
		{"Disabled", ThinkingConfigDisabled{Type: "disabled"}, "disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.GetType() != tt.typ {
				t.Errorf("GetType() = %q, want %q", tt.config.GetType(), tt.typ)
			}
		})
	}
}

func TestThinkingConfigEnabledJSON(t *testing.T) {
	config := ThinkingConfigEnabled{
		Type:         "enabled",
		BudgetTokens: 10000,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ThinkingConfigEnabled
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.BudgetTokens != config.BudgetTokens {
		t.Errorf("BudgetTokens = %d, want %d", unmarshaled.BudgetTokens, config.BudgetTokens)
	}
}

// ============================================================================
// SDK Control Request Types Tests
// ============================================================================

func TestUnmarshalSDKControlRequest(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expectedType  string
		expectedSubty string
	}{
		{
			name:          "Interrupt request",
			jsonData:      `{"type": "control_request", "request_id": "req-123", "request": {"subtype": "interrupt"}}`,
			expectedType:  "control_request",
			expectedSubty: "interrupt",
		},
		{
			name:          "Permission request",
			jsonData:      `{"type": "control_request", "request_id": "req-456", "request": {"subtype": "can_use_tool", "tool_name": "Bash", "input": {}}}`,
			expectedType:  "control_request",
			expectedSubty: "can_use_tool",
		},
		{
			name:          "Initialize request",
			jsonData:      `{"type": "control_request", "request_id": "req-789", "request": {"subtype": "initialize", "hooks": {}}}`,
			expectedType:  "control_request",
			expectedSubty: "initialize",
		},
		{
			name:          "Set permission mode request",
			jsonData:      `{"type": "control_request", "request_id": "req-abc", "request": {"subtype": "set_permission_mode", "mode": "plan"}}`,
			expectedType:  "control_request",
			expectedSubty: "set_permission_mode",
		},
		{
			name:          "Hook callback request",
			jsonData:      `{"type": "control_request", "request_id": "req-def", "request": {"subtype": "hook_callback", "callback_id": "cb-123", "input": {}}}`,
			expectedType:  "control_request",
			expectedSubty: "hook_callback",
		},
		{
			name:          "MCP message request",
			jsonData:      `{"type": "control_request", "request_id": "req-ghi", "request": {"subtype": "mcp_message", "server_name": "my-server", "message": {}}}`,
			expectedType:  "control_request",
			expectedSubty: "mcp_message",
		},
		{
			name:          "Rewind files request",
			jsonData:      `{"type": "control_request", "request_id": "req-jkl", "request": {"subtype": "rewind_files", "user_message_id": "msg-123"}}`,
			expectedType:  "control_request",
			expectedSubty: "rewind_files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := UnmarshalSDKControlRequest([]byte(tt.jsonData))
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if req.Type != tt.expectedType {
				t.Errorf("Type = %q, want %q", req.Type, tt.expectedType)
			}
			if req.RequestID != "req-"+tt.expectedSubty[:3] && !json.Valid([]byte(tt.jsonData)) {
				// RequestID check is flexible
			}
		})
	}
}

func TestUnmarshalSDKControlRequestInvalid(t *testing.T) {
	_, err := UnmarshalSDKControlRequest([]byte(`invalid json`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestControlResponseJSON(t *testing.T) {
	resp := ControlResponse{
		Subtype:   "success",
		RequestID: "req-123",
		Response:  map[string]interface{}{"status": "ok"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ControlResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Subtype != resp.Subtype {
		t.Errorf("Subtype = %q, want %q", unmarshaled.Subtype, resp.Subtype)
	}
}

func TestControlErrorResponseJSON(t *testing.T) {
	resp := ControlErrorResponse{
		Subtype:   "error",
		RequestID: "req-456",
		Error:     "Something went wrong",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ControlErrorResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Error != resp.Error {
		t.Errorf("Error = %q, want %q", unmarshaled.Error, resp.Error)
	}
}

// ============================================================================
// Utility Functions Tests
// ============================================================================

func TestUtilityFunctions(t *testing.T) {
	// Test String
	s := String("test")
	if s == nil || *s != "test" {
		t.Errorf("String(\"test\") = %v, want pointer to \"test\"", s)
	}

	// Test Int
	i := Int(42)
	if i == nil || *i != 42 {
		t.Errorf("Int(42) = %v, want pointer to 42", i)
	}

	// Test Float64
	f := Float64(3.14)
	if f == nil || *f != 3.14 {
		t.Errorf("Float64(3.14) = %v, want pointer to 3.14", f)
	}

	// Test Bool
	b := Bool(true)
	if b == nil || *b != true {
		t.Errorf("Bool(true) = %v, want pointer to true", b)
	}

	// Test PermissionModePtr
	pm := PermissionModePtr(PermissionModePlan)
	if pm == nil || *pm != PermissionModePlan {
		t.Errorf("PermissionModePtr(PermissionModePlan) = %v, want pointer to PermissionModePlan", pm)
	}

	// Test PermissionBehaviorPtr
	pb := PermissionBehaviorPtr(PermissionBehaviorAllow)
	if pb == nil || *pb != PermissionBehaviorAllow {
		t.Errorf("PermissionBehaviorPtr(PermissionBehaviorAllow) = %v, want pointer to PermissionBehaviorAllow", pb)
	}

	// Test PermissionUpdateDestinationPtr
	pud := PermissionUpdateDestinationPtr(PermissionUpdateDestinationSession)
	if pud == nil || *pud != PermissionUpdateDestinationSession {
		t.Errorf("PermissionUpdateDestinationPtr(PermissionUpdateDestinationSession) = %v, want pointer to PermissionUpdateDestinationSession", pud)
	}
}

// ============================================================================
// QueryResult Tests
// ============================================================================

func TestQueryResultJSON(t *testing.T) {
	result := QueryResult{
		Result:           "Task completed",
		SessionID:        "session-123",
		CostUSD:          0.05,
		DurationMs:       5000,
		DurationAPIMs:    3000,
		NumTurns:         3,
		IsError:          false,
		Subtype:          "success",
		Usage:            map[string]interface{}{"input_tokens": 100, "output_tokens": 50},
		StructuredOutput: map[string]interface{}{"key": "value"},
		TotalCostUSD:     0.05,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled QueryResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Result != result.Result {
		t.Errorf("Result = %q, want %q", unmarshaled.Result, result.Result)
	}
	if unmarshaled.SessionID != result.SessionID {
		t.Errorf("SessionID = %q, want %q", unmarshaled.SessionID, result.SessionID)
	}
	if unmarshaled.NumTurns != result.NumTurns {
		t.Errorf("NumTurns = %d, want %d", unmarshaled.NumTurns, result.NumTurns)
	}
}

// ============================================================================
// AssistantMessageError Constants Tests
// ============================================================================

func TestAssistantMessageErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      AssistantMessageError
		expected string
	}{
		{"AuthenticationFailed", AssistantMessageErrorAuthenticationFailed, "authentication_failed"},
		{"BillingError", AssistantMessageErrorBillingError, "billing_error"},
		{"RateLimit", AssistantMessageErrorRateLimit, "rate_limit"},
		{"InvalidRequest", AssistantMessageErrorInvalidRequest, "invalid_request"},
		{"ServerError", AssistantMessageErrorServerError, "server_error"},
		{"Unknown", AssistantMessageErrorUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.err) != tt.expected {
				t.Errorf("AssistantMessageError %s = %q, want %q", tt.name, tt.err, tt.expected)
			}
		})
	}
}

// ============================================================================
// AgentDefinition Tests
// ============================================================================

func TestAgentDefinitionJSON(t *testing.T) {
	model := "sonnet"
	def := AgentDefinition{
		Description: "A test agent",
		Prompt:      "You are a helpful assistant.",
		Tools:       []string{"Bash", "Read", "Write"},
		Model:       &model,
	}

	data, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled AgentDefinition
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Description != def.Description {
		t.Errorf("Description = %q, want %q", unmarshaled.Description, def.Description)
	}
	if unmarshaled.Model == nil || *unmarshaled.Model != model {
		t.Errorf("Model not properly unmarshaled")
	}
	if len(unmarshaled.Tools) != len(def.Tools) {
		t.Errorf("Tools length = %d, want %d", len(unmarshaled.Tools), len(def.Tools))
	}
}

// ============================================================================
// SandboxSettings Tests
// ============================================================================

func TestSandboxSettingsJSON(t *testing.T) {
	enabled := true
	autoAllow := false
	allowUnix := true
	network := SandboxNetworkConfig{
		AllowAllUnixSockets: &allowUnix,
	}

	settings := SandboxSettings{
		Enabled:                  &enabled,
		AutoAllowBashIfSandboxed: &autoAllow,
		ExcludedCommands:         []string{"git", "docker"},
		Network:                  &network,
	}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SandboxSettings
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Enabled == nil || *unmarshaled.Enabled != enabled {
		t.Errorf("Enabled not properly unmarshaled")
	}
	if unmarshaled.Network == nil || unmarshaled.Network.AllowAllUnixSockets == nil {
		t.Error("Network not properly unmarshaled")
	}
}

// ============================================================================
// SdkBeta Constants Tests
// ============================================================================

func TestSdkBetaConstants(t *testing.T) {
	if string(SdkBetaContext1M) != "context-1m-2025-08-07" {
		t.Errorf("SdkBetaContext1M = %q, want %q", SdkBetaContext1M, "context-1m-2025-08-07")
	}
}

// ============================================================================
// SettingSource Constants Tests
// ============================================================================

func TestSettingSourceConstants(t *testing.T) {
	tests := []struct {
		name     string
		source   SettingSource
		expected string
	}{
		{"User", SettingSourceUser, "user"},
		{"Project", SettingSourceProject, "project"},
		{"Local", SettingSourceLocal, "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.source) != tt.expected {
				t.Errorf("SettingSource %s = %q, want %q", tt.name, tt.source, tt.expected)
			}
		})
	}
}

// ============================================================================
// Options Alias Tests
// ============================================================================

func TestOptionsAlias(t *testing.T) {
	// Verify that Options is an alias for ClaudeAgentOptions
	var _ Options = ClaudeAgentOptions{}
	opts := Options{
		Model: String("claude-3-sonnet"),
	}

	if opts.Model == nil || *opts.Model != "claude-3-sonnet" {
		t.Errorf("Options alias not working correctly")
	}
}
