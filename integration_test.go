// Package claude_test tests integration of middleware and functional options with SDK.
package claude_test

import (
	"context"
	"testing"
	"time"

	"github.com/next-bin/claude-agent-sdk-golang/option"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func TestFunctionalOptionsWithTypes(t *testing.T) {
	// Test that functional options can be converted to ClaudeAgentOptions
	config, err := option.NewRequestConfig(
		option.WithSystemPrompt("Test prompt"),
		option.WithModel(types.ModelSonnet),
		option.WithMaxTurns(3),
		option.WithPermissionMode(types.PermissionModeAcceptEdits),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	// Convert to ClaudeAgentOptions
	opts := configToOptions(config)

	// Verify conversion
	if opts.SystemPrompt == nil || opts.SystemPrompt != "Test prompt" {
		t.Errorf("SystemPrompt mismatch")
	}

	if opts.Model == nil || *opts.Model != types.ModelSonnet {
		t.Errorf("Model mismatch: got %v", opts.Model)
	}

	if opts.MaxTurns == nil || *opts.MaxTurns != 3 {
		t.Errorf("MaxTurns mismatch: got %v", opts.MaxTurns)
	}

	if opts.PermissionMode == nil || *opts.PermissionMode != types.PermissionModeAcceptEdits {
		t.Errorf("PermissionMode mismatch: got %v", opts.PermissionMode)
	}
}

func TestFunctionalOptionsComposition(t *testing.T) {
	// Test composing multiple option sets
	baseConfig, err := option.NewRequestConfig(
		option.WithSystemPrompt("Base"),
		option.WithMaxTurns(5),
	)
	if err != nil {
		t.Fatalf("Base config failed: %v", err)
	}

	// Apply additional options
	err = baseConfig.Apply(
		option.WithModel(types.ModelOpus),
		option.WithPermissionMode(types.PermissionModeDefault),
	)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify all options applied
	if baseConfig.SystemPrompt != "Base" {
		t.Errorf("SystemPrompt should be preserved")
	}

	if baseConfig.MaxTurns == nil || *baseConfig.MaxTurns != 5 {
		t.Errorf("MaxTurns should be preserved")
	}

	if baseConfig.Model == nil || *baseConfig.Model != types.ModelOpus {
		t.Errorf("Model should be applied")
	}

	if baseConfig.PermissionMode == nil || *baseConfig.PermissionMode != types.PermissionModeDefault {
		t.Errorf("PermissionMode should be applied")
	}
}

func TestFunctionalOptionsOverride(t *testing.T) {
	// Test that later options override earlier ones
	config, err := option.NewRequestConfig(
		option.WithMaxTurns(5),
		option.WithMaxTurns(10), // Override
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	if config.MaxTurns == nil || *config.MaxTurns != 10 {
		t.Errorf("MaxTurns should be 10 (overridden), got %v", config.MaxTurns)
	}
}

func TestFunctionalOptionsWithHooks(t *testing.T) {
	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: []types.HookMatcher{
			{Matcher: "Bash"},
		},
	}

	config, err := option.NewRequestConfig(
		option.WithHooks(hooks),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	if len(config.Hooks) != 1 {
		t.Errorf("Expected 1 hook event, got %d", len(config.Hooks))
	}

	if len(config.Hooks[types.HookEventPreToolUse]) != 1 {
		t.Errorf("Expected 1 PreToolUse matcher")
	}
}

func TestFunctionalOptionsWithTools(t *testing.T) {
	config, err := option.NewRequestConfig(
		option.WithTools([]string{"Read", "Write"}),
		option.WithAllowedTools([]string{"Read"}),
		option.WithDisallowedTools([]string{"Bash"}),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	if len(config.AllowedTools) != 1 || config.AllowedTools[0] != "Read" {
		t.Errorf("AllowedTools mismatch")
	}

	if len(config.DisallowedTools) != 1 || config.DisallowedTools[0] != "Bash" {
		t.Errorf("DisallowedTools mismatch")
	}
}

func TestFunctionalOptionsWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that config creation respects context timeout (should not block)
	done := make(chan struct{})
	go func() {
		_, err := option.NewRequestConfig(
			option.WithSystemPrompt("Test"),
		)
		if err != nil {
			t.Errorf("NewRequestConfig failed: %v", err)
		}
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-ctx.Done():
		t.Fatal("NewRequestConfig blocked too long")
	}
}

// Helper function to convert RequestConfig to ClaudeAgentOptions
func configToOptions(config *option.RequestConfig) *types.ClaudeAgentOptions {
	opts := &types.ClaudeAgentOptions{}

	if config.SystemPrompt != nil {
		opts.SystemPrompt = config.SystemPrompt
	}

	if config.Model != nil {
		opts.Model = config.Model
	}

	if config.MaxTurns != nil {
		opts.MaxTurns = config.MaxTurns
	}

	if config.PermissionMode != nil {
		opts.PermissionMode = config.PermissionMode
	}

	if config.Tools != nil {
		opts.Tools = config.Tools
	}

	if len(config.AllowedTools) > 0 {
		opts.AllowedTools = config.AllowedTools
	}

	if len(config.DisallowedTools) > 0 {
		opts.DisallowedTools = config.DisallowedTools
	}

	if config.MCPServers != nil {
		opts.MCPServers = config.MCPServers
	}

	if len(config.Hooks) > 0 {
		opts.Hooks = config.Hooks
	}

	if config.CWD != nil {
		opts.CWD = config.CWD
	}

	if config.CLIPath != nil {
		opts.CLIPath = config.CLIPath
	}

	if len(config.Env) > 0 {
		opts.Env = config.Env
	}

	return opts
}
