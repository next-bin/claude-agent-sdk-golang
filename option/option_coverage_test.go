// Package option provides comprehensive test coverage for the functional options.
package option

import (
	"errors"
	"reflect"
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// RequestConfig Tests
// ============================================================================

func TestNewRequestConfig(t *testing.T) {
	tests := []struct {
		name    string
		opts    []RequestOption
		wantErr bool
	}{
		{
			name:    "empty options",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "single valid option",
			opts:    []RequestOption{WithModel("sonnet")},
			wantErr: false,
		},
		{
			name:    "multiple valid options",
			opts:    []RequestOption{WithModel("sonnet"), WithMaxTurns(5)},
			wantErr: false,
		},
		{
			name: "option that returns error",
			opts: []RequestOption{func(c *RequestConfig) error {
				return errors.New("test error")
			}},
			wantErr: true,
		},
		{
			name: "first option returns error",
			opts: []RequestOption{
				func(c *RequestConfig) error { return errors.New("error1") },
				WithModel("sonnet"),
			},
			wantErr: true,
		},
		{
			name: "middle option returns error",
			opts: []RequestOption{
				WithModel("sonnet"),
				func(c *RequestConfig) error { return errors.New("error2") },
				WithMaxTurns(5),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewRequestConfig(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRequestConfig() expected error, got nil")
				}
				if config != nil {
					t.Errorf("NewRequestConfig() expected nil config on error, got %v", config)
				}
			} else {
				if err != nil {
					t.Errorf("NewRequestConfig() unexpected error: %v", err)
				}
				if config == nil {
					t.Errorf("NewRequestConfig() expected config, got nil")
				}
			}
		})
	}
}

func TestRequestConfigApply(t *testing.T) {
	tests := []struct {
		name    string
		initial *RequestConfig
		opts    []RequestOption
		wantErr bool
	}{
		{
			name:    "apply to nil config should panic or work",
			initial: nil,
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "apply empty options",
			initial: &RequestConfig{},
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "apply single option",
			initial: &RequestConfig{},
			opts:    []RequestOption{WithModel("opus")},
			wantErr: false,
		},
		{
			name:    "apply multiple options",
			initial: &RequestConfig{},
			opts: []RequestOption{
				WithModel("sonnet"),
				WithMaxTurns(10),
				WithMaxBudgetUSD(5.0),
			},
			wantErr: false,
		},
		{
			name:    "apply option that returns error",
			initial: &RequestConfig{},
			opts: []RequestOption{
				func(c *RequestConfig) error { return errors.New("apply error") },
			},
			wantErr: true,
		},
		{
			name:    "apply stops on first error",
			initial: &RequestConfig{},
			opts: []RequestOption{
				WithModel("sonnet"),
				func(c *RequestConfig) error { return errors.New("stop here") },
				WithMaxTurns(5), // This should not be applied
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle nil config case - this would panic, so we need to skip
			if tt.initial == nil && len(tt.opts) > 0 {
				t.Skip("cannot apply options to nil config")
			}

			if tt.initial == nil {
				return // Nothing to test
			}

			err := tt.initial.Apply(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Apply() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Apply() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRequestConfigApplyChained(t *testing.T) {
	config := &RequestConfig{}

	// Apply first batch
	err := config.Apply(WithModel("sonnet"), WithMaxTurns(5))
	if err != nil {
		t.Fatalf("First Apply failed: %v", err)
	}

	// Verify first batch
	if config.Model == nil || *config.Model != "sonnet" {
		t.Errorf("Expected Model='sonnet', got %v", config.Model)
	}
	if config.MaxTurns == nil || *config.MaxTurns != 5 {
		t.Errorf("Expected MaxTurns=5, got %v", config.MaxTurns)
	}

	// Apply second batch
	err = config.Apply(WithSystemPrompt("test"), WithMaxBudgetUSD(10.0))
	if err != nil {
		t.Fatalf("Second Apply failed: %v", err)
	}

	// Verify both batches
	if config.SystemPrompt != "test" {
		t.Errorf("Expected SystemPrompt='test', got %v", config.SystemPrompt)
	}
	if config.MaxBudgetUSD == nil || *config.MaxBudgetUSD != 10.0 {
		t.Errorf("Expected MaxBudgetUSD=10.0, got %v", config.MaxBudgetUSD)
	}
	// Previous values should still be there
	if config.Model == nil || *config.Model != "sonnet" {
		t.Errorf("Model should still be 'sonnet'")
	}
}

// ============================================================================
// Core Options Tests
// ============================================================================

func TestWithSystemPrompt(t *testing.T) {
	tests := []struct {
		name   string
		prompt interface{}
	}{
		{
			name:   "string prompt",
			prompt: "You are a helpful assistant",
		},
		{
			name:   "empty string prompt",
			prompt: "",
		},
		{
			name: "system prompt preset",
			prompt: types.SystemPromptPreset{
				Type:   "preset",
				Preset: "claude_code",
			},
		},
		{
			name: "system prompt preset with append",
			prompt: types.SystemPromptPreset{
				Type:   "preset",
				Preset: "claude_code",
				Append: types.String("extra instructions"),
			},
		},
		{
			name: "system prompt file",
			prompt: types.SystemPromptFile{
				Type: "file",
				Path: "/path/to/prompt.txt",
			},
		},
		{
			name:   "nil prompt",
			prompt: nil,
		},
		{
			name:   "map prompt (invalid but accepted)",
			prompt: map[string]string{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithSystemPrompt(tt.prompt)
			err := opt(config)

			if err != nil {
				t.Errorf("WithSystemPrompt() returned error: %v", err)
			}
			if !reflect.DeepEqual(config.SystemPrompt, tt.prompt) {
				t.Errorf("Expected SystemPrompt=%v, got %v", tt.prompt, config.SystemPrompt)
			}
		})
	}
}

func TestWithModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{name: "opus model", model: "opus"},
		{name: "sonnet model", model: "sonnet"},
		{name: "haiku model", model: "haiku"},
		{name: "concrete model", model: "claude-opus-4-6"},
		{name: "empty model", model: ""},
		{name: "custom model", model: "custom-model-name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithModel(tt.model)
			err := opt(config)

			if err != nil {
				t.Errorf("WithModel() returned error: %v", err)
			}
			if config.Model == nil {
				t.Errorf("Expected Model to be set, got nil")
			} else if *config.Model != tt.model {
				t.Errorf("Expected Model='%s', got '%s'", tt.model, *config.Model)
			}
		})
	}
}

func TestWithFallbackModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{name: "fallback to opus", model: "opus"},
		{name: "fallback to sonnet", model: "sonnet"},
		{name: "fallback empty", model: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithFallbackModel(tt.model)
			err := opt(config)

			if err != nil {
				t.Errorf("WithFallbackModel() returned error: %v", err)
			}
			if config.FallbackModel == nil {
				t.Errorf("Expected FallbackModel to be set, got nil")
			} else if *config.FallbackModel != tt.model {
				t.Errorf("Expected FallbackModel='%s', got '%s'", tt.model, *config.FallbackModel)
			}
		})
	}
}

