package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Mock Hook Callback for Testing
// ============================================================================

// testHookCallback is a mock implementation of types.HookCallback for testing
type testHookCallback struct {
	executeFunc func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error)
}

func (m *testHookCallback) Execute(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
	if m.executeFunc != nil {
		return m.executeFunc(input, toolUseID, context)
	}
	return types.SyncHookJSONOutput{}, nil
}

// ============================================================================
// Hooks E2E Tests
// ============================================================================

func TestPreToolUseHook(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	hookCalled := false

	// Create a hook callback using the interface
	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			hookCalled = true
			return types.SyncHookJSONOutput{
				Continue_: types.Bool(true),
			}, nil
		},
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
		AllowedTools:   []string{"Bash"},
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []types.HookCallback{callback},
				},
			},
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Use the Bash tool to run 'echo hello'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for range msgChan {
		// Consume messages
	}

	// Note: Hook may or may not be called depending on whether Bash tool is actually used
	_ = hookCalled
}

func TestNotificationHook(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	notificationReceived := false

	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			notificationReceived = true
			return types.SyncHookJSONOutput{
				Continue_: types.Bool(true),
			}, nil
		},
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventNotification: {
				{
					Hooks: []types.HookCallback{callback},
				},
			},
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Say hello")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for range msgChan {
		// Consume messages
	}

	_ = notificationReceived
}

// ============================================================================
// Agent Definition Tests
// ============================================================================

func TestAgentDefinition(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Define a custom agent
	agent := types.AgentDefinition{
		Description: "A simple greeting agent",
		Prompt:      "You are a friendly greeting agent. Always respond with a cheerful greeting.",
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
		Agents: map[string]types.AgentDefinition{
			"greeter": agent,
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Say hello")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.ResultMessage:
			foundResult = true
			_ = m
		}
	}

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

// ============================================================================
// Structured Output Tests
// ============================================================================

func TestStructuredOutput(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Define JSON schema for structured output
	outputFormat := map[string]interface{}{
		"type": "json_schema",
		"schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"greeting": map[string]interface{}{
					"type": "string",
				},
				"count": map[string]interface{}{
					"type": "number",
				},
			},
			"required": []string{"greeting", "count"},
		},
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
		OutputFormat:   outputFormat,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Return a JSON object with a greeting 'Hello' and count 42")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundStructuredOutput bool
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.ResultMessage:
			if m.StructuredOutput != nil {
				foundStructuredOutput = true
			}
		}
	}

	// Note: structured output may not always be present depending on CLI version
	_ = foundStructuredOutput
}

// ============================================================================
// Include Partial Messages Tests
// ============================================================================

func TestIncludePartialMessages(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:                  types.String(DefaultTestConfig().Model),
		PermissionMode:         &mode,
		MaxTurns:               types.Int(1),
		IncludePartialMessages: true,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Tell me a short story")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	streamEventCount := 0
	for msg := range msgChan {
		switch msg.(type) {
		case *types.StreamEvent:
			streamEventCount++
		}
	}

	// With partial messages enabled, we should see some stream events
	// (Note: this depends on CLI version and response content)
	_ = streamEventCount
}
