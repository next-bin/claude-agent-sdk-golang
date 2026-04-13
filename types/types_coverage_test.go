// Package types contains comprehensive tests for types.go to achieve 90%+ coverage.
package types

import (
	"encoding/json"
	"testing"
)

// ============================================================================
// McpServerRef and McpServerInlineConfig Tests
// ============================================================================

func TestMcpServerRefIsMcpServerRefOrConfig(t *testing.T) {
	ref := McpServerRef("my-server")
	// The isMcpServerRefOrConfig method is a marker method that satisfies the interface
	ref.isMcpServerRefOrConfig() // This call covers the method

	// Test that it satisfies the interface
	var _ McpServerRefOrConfig = ref

	// Test JSON marshaling/unmarshaling
	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal McpServerRef: %v", err)
	}
	if string(data) != `"my-server"` {
		t.Errorf("McpServerRef JSON = %s, want %q", data, `"my-server"`)
	}

	var unmarshaled McpServerRef
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal McpServerRef: %v", err)
	}
	if unmarshaled != ref {
		t.Errorf("McpServerRef unmarshaled = %q, want %q", unmarshaled, ref)
	}
}

func TestMcpServerInlineConfigIsMcpServerRefOrConfig(t *testing.T) {
	config := McpServerInlineConfig{
		Name:   "inline-server",
		Config: map[string]interface{}{"type": "stdio", "command": "my-cmd"},
	}
	// The isMcpServerRefOrConfig method is a marker method that satisfies the interface
	config.isMcpServerRefOrConfig() // This call covers the method

	// Test that it satisfies the interface
	var _ McpServerRefOrConfig = config

	// Test JSON marshaling
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal McpServerInlineConfig: %v", err)
	}

	var unmarshaled McpServerInlineConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal McpServerInlineConfig: %v", err)
	}
	if unmarshaled.Name != config.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, config.Name)
	}
}

// ============================================================================
// Hook Input GetHookEventName Methods Tests (lines 409-436)
// ============================================================================

func TestHookInputGetHookEventNameMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    HookInput
		expected string
	}{
		{
			name: "PreToolUseHookInput",
			input: PreToolUseHookInput{
				HookEventName: "PreToolUse",
			},
			expected: "PreToolUse",
		},
		{
			name: "PostToolUseHookInput",
			input: PostToolUseHookInput{
				HookEventName: "PostToolUse",
			},
			expected: "PostToolUse",
		},
		{
			name: "PostToolUseFailureHookInput",
			input: PostToolUseFailureHookInput{
				HookEventName: "PostToolUseFailure",
			},
			expected: "PostToolUseFailure",
		},
		{
			name: "UserPromptSubmitHookInput",
			input: UserPromptSubmitHookInput{
				HookEventName: "UserPromptSubmit",
			},
			expected: "UserPromptSubmit",
		},
		{
			name: "StopHookInput",
			input: StopHookInput{
				HookEventName: "Stop",
			},
			expected: "Stop",
		},
		{
			name: "SubagentStopHookInput",
			input: SubagentStopHookInput{
				HookEventName: "SubagentStop",
			},
			expected: "SubagentStop",
		},
		{
			name: "PreCompactHookInput",
			input: PreCompactHookInput{
				HookEventName: "PreCompact",
			},
			expected: "PreCompact",
		},
		{
			name: "NotificationHookInput",
			input: NotificationHookInput{
				HookEventName: "Notification",
			},
			expected: "Notification",
		},
		{
			name: "SubagentStartHookInput",
			input: SubagentStartHookInput{
				HookEventName: "SubagentStart",
			},
			expected: "SubagentStart",
		},
		{
			name: "PermissionRequestHookInput",
			input: PermissionRequestHookInput{
				HookEventName: "PermissionRequest",
			},
			expected: "PermissionRequest",
		},
		{
			name: "SessionStartHookInput",
			input: SessionStartHookInput{
				HookEventName: "SessionStart",
			},
			expected: "SessionStart",
		},
		{
			name: "SessionEndHookInput",
			input: SessionEndHookInput{
				HookEventName: "SessionEnd",
			},
			expected: "SessionEnd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.GetHookEventName() != tt.expected {
				t.Errorf("GetHookEventName() = %q, want %q", tt.input.GetHookEventName(), tt.expected)
			}
		})
	}
}

// ============================================================================
// Hook Specific Output GetHookEventName Methods Tests (lines 512-530)
// ============================================================================

func TestHookSpecificOutputGetHookEventNameMethods(t *testing.T) {
	tests := []struct {
		name     string
		output   HookSpecificOutput
		expected string
	}{
		{
			name: "PreToolUseHookSpecificOutput",
			output: PreToolUseHookSpecificOutput{
				HookEventName: "PreToolUse",
			},
			expected: "PreToolUse",
		},
		{
			name: "PostToolUseHookSpecificOutput",
			output: PostToolUseHookSpecificOutput{
				HookEventName: "PostToolUse",
			},
			expected: "PostToolUse",
		},
		{
			name: "PostToolUseFailureHookSpecificOutput",
			output: PostToolUseFailureHookSpecificOutput{
				HookEventName: "PostToolUseFailure",
			},
			expected: "PostToolUseFailure",
		},
		{
			name: "UserPromptSubmitHookSpecificOutput",
			output: UserPromptSubmitHookSpecificOutput{
				HookEventName: "UserPromptSubmit",
			},
			expected: "UserPromptSubmit",
		},
		{
			name: "SessionStartHookSpecificOutput",
			output: SessionStartHookSpecificOutput{
				HookEventName: "SessionStart",
			},
			expected: "SessionStart",
		},
		{
			name: "NotificationHookSpecificOutput",
			output: NotificationHookSpecificOutput{
				HookEventName: "Notification",
			},
			expected: "Notification",
		},
		{
			name: "SubagentStartHookSpecificOutput",
			output: SubagentStartHookSpecificOutput{
				HookEventName: "SubagentStart",
			},
			expected: "SubagentStart",
		},
		{
			name: "PermissionRequestHookSpecificOutput",
			output: PermissionRequestHookSpecificOutput{
				HookEventName: "PermissionRequest",
			},
			expected: "PermissionRequest",
		},
		{
			name: "SessionEndHookSpecificOutput",
			output: SessionEndHookSpecificOutput{
				HookEventName: "SessionEnd",
			},
			expected: "SessionEnd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.output.GetHookEventName() != tt.expected {
				t.Errorf("GetHookEventName() = %q, want %q", tt.output.GetHookEventName(), tt.expected)
			}
		})
	}
}

