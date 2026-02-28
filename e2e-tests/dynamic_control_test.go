package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Dynamic Control E2E Tests
// ============================================================================

// TestSetPermissionMode tests that permission mode can be changed dynamically
// during a session.
func TestSetPermissionMode(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeDefault
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Change permission mode to acceptEdits
	if err := client.SetPermissionMode(ctx, "acceptEdits"); err != nil {
		t.Logf("SetPermissionMode failed: %v (may not be supported)", err)
	}

	// Make a query
	msgChan, err := client.Query(ctx, "What is 2+2? Just respond with the number.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for range msgChan {
		// Consume messages
	}

	// Change back to default
	if err := client.SetPermissionMode(ctx, "default"); err != nil {
		t.Logf("SetPermissionMode failed: %v (may not be supported)", err)
	}

	// Make another query
	msgChan, err = client.Query(ctx, "What is 3+3? Just respond with the number.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for range msgChan {
		// Consume messages
	}
}

// TestSetModel tests that model can be changed dynamically during a session.
func TestSetModel(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Start with default model
	msgChan, err := client.Query(ctx, "What is 1+1? Just the number.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for range msgChan {
		// Consume messages
	}

	// Switch to Haiku model
	if err := client.SetModel(ctx, types.ModelHaiku); err != nil {
		t.Logf("SetModel failed: %v (may not be supported)", err)
	}

	msgChan, err = client.Query(ctx, "What is 2+2? Just the number.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for range msgChan {
		// Consume messages
	}

	// Switch back to default (empty string means default)
	if err := client.SetModel(ctx, ""); err != nil {
		t.Logf("SetModel failed: %v (may not be supported)", err)
	}

	msgChan, err = client.Query(ctx, "What is 3+3? Just the number.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for range msgChan {
		// Consume messages
	}
}

// TestInterrupt tests that interrupt can be sent during a session.
func TestInterrupt(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Start a query
	msgChan, err := client.Query(ctx, "Count from 1 to 100 slowly.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Send interrupt in a goroutine
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := client.Interrupt(ctx); err != nil {
			t.Logf("Interrupt resulted in: %v", err)
		}
	}()

	// Consume any remaining messages
	messageCount := 0
	for range msgChan {
		messageCount++
	}

	t.Logf("Received %d messages before/after interrupt", messageCount)
}

// TestGetMCPStatus tests that MCP server status can be retrieved.
func TestGetMCPStatus(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Initial query
	msgChan, err := client.Query(ctx, "Say hello")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	for range msgChan {
	}

	// Change model
	_ = client.SetModel(ctx, types.ModelHaiku)

	// Another query
	msgChan, err = client.Query(ctx, "Say goodbye")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	for range msgChan {
	}

	// Change permission mode
	_ = client.SetPermissionMode(ctx, "default")

	// Final query
	msgChan, err = client.Query(ctx, "Say done")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	for range msgChan {
	}
}