func TestWithMaxTurns(t *testing.T) {
	tests := []struct {
		name  string
		turns int
	}{
		{name: "zero turns", turns: 0},
		{name: "one turn", turns: 1},
		{name: "five turns", turns: 5},
		{name: "negative turns", turns: -1},
		{name: "large turns", turns: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithMaxTurns(tt.turns)
			err := opt(config)

			if err != nil {
				t.Errorf("WithMaxTurns() returned error: %v", err)
			}
			if config.MaxTurns == nil {
				t.Errorf("Expected MaxTurns to be set, got nil")
			} else if *config.MaxTurns != tt.turns {
				t.Errorf("Expected MaxTurns=%d, got %d", tt.turns, *config.MaxTurns)
			}
		})
	}
}

func TestWithMaxBudgetUSD(t *testing.T) {
	tests := []struct {
		name   string
		budget float64
	}{
		{name: "zero budget", budget: 0.0},
		{name: "small budget", budget: 0.5},
		{name: "medium budget", budget: 5.0},
		{name: "large budget", budget: 100.0},
		{name: "negative budget", budget: -10.0},
		{name: "very precise budget", budget: 1.234567},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithMaxBudgetUSD(tt.budget)
			err := opt(config)

			if err != nil {
				t.Errorf("WithMaxBudgetUSD() returned error: %v", err)
			}
			if config.MaxBudgetUSD == nil {
				t.Errorf("Expected MaxBudgetUSD to be set, got nil")
			} else if *config.MaxBudgetUSD != tt.budget {
				t.Errorf("Expected MaxBudgetUSD=%f, got %f", tt.budget, *config.MaxBudgetUSD)
			}
		})
	}
}

func TestWithPermissionMode(t *testing.T) {
	tests := []struct {
		name string
		mode types.PermissionMode
	}{
		{name: "default mode", mode: types.PermissionModeDefault},
		{name: "acceptEdits mode", mode: types.PermissionModeAcceptEdits},
		{name: "plan mode", mode: types.PermissionModePlan},
		{name: "bypassPermissions mode", mode: types.PermissionModeBypassPermissions},
		{name: "dontAsk mode", mode: types.PermissionModeDontAsk},
		{name: "auto mode", mode: types.PermissionModeAuto},
		{name: "custom mode", mode: types.PermissionMode("custom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithPermissionMode(tt.mode)
			err := opt(config)

			if err != nil {
				t.Errorf("WithPermissionMode() returned error: %v", err)
			}
			if config.PermissionMode == nil {
				t.Errorf("Expected PermissionMode to be set, got nil")
			} else if *config.PermissionMode != tt.mode {
				t.Errorf("Expected PermissionMode='%s', got '%s'", tt.mode, *config.PermissionMode)
			}
		})
	}
}

func TestWithContinueConversation(t *testing.T) {
	config := &RequestConfig{}
	opt := WithContinueConversation()
	err := opt(config)

	if err != nil {
		t.Errorf("WithContinueConversation() returned error: %v", err)
	}
	if !config.ContinueConversation {
		t.Errorf("Expected ContinueConversation=true, got %v", config.ContinueConversation)
	}

	// Apply again to verify it stays true
	err = opt(config)
	if err != nil {
		t.Errorf("Second WithContinueConversation() returned error: %v", err)
	}
	if !config.ContinueConversation {
		t.Errorf("ContinueConversation should still be true after second apply")
	}
}

