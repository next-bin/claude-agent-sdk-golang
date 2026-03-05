package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// MCP Control E2E Tests - CLI Plugins and Custom MCP Servers
// ============================================================================

// TestMCPControlWithCLIPlugins tests MCP control methods with CLI plugins.
// CLI plugins like context7, playwright are configured in ~/.claude/settings.json
func TestMCPControlWithCLIPlugins(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First, make a query to trigger MCP server connections
	t.Log("=== Step 0: Initial query to trigger MCP connections ===")
	msgChan, err := client.Query(ctx, "What tools are available? List them.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Consume messages until result
	drainUntilResult(ctx, msgChan)

	// Step 1: Get MCP status - should show CLI plugin MCP servers
	t.Log("\n=== Step 1: Get MCP status ===")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Fatalf("GetMCPStatus failed: %v", err)
	}

	t.Logf("MCP Status response: %+v", status)

	// Parse and display server info
	servers, ok := status["mcpServers"].([]interface{})
	if !ok {
		t.Log("No mcpServers found or unexpected format")
		return
	}

	t.Logf("Found %d MCP server(s)", len(servers))

	var connectedServerName string
	for i, s := range servers {
		if serverInfo, ok := s.(map[string]interface{}); ok {
			name, _ := serverInfo["name"].(string)
			serverStatus, _ := serverInfo["status"].(string)
			t.Logf("  [%d] Server: %s, Status: %s", i, name, serverStatus)

			// Find a connected server to test with
			if serverStatus == "connected" && connectedServerName == "" {
				connectedServerName = name
			}
		}
	}

	if connectedServerName == "" {
		t.Log("No connected MCP server found to test control methods")
		return
	}

	// Step 2: Test ToggleMCPServer
	t.Logf("\n=== Step 2: ToggleMCPServer on '%s' ===", connectedServerName)
	err = client.ToggleMCPServer(ctx, connectedServerName, false)
	if err != nil {
		t.Logf("ToggleMCPServer(false) error: %v", err)
	} else {
		t.Logf("✅ Server '%s' disabled successfully", connectedServerName)

		// Verify status changed
		time.Sleep(1 * time.Second)
		status2, err := client.GetMCPStatus(ctx)
		if err == nil {
			t.Logf("Status after disable: %+v", status2)
		}

		// Re-enable
		t.Logf("Re-enabling server '%s'", connectedServerName)
		err = client.ToggleMCPServer(ctx, connectedServerName, true)
		if err != nil {
			t.Logf("ToggleMCPServer(true) error: %v", err)
		} else {
			t.Logf("✅ Server '%s' re-enabled successfully", connectedServerName)
		}
	}

	// Step 3: Test ReconnectMCPServer
	t.Logf("\n=== Step 3: ReconnectMCPServer on '%s' ===", connectedServerName)
	err = client.ReconnectMCPServer(ctx, connectedServerName)
	if err != nil {
		t.Logf("ReconnectMCPServer error: %v", err)
	} else {
		t.Logf("✅ Server '%s' reconnected successfully", connectedServerName)
	}

	// Step 4: Final query to verify everything works
	t.Log("\n=== Step 4: Final query to verify client works ===")
	msgChan2, err := client.Query(ctx, "Say 'MCP control test complete'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	drainUntilResult(ctx, msgChan2)
}

// TestMCPControlWithConfiguredMCPServer tests MCP control with the MiniMax MCP server
// that is configured in ~/.claude.json
func TestMCPControlWithConfiguredMCPServer(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Use the MiniMax MCP server that's configured in ~/.claude.json
	// The CLI will automatically load MCP servers from config
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First query to trigger MCP connections
	t.Log("=== Step 0: Initial query to trigger MCP ===")
	msgChan, err := client.Query(ctx, "What is 2+2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	drainUntilResult(ctx, msgChan)

	// Step 1: Get MCP status to find the MiniMax server
	t.Log("\n=== Step 1: Get MCP status ===")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Fatalf("GetMCPStatus failed: %v", err)
	}

	t.Logf("MCP Status: %+v", status)

	servers, ok := status["mcpServers"].([]interface{})
	if !ok {
		t.Fatal("No mcpServers found in response")
	}

	t.Logf("Found %d MCP server(s)", len(servers))

	// Find the MiniMax server or any connected server
	var serverToTest string
	for _, s := range servers {
		if serverInfo, ok := s.(map[string]interface{}); ok {
			name, _ := serverInfo["name"].(string)
			serverStatus, _ := serverInfo["status"].(string)
			t.Logf("  Server: %s, Status: %s", name, serverStatus)
			if serverStatus == "connected" {
				serverToTest = name
			}
		}
	}

	if serverToTest == "" {
		t.Log("No connected MCP server found in status")
		return
	}

	// Step 2: Test ReconnectMCPServer
	t.Logf("\n=== Step 2: ReconnectMCPServer on '%s' ===", serverToTest)
	err = client.ReconnectMCPServer(ctx, serverToTest)
	if err != nil {
		t.Logf("ReconnectMCPServer error: %v", err)
	} else {
		t.Logf("✅ Server '%s' reconnected successfully", serverToTest)
	}

	// Step 3: Test ToggleMCPServer
	t.Logf("\n=== Step 3: ToggleMCPServer on '%s' ===", serverToTest)

	// Disable
	err = client.ToggleMCPServer(ctx, serverToTest, false)
	if err != nil {
		t.Logf("ToggleMCPServer(false) error: %v", err)
	} else {
		t.Logf("✅ Server '%s' disabled successfully", serverToTest)
	}

	// Wait a moment
	time.Sleep(500 * time.Millisecond)

	// Re-enable
	err = client.ToggleMCPServer(ctx, serverToTest, true)
	if err != nil {
		t.Logf("ToggleMCPServer(true) error: %v", err)
	} else {
		t.Logf("✅ Server '%s' re-enabled successfully", serverToTest)
	}

	// Step 4: Final query to verify everything works
	t.Log("\n=== Step 4: Final query ===")
	msgChan2, err := client.Query(ctx, "Say 'MiniMax MCP control test complete'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	drainUntilResult(ctx, msgChan2)
}

// TestMCPControlWithCustomStdioServer tests MCP control with a custom stdio MCP server.
func TestMCPControlWithCustomStdioServer(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Configure a simple stdio MCP server using uvx
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
		MCPServers: map[string]types.McpServerConfig{
			"custom-test": types.McpStdioServerConfig{
				Type:    "stdio",
				Command: "echo",
				Args:    []string{"MCP server response"},
			},
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First query to trigger MCP
	t.Log("=== Step 0: Initial query ===")
	msgChan, err := client.Query(ctx, "What is 1+1?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	drainUntilResult(ctx, msgChan)

	// Step 1: Get MCP status
	t.Log("\n=== Step 1: Get MCP status ===")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Logf("GetMCPStatus error: %v", err)
	} else {
		t.Logf("MCP Status: %+v", status)

		// Find our custom server
		servers, ok := status["mcpServers"].([]interface{})
		if ok {
			t.Logf("Found %d servers in status", len(servers))
			for _, s := range servers {
				if serverInfo, ok := s.(map[string]interface{}); ok {
					name, _ := serverInfo["name"].(string)
					serverStatus, _ := serverInfo["status"].(string)
					t.Logf("  Server: %s, Status: %s", name, serverStatus)
				}
			}
		}
	}

	// Step 2: Test ReconnectMCPServer
	t.Log("\n=== Step 2: Test ReconnectMCPServer ===")
	err = client.ReconnectMCPServer(ctx, "custom-test")
	if err != nil {
		t.Logf("ReconnectMCPServer error: %v", err)
	} else {
		t.Log("✅ ReconnectMCPServer succeeded")
	}

	// Step 3: Test ToggleMCPServer
	t.Log("\n=== Step 3: Test ToggleMCPServer ===")
	err = client.ToggleMCPServer(ctx, "custom-test", false)
	if err != nil {
		t.Logf("ToggleMCPServer(false) error: %v", err)
	} else {
		t.Log("✅ ToggleMCPServer(false) succeeded")
	}

	err = client.ToggleMCPServer(ctx, "custom-test", true)
	if err != nil {
		t.Logf("ToggleMCPServer(true) error: %v", err)
	} else {
		t.Log("✅ ToggleMCPServer(true) succeeded")
	}

	// Step 4: Final query
	t.Log("\n=== Step 4: Final query ===")
	msgChan2, err := client.Query(ctx, "Say 'custom MCP test complete'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	drainUntilResult(ctx, msgChan2)
}

// TestMCPStatusDetailed tests GetMCPStatus returns detailed server info.
func TestMCPStatusDetailed(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First query to trigger MCP connections
	t.Log("=== Step 0: Initial query to trigger MCP ===")
	msgChan, err := client.Query(ctx, "List the available tools")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	drainUntilResult(ctx, msgChan)

	// Get MCP status
	t.Log("\n=== Step 1: Get MCP Status ===")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Fatalf("GetMCPStatus failed: %v", err)
	}

	t.Log("=== MCP Status Details ===")

	// Pretty print the status
	servers, ok := status["mcpServers"].([]interface{})
	if !ok {
		t.Log("No mcpServers found in response")
		return
	}

	t.Logf("Total MCP Servers: %d", len(servers))

	for i, s := range servers {
		if serverInfo, ok := s.(map[string]interface{}); ok {
			t.Logf("\n--- Server %d ---", i+1)
			t.Logf("  Name: %v", serverInfo["name"])
			t.Logf("  Status: %v", serverInfo["status"])

			if si, ok := serverInfo["serverInfo"].(map[string]interface{}); ok {
				t.Logf("  ServerInfo.Name: %v", si["name"])
				t.Logf("  ServerInfo.Version: %v", si["version"])
			}

			if tools, ok := serverInfo["tools"].([]interface{}); ok {
				t.Logf("  Tools: %d", len(tools))
				for j, tool := range tools {
					if toolInfo, ok := tool.(map[string]interface{}); ok {
						t.Logf("    [%d] %v", j+1, toolInfo["name"])
					}
				}
			}

			if cfg, ok := serverInfo["config"].(map[string]interface{}); ok {
				t.Logf("  Config.Type: %v", cfg["type"])
			}
		}
	}
}

// TestStopTaskWithRealTask tests StopTask with a real task if possible.
func TestStopTaskWithRealTask(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(10),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Start a query that might trigger a subagent/task
	msgChan, err := client.Query(ctx, "Use the Bash tool to run a long command like 'sleep 5 && echo done'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Look for task_started message to get a real task ID
	var taskID string
	resultFound := false
	timeoutChan := time.After(30 * time.Second)

	for taskID == "" && !resultFound {
		select {
		case <-timeoutChan:
			t.Log("Timeout waiting for task_started message")
			resultFound = true
		case msg, ok := <-msgChan:
			if !ok {
				t.Log("Channel closed")
				resultFound = true
				break
			}

			switch m := msg.(type) {
			case *types.TaskStartedMessage:
				taskID = m.TaskID
				t.Logf("Found task: %s - %s", m.TaskID, m.Description)

				// Try to stop this task
				err := client.StopTask(ctx, taskID)
				if err != nil {
					t.Logf("StopTask error: %v", err)
				} else {
					t.Logf("✅ StopTask succeeded for task %s", taskID)
				}

			case *types.TaskNotificationMessage:
				t.Logf("Task notification: %s - %s", m.TaskID, m.Status)

			case *types.ResultMessage:
				t.Logf("Result received: IsError=%v", m.IsError)
				resultFound = true
			}
		}
	}

	// Drain remaining messages
	go func() {
		for range msgChan {
		}
	}()

	if taskID == "" {
		t.Log("No task was started during this query")
	}
}

// drainUntilResult drains messages from channel until ResultMessage is found.
func drainUntilResult(ctx context.Context, msgChan <-chan types.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			if _, isResult := msg.(*types.ResultMessage); isResult {
				// Drain remaining in background
				go func() {
					for range msgChan {
					}
				}()
				return
			}
		}
	}
}

// queryAndDrainResult executes a query and waits for result.
func queryAndDrainResult(ctx context.Context, t *testing.T, client *claude.Client, prompt string) {
	msgChan, err := client.Query(ctx, prompt)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	resultFound := false
	for !resultFound {
		select {
		case <-ctx.Done():
			t.Log("Context timeout")
			resultFound = true
		case msg, ok := <-msgChan:
			if !ok {
				t.Log("Channel closed")
				resultFound = true
				break
			}
			if _, isResult := msg.(*types.ResultMessage); isResult {
				t.Logf("✅ Query completed: %s", prompt)
				resultFound = true
				// Drain remaining
				go func() {
					for range msgChan {
					}
				}()
			}
		}
	}
}
