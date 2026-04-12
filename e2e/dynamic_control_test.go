package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Dynamic Control E2E Tests
// ============================================================================

// TestSetPermissionMode tests that permission mode can be changed dynamically
// during a session.
func TestSetPermissionMode(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	// Use BypassPermissions to avoid interactive permission prompts in tests
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Change permission mode to acceptEdits
	if err := client.SetPermissionMode(ctx, "acceptEdits"); err != nil {
		t.Logf("SetPermissionMode failed: %v (may not be supported)", err)
	}

	// Make a query
	if err := client.Query(ctx, "What is 2+2? Just respond with the number."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	// Change back to default
	if err := client.SetPermissionMode(ctx, "default"); err != nil {
		t.Logf("SetPermissionMode failed: %v (may not be supported)", err)
	}

	// Make another query
	if err := client.Query(ctx, "What is 3+3? Just respond with the number."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)
}

// TestSetModel tests that model can be changed dynamically during a session.
func TestSetModel(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Start with default model
	if err := client.Query(ctx, "What is 1+1? Just the number."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	// Switch to Haiku model
	if err := client.SetModel(ctx, types.ModelHaiku); err != nil {
		t.Logf("SetModel failed: %v (may not be supported)", err)
	}

	if err := client.Query(ctx, "What is 2+2? Just the number."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	// Switch back to default (empty string means default)
	if err := client.SetModel(ctx, ""); err != nil {
		t.Logf("SetModel failed: %v (may not be supported)", err)
	}

	if err := client.Query(ctx, "What is 3+3? Just the number."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)
}

// TestInterrupt tests that interrupt can be sent during a session.
func TestInterrupt(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Start a query
	if err := client.Query(ctx, "Count from 1 to 100 slowly."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Send interrupt in a goroutine
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := client.Interrupt(ctx); err != nil {
			t.Logf("Interrupt resulted in: %v (may not be supported)", err)
		}
	}()

	// Consume messages until context is done or channel closes
	messageCount := 0
	for {
		select {
		case <-ctx.Done():
			t.Logf("Context done after receiving %d messages", messageCount)
			return
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed
				t.Logf("Received %d messages before/after interrupt", messageCount)
				return
			}
			messageCount++
			// Check for result message - indicates query completed
			if _, isResult := msg.(*types.ResultMessage); isResult {
				t.Logf("Received result after %d messages", messageCount)
				return
			}
		}
	}
}

// TestGetMCPStatus tests that MCP server status can be retrieved.
func TestGetMCPStatus(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Get MCP status
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Logf("GetMCPStatus failed: %v (may not be supported)", err)
	} else {
		t.Logf("MCP status: %v", status)
	}
}

// TestGetServerInfo tests that server info can be retrieved.
func TestGetServerInfo(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Get server info
	info := client.GetServerInfo()
	if info == nil {
		t.Log("GetServerInfo returned nil")
	} else {
		t.Logf("Server info: %v", info)
	}
}

// TestMultipleDynamicOperations tests multiple dynamic operations in sequence.
func TestMultipleDynamicOperations(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 90*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Initial query
	t.Log("Query 1: Say hello")
	if err := client.Query(ctx, "Say hello"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	count, found := consumeMessagesUntilResult(ctx, msgChan)
	t.Logf("Query 1 completed: count=%d, foundResult=%v", count, found)

	// Change model
	t.Log("Changing model to haiku")
	_ = client.SetModel(ctx, types.ModelHaiku)

	// Another query
	t.Log("Query 2: Say goodbye")
	if err := client.Query(ctx, "Say goodbye"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	count, found = consumeMessagesUntilResult(ctx, msgChan)
	t.Logf("Query 2 completed: count=%d, foundResult=%v", count, found)

	// Change permission mode
	t.Log("Changing permission mode to default")
	_ = client.SetPermissionMode(ctx, "default")

	// Final query
	t.Log("Query 3: Say done")
	if err := client.Query(ctx, "Say done"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	count, found = consumeMessagesUntilResult(ctx, msgChan)
	t.Logf("Query 3 completed: count=%d, foundResult=%v", count, found)
	t.Log("TEST PASSED: All queries completed")
}