func TestWithResume(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{name: "valid session ID", sessionID: "session-123-abc"},
		{name: "empty session ID", sessionID: ""},
		{name: "UUID session ID", sessionID: "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithResume(tt.sessionID)
			err := opt(config)

			if err != nil {
				t.Errorf("WithResume() returned error: %v", err)
			}
			if config.Resume == nil {
				t.Errorf("Expected Resume to be set, got nil")
			} else if *config.Resume != tt.sessionID {
				t.Errorf("Expected Resume='%s', got '%s'", tt.sessionID, *config.Resume)
			}
		})
	}
}

func TestWithSessionID(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
	}{
		{name: "valid session ID", sessionID: "my-session-id"},
		{name: "empty session ID", sessionID: ""},
		{name: "numeric session ID", sessionID: "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithSessionID(tt.sessionID)
			err := opt(config)

			if err != nil {
				t.Errorf("WithSessionID() returned error: %v", err)
			}
			if config.SessionID == nil {
				t.Errorf("Expected SessionID to be set, got nil")
			} else if *config.SessionID != tt.sessionID {
				t.Errorf("Expected SessionID='%s', got '%s'", tt.sessionID, *config.SessionID)
			}
		})
	}
}

// ============================================================================
// Tool Options Tests
// ============================================================================

func TestWithTools(t *testing.T) {
	tests := []struct {
		name  string
		tools interface{}
	}{
		{
			name:  "string slice tools",
			tools: []string{"Bash", "Read", "Write"},
		},
		{
			name:  "empty string slice",
			tools: []string{},
		},
		{
			name:  "nil tools",
			tools: nil,
		},
		{
			name: "tools preset",
			tools: types.ToolsPreset{
				Type:   "preset",
				Preset: "claude_code",
			},
		},
		{
			name:  "single tool",
			tools: []string{"Bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithTools(tt.tools)
			err := opt(config)

			if err != nil {
				t.Errorf("WithTools() returned error: %v", err)
			}
			if !reflect.DeepEqual(config.Tools, tt.tools) {
				t.Errorf("Expected Tools=%v, got %v", tt.tools, config.Tools)
			}
		})
	}
}

func TestWithAllowedTools(t *testing.T) {
	tests := []struct {
		name  string
		tools []string
	}{
		{
			name:  "multiple allowed tools",
			tools: []string{"Bash", "Read", "Write", "Edit"},
		},
		{
			name:  "single allowed tool",
			tools: []string{"Bash"},
		},
		{
			name:  "empty allowed tools",
			tools: []string{},
		},
		{
			name:  "nil allowed tools",
			tools: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithAllowedTools(tt.tools)
			err := opt(config)

			if err != nil {
				t.Errorf("WithAllowedTools() returned error: %v", err)
			}
			// Compare slices
			if len(config.AllowedTools) != len(tt.tools) {
				t.Errorf("Expected AllowedTools length=%d, got %d", len(tt.tools), len(config.AllowedTools))
			}
			for i, tool := range tt.tools {
				if i >= len(config.AllowedTools) || config.AllowedTools[i] != tool {
					t.Errorf("Expected AllowedTools[%d]='%s'", i, tool)
				}
			}
		})
	}
}

func TestWithDisallowedTools(t *testing.T) {
	tests := []struct {
		name  string
		tools []string
	}{
		{
			name:  "multiple disallowed tools",
			tools: []string{"Bash", "WebFetch"},
		},
		{
			name:  "single disallowed tool",
			tools: []string{"Bash"},
		},
		{
			name:  "empty disallowed tools",
			tools: []string{},
		},
		{
			name:  "nil disallowed tools",
			tools: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithDisallowedTools(tt.tools)
			err := opt(config)

			if err != nil {
				t.Errorf("WithDisallowedTools() returned error: %v", err)
			}
			if len(config.DisallowedTools) != len(tt.tools) {
				t.Errorf("Expected DisallowedTools length=%d, got %d", len(tt.tools), len(config.DisallowedTools))
			}
			for i, tool := range tt.tools {
				if i >= len(config.DisallowedTools) || config.DisallowedTools[i] != tool {
					t.Errorf("Expected DisallowedTools[%d]='%s'", i, tool)
				}
			}
		})
	}
}

// ============================================================================
// MCP Options Tests
// ============================================================================

