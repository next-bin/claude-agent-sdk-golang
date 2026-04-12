// Package option_test tests the functional options functionality.
package option_test

import (
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang/option"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func TestNewRequestConfig(t *testing.T) {
	config, err := option.NewRequestConfig()
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}
	if config == nil {
		t.Fatal("Config is nil")
	}
}

func TestWithOptions(t *testing.T) {
	config, err := option.NewRequestConfig(
		option.WithSystemPrompt("You are a helpful assistant"),
		option.WithModel(types.ModelSonnet),
		option.WithMaxTurns(5),
		option.WithPermissionMode(types.PermissionModeAcceptEdits),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	// Check system prompt
	if config.SystemPrompt != "You are a helpful assistant" {
		t.Errorf("Expected system prompt, got %v", config.SystemPrompt)
	}

	// Check model
	if config.Model == nil || *config.Model != types.ModelSonnet {
		t.Errorf("Expected model=%s, got %v", types.ModelSonnet, config.Model)
	}

	// Check max turns
	if config.MaxTurns == nil || *config.MaxTurns != 5 {
		t.Errorf("Expected maxTurns=5, got %v", config.MaxTurns)
	}

	// Check permission mode
	if config.PermissionMode == nil || *config.PermissionMode != types.PermissionModeAcceptEdits {
		t.Errorf("Expected permissionMode=%s, got %v", types.PermissionModeAcceptEdits, config.PermissionMode)
	}
}

func TestWithTools(t *testing.T) {
	tools := []string{"Read", "Write", "Bash"}
	config, err := option.NewRequestConfig(
		option.WithTools(tools),
		option.WithAllowedTools([]string{"Read"}),
		option.WithDisallowedTools([]string{"Bash"}),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	// Check tools
	if config.Tools == nil {
		t.Error("Expected tools to be set")
	}

	// Check allowed tools
	if len(config.AllowedTools) != 1 || config.AllowedTools[0] != "Read" {
		t.Errorf("Expected allowedTools=[Read], got %v", config.AllowedTools)
	}

	// Check disallowed tools
	if len(config.DisallowedTools) != 1 || config.DisallowedTools[0] != "Bash" {
		t.Errorf("Expected disallowedTools=[Bash], got %v", config.DisallowedTools)
	}
}

func TestWithHooks(t *testing.T) {
	hooks := map[types.HookEvent][]types.HookMatcher{
		types.HookEventPreToolUse: []types.HookMatcher{
			{Matcher: "Bash", Hooks: nil},
		},
	}

	config, err := option.NewRequestConfig(
		option.WithHooks(hooks),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	if config.Hooks == nil {
		t.Error("Expected hooks to be set")
	}

	if len(config.Hooks[types.HookEventPreToolUse]) != 1 {
		t.Errorf("Expected 1 PreToolUse hook matcher, got %d", len(config.Hooks[types.HookEventPreToolUse]))
	}
}

func TestWithThinking(t *testing.T) {
	config, err := option.NewRequestConfig(
		option.WithEffort("high"),
	)
	if err != nil {
		t.Fatalf("NewRequestConfig failed: %v", err)
	}

	if config.Effort == nil || *config.Effort != "high" {
		t.Errorf("Expected effort=high, got %v", config.Effort)
	}
}

func TestApply(t *testing.T) {
	config := &option.RequestConfig{}

	err := config.Apply(
		option.WithMaxTurns(10),
		option.WithContinueConversation(),
	)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if config.MaxTurns == nil || *config.MaxTurns != 10 {
		t.Errorf("Expected maxTurns=10, got %v", config.MaxTurns)
	}

	if !config.ContinueConversation {
		t.Error("Expected ContinueConversation=true")
	}
}

func TestMultipleApply(t *testing.T) {
	config := &option.RequestConfig{}

	// Apply first set of options
	err := config.Apply(
		option.WithMaxTurns(5),
	)
	if err != nil {
		t.Fatalf("First Apply failed: %v", err)
	}

	// Apply second set of options (should override)
	err = config.Apply(
		option.WithMaxTurns(10),
	)
	if err != nil {
		t.Fatalf("Second Apply failed: %v", err)
	}

	if config.MaxTurns == nil || *config.MaxTurns != 10 {
		t.Errorf("Expected maxTurns=10 (overridden), got %v", config.MaxTurns)
	}
}
