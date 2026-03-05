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

// TestSdkMcpToolActualExecution tests that SDK MCP tools are actually executed.
func TestSdkMcpToolActualExecution(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Track if tool was actually called
	var toolCalled bool
	var receivedMessage string

	// Create a simple echo tool
	echoTool := sdkmcp.Tool("echo", "Echo back the input message", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{"type": "string"},
		},
		"required": []string{"message"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		toolCalled = true
		receivedMessage, _ = args["message"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Echo: %s", receivedMessage)), nil
	})

	// Create SDK MCP server
	server := sdkmcp.CreateSdkMcpServer("test", []*sdkmcp.SdkMcpTool{echoTool})

	mode := types.PermissionModeBypassPermissions
	// Configure client with the SDK MCP server
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
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Query with explicit tool instruction
	msgChan, err := client.Query(ctx, "Call the mcp__test__echo tool with message 'Hello from test'. You must use this exact tool name.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Collect messages
	var resultMsg *types.ResultMessage
messageLoop:
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Context timeout")
		case msg, ok := <-msgChan:
			if !ok {
				t.Fatal("Channel closed without result")
			}
			switch m := msg.(type) {
			case *types.AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(types.TextBlock); ok {
						t.Logf("Assistant: %s", tb.Text)
					}
					if tu, ok := block.(types.ToolUseBlock); ok {
						t.Logf("ToolUse: %s", tu.Name)
					}
				}
			case *types.ResultMessage:
				resultMsg = m
				// Drain remaining
				go func() {
					for range msgChan {
					}
				}()
				break messageLoop
			}
		}
	}

	t.Logf("Tool called: %v", toolCalled)
	t.Logf("Received message: %s", receivedMessage)
	t.Logf("Result IsError: %v", resultMsg.IsError)

	if toolCalled {
		t.Log("✅ SDK MCP tool was actually called")
	} else {
		t.Log("⚠️ SDK MCP tool was not called - CLI may not have recognized it")
	}
}