func TestWithMCPServers(t *testing.T) {
	tests := []struct {
		name    string
		servers interface{}
	}{
		{
			name: "map of stdio configs",
			servers: map[string]types.McpStdioServerConfig{
				"test-server": {
					Command: "test-command",
					Args:    []string{"arg1", "arg2"},
				},
			},
		},
		{
			name: "map of SSE configs",
			servers: map[string]types.McpSSEServerConfig{
				"sse-server": {
					Type: "sse",
					URL:  "http://localhost:8080/sse",
				},
			},
		},
		{
			name:    "string path",
			servers: "/path/to/mcp/config.json",
		},
		{
			name:    "empty string",
			servers: "",
		},
		{
			name:    "nil servers",
			servers: nil,
		},
		{
			name:    "empty map",
			servers: map[string]types.McpStdioServerConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithMCPServers(tt.servers)
			err := opt(config)

			if err != nil {
				t.Errorf("WithMCPServers() returned error: %v", err)
			}
			if !reflect.DeepEqual(config.MCPServers, tt.servers) {
				t.Errorf("Expected MCPServers=%v, got %v", tt.servers, config.MCPServers)
			}
		})
	}
}

// ============================================================================
// Hook Options Tests
// ============================================================================

func TestWithHooks(t *testing.T) {
	tests := []struct {
		name  string
		hooks map[types.HookEvent][]types.HookMatcher
	}{
		{
			name: "single hook event",
			hooks: map[types.HookEvent][]types.HookMatcher{
				types.HookEventPreToolUse: []types.HookMatcher{
					{Matcher: "Bash"},
				},
			},
		},
		{
			name: "multiple hook events",
			hooks: map[types.HookEvent][]types.HookMatcher{
				types.HookEventPreToolUse: []types.HookMatcher{
					{Matcher: "Bash"},
				},
				types.HookEventPostToolUse: []types.HookMatcher{
					{Matcher: "Read|Write"},
				},
			},
		},
		{
			name:  "nil hooks",
			hooks: nil,
		},
		{
			name:  "empty hooks",
			hooks: map[types.HookEvent][]types.HookMatcher{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithHooks(tt.hooks)
			err := opt(config)

			if err != nil {
				t.Errorf("WithHooks() returned error: %v", err)
			}
			if config.Hooks == nil && tt.hooks != nil {
				t.Errorf("Expected Hooks to be set, got nil")
			}
			if len(config.Hooks) != len(tt.hooks) {
				t.Errorf("Expected Hooks length=%d, got %d", len(tt.hooks), len(config.Hooks))
			}
		})
	}
}

// ============================================================================
// Environment Options Tests
// ============================================================================

func TestWithCWD(t *testing.T) {
	tests := []struct {
		name string
		dir  interface{}
	}{
		{name: "string path", dir: "/home/user/project"},
		{name: "relative path", dir: "./subdir"},
		{name: "empty string", dir: ""},
		{name: "nil dir", dir: nil},
		{name: "map (invalid but accepted)", dir: map[string]string{"path": "/test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithCWD(tt.dir)
			err := opt(config)

			if err != nil {
				t.Errorf("WithCWD() returned error: %v", err)
			}
			if !reflect.DeepEqual(config.CWD, tt.dir) {
				t.Errorf("Expected CWD=%v, got %v", tt.dir, config.CWD)
			}
		})
	}
}

func TestWithCLIPath(t *testing.T) {
	tests := []struct {
		name string
		path interface{}
	}{
		{name: "absolute path", path: "/usr/local/bin/claude"},
		{name: "relative path", path: "./claude"},
		{name: "empty string", path: ""},
		{name: "nil path", path: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithCLIPath(tt.path)
			err := opt(config)

			if err != nil {
				t.Errorf("WithCLIPath() returned error: %v", err)
			}
			if !reflect.DeepEqual(config.CLIPath, tt.path) {
				t.Errorf("Expected CLIPath=%v, got %v", tt.path, config.CLIPath)
			}
		})
	}
}

func TestWithEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "multiple env vars",
			env: map[string]string{
				"API_KEY":    "secret",
				"DEBUG_MODE": "true",
			},
		},
		{
			name: "single env var",
			env: map[string]string{
				"HOME": "/home/user",
			},
		},
		{
			name: "empty env vars",
			env:  map[string]string{},
		},
		{
			name: "nil env vars",
			env:  nil,
		},
		{
			name: "env with empty value",
			env: map[string]string{
				"EMPTY_VAR": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithEnv(tt.env)
			err := opt(config)

			if err != nil {
				t.Errorf("WithEnv() returned error: %v", err)
			}
			if config.Env == nil && tt.env != nil {
				t.Errorf("Expected Env to be set, got nil")
			}
			if len(config.Env) != len(tt.env) {
				t.Errorf("Expected Env length=%d, got %d", len(tt.env), len(config.Env))
			}
		})
	}
}

func TestWithBetas(t *testing.T) {
	tests := []struct {
		name  string
		betas []types.SdkBeta
	}{
		{
			name:  "single beta",
			betas: []types.SdkBeta{types.SdkBetaContext1M},
		},
		{
			name: "multiple betas",
			betas: []types.SdkBeta{
				types.SdkBetaContext1M,
				types.SdkBeta("custom-beta"),
			},
		},
		{
			name:  "empty betas",
			betas: []types.SdkBeta{},
		},
		{
			name:  "nil betas",
			betas: nil,
		},
		{
			name:  "custom beta string",
			betas: []types.SdkBeta{types.SdkBeta("experimental-feature")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithBetas(tt.betas)
			err := opt(config)

			if err != nil {
				t.Errorf("WithBetas() returned error: %v", err)
			}
			if len(config.Betas) != len(tt.betas) {
				t.Errorf("Expected Betas length=%d, got %d", len(tt.betas), len(config.Betas))
			}
		})
	}
}

