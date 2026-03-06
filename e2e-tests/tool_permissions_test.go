package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Tool Permission E2E Tests
// ============================================================================

// TestPermissionCallbackGetsCalled tests that can_use_tool callback gets invoked
// for non-read-only commands.
//
// Note: The CLI auto-allows certain read-only commands (like 'echo') without
// consulting the SDK callback. We use 'touch' which requires permission.
func TestPermissionCallbackGetsCalled(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	callbackInvocations := make([]struct {
		toolName  string
		inputData map[string]interface{}
	}, 0)

	permissionCallback := func(toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		t.Logf("Permission callback called for: %s", toolName)
		callbackInvocations = append(callbackInvocations, struct {
			toolName  string
			inputData map[string]interface{}
		}{toolName: toolName, inputData: input})
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		CanUseTool:     permissionCallback,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Use a simple query that should trigger the callback if tools are used
	if err := client.Query(ctx, "List files in /tmp directory"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Use ConsumeMessagesVerbose for proper message handling
	ConsumeMessagesVerbose(ctx, t, msgChan, "TestPermissionCallbackGetsCalled")

	t.Logf("Callback invocations: %d", len(callbackInvocations))

	// Log all tool names
	for _, inv := range callbackInvocations {
		t.Logf("Tool called: %s", inv.toolName)
	}
}

// TestPermissionCallbackAllow tests that permission callback can allow tools.
func TestPermissionCallbackAllow(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	allowedTools := []string{"Read", "Glob", "Grep"}
	deniedTools := []string{"Bash", "Edit", "Write"}

	permissionCallback := func(toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		for _, allowed := range allowedTools {
			if toolName == allowed {
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			}
		}
		for _, denied := range deniedTools {
			if toolName == denied {
				return types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   "Tool " + toolName + " is not allowed by security policy",
					Interrupt: false,
				}, nil
			}
		}
		return types.PermissionResultDeny{
			Behavior:  "deny",
			Message:   "Unknown tool: " + toolName,
			Interrupt: false,
		}, nil
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		CanUseTool:     permissionCallback,
		MaxTurns:       types.Int(1),
		AllowedTools:   []string{"Read", "Glob", "Grep"},
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Say 'hello'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	_, foundResult, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestPermissionCallbackAllow")

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

// TestPermissionCallbackWithUpdatedPermissions tests that permission callback
// can return PermissionResultAllow with updated permissions.
func TestPermissionCallbackWithUpdatedPermissions(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	permissionCallback := func(toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		// Allow with updated permissions
		session := types.PermissionUpdateDestinationSession
		return types.PermissionResultAllow{
			Behavior: "allow",
			UpdatedPermissions: []types.PermissionUpdate{
				{
					Type: types.PermissionUpdateTypeAddRules,
					Rules: []types.PermissionRuleValue{
						{ToolName: toolName},
					},
					Behavior:    permissionBehaviorPtr(types.PermissionBehaviorAllow),
					Destination: &session,
				},
			},
		}, nil
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		CanUseTool:     permissionCallback,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Say 'test passed'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	_, foundResult, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestPermissionCallbackWithUpdatedPermissions")

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

// TestPermissionCallbackDeny tests that permission callback can deny tools.
func TestPermissionCallbackDeny(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	permissionCallback := func(toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		if toolName == "Bash" {
			return types.PermissionResultDeny{
				Behavior:  "deny",
				Message:   "Bash is not allowed",
				Interrupt: false,
			}, nil
		}
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		CanUseTool:     permissionCallback,
		MaxTurns:       types.Int(1),
		AllowedTools:   []string{"Bash"},
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

	_, foundResult, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestPermissionCallbackDeny")

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

// TestPermissionCallbackDenyWithInterrupt tests that permission callback
// can deny tools with interrupt.
func TestPermissionCallbackDenyWithInterrupt(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	permissionCallback := func(toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
		// Deny all tools with interrupt
		return types.PermissionResultDeny{
			Behavior:  "deny",
			Message:   "Tool " + toolName + " is not allowed",
			Interrupt: true,
		}, nil
	}

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		CanUseTool:     permissionCallback,
		MaxTurns:       types.Int(1),
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

	// Just consume messages
	ConsumeMessagesVerbose(ctx, t, msgChan, "TestPermissionCallbackDenyWithInterrupt")
}

// Helper function to create a pointer to PermissionBehavior
func permissionBehaviorPtr(b types.PermissionBehavior) *types.PermissionBehavior {
	return &b
}
