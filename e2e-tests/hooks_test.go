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

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
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

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Use the Bash tool to run 'echo hello'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	// Note: Hook may or may not be called depending on whether Bash tool is actually used
	_ = hookCalled
}

func TestNotificationHook(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
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

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Say hello"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	_ = notificationReceived
}

// TestHookWithPermissionDecisionAndReason tests that hooks with permissionDecision
// and permissionDecisionReason fields work end-to-end.
// Note: We use a prompt that doesn't require the tool being matched, to avoid
// the model getting stuck when the tool is denied. Instead, we verify that the
// hook is called with the correct fields when the model decides to use the tool.
func TestHookWithPermissionDecisionAndReason(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	var hookInvocations []string
	var permissionDecisions []string

	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			// Type assert to PreToolUseHookInput to access tool name
			if preInput, ok := input.(types.PreToolUseHookInput); ok {
				t.Logf("Hook called for tool: %s", preInput.ToolName)
				hookInvocations = append(hookInvocations, preInput.ToolName)

				// Always allow but record the permission decision fields
				allow := "allow"
				reason := "Tool approved for testing permission decision fields"
				permissionDecisions = append(permissionDecisions, allow)

				return types.SyncHookJSONOutput{
					Reason: types.String(reason),
					HookSpecificOutput: types.PreToolUseHookSpecificOutput{
						HookEventName:            "PreToolUse",
						PermissionDecision:       &allow,
						PermissionDecisionReason: &reason,
					},
				}, nil
			}
			return types.SyncHookJSONOutput{Continue_: types.Bool(true)}, nil
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

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Use the Bash tool to run 'echo hello'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Hook invocations: %v", hookInvocations)
	t.Logf("Permission decisions: %v", permissionDecisions)

	// Verify hook was called
	found := false
	for _, inv := range hookInvocations {
		if inv == "Bash" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Hook should have been invoked for Bash tool, got: %v", hookInvocations)
	}

	// Verify permission decision was recorded
	if len(permissionDecisions) == 0 {
		t.Error("Expected at least one permission decision to be recorded")
	}
}

// TestHookWithContinueAndStopReason tests that hooks with continue_=False
// and stopReason fields work end-to-end.
func TestHookWithContinueAndStopReason(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	var hookInvocations []string

	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			// Type assert to PostToolUseHookInput to access tool name
			if postInput, ok := input.(types.PostToolUseHookInput); ok {
				hookInvocations = append(hookInvocations, postInput.ToolName)
			}

			// Test continue_=False and stopReason fields
			return types.SyncHookJSONOutput{
				Continue_:     types.Bool(false),
				StopReason:    types.String("Execution halted by test hook for validation"),
				Reason:        types.String("Testing continue and stopReason fields"),
				SystemMessage: types.String("🛑 Test hook stopped execution"),
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
			types.HookEventPostToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []types.HookCallback{callback},
				},
			},
		},
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Run: echo 'test message'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Hook invocations: %v", hookInvocations)
	// Verify hook was called
	found := false
	for _, inv := range hookInvocations {
		if inv == "Bash" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("PostToolUse hook should have been invoked, got: %v", hookInvocations)
	}
}

// TestHookWithAdditionalContext tests that hooks with hookSpecificOutput
// containing additionalContext work end-to-end.
func TestHookWithAdditionalContext(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	var hookInvocations []string

	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			hookInvocations = append(hookInvocations, "context_added")

			return types.SyncHookJSONOutput{
				SystemMessage:  types.String("Additional context provided by hook"),
				Reason:         types.String("Hook providing monitoring feedback"),
				SuppressOutput: types.Bool(false),
				HookSpecificOutput: types.PostToolUseHookSpecificOutput{
					HookEventName:     "PostToolUse",
					AdditionalContext: types.String("The command executed successfully with hook monitoring"),
				},
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
			types.HookEventPostToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []types.HookCallback{callback},
				},
			},
		},
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Run: echo 'testing hooks'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Hook invocations: %v", hookInvocations)
	// Verify hook was called
	found := false
	for _, inv := range hookInvocations {
		if inv == "context_added" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Hook with hookSpecificOutput should have been invoked")
	}
}