// ============================================================================
// Thinking Options Tests
// ============================================================================

func TestWithThinking(t *testing.T) {
	tests := []struct {
		name   string
		config types.ThinkingConfig
	}{
		{
			name:   "adaptive thinking",
			config: types.ThinkingConfigAdaptive{Type: "adaptive"},
		},
		{
			name:   "enabled thinking with budget",
			config: types.ThinkingConfigEnabled{Type: "enabled", BudgetTokens: 10000},
		},
		{
			name:   "disabled thinking",
			config: types.ThinkingConfigDisabled{Type: "disabled"},
		},
		{
			name:   "enabled with zero budget",
			config: types.ThinkingConfigEnabled{Type: "enabled", BudgetTokens: 0},
		},
		{
			name:   "enabled with large budget",
			config: types.ThinkingConfigEnabled{Type: "enabled", BudgetTokens: 100000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithThinking(tt.config)
			err := opt(config)

			if err != nil {
				t.Errorf("WithThinking() returned error: %v", err)
			}
			if config.Thinking == nil {
				t.Errorf("Expected Thinking to be set, got nil")
			}
			if config.Thinking.GetType() != tt.config.GetType() {
				t.Errorf("Expected Thinking type='%s', got '%s'", tt.config.GetType(), config.Thinking.GetType())
			}
		})
	}
}

func TestWithEffort(t *testing.T) {
	tests := []struct {
		name   string
		effort string
	}{
		{name: "low effort", effort: "low"},
		{name: "medium effort", effort: "medium"},
		{name: "high effort", effort: "high"},
		{name: "max effort", effort: "max"},
		{name: "empty effort", effort: ""},
		{name: "custom effort", effort: "custom-value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithEffort(tt.effort)
			err := opt(config)

			if err != nil {
				t.Errorf("WithEffort() returned error: %v", err)
			}
			if config.Effort == nil {
				t.Errorf("Expected Effort to be set, got nil")
			} else if *config.Effort != tt.effort {
				t.Errorf("Expected Effort='%s', got '%s'", tt.effort, *config.Effort)
			}
		})
	}
}

// ============================================================================
// Output Options Tests
// ============================================================================