// ============================================================================
// MCP Server Config ToMap Tests
// ============================================================================

func TestMcpStdioServerConfigToMap(t *testing.T) {
	tests := []struct {
		name   string
		config McpStdioServerConfig
		check  func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "Full config",
			config: McpStdioServerConfig{
				Command: "claude",
				Args:    []string{"--mcp", "--port", "8080"},
				Env:     map[string]string{"API_KEY": "secret", "DEBUG": "true"},
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != "stdio" {
					t.Errorf("type = %v, want %q", result["type"], "stdio")
				}
				if result["command"] != "claude" {
					t.Errorf("command = %v, want %q", result["command"], "claude")
				}
				args, ok := result["args"].([]string)
				if !ok || len(args) != 3 {
					t.Errorf("args = %v, want 3 elements", result["args"])
				}
				env, ok := result["env"].(map[string]string)
				if !ok || len(env) != 2 {
					t.Errorf("env = %v, want 2 elements", result["env"])
				}
			},
		},
		{
			name: "Minimal config",
			config: McpStdioServerConfig{
				Command: "my-command",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != "stdio" {
					t.Errorf("type = %v, want %q", result["type"], "stdio")
				}
				if result["command"] != "my-command" {
					t.Errorf("command = %v, want %q", result["command"], "my-command")
				}
				if _, exists := result["args"]; exists {
					t.Error("args should not be present for empty slice")
				}
				if _, exists := result["env"]; exists {
					t.Error("env should not be present for empty map")
				}
			},
		},
		{
			name: "Empty args and env",
			config: McpStdioServerConfig{
				Command: "cmd",
				Args:    []string{},
				Env:     map[string]string{},
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if _, exists := result["args"]; exists {
					t.Error("args should not be present for empty slice")
				}
				if _, exists := result["env"]; exists {
					t.Error("env should not be present for empty map")
				}
			},
		},
		{
			name: "Explicit type",
			config: McpStdioServerConfig{
				Type:    "custom-stdio",
				Command: "cmd",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != "custom-stdio" {
					t.Errorf("type = %v, want %q", result["type"], "custom-stdio")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ToMap()
			tt.check(t, result)
		})
	}
}

func TestMcpStdioServerConfigGetTypeEmpty(t *testing.T) {
	config := McpStdioServerConfig{
		Command: "cmd",
	}
	// When Type is empty, GetType should return "stdio"
	if config.GetType() != "stdio" {
		t.Errorf("GetType() = %q, want %q", config.GetType(), "stdio")
	}
}

func TestMcpStdioServerConfigGetTypeNonEmpty(t *testing.T) {
	config := McpStdioServerConfig{
		Type:    "custom",
		Command: "cmd",
	}
	if config.GetType() != "custom" {
		t.Errorf("GetType() = %q, want %q", config.GetType(), "custom")
	}
}

func TestMcpSSEServerConfigToMap(t *testing.T) {
	tests := []struct {
		name   string
		config McpSSEServerConfig
		check  func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "Full config",
			config: McpSSEServerConfig{
				Type:    "sse",
				URL:     "http://localhost:8080/mcp",
				Headers: map[string]string{"Authorization": "Bearer token"},
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != "sse" {
					t.Errorf("type = %v, want %q", result["type"], "sse")
				}
				if result["url"] != "http://localhost:8080/mcp" {
					t.Errorf("url = %v, want %q", result["url"], "http://localhost:8080/mcp")
				}
				headers, ok := result["headers"].(map[string]string)
				if !ok || len(headers) != 1 {
					t.Errorf("headers = %v, want 1 element", result["headers"])
				}
			},
		},
		{
			name: "Minimal config",
			config: McpSSEServerConfig{
				Type: "sse",
				URL:  "http://localhost/mcp",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if _, exists := result["headers"]; exists {
					t.Error("headers should not be present for empty map")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ToMap()
			tt.check(t, result)
		})
	}
}

func TestMcpHttpServerConfigToMap(t *testing.T) {
	tests := []struct {
		name   string
		config McpHttpServerConfig
		check  func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "Full config",
			config: McpHttpServerConfig{
				Type:    "http",
				URL:     "http://localhost:3000/mcp",
				Headers: map[string]string{"X-Custom": "value"},
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if result["type"] != "http" {
					t.Errorf("type = %v, want %q", result["type"], "http")
				}
				if result["url"] != "http://localhost:3000/mcp" {
					t.Errorf("url = %v, want %q", result["url"], "http://localhost:3000/mcp")
				}
			},
		},
		{
			name: "No headers",
			config: McpHttpServerConfig{
				Type: "http",
				URL:  "http://localhost/mcp",
			},
			check: func(t *testing.T, result map[string]interface{}) {
				if _, exists := result["headers"]; exists {
					t.Error("headers should not be present for nil map")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ToMap()
			tt.check(t, result)
		})
	}
}

func TestMcpSdkServerConfigToMap(t *testing.T) {
	config := McpSdkServerConfig{
		Type:     "sdk",
		Name:     "my-sdk-server",
		Instance: "instance-data",
	}

	result := config.ToMap()
	if result["type"] != "sdk" {
		t.Errorf("type = %v, want %q", result["type"], "sdk")
	}
	if result["name"] != "my-sdk-server" {
		t.Errorf("name = %v, want %q", result["name"], "my-sdk-server")
	}
	if result["instance"] != "instance-data" {
		t.Errorf("instance = %v, want %q", result["instance"], "instance-data")
	}
}

// ============================================================================
// UnmarshalContentBlock Edge Cases Tests
// ============================================================================

