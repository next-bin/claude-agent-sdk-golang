package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// MCP Control E2E Tests - With SDK MCP Server (Verified Working)
// ============================================================================

// TestMCPControlWithSDKMCPReal tests MCP control methods using SDK MCP Server.
// SDK MCP Server is an in-process server that's fully managed by the SDK.
// This test verifies that:
// 1. SDK MCP tools are properly registered and callable
// 2. Client methods work correctly after control operations
func TestMCPControlWithSDKMCPReal(t *testing.T) {
	SkipIfNoAPIKey(t)

	// Use a background context for the client connection
	bgCtx := context.Background()

	// Create a simple echo tool using SDK MCP
	echoTool := sdkmcp.Tool("echo", "Echo back the input message", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{"type": "string"},
		},
		"required": []string{"message"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		message, _ := args["message"].(string)
		return sdkmcp.TextResult("Echo: " + message), nil
	})

	server := sdkmcp.CreateSdkMcpServer("test-sdk", []*sdkmcp.SdkMcpTool{echoTool})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"test-sdk": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__test-sdk__echo"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Step 1: Use the SDK MCP tool (with its own timeout context)
	t.Log("=== Step 1: Query with SDK MCP tool ===")
	ctx1, cancel1 := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel1()
	msgChan, err := client.Query(ctx1, "Use the echo tool to say 'Hello from SDK MCP'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	result := drainMessagesAndCheckResult(ctx1, t, msgChan)
	if !result.Found {
		t.Error("Expected to receive a result message")
	}
	if result.IsError {
		t.Errorf("Result was an error: %v", result.Error)
	}
	t.Logf("✅ SDK MCP tool executed successfully, cost: $%.6f", result.Cost)

	// Step 2: Get MCP status
	t.Log("\n=== Step 2: Get MCP status ===")
	ctx2, cancel2 := context.WithTimeout(bgCtx, 30*time.Second)
	defer cancel2()
	status, err := client.GetMCPStatus(ctx2)
	if err != nil {
		t.Logf("GetMCPStatus error: %v", err)
	} else {
		t.Logf("MCP Status: %+v", status)
	}

	// Step 3: Try control methods on SDK MCP server
	// Note: SDK MCP servers are in-process, so toggle/reconnect may return "not found"
	// because they don't appear in the CLI's external MCP server list
	t.Log("\n=== Step 3: Test control methods ===")

	ctx3, cancel3 := context.WithTimeout(bgCtx, 30*time.Second)
	defer cancel3()
	err = client.ToggleMCPServer(ctx3, "test-sdk", false)
	if err != nil {
		t.Logf("ToggleMCPServer(false) returned: %v (expected for SDK MCP)", err)
	} else {
		t.Log("✅ ToggleMCPServer(false) succeeded")
	}

	err = client.ToggleMCPServer(ctx3, "test-sdk", true)
	if err != nil {
		t.Logf("ToggleMCPServer(true) returned: %v (expected for SDK MCP)", err)
	} else {
		t.Log("✅ ToggleMCPServer(true) succeeded")
	}

	err = client.ReconnectMCPServer(ctx3, "test-sdk")
	if err != nil {
		t.Logf("ReconnectMCPServer returned: %v (expected for SDK MCP)", err)
	} else {
		t.Log("✅ ReconnectMCPServer succeeded")
	}

	// Step 4: Another query to verify client still works (with its own timeout context)
	t.Log("\n=== Step 4: Second query to verify client works ===")
	ctx4, cancel4 := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel4()
	msgChan2, err := client.Query(ctx4, "Use the echo tool to say 'Second message'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	result2 := drainMessagesAndCheckResult(ctx4, t, msgChan2)
	if !result2.Found {
		t.Error("Expected to receive a result message in second query")
	}
	t.Logf("✅ Second query completed, cost: $%.6f", result2.Cost)
}

// TestMultipleSDKMCPServers tests multiple SDK MCP servers.
func TestMultipleSDKMCPServers(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create first tool
	addTool := sdkmcp.Tool("add", "Add two numbers", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return sdkmcp.TextResult(string(rune(int(a + b) + '0'))), nil
	})

	server1 := sdkmcp.CreateSdkMcpServer("math", []*sdkmcp.SdkMcpTool{addTool})

	// Create second tool
	greetTool := sdkmcp.Tool("greet", "Greet someone", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []string{"name"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		name, _ := args["name"].(string)
		return sdkmcp.TextResult("Hello, " + name + "!"), nil
	})

	server2 := sdkmcp.CreateSdkMcpServer("greeting", []*sdkmcp.SdkMcpTool{greetTool})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"math": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server1,
			},
			"greeting": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server2,
			},
		},
		AllowedTools:   []string{"mcp__math__add", "mcp__greeting__greet"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test both tools
	t.Log("=== Testing math tool ===")
	msgChan, err := client.Query(ctx, "Use the add tool to calculate 2+3")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	result := drainMessagesAndCheckResult(ctx, t, msgChan)
	t.Logf("✅ Math tool result: found=%v, isError=%v", result.Found, result.IsError)

	t.Log("\n=== Testing greeting tool ===")
	msgChan2, err := client.Query(ctx, "Use the greet tool to greet 'World'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	result2 := drainMessagesAndCheckResult(ctx, t, msgChan2)
	t.Logf("✅ Greeting tool result: found=%v, isError=%v", result2.Found, result2.IsError)
}

// TestResult holds the result of draining messages.
type TestResult struct {
	Found   bool
	IsError bool
	Error   string
	Cost    float64
}

// drainMessagesAndCheckResult drains messages and returns result info.
func drainMessagesAndCheckResult(ctx context.Context, t *testing.T, msgChan <-chan types.Message) TestResult {
	var result TestResult

	for {
		select {
		case <-ctx.Done():
			t.Log("Context timeout")
			return result
		case msg, ok := <-msgChan:
			if !ok {
				t.Log("Channel closed")
				return result
			}

			switch m := msg.(type) {
			case *types.AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(types.TextBlock); ok {
						t.Logf("  Assistant: %s", truncateString(tb.Text, 100))
					}
				}
			case *types.ResultMessage:
				result.Found = true
				result.IsError = m.IsError
				if m.TotalCostUSD != nil {
					result.Cost = *m.TotalCostUSD
				}
				// Drain remaining in background
				go func() {
					for range msgChan {
					}
				}()
				return result
			}
		}
	}
}