func TestWithOutputFormat(t *testing.T) {
	tests := []struct {
		name   string
		format map[string]interface{}
	}{
		{
			name: "json schema format",
			format: map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
		{
			name: "simple format",
			format: map[string]interface{}{
				"type": "text",
			},
		},
		{
			name:   "empty format",
			format: map[string]interface{}{},
		},
		{
			name:   "nil format",
			format: nil,
		},
		{
			name: "complex nested format",
			format: map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name":   "result",
					"strict": true,
					"schema": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
						"required":   []string{"status"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithOutputFormat(tt.format)
			err := opt(config)

			if err != nil {
				t.Errorf("WithOutputFormat() returned error: %v", err)
			}
			if config.OutputFormat == nil && tt.format != nil {
				t.Errorf("Expected OutputFormat to be set, got nil")
			}
			if len(config.OutputFormat) != len(tt.format) {
				t.Errorf("Expected OutputFormat length=%d, got %d", len(tt.format), len(config.OutputFormat))
			}
		})
	}
}

// ============================================================================
// Advanced Options Tests
// ============================================================================

func TestWithAgents(t *testing.T) {
	tests := []struct {
		name   string
		agents map[string]types.AgentDefinition
	}{
		{
			name: "single agent",
			agents: map[string]types.AgentDefinition{
				"researcher": {
					Description: "Research agent",
					Prompt:      "You are a researcher",
					Tools:       []string{"WebSearch", "WebFetch"},
				},
			},
		},
		{
			name: "multiple agents",
			agents: map[string]types.AgentDefinition{
				"researcher": {
					Description: "Research agent",
					Prompt:      "You are a researcher",
				},
				"coder": {
					Description: "Coding agent",
					Prompt:      "You are a coder",
					Model:       types.String("opus"),
				},
			},
		},
		{
			name:   "empty agents",
			agents: map[string]types.AgentDefinition{},
		},
		{
			name:   "nil agents",
			agents: nil,
		},
		{
			name: "agent with all fields",
			agents: map[string]types.AgentDefinition{
				"full-agent": {
					Description: "Full featured agent",
					Prompt:      "Complete agent",
					Tools:       []string{"Bash", "Read"},
					Model:       types.String("sonnet"),
					Skills:      []string{"skill1", "skill2"},
					Memory:      types.String("user"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithAgents(tt.agents)
			err := opt(config)

			if err != nil {
				t.Errorf("WithAgents() returned error: %v", err)
			}
			if config.Agents == nil && tt.agents != nil {
				t.Errorf("Expected Agents to be set, got nil")
			}
			if len(config.Agents) != len(tt.agents) {
				t.Errorf("Expected Agents length=%d, got %d", len(tt.agents), len(config.Agents))
			}
		})
	}
}

func TestWithSettingSources(t *testing.T) {
	tests := []struct {
		name    string
		sources []types.SettingSource
	}{
		{
			name:    "single source",
			sources: []types.SettingSource{types.SettingSourceUser},
		},
		{
			name:    "multiple sources",
			sources: []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal},
		},
		{
			name:    "empty sources",
			sources: []types.SettingSource{},
		},
		{
			name:    "nil sources",
			sources: nil,
		},
		{
			name:    "custom source",
			sources: []types.SettingSource{types.SettingSource("custom")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithSettingSources(tt.sources)
			err := opt(config)

			if err != nil {
				t.Errorf("WithSettingSources() returned error: %v", err)
			}
			if len(config.SettingSources) != len(tt.sources) {
				t.Errorf("Expected SettingSources length=%d, got %d", len(tt.sources), len(config.SettingSources))
			}
		})
	}
}

func TestWithSandbox(t *testing.T) {
	tests := []struct {
		name    string
		sandbox *types.SandboxSettings
	}{
		{
			name: "enabled sandbox",
			sandbox: &types.SandboxSettings{
				Enabled: types.Bool(true),
			},
		},
		{
			name: "disabled sandbox",
			sandbox: &types.SandboxSettings{
				Enabled: types.Bool(false),
			},
		},
		{
			name: "sandbox with network config",
			sandbox: &types.SandboxSettings{
				Enabled: types.Bool(true),
				Network: &types.SandboxNetworkConfig{
					AllowLocalBinding: types.Bool(true),
				},
			},
		},
		{
			name: "sandbox with excluded commands",
			sandbox: &types.SandboxSettings{
				ExcludedCommands: []string{"git", "docker"},
			},
		},
		{
			name:    "nil sandbox",
			sandbox: nil,
		},
		{
			name:    "empty sandbox settings",
			sandbox: &types.SandboxSettings{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithSandbox(tt.sandbox)
			err := opt(config)

			if err != nil {
				t.Errorf("WithSandbox() returned error: %v", err)
			}
			if config.Sandbox != tt.sandbox {
				t.Errorf("Expected Sandbox=%v, got %v", tt.sandbox, config.Sandbox)
			}
		})
	}
}

func TestWithPlugins(t *testing.T) {
	tests := []struct {
		name    string
		plugins []types.SdkPluginConfig
	}{
		{
			name: "single plugin",
			plugins: []types.SdkPluginConfig{
				{Type: "local", Path: "/path/to/plugin"},
			},
		},
		{
			name: "multiple plugins",
			plugins: []types.SdkPluginConfig{
				{Type: "local", Path: "/plugin1"},
				{Type: "local", Path: "/plugin2"},
			},
		},
		{
			name:    "empty plugins",
			plugins: []types.SdkPluginConfig{},
		},
		{
			name:    "nil plugins",
			plugins: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithPlugins(tt.plugins)
			err := opt(config)

			if err != nil {
				t.Errorf("WithPlugins() returned error: %v", err)
			}
			if len(config.Plugins) != len(tt.plugins) {
				t.Errorf("Expected Plugins length=%d, got %d", len(tt.plugins), len(config.Plugins))
			}
		})
	}
}

func TestWithFileCheckpointing(t *testing.T) {
	config := &RequestConfig{}
	opt := WithFileCheckpointing()
	err := opt(config)

	if err != nil {
		t.Errorf("WithFileCheckpointing() returned error: %v", err)
	}
	if !config.EnableFileCheckpointing {
		t.Errorf("Expected EnableFileCheckpointing=true, got %v", config.EnableFileCheckpointing)
	}

	// Apply again
	err = opt(config)
	if err != nil {
		t.Errorf("Second WithFileCheckpointing() returned error: %v", err)
	}
	if !config.EnableFileCheckpointing {
		t.Errorf("EnableFileCheckpointing should still be true")
	}
}

func TestWithTaskBudget(t *testing.T) {
	tests := []struct {
		name   string
		budget *types.TaskBudget
	}{
		{
			name:   "budget with tokens",
			budget: &types.TaskBudget{Total: 100000},
		},
		{
			name:   "budget with zero tokens",
			budget: &types.TaskBudget{Total: 0},
		},
		{
			name:   "large budget",
			budget: &types.TaskBudget{Total: 1000000},
		},
		{
			name:   "nil budget",
			budget: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RequestConfig{}
			opt := WithTaskBudget(tt.budget)
			err := opt(config)

			if err != nil {
				t.Errorf("WithTaskBudget() returned error: %v", err)
			}
			if config.TaskBudget != tt.budget {
				t.Errorf("Expected TaskBudget=%v, got %v", tt.budget, config.TaskBudget)
			}
		})
	}
}

// ============================================================================
// Combined Options Tests
// ============================================================================