func TestUnmarshalContentBlockInvalidInnerJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "Invalid JSON structure",
			jsonData: `{}`,
			wantErr:  false, // Will return generic block with empty type
		},
		{
			name:     "Valid type but invalid inner content for text",
			jsonData: `{"type": "text", "text": invalid}`,
			wantErr:  true,
		},
		{
			name:     "Valid type but invalid inner content for thinking",
			jsonData: `{"type": "thinking", "thinking": invalid}`,
			wantErr:  true,
		},
		{
			name:     "Valid type but invalid inner content for tool_use",
			jsonData: `{"type": "tool_use", "id": invalid}`,
			wantErr:  true,
		},
		{
			name:     "Valid type but invalid inner content for tool_result",
			jsonData: `{"type": "tool_result", "tool_use_id": invalid}`,
			wantErr:  true,
		},
		{
			name:     "Unknown type returns generic block",
			jsonData: `{"type": "unknown_type", "custom_field": "value"}`,
			wantErr:  false,
		},
		{
			name:     "Invalid JSON in unknown block",
			jsonData: `{"type": "custom", invalid}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := UnmarshalContentBlock([]byte(tt.jsonData))
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if block == nil {
					t.Error("Expected non-nil block")
				}
			}
		})
	}
}

// ============================================================================
// GenericContentBlock GetType Edge Cases
// ============================================================================

func TestGenericContentBlockGetTypeNoTypeField(t *testing.T) {
	block := GenericContentBlock{
		Data: map[string]interface{}{
			"custom_field": "value",
		},
	}
	// When "type" key doesn't exist, GetType should return empty string
	if block.GetType() != "" {
		t.Errorf("GetType() = %q, want %q", block.GetType(), "")
	}
}

func TestGenericContentBlockGetTypeNonStringType(t *testing.T) {
	block := GenericContentBlock{
		Data: map[string]interface{}{
			"type": 123, // Non-string type
		},
	}
	// When "type" is not a string, GetType should return empty string
	if block.GetType() != "" {
		t.Errorf("GetType() = %q, want %q", block.GetType(), "")
	}
}

func TestGenericContentBlockGetTypeValidString(t *testing.T) {
	block := GenericContentBlock{
		Data: map[string]interface{}{
			"type":         "custom_type",
			"custom_field": "value",
		},
	}
	if block.GetType() != "custom_type" {
		t.Errorf("GetType() = %q, want %q", block.GetType(), "custom_type")
	}
}

// ============================================================================
// SystemMessage and RateLimitEvent GetSessionID Tests
// ============================================================================

func TestSystemMessageGetSessionID(t *testing.T) {
	msg := SystemMessage{
		Subtype: "init",
		Data:    map[string]interface{}{"key": "value"},
	}
	// SystemMessage.GetSessionID should return empty string
	if msg.GetSessionID() != "" {
		t.Errorf("GetSessionID() = %q, want %q", msg.GetSessionID(), "")
	}
}

func TestRateLimitEventGetSessionID(t *testing.T) {
	event := RateLimitEvent{
		RateLimitInfo: RateLimitInfo{
			Status: RateLimitStatusAllowed,
		},
		UUID:      "uuid-123",
		SessionID: "session-456",
	}
	if event.GetSessionID() != "session-456" {
		t.Errorf("GetSessionID() = %q, want %q", event.GetSessionID(), "session-456")
	}
}

// ============================================================================
// UnmarshalSDKControlRequest Additional Subtypes Tests
// ============================================================================

func TestUnmarshalSDKControlRequestMcpReconnect(t *testing.T) {
	jsonData := `{"type": "control_request", "request_id": "req-123", "request": {"subtype": "mcp_reconnect", "serverName": "my-server"}}`
	req, err := UnmarshalSDKControlRequest([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.Type != "control_request" {
		t.Errorf("Type = %q, want %q", req.Type, "control_request")
	}
	if req.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want %q", req.RequestID, "req-123")
	}

	mcpReq, ok := req.Request.(SDKControlMcpReconnectRequest)
	if !ok {
		t.Fatalf("Request is not SDKControlMcpReconnectRequest, got %T", req.Request)
	}
	if mcpReq.Subtype != "mcp_reconnect" {
		t.Errorf("Subtype = %q, want %q", mcpReq.Subtype, "mcp_reconnect")
	}
	if mcpReq.ServerName != "my-server" {
		t.Errorf("ServerName = %q, want %q", mcpReq.ServerName, "my-server")
	}
}

func TestUnmarshalSDKControlRequestMcpToggle(t *testing.T) {
	jsonData := `{"type": "control_request", "request_id": "req-456", "request": {"subtype": "mcp_toggle", "serverName": "another-server", "enabled": true}}`
	req, err := UnmarshalSDKControlRequest([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	mcpReq, ok := req.Request.(SDKControlMcpToggleRequest)
	if !ok {
		t.Fatalf("Request is not SDKControlMcpToggleRequest, got %T", req.Request)
	}
	if mcpReq.ServerName != "another-server" {
		t.Errorf("ServerName = %q, want %q", mcpReq.ServerName, "another-server")
	}
	if mcpReq.Enabled != true {
		t.Errorf("Enabled = %v, want %v", mcpReq.Enabled, true)
	}
}

func TestUnmarshalSDKControlRequestStopTask(t *testing.T) {
	jsonData := `{"type": "control_request", "request_id": "req-789", "request": {"subtype": "stop_task", "task_id": "task-123"}}`
	req, err := UnmarshalSDKControlRequest([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	stopReq, ok := req.Request.(SDKControlStopTaskRequest)
	if !ok {
		t.Fatalf("Request is not SDKControlStopTaskRequest, got %T", req.Request)
	}
	if stopReq.TaskID != "task-123" {
		t.Errorf("TaskID = %q, want %q", stopReq.TaskID, "task-123")
	}
}

func TestUnmarshalSDKControlRequestUnknownSubtype(t *testing.T) {
	jsonData := `{"type": "control_request", "request_id": "req-abc", "request": {"subtype": "unknown_subtype", "custom_field": "value"}}`
	req, err := UnmarshalSDKControlRequest([]byte(jsonData))
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Unknown subtypes should be kept as json.RawMessage
	if req.Type != "control_request" {
		t.Errorf("Type = %q, want %q", req.Type, "control_request")
	}
}

func TestUnmarshalSDKControlRequestInvalidInnerJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "Invalid outer JSON",
			jsonData: `invalid json`,
			wantErr:  true,
		},
		{
			name:     "Invalid request JSON",
			jsonData: `{"type": "control_request", "request_id": "req", "request": invalid}`,
			wantErr:  true,
		},
		{
			name:     "Missing subtype",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"field": "value"}}`,
			wantErr:  false, // Empty subtype will fall through to default case
		},
		{
			name:     "Invalid subtype JSON",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid interrupt request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "interrupt", invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid can_use_tool request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "can_use_tool", "tool_name": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid initialize request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "initialize", "hooks": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid set_permission_mode request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "set_permission_mode", "mode": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid hook_callback request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "hook_callback", "callback_id": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid mcp_message request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "mcp_message", "server_name": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid rewind_files request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "rewind_files", "user_message_id": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid mcp_reconnect request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "mcp_reconnect", "serverName": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid mcp_toggle request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "mcp_toggle", "serverName": invalid}}`,
			wantErr:  true,
		},
		{
			name:     "Invalid stop_task request",
			jsonData: `{"type": "control_request", "request_id": "req", "request": {"subtype": "stop_task", "task_id": invalid}}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalSDKControlRequest([]byte(tt.jsonData))
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// ============================================================================
// PermissionUpdate ToDict Additional Cases Tests
// ============================================================================

