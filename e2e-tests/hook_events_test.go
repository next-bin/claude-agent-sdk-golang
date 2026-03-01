package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Hook Events E2E Tests
// ============================================================================

// hookInvocation tracks hook invocations for testing
type hookInvocation struct {
	hookEventName string
	toolName      string
	toolUseID     string
	message       string
}

// TestPreToolUseHookWithAdditionalContext tests PreToolUse hook returning
// additionalContext field end-to-end.
func TestPreToolUseHookWithAdditionalContext(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	hookInvocations := make([]hookInvocation, 0)

	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			toolName := ""
			toolUseIDStr := ""
			if preInput, ok := input.(*types.PreToolUseHookInput); ok {
				toolName = preInput.ToolName
				if preInput.ToolUseID != "" {
					toolUseIDStr = preInput.ToolUseID
				}
			}

			hookInvocations = append(hookInvocations, hookInvocation{
				hookEventName: "PreToolUse",
				toolName:      toolName,
				toolUseID:     toolUseIDStr,
			})

			return types.SyncHookJSONOutput{
				HookSpecificOutput: types.PreToolUseHookSpecificOutput{
					HookEventName:      "PreToolUse",
					PermissionDecision: strPtr("allow"),
					AdditionalContext:  strPtr("This command is running in a test environment"),
				},
			}, nil
		},
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		AllowedTools:   []string{"Bash"},
		MaxTurns:       types.Int(2),
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

	msgChan, err := client.Query(ctx, "Run: echo 'test additional context'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Hook invocations: %v", hookInvocations)

	if len(hookInvocations) == 0 {
		t.Error("PreToolUse hook should have been invoked")
	} else {
		// Note: tool_use_id may not always be present depending on CLI version
		t.Logf("Tool name: %s, tool_use_id: %s", hookInvocations[0].toolName, hookInvocations[0].toolUseID)
	}
}

// TestPostToolUseHookWithToolUseID tests PostToolUse hook receives
// tool_use_id field end-to-end.
func TestPostToolUseHookWithToolUseID(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	hookInvocations := make([]hookInvocation, 0)

	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			toolName := ""
			toolUseIDStr := ""
			if postInput, ok := input.(*types.PostToolUseHookInput); ok {
				toolName = postInput.ToolName
				if postInput.ToolUseID != "" {
					toolUseIDStr = postInput.ToolUseID
				}
			}

			hookInvocations = append(hookInvocations, hookInvocation{
				hookEventName: "PostToolUse",
				toolName:      toolName,
				toolUseID:     toolUseIDStr,
			})

			return types.SyncHookJSONOutput{
				HookSpecificOutput: types.PostToolUseHookSpecificOutput{
					HookEventName:     "PostToolUse",
					AdditionalContext: strPtr("Post-tool monitoring active"),
				},
			}, nil
		},
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		AllowedTools:   []string{"Bash"},
		MaxTurns:       types.Int(2),
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

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Run: echo 'test tool_use_id'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Hook invocations: %v", hookInvocations)

	if len(hookInvocations) == 0 {
		t.Error("PostToolUse hook should have been invoked")
	} else {
		// Note: tool_use_id may not always be present depending on CLI version
		t.Logf("Tool name: %s, tool_use_id: %s", hookInvocations[0].toolName, hookInvocations[0].toolUseID)
	}
}

// TestNotificationHookEvents tests Notification hook fires end-to-end.
func TestNotificationHookEvents(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	hookInvocations := make([]hookInvocation, 0)

	callback := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			notificationType := ""
			message := ""
			if notifInput, ok := input.(*types.NotificationHookInput); ok {
				notificationType = notifInput.NotificationType
				message = notifInput.Message
			}

			hookInvocations = append(hookInvocations, hookInvocation{
				hookEventName: "Notification",
				message:       message,
			})

			t.Logf("Notification type: %s, message: %s", notificationType, message)

			return types.SyncHookJSONOutput{
				HookSpecificOutput: types.NotificationHookSpecificOutput{
					HookEventName:     "Notification",
					AdditionalContext: strPtr("Notification received"),
				},
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

	msgChan, err := client.Query(ctx, "Say hello in one word")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Notification hook invocations: %v", hookInvocations)

	// Notification hooks may or may not fire depending on CLI behavior.
	// This test verifies the hook registration doesn't cause errors.
	// If it fires, verify the shape is correct.
	for _, inv := range hookInvocations {
		if inv.hookEventName != "Notification" {
			t.Errorf("Expected Notification event, got: %s", inv.hookEventName)
		}
	}
}

// TestMultipleHooksTogether tests registering multiple hook event types together.
func TestMultipleHooksTogether(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	allInvocations := make([]hookInvocation, 0)

	trackHook := &testHookCallback{
		executeFunc: func(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
			allInvocations = append(allInvocations, hookInvocation{
				hookEventName: input.GetHookEventName(),
			})
			return types.SyncHookJSONOutput{}, nil
		},
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		AllowedTools:   []string{"Bash"},
		MaxTurns:       types.Int(2),
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventNotification: {
				{
					Hooks: []types.HookCallback{trackHook},
				},
			},
			types.HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []types.HookCallback{trackHook},
				},
			},
			types.HookEventPostToolUse: {
				{
					Matcher: "Bash",
					Hooks:   []types.HookCallback{trackHook},
				},
			},
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Run: echo 'multi-hook test'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("All hook invocations: %v", allInvocations)

	eventNames := make([]string, 0, len(allInvocations))
	for _, inv := range allInvocations {
		eventNames = append(eventNames, inv.hookEventName)
	}

	// At minimum, PreToolUse and PostToolUse should fire for the Bash command
	hasPreToolUse := false
	hasPostToolUse := false
	for _, name := range eventNames {
		if name == "PreToolUse" {
			hasPreToolUse = true
		}
		if name == "PostToolUse" {
			hasPostToolUse = true
		}
	}

	if !hasPreToolUse {
		t.Error("PreToolUse hook should have fired")
	}
	if !hasPostToolUse {
		t.Error("PostToolUse hook should have fired")
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