func TestCombinedOptions(t *testing.T) {
	config, err := NewRequestConfig(
		WithModel("sonnet"),
		WithSystemPrompt("You are helpful"),
		WithMaxTurns(10),
		WithMaxBudgetUSD(5.0),
		WithPermissionMode(types.PermissionModeAuto),
		WithTools([]string{"Bash", "Read"}),
		WithAllowedTools([]string{"Bash"}),
		WithDisallowedTools([]string{"WebFetch"}),
		WithEnv(map[string]string{"DEBUG": "true"}),
		WithEffort("high"),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig with combined options failed: %v", err)
	}

	// Verify all options were applied
	if config.Model == nil || *config.Model != "sonnet" {
		t.Errorf("Model not set correctly")
	}
	if config.SystemPrompt != "You are helpful" {
		t.Errorf("SystemPrompt not set correctly")
	}
	if config.MaxTurns == nil || *config.MaxTurns != 10 {
		t.Errorf("MaxTurns not set correctly")
	}
	if config.MaxBudgetUSD == nil || *config.MaxBudgetUSD != 5.0 {
		t.Errorf("MaxBudgetUSD not set correctly")
	}
	if config.PermissionMode == nil || *config.PermissionMode != types.PermissionModeAuto {
		t.Errorf("PermissionMode not set correctly")
	}
	if config.Effort == nil || *config.Effort != "high" {
		t.Errorf("Effort not set correctly")
	}
}

func TestOptionComposition(t *testing.T) {
	// Create base options
	baseOpts := []RequestOption{
		WithModel("sonnet"),
		WithMaxTurns(5),
	}

	// Create extra options
	extraOpts := []RequestOption{
		WithSystemPrompt("extra prompt"),
		WithMaxBudgetUSD(10.0),
	}

	// Combine options
	allOpts := append(baseOpts, extraOpts...)

	config, err := NewRequestConfig(allOpts...)
	if err != nil {
		t.Fatalf("NewRequestConfig with composed options failed: %v", err)
	}

	// Verify all composed options were applied
	if config.Model == nil || *config.Model != "sonnet" {
		t.Errorf("Base option Model not applied")
	}
	if config.MaxTurns == nil || *config.MaxTurns != 5 {
		t.Errorf("Base option MaxTurns not applied")
	}
	if config.SystemPrompt != "extra prompt" {
		t.Errorf("Extra option SystemPrompt not applied")
	}
	if config.MaxBudgetUSD == nil || *config.MaxBudgetUSD != 10.0 {
		t.Errorf("Extra option MaxBudgetUSD not applied")
	}
}

func TestOptionOverride(t *testing.T) {
	// Apply same option multiple times to test override behavior
	config := &RequestConfig{}

	// First application
	err := config.Apply(WithModel("sonnet"))
	if err != nil {
		t.Fatalf("First apply failed: %v", err)
	}
	if *config.Model != "sonnet" {
		t.Errorf("First Model should be 'sonnet'")
	}

	// Override with new value
	err = config.Apply(WithModel("opus"))
	if err != nil {
		t.Fatalf("Second apply failed: %v", err)
	}
	if *config.Model != "opus" {
		t.Errorf("Model should be overridden to 'opus'")
	}

	// Override again
	err = config.Apply(WithModel("haiku"))
	if err != nil {
		t.Fatalf("Third apply failed: %v", err)
	}
	if *config.Model != "haiku" {
		t.Errorf("Model should be overridden to 'haiku'")
	}
}

// ============================================================================
// Edge Cases and Error Handling Tests
// ============================================================================

func TestErrorOptionPropagation(t *testing.T) {
	// Create an option that returns an error
	errorOpt := func(c *RequestConfig) error {
		return errors.New("validation failed")
	}

	config, err := NewRequestConfig(
		WithModel("sonnet"),
		errorOpt,
		WithMaxTurns(5), // This should not be applied
	)

	if err == nil {
		t.Errorf("Expected error from errorOpt, got nil")
	}
	if config != nil {
		t.Errorf("Expected nil config on error, got %v", config)
	}
}

func TestNilOptionHandling(t *testing.T) {
	// Test that nil values in options don't cause issues
	config := &RequestConfig{}

	err := config.Apply(
		WithSystemPrompt(nil),
		WithTools(nil),
		WithMCPServers(nil),
		WithCWD(nil),
		WithCLIPath(nil),
		WithHooks(nil),
		WithAgents(nil),
		WithSettingSources(nil),
		WithSandbox(nil),
		WithPlugins(nil),
		WithTaskBudget(nil),
		WithEnv(nil),
		WithOutputFormat(nil),
		WithBetas(nil),
	)
	if err != nil {
		t.Errorf("Applying nil options should not return error: %v", err)
	}

	// Verify nil values are set
	if config.SystemPrompt != nil {
		t.Errorf("SystemPrompt should be nil")
	}
	if config.Tools != nil {
		t.Errorf("Tools should be nil")
	}
	if config.MCPServers != nil {
		t.Errorf("MCPServers should be nil")
	}
}

func TestEmptyValuesHandling(t *testing.T) {
	config := &RequestConfig{}

	err := config.Apply(
		WithModel(""),
		WithFallbackModel(""),
		WithResume(""),
		WithSessionID(""),
		WithEffort(""),
		WithAllowedTools([]string{}),
		WithDisallowedTools([]string{}),
	)
	if err != nil {
		t.Errorf("Applying empty values should not return error: %v", err)
	}

	// Verify empty values are set (not nil)
	if config.Model == nil {
		t.Errorf("Model should be set (even if empty)")
	}
	if *config.Model != "" {
		t.Errorf("Model should be empty string")
	}
	if len(config.AllowedTools) != 0 {
		t.Errorf("AllowedTools should be empty slice")
	}
}