func TestPermissionUpdateToDictReplaceRules(t *testing.T) {
	update := PermissionUpdate{
		Type: PermissionUpdateTypeReplaceRules,
		Rules: []PermissionRuleValue{
			{ToolName: "Bash"},
			{ToolName: "Read", RuleContent: String("allow read")},
		},
		Behavior: PermissionBehaviorPtr(PermissionBehaviorAllow),
	}

	result := update.ToDict()
	if result["type"] != PermissionUpdateTypeReplaceRules {
		t.Errorf("type = %v, want %v", result["type"], PermissionUpdateTypeReplaceRules)
	}

	rules, ok := result["rules"].([]map[string]interface{})
	if !ok {
		t.Fatal("rules is not the expected type")
	}
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	// First rule should not have ruleContent
	if _, exists := rules[0]["ruleContent"]; exists {
		t.Error("First rule should not have ruleContent")
	}

	// Second rule should have ruleContent
	if rules[1]["ruleContent"] != "allow read" {
		t.Errorf("ruleContent = %v, want %q", rules[1]["ruleContent"], "allow read")
	}
}

func TestPermissionUpdateToDictRemoveRules(t *testing.T) {
	update := PermissionUpdate{
		Type: PermissionUpdateTypeRemoveRules,
		Rules: []PermissionRuleValue{
			{ToolName: "WebFetch"},
		},
	}

	result := update.ToDict()
	if result["type"] != PermissionUpdateTypeRemoveRules {
		t.Errorf("type = %v, want %v", result["type"], PermissionUpdateTypeRemoveRules)
	}

	rules, ok := result["rules"].([]map[string]interface{})
	if !ok {
		t.Fatal("rules is not the expected type")
	}
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}
	if rules[0]["toolName"] != "WebFetch" {
		t.Errorf("toolName = %v, want %q", rules[0]["toolName"], "WebFetch")
	}
}

func TestPermissionUpdateToDictRemoveDirectories(t *testing.T) {
	update := PermissionUpdate{
		Type:        PermissionUpdateTypeRemoveDirectories,
		Directories: []string{"/tmp", "/var"},
	}

	result := update.ToDict()
	if result["type"] != PermissionUpdateTypeRemoveDirectories {
		t.Errorf("type = %v, want %v", result["type"], PermissionUpdateTypeRemoveDirectories)
	}

	dirs, ok := result["directories"].([]string)
	if !ok {
		t.Fatal("directories is not the expected type")
	}
	if len(dirs) != 2 {
		t.Errorf("Expected 2 directories, got %d", len(dirs))
	}
}

func TestPermissionUpdateToDictNoRules(t *testing.T) {
	update := PermissionUpdate{
		Type: PermissionUpdateTypeAddRules,
	}

	result := update.ToDict()
	if _, exists := result["rules"]; exists {
		t.Error("rules should not be present when nil")
	}
	if _, exists := result["behavior"]; exists {
		t.Error("behavior should not be present when nil")
	}
}

func TestPermissionUpdateToDictNoMode(t *testing.T) {
	update := PermissionUpdate{
		Type: PermissionUpdateTypeSetMode,
	}

	result := update.ToDict()
	if _, exists := result["mode"]; exists {
		t.Error("mode should not be present when nil")
	}
}

func TestPermissionUpdateToDictNoDirectories(t *testing.T) {
	update := PermissionUpdate{
		Type: PermissionUpdateTypeAddDirectories,
	}

	result := update.ToDict()
	if _, exists := result["directories"]; exists {
		t.Error("directories should not be present when nil")
	}
}

func TestPermissionUpdateToDictNoDestination(t *testing.T) {
	update := PermissionUpdate{
		Type: PermissionUpdateTypeAddRules,
	}

	result := update.ToDict()
	if _, exists := result["destination"]; exists {
		t.Error("destination should not be present when nil")
	}
}

// ============================================================================
// RateLimitInfo and RateLimitStatus Tests
// ============================================================================

func TestRateLimitStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   RateLimitStatus
		expected string
	}{
		{"Allowed", RateLimitStatusAllowed, "allowed"},
		{"AllowedWarning", RateLimitStatusAllowedWarning, "allowed_warning"},
		{"Rejected", RateLimitStatusRejected, "rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("RateLimitStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestRateLimitTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		typ      RateLimitType
		expected string
	}{
		{"FiveHour", RateLimitTypeFiveHour, "five_hour"},
		{"SevenDay", RateLimitTypeSevenDay, "seven_day"},
		{"SevenDayOpus", RateLimitTypeSevenDayOpus, "seven_day_opus"},
		{"SevenDaySonnet", RateLimitTypeSevenDaySonnet, "seven_day_sonnet"},
		{"Overage", RateLimitTypeOverage, "overage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.typ) != tt.expected {
				t.Errorf("RateLimitType %s = %q, want %q", tt.name, tt.typ, tt.expected)
			}
		})
	}
}

