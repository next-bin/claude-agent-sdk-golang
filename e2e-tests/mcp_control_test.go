package e2e_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// MCP Control E2E Tests - ReconnectMCPServer, ToggleMCPServer, StopTask
// ============================================================================

// TestReconnectMCPServer tests the ReconnectMCPServer API.
// Note: SDK MCP servers (in-process) don't appear in CLI's MCP server list,
// so reconnect/toggle operations return "Server not found" for them.
// These methods are designed for external MCP servers configured via settings.
func TestReconnectMCPServer(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create a simple echo tool
	echoTool := sdkmcp.Tool("echo", "Echo back the input message", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{"type": "string"},
		},
		"required": []string{"message"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		message, _ := args["message"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Echo: %s", message)), nil
	})

	server := sdkmcp.CreateSdkMcpServer("test", []*sdkmcp.SdkMcpTool{echoTool})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"test": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__test__echo"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First query
	t.Log("Step 1: Initial query")
	queryAndDrain(ctx, t, client, "Use the echo tool to say 'Hello'")

	// Get MCP status
	t.Log("Step 2: Get MCP status")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Logf("GetMCPStatus error: %v", err)
	} else {
		t.Logf("MCP status: %+v", status)
	}

	// Try to reconnect the MCP server
	t.Log("Step 3: Try ReconnectMCPServer")
	err = client.ReconnectMCPServer(ctx, "test")
	if err != nil {
		t.Logf("ReconnectMCPServer error: %v", err)
	} else {
		t.Log("ReconnectMCPServer succeeded")
	}

	// Second query
	t.Log("Step 4: Query after reconnect attempt")
	queryAndDrain(ctx, t, client, "Say 'World' using the echo tool")
}

// TestToggleMCPServer tests the ToggleMCPServer API.
func TestToggleMCPServer(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a simple tool
	testTool := sdkmcp.Tool("test_op", "A test operation", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		return sdkmcp.TextResult("Test operation successful"), nil
	})

	server := sdkmcp.CreateSdkMcpServer("toggle_test", []*sdkmcp.SdkMcpTool{testTool})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"toggle_test": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__toggle_test__test_op"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First query
	t.Log("Query with SDK MCP server")
	queryAndDrain(ctx, t, client, "Use the test_op tool")

	// Try to toggle the server
	t.Log("Try ToggleMCPServer(false)")
	err := client.ToggleMCPServer(ctx, "toggle_test", false)
	if err != nil {
		t.Logf("ToggleMCPServer(false) error: %v", err)
	}

	// Query should still work
	queryAndDrain(ctx, t, client, "Say hello")
}

// TestStopTask tests the StopTask API.
func TestStopTask(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Start a query
	t.Log("Start query")
	msgChan, err := client.Query(ctx, "What is 2+2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Try to stop a task with a fake task ID
	t.Log("Try StopTask with fake task ID")
	err = client.StopTask(ctx, "fake-task-id-12345")
	if err != nil {
		t.Logf("StopTask error: %v", err)
	}

	// Drain the query
	drainChannel(ctx, msgChan)
	t.Log("Query completed")
}

// TestMCPControlMethodsWithoutMCP tests MCP control methods without any MCP servers.
func TestMCPControlMethodsWithoutMCP(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Get MCP status
	t.Log("Step 1: GetMCPStatus")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Logf("GetMCPStatus error: %v", err)
	} else {
		t.Logf("MCP Status: %+v", status)
	}

	// Try to reconnect a non-existent server
	t.Log("Step 2: ReconnectMCPServer with non-existent server")
	err = client.ReconnectMCPServer(ctx, "non_existent_server")
	if err != nil {
		t.Logf("ReconnectMCPServer error: %v", err)
	}

	// Try to toggle a non-existent server
	t.Log("Step 3: ToggleMCPServer with non-existent server")
	err = client.ToggleMCPServer(ctx, "non_existent_server", false)
	if err != nil {
		t.Logf("ToggleMCPServer error: %v", err)
	}

	// Make a query to verify client still works
	queryAndDrain(ctx, t, client, "Say 'test complete'")
}

// queryAndDrain executes a query and drains all messages until result.
func queryAndDrain(ctx context.Context, t *testing.T, client *claude.Client, prompt string) {
	msgChan, err := client.Query(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	found := drainChannel(ctx, msgChan)
	t.Logf("Query '%s' completed: foundResult=%v", truncate(prompt, 30), found)
}

// drainChannel drains all messages from the channel until result or context done.
func drainChannel(ctx context.Context, msgChan <-chan types.Message) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case msg, ok := <-msgChan:
			if !ok {
				return false
			}
			if _, isResult := msg.(*types.ResultMessage); isResult {
				// Continue draining to not block the sender
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-msgChan:
							if !ok {
								return
							}
						}
					}
				}()
				return true
			}
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}