func TestPointerTypeOptions(t *testing.T) {
	// Test that pointer types are correctly handled
	model := "sonnet"
	config := &RequestConfig{}

	err := config.Apply(WithModel(model))
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Modify original - should not affect config since WithModel creates a new pointer
	model = "opus"
	if *config.Model != "sonnet" {
		t.Errorf("Config Model should remain 'sonnet', got '%s'", *config.Model)
	}
}

func TestRequestConfigDefaults(t *testing.T) {
	// Test that empty RequestConfig has expected default values
	config, err := NewRequestConfig()
	if err != nil {
		t.Fatalf("NewRequestConfig() failed: %v", err)
	}

	// All fields should be their zero values
	if config.Model != nil {
		t.Errorf("Model should be nil by default")
	}
	if config.MaxTurns != nil {
		t.Errorf("MaxTurns should be nil by default")
	}
	if config.ContinueConversation != false {
		t.Errorf("ContinueConversation should be false by default")
	}
	if config.EnableFileCheckpointing != false {
		t.Errorf("EnableFileCheckpointing should be false by default")
	}
	if len(config.AllowedTools) != 0 {
		t.Errorf("AllowedTools should be empty by default")
	}
	if len(config.DisallowedTools) != 0 {
		t.Errorf("DisallowedTools should be empty by default")
	}
}

func TestBooleanOptions(t *testing.T) {
	config := &RequestConfig{}

	// Test WithContinueConversation (sets to true)
	err := config.Apply(WithContinueConversation())
	if err != nil {
		t.Errorf("WithContinueConversation failed: %v", err)
	}
	if !config.ContinueConversation {
		t.Errorf("ContinueConversation should be true")
	}

	// Test WithFileCheckpointing (sets to true)
	err = config.Apply(WithFileCheckpointing())
	if err != nil {
		t.Errorf("WithFileCheckpointing failed: %v", err)
	}
	if !config.EnableFileCheckpointing {
		t.Errorf("EnableFileCheckpointing should be true")
	}

	// Boolean options cannot be unset directly - need to create new config
	config2, err := NewRequestConfig()
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}
	if config2.ContinueConversation {
		t.Errorf("New config ContinueConversation should be false")
	}
	if config2.EnableFileCheckpointing {
		t.Errorf("New config EnableFileCheckpointing should be false")
	}
}

func TestComplexMapOptions(t *testing.T) {
	// Test complex map structures
	complexOutputFormat := map[string]interface{}{
		"type": "json_schema",
		"json_schema": map[string]interface{}{
			"name":   "response",
			"strict": true,
			"schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type": "string",
					},
					"data": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
						},
					},
				},
				"required": []interface{}{"status"},
			},
		},
	}

	config, err := NewRequestConfig(WithOutputFormat(complexOutputFormat))
	if err != nil {
		t.Fatalf("NewRequestConfig with complex format failed: %v", err)
	}

	if config.OutputFormat["type"] != "json_schema" {
		t.Errorf("OutputFormat type not set correctly")
	}
}

func TestHooksWithMatchers(t *testing.T) {
	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: []types.HookMatcher{
			{
				Matcher: "Bash|Edit|Write",
				Timeout: types.Float64(30.0),
			},
			{
				Matcher: "WebFetch",
				Timeout: types.Float64(60.0),
			},
		},
		types.HookEventPostToolUse: []types.HookMatcher{
			{
				Matcher: "Read",
			},
		},
	}

	config, err := NewRequestConfig(WithHooks(hooks))
	if err != nil {
		t.Fatalf("NewRequestConfig with hooks failed: %v", err)
	}

	if len(config.Hooks) != 2 {
		t.Errorf("Expected 2 hook events, got %d", len(config.Hooks))
	}
	if len(config.Hooks[types.HookEventPreToolUse]) != 2 {
		t.Errorf("Expected 2 PreToolUse matchers")
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkNewRequestConfig(b *testing.B) {
	opts := []RequestOption{
		WithModel("sonnet"),
		WithMaxTurns(10),
		WithSystemPrompt("benchmark test"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewRequestConfig(opts...)
	}
}

func BenchmarkApply(b *testing.B) {
	config := &RequestConfig{}
	opts := []RequestOption{
		WithModel("sonnet"),
		WithMaxTurns(5),
		WithMaxBudgetUSD(10.0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Apply(opts...)
	}
}

func BenchmarkWithModel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = WithModel("sonnet")
	}
}

func BenchmarkCombinedOptions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = NewRequestConfig(
			WithModel("sonnet"),
			WithSystemPrompt("test"),
			WithMaxTurns(5),
			WithMaxBudgetUSD(10.0),
			WithPermissionMode(types.PermissionModeAuto),
			WithTools([]string{"Bash", "Read"}),
			WithEnv(map[string]string{"KEY": "value"}),
		)
	}
}