func TestRateLimitInfoJSON(t *testing.T) {
	resetsAt := 1234567890
	rateLimitType := RateLimitTypeFiveHour
	utilization := 0.75
	overageStatus := RateLimitStatusAllowed
	overageResetsAt := 1234567900
	overageDisabledReason := "billing_inactive"

	info := RateLimitInfo{
		Status:                RateLimitStatusAllowedWarning,
		ResetsAt:              &resetsAt,
		RateLimitType:         &rateLimitType,
		Utilization:           &utilization,
		OverageStatus:         &overageStatus,
		OverageResetsAt:       &overageResetsAt,
		OverageDisabledReason: &overageDisabledReason,
		Raw:                   map[string]interface{}{"extra": "data"},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled RateLimitInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Status != info.Status {
		t.Errorf("Status = %q, want %q", unmarshaled.Status, info.Status)
	}
	if unmarshaled.ResetsAt == nil || *unmarshaled.ResetsAt != resetsAt {
		t.Errorf("ResetsAt not properly unmarshaled")
	}
	if unmarshaled.Utilization == nil || *unmarshaled.Utilization != utilization {
		t.Errorf("Utilization not properly unmarshaled")
	}
}

func TestRateLimitEventJSON(t *testing.T) {
	info := RateLimitInfo{
		Status: RateLimitStatusRejected,
	}
	event := RateLimitEvent{
		RateLimitInfo: info,
		UUID:          "uuid-123",
		SessionID:     "session-456",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled RateLimitEvent
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.UUID != event.UUID {
		t.Errorf("UUID = %q, want %q", unmarshaled.UUID, event.UUID)
	}
	if unmarshaled.SessionID != event.SessionID {
		t.Errorf("SessionID = %q, want %q", unmarshaled.SessionID, event.SessionID)
	}
}

// ============================================================================
// MCP Server Status Types Tests
// ============================================================================

func TestMcpServerConnectionStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   McpServerConnectionStatus
		expected string
	}{
		{"Connected", McpServerStatusConnected, "connected"},
		{"Failed", McpServerStatusFailed, "failed"},
		{"NeedsAuth", McpServerStatusNeedsAuth, "needs-auth"},
		{"Pending", McpServerStatusPending, "pending"},
		{"Disabled", McpServerStatusDisabled, "disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("McpServerConnectionStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestMcpSdkServerConfigStatusJSON(t *testing.T) {
	config := McpSdkServerConfigStatus{
		Type: "sdk",
		Name: "my-sdk-server",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpSdkServerConfigStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != config.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, config.Type)
	}
}

func TestMcpClaudeAIProxyServerConfigJSON(t *testing.T) {
	config := McpClaudeAIProxyServerConfig{
		Type: "claudeai-proxy",
		URL:  "https://claude.ai/proxy",
		ID:   "proxy-123",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpClaudeAIProxyServerConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ID != config.ID {
		t.Errorf("ID = %q, want %q", unmarshaled.ID, config.ID)
	}
}

func TestMcpToolAnnotationsJSON(t *testing.T) {
	readOnly := true
	destructive := false
	openWorld := true

	annotations := McpToolAnnotations{
		ReadOnly:    &readOnly,
		Destructive: &destructive,
		OpenWorld:   &openWorld,
	}

	data, err := json.Marshal(annotations)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpToolAnnotations
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ReadOnly == nil || *unmarshaled.ReadOnly != readOnly {
		t.Errorf("ReadOnly not properly unmarshaled")
	}
}

func TestMcpToolInfoJSON(t *testing.T) {
	desc := "A test tool"
	readOnly := true
	tool := McpToolInfo{
		Name:        "test_tool",
		Description: &desc,
		Annotations: &McpToolAnnotations{
			ReadOnly: &readOnly,
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpToolInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != tool.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, tool.Name)
	}
}

func TestMcpServerInfoJSON(t *testing.T) {
	info := McpServerInfo{
		Name:    "my-mcp-server",
		Version: "1.0.0",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpServerInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Version != info.Version {
		t.Errorf("Version = %q, want %q", unmarshaled.Version, info.Version)
	}
}

func TestMcpServerStatusJSON(t *testing.T) {
	errMsg := "Connection failed"
	scope := "project"
	serverInfo := McpServerInfo{
		Name:    "server",
		Version: "1.0",
	}

	status := McpServerStatus{
		Name:       "my-server",
		Status:     McpServerStatusFailed,
		ServerInfo: &serverInfo,
		Error:      &errMsg,
		Config:     map[string]interface{}{"type": "stdio"},
		Scope:      &scope,
		Tools: []McpToolInfo{
			{Name: "tool1"},
			{Name: "tool2"},
		},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpServerStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != status.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, status.Name)
	}
	if unmarshaled.Status != status.Status {
		t.Errorf("Status = %q, want %q", unmarshaled.Status, status.Status)
	}
	if unmarshaled.Error == nil || *unmarshaled.Error != errMsg {
		t.Errorf("Error not properly unmarshaled")
	}
	if len(unmarshaled.Tools) != 2 {
		t.Errorf("Tools length = %d, want 2", len(unmarshaled.Tools))
	}
}

func TestMcpStatusResponseJSON(t *testing.T) {
	response := McpStatusResponse{
		McpServers: []McpServerStatus{
			{Name: "server1", Status: McpServerStatusConnected},
			{Name: "server2", Status: McpServerStatusPending},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled McpStatusResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(unmarshaled.McpServers) != 2 {
		t.Errorf("McpServers length = %d, want 2", len(unmarshaled.McpServers))
	}
}

// ============================================================================
// Task Notification Status Tests
// ============================================================================

func TestTaskNotificationStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskNotificationStatus
		expected string
	}{
		{"Completed", TaskNotificationStatusCompleted, "completed"},
		{"Failed", TaskNotificationStatusFailed, "failed"},
		{"Stopped", TaskNotificationStatusStopped, "stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("TaskNotificationStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestTaskUsageJSON(t *testing.T) {
	usage := TaskUsage{
		TotalTokens: 10000,
		ToolUses:    5,
		DurationMs:  30000,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled TaskUsage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.TotalTokens != usage.TotalTokens {
		t.Errorf("TotalTokens = %d, want %d", unmarshaled.TotalTokens, usage.TotalTokens)
	}
	if unmarshaled.ToolUses != usage.ToolUses {
		t.Errorf("ToolUses = %d, want %d", unmarshaled.ToolUses, usage.ToolUses)
	}
}

func TestTaskStartedMessageJSON(t *testing.T) {
	toolUseID := "tool-123"
	taskType := "research"

	msg := TaskStartedMessage{
		SystemMessage: SystemMessage{
			Subtype: "task_started",
			Data:    map[string]interface{}{"key": "value"},
		},
		TaskID:      "task-456",
		Description: "A test task",
		UUID:        "uuid-789",
		SessionID:   "session-abc",
		ToolUseID:   &toolUseID,
		TaskType:    &taskType,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled TaskStartedMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.TaskID != msg.TaskID {
		t.Errorf("TaskID = %q, want %q", unmarshaled.TaskID, msg.TaskID)
	}
	if unmarshaled.Description != msg.Description {
		t.Errorf("Description = %q, want %q", unmarshaled.Description, msg.Description)
	}
}

func TestTaskProgressMessageJSON(t *testing.T) {
	toolUseID := "tool-123"
	lastToolName := "Bash"

	msg := TaskProgressMessage{
		SystemMessage: SystemMessage{
			Subtype: "task_progress",
		},
		TaskID:      "task-456",
		Description: "Progress update",
		Usage: TaskUsage{
			TotalTokens: 5000,
			ToolUses:    3,
			DurationMs:  15000,
		},
		UUID:         "uuid-789",
		SessionID:    "session-abc",
		ToolUseID:    &toolUseID,
		LastToolName: &lastToolName,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled TaskProgressMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Usage.TotalTokens != msg.Usage.TotalTokens {
		t.Errorf("Usage.TotalTokens = %d, want %d", unmarshaled.Usage.TotalTokens, msg.Usage.TotalTokens)
	}
	if unmarshaled.LastToolName == nil || *unmarshaled.LastToolName != lastToolName {
		t.Errorf("LastToolName not properly unmarshaled")
	}
}

func TestTaskNotificationMessageJSON(t *testing.T) {
	toolUseID := "tool-123"
	usage := TaskUsage{
		TotalTokens: 10000,
		ToolUses:    10,
		DurationMs:  60000,
	}

	msg := TaskNotificationMessage{
		SystemMessage: SystemMessage{
			Subtype: "task_notification",
		},
		TaskID:     "task-456",
		Status:     TaskNotificationStatusCompleted,
		OutputFile: "/path/to/output.txt",
		Summary:    "Task completed successfully",
		UUID:       "uuid-789",
		SessionID:  "session-abc",
		ToolUseID:  &toolUseID,
		Usage:      &usage,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled TaskNotificationMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Status != msg.Status {
		t.Errorf("Status = %q, want %q", unmarshaled.Status, msg.Status)
	}
	if unmarshaled.OutputFile != msg.OutputFile {
		t.Errorf("OutputFile = %q, want %q", unmarshaled.OutputFile, msg.OutputFile)
	}
	if unmarshaled.Usage == nil || unmarshaled.Usage.TotalTokens != usage.TotalTokens {
		t.Errorf("Usage not properly unmarshaled")
	}
}

// ============================================================================
// SDKSessionInfo and SessionMessage Tests
// ============================================================================

func TestSDKSessionInfoJSON(t *testing.T) {
	createdAt := int64(1234567890)
	fileSize := int64(1024)
	customTitle := "My Session"
	tag := "important"
	firstPrompt := "Hello"
	gitBranch := "main"
	cwd := "/home/user"

	info := SDKSessionInfo{
		SessionID:    "session-123",
		Summary:      "Test session",
		LastModified: 1234567900,
		CreatedAt:    &createdAt,
		FileSize:     &fileSize,
		CustomTitle:  &customTitle,
		Tag:          &tag,
		FirstPrompt:  &firstPrompt,
		GitBranch:    &gitBranch,
		CWD:          &cwd,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKSessionInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.SessionID != info.SessionID {
		t.Errorf("SessionID = %q, want %q", unmarshaled.SessionID, info.SessionID)
	}
	if unmarshaled.CreatedAt == nil || *unmarshaled.CreatedAt != createdAt {
		t.Errorf("CreatedAt not properly unmarshaled")
	}
	if unmarshaled.CustomTitle == nil || *unmarshaled.CustomTitle != customTitle {
		t.Errorf("CustomTitle not properly unmarshaled")
	}
}

func TestSessionMessageJSON(t *testing.T) {
	parentToolUseID := "tool-123"

	msg := SessionMessage{
		Type:            "user",
		UUID:            "uuid-456",
		SessionID:       "session-789",
		Message:         map[string]interface{}{"content": "Hello"},
		ParentToolUseID: &parentToolUseID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SessionMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != msg.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, msg.Type)
	}
	if unmarshaled.ParentToolUseID == nil || *unmarshaled.ParentToolUseID != parentToolUseID {
		t.Errorf("ParentToolUseID not properly unmarshaled")
	}
}

// ============================================================================
// Context Usage Types Tests
// ============================================================================

func TestContextUsageCategoryJSON(t *testing.T) {
	isDeferred := true
	category := ContextUsageCategory{
		Name:       "system_prompt",
		Tokens:     5000,
		Color:      "#FF0000",
		IsDeferred: &isDeferred,
	}

	data, err := json.Marshal(category)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ContextUsageCategory
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Name != category.Name {
		t.Errorf("Name = %q, want %q", unmarshaled.Name, category.Name)
	}
	if unmarshaled.Tokens != category.Tokens {
		t.Errorf("Tokens = %d, want %d", unmarshaled.Tokens, category.Tokens)
	}
}

func TestContextUsageResponseJSON(t *testing.T) {
	threshold := 80
	apiUsage := map[string]interface{}{"input": 100, "output": 50}

	response := ContextUsageResponse{
		Categories: []ContextUsageCategory{
			{Name: "system", Tokens: 1000, Color: "#000"},
			{Name: "tools", Tokens: 2000, Color: "#FFF"},
		},
		TotalTokens:          5000,
		MaxTokens:            100000,
		RawMaxTokens:         200000,
		Percentage:           5.0,
		Model:                "claude-3-sonnet",
		IsAutoCompactEnabled: true,
		MemoryFiles: []map[string]interface{}{
			{"path": "/memory/file1.txt"},
		},
		MCPTools: []map[string]interface{}{
			{"name": "tool1"},
		},
		Agents: []map[string]interface{}{
			{"name": "agent1"},
		},
		GridRows: [][]map[string]interface{}{
			{{"cell": "value"}},
		},
		AutoCompactThreshold: &threshold,
		DeferredBuiltinTools: []map[string]interface{}{
			{"name": "deferred_tool"},
		},
		SystemTools: []map[string]interface{}{
			{"name": "system_tool"},
		},
		SystemPromptSections: []map[string]interface{}{
			{"section": "intro"},
		},
		SlashCommands: map[string]interface{}{
			"help": "Show help",
		},
		Skills: map[string]interface{}{
			"skill1": "enabled",
		},
		MessageBreakdown: map[string]interface{}{
			"user":      10,
			"assistant": 20,
		},
		APIUsage: &apiUsage,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ContextUsageResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.TotalTokens != response.TotalTokens {
		t.Errorf("TotalTokens = %d, want %d", unmarshaled.TotalTokens, response.TotalTokens)
	}
	if unmarshaled.Model != response.Model {
		t.Errorf("Model = %q, want %q", unmarshaled.Model, response.Model)
	}
	if len(unmarshaled.Categories) != 2 {
		t.Errorf("Categories length = %d, want 2", len(unmarshaled.Categories))
	}
}

// ============================================================================
// ForkSessionResult Tests
// ============================================================================

func TestForkSessionResultJSON(t *testing.T) {
	result := ForkSessionResult{
		SessionID: "forked-session-123",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ForkSessionResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.SessionID != result.SessionID {
		t.Errorf("SessionID = %q, want %q", unmarshaled.SessionID, result.SessionID)
	}
}

// ============================================================================
// SdkPluginConfig Tests
// ============================================================================

func TestSdkPluginConfigJSON(t *testing.T) {
	config := SdkPluginConfig{
		Type: "local",
		Path: "/path/to/plugin",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SdkPluginConfig
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Path != config.Path {
		t.Errorf("Path = %q, want %q", unmarshaled.Path, config.Path)
	}
}

// ============================================================================
// SandboxIgnoreViolations Tests
// ============================================================================

func TestSandboxIgnoreViolationsJSON(t *testing.T) {
	violations := SandboxIgnoreViolations{
		File:    []string{"/tmp/sensitive.txt"},
		Network: []string{"internal-api.com"},
	}

	data, err := json.Marshal(violations)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SandboxIgnoreViolations
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(unmarshaled.File) != 1 {
		t.Errorf("File length = %d, want 1", len(unmarshaled.File))
	}
	if len(unmarshaled.Network) != 1 {
		t.Errorf("Network length = %d, want 1", len(unmarshaled.Network))
	}
}

// ============================================================================
// TaskBudget Tests
// ============================================================================

func TestTaskBudgetJSON(t *testing.T) {
	budget := TaskBudget{
		Total: 50000,
	}

	data, err := json.Marshal(budget)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled TaskBudget
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Total != budget.Total {
		t.Errorf("Total = %d, want %d", unmarshaled.Total, budget.Total)
	}
}

// ============================================================================
// SystemPromptPreset, ToolsPreset, SystemPromptFile Tests
// ============================================================================

func TestSystemPromptPresetJSON(t *testing.T) {
	appendText := "Custom append text"
	preset := SystemPromptPreset{
		Type:   "preset",
		Preset: "claude_code",
		Append: &appendText,
	}

	data, err := json.Marshal(preset)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SystemPromptPreset
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Preset != preset.Preset {
		t.Errorf("Preset = %q, want %q", unmarshaled.Preset, preset.Preset)
	}
	if unmarshaled.Append == nil || *unmarshaled.Append != appendText {
		t.Errorf("Append not properly unmarshaled")
	}
}

func TestToolsPresetJSON(t *testing.T) {
	preset := ToolsPreset{
		Type:   "preset",
		Preset: "claude_code",
	}

	data, err := json.Marshal(preset)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ToolsPreset
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != preset.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, preset.Type)
	}
}

func TestSystemPromptFileJSON(t *testing.T) {
	file := SystemPromptFile{
		Type: "file",
		Path: "/path/to/prompt.txt",
	}

	data, err := json.Marshal(file)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SystemPromptFile
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Path != file.Path {
		t.Errorf("Path = %q, want %q", unmarshaled.Path, file.Path)
	}
}

// ============================================================================
// SDKControlResponse Tests
// ============================================================================

func TestSDKControlResponseJSON(t *testing.T) {
	resp := SDKControlResponse{
		Type: "control_response",
		Response: ControlResponse{
			Subtype:   "success",
			RequestID: "req-123",
			Response:  map[string]interface{}{"status": "ok"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKControlResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != resp.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, resp.Type)
	}
}

func TestSDKControlResponseErrorJSON(t *testing.T) {
	resp := SDKControlResponse{
		Type: "control_response",
		Response: ControlErrorResponse{
			Subtype:   "error",
			RequestID: "req-456",
			Error:     "Something went wrong",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKControlResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != resp.Type {
		t.Errorf("Type = %q, want %q", unmarshaled.Type, resp.Type)
	}
}

// ============================================================================
// Additional Hook Input Tests with Optional Fields
// ============================================================================

func TestPreToolUseHookInputWithOptionalFields(t *testing.T) {
	permMode := "auto"
	agentID := "agent-123"
	agentType := "research"

	input := PreToolUseHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
			PermissionMode: &permMode,
		},
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput:     map[string]interface{}{"command": "ls"},
		ToolUseID:     "tool-456",
		AgentID:       &agentID,
		AgentType:     &agentType,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PreToolUseHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.PermissionMode == nil || *unmarshaled.PermissionMode != permMode {
		t.Errorf("PermissionMode not properly unmarshaled")
	}
	if unmarshaled.AgentID == nil || *unmarshaled.AgentID != agentID {
		t.Errorf("AgentID not properly unmarshaled")
	}
}

func TestPostToolUseHookInputWithOptionalFields(t *testing.T) {
	agentID := "agent-123"
	agentType := "task"

	input := PostToolUseHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName: "PostToolUse",
		ToolName:      "Read",
		ToolInput:     map[string]interface{}{"file_path": "/test.txt"},
		ToolResponse:  "file contents",
		ToolUseID:     "tool-789",
		AgentID:       &agentID,
		AgentType:     &agentType,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PostToolUseHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.AgentType == nil || *unmarshaled.AgentType != agentType {
		t.Errorf("AgentType not properly unmarshaled")
	}
}

func TestPermissionRequestHookInputWithOptionalFields(t *testing.T) {
	agentID := "agent-123"
	agentType := "task"

	input := PermissionRequestHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName:         "PermissionRequest",
		ToolName:              "Bash",
		ToolInput:             map[string]interface{}{"command": "rm -rf"},
		PermissionSuggestions: []interface{}{map[string]interface{}{"behavior": "deny"}},
		AgentID:               &agentID,
		AgentType:             &agentType,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled PermissionRequestHookInput
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.AgentID == nil || *unmarshaled.AgentID != agentID {
		t.Errorf("AgentID not properly unmarshaled")
	}
}

// ============================================================================
// PermissionUpdateDestination Constants Tests
// ============================================================================

func TestAllPermissionUpdateDestinationConstants(t *testing.T) {
	constants := []struct {
		name     string
		constant PermissionUpdateDestination
		expected string
	}{
		{"UserSettings", PermissionUpdateDestinationUserSettings, "userSettings"},
		{"ProjectSettings", PermissionUpdateDestinationProjectSettings, "projectSettings"},
		{"LocalSettings", PermissionUpdateDestinationLocalSettings, "localSettings"},
		{"Session", PermissionUpdateDestinationSession, "session"},
	}

	for _, c := range constants {
		t.Run(c.name, func(t *testing.T) {
			if string(c.constant) != c.expected {
				t.Errorf("%s = %q, want %q", c.name, c.constant, c.expected)
			}
		})
	}
}

// ============================================================================
// PermissionMode Additional Constants Tests
// ============================================================================

func TestAllPermissionModeConstants(t *testing.T) {
	constants := []struct {
		name     string
		constant PermissionMode
		expected string
	}{
		{"Default", PermissionModeDefault, "default"},
		{"AcceptEdits", PermissionModeAcceptEdits, "acceptEdits"},
		{"Plan", PermissionModePlan, "plan"},
		{"BypassPermissions", PermissionModeBypassPermissions, "bypassPermissions"},
		{"DontAsk", PermissionModeDontAsk, "dontAsk"},
		{"Auto", PermissionModeAuto, "auto"},
	}

	for _, c := range constants {
		t.Run(c.name, func(t *testing.T) {
			if string(c.constant) != c.expected {
				t.Errorf("%s = %q, want %q", c.name, c.constant, c.expected)
			}
		})
	}
}

// ============================================================================
// Hook Event Additional Constants Tests
// ============================================================================

func TestAllHookEventConstants(t *testing.T) {
	constants := []struct {
		name     string
		constant HookEvent
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
		{"SessionStart", HookEventSessionStart, "SessionStart"},
		{"SessionEnd", HookEventSessionEnd, "SessionEnd"},
	}

	for _, c := range constants {
		t.Run(c.name, func(t *testing.T) {
			if string(c.constant) != c.expected {
				t.Errorf("%s = %q, want %q", c.name, c.constant, c.expected)
			}
		})
	}
}

// ============================================================================
// ToolPermissionContext Tests
// ============================================================================

func TestToolPermissionContextJSON(t *testing.T) {
	ctx := ToolPermissionContext{
		Signal: nil,
		Suggestions: []PermissionUpdate{
			{Type: PermissionUpdateTypeAddRules},
		},
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled ToolPermissionContext
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(unmarshaled.Suggestions) != 1 {
		t.Errorf("Suggestions length = %d, want 1", len(unmarshaled.Suggestions))
	}
}

// ============================================================================
// HookMatcher Tests
// ============================================================================

func TestHookMatcherJSON(t *testing.T) {
	timeout := 120.0
	matcher := HookMatcher{
		Matcher: "Bash|Read|Write",
		Hooks:   []HookCallback{},
		Timeout: &timeout,
	}

	data, err := json.Marshal(matcher)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled HookMatcher
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Matcher != matcher.Matcher {
		t.Errorf("Matcher = %q, want %q", unmarshaled.Matcher, matcher.Matcher)
	}
	if unmarshaled.Timeout == nil || *unmarshaled.Timeout != timeout {
		t.Errorf("Timeout not properly unmarshaled")
	}
}

// ============================================================================
// Additional SDK Control Request Types JSON Tests
// ============================================================================

func TestSDKControlInterruptRequestJSON(t *testing.T) {
	req := SDKControlInterruptRequest{
		Subtype: "interrupt",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKControlInterruptRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Subtype != req.Subtype {
		t.Errorf("Subtype = %q, want %q", unmarshaled.Subtype, req.Subtype)
	}
}

func TestSDKControlPermissionRequestJSON(t *testing.T) {
	blockedPath := "/blocked/path"
	req := SDKControlPermissionRequest{
		Subtype:               "can_use_tool",
		ToolName:              "Bash",
		Input:                 map[string]interface{}{"command": "ls"},
		PermissionSuggestions: []interface{}{map[string]interface{}{"behavior": "allow"}},
		BlockedPath:           &blockedPath,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKControlPermissionRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ToolName != req.ToolName {
		t.Errorf("ToolName = %q, want %q", unmarshaled.ToolName, req.ToolName)
	}
	if unmarshaled.BlockedPath == nil || *unmarshaled.BlockedPath != blockedPath {
		t.Errorf("BlockedPath not properly unmarshaled")
	}
}

func TestSDKControlInitializeRequestJSON(t *testing.T) {
	req := SDKControlInitializeRequest{
		Subtype: "initialize",
		Hooks: map[HookEvent]interface{}{
			HookEventPreToolUse: map[string]interface{}{"matcher": "Bash"},
		},
		Agents: map[string]map[string]interface{}{
			"research": map[string]interface{}{"model": "sonnet"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKControlInitializeRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Subtype != req.Subtype {
		t.Errorf("Subtype = %q, want %q", unmarshaled.Subtype, req.Subtype)
	}
}

func TestSDKHookCallbackRequestJSON(t *testing.T) {
	toolUseID := "tool-123"
	req := SDKHookCallbackRequest{
		Subtype:    "hook_callback",
		CallbackID: "cb-456",
		Input:      map[string]interface{}{"event": "PreToolUse"},
		ToolUseID:  &toolUseID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKHookCallbackRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.CallbackID != req.CallbackID {
		t.Errorf("CallbackID = %q, want %q", unmarshaled.CallbackID, req.CallbackID)
	}
}

func TestSDKControlMcpMessageRequestJSON(t *testing.T) {
	req := SDKControlMcpMessageRequest{
		Subtype:    "mcp_message",
		ServerName: "my-server",
		Message:    map[string]interface{}{"method": "tools/list"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKControlMcpMessageRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.ServerName != req.ServerName {
		t.Errorf("ServerName = %q, want %q", unmarshaled.ServerName, req.ServerName)
	}
}

func TestSDKControlRewindFilesRequestJSON(t *testing.T) {
	req := SDKControlRewindFilesRequest{
		Subtype:       "rewind_files",
		UserMessageID: "msg-123",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var unmarshaled SDKControlRewindFilesRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.UserMessageID != req.UserMessageID {
		t.Errorf("UserMessageID = %q, want %q", unmarshaled.UserMessageID, req.UserMessageID)
	}
}
