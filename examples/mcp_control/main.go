// Example: MCP Control Operations
//
// This example demonstrates how to use the new MCP control methods:
// - ReconnectMCPServer: Reconnect a disconnected MCP server
// - ToggleMCPServer: Enable/disable an MCP server
// - StopTask: Stop a running task
//
// Run with: go run examples/mcp_control/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create SDK MCP tools
	echoTool := sdkmcp.Tool("echo", "Echo back a message", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{"type": "string"},
		},
		"required": []string{"message"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		message, _ := args["message"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Echo: %s", message)), nil
	})

	timeTool := sdkmcp.Tool("get_time", "Get the current time", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		return sdkmcp.TextResult(time.Now().Format(time.RFC3339)), nil
	})

	// Create SDK MCP server
	server := sdkmcp.CreateSdkMcpServer("demo", []*sdkmcp.SdkMcpTool{echoTool, timeTool})

	// Configure client
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"demo": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__demo__echo", "mcp__demo__get_time"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(5),
	})
	defer client.Close()

	// Connect
	fmt.Println("=== Connecting to Claude CLI ===")
	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected successfully!")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(ctx)

	// Initial query
	fmt.Println("\n=== Initial Query ===")
	queryAndConsume(ctx, client, msgChan, "Use the echo tool to say 'Hello, World!'")

	// Get MCP status
	fmt.Println("\n=== Getting MCP Status ===")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		fmt.Printf("GetMCPStatus error: %v\n", err)
	} else {
		// Status is a map[string]interface{} containing "mcpServers" key
		fmt.Printf("MCP Status: %+v\n", status)
		if servers, ok := status["mcpServers"].([]interface{}); ok {
			fmt.Printf("MCP Servers: %d\n", len(servers))
			for _, s := range servers {
				if serverInfo, ok := s.(map[string]interface{}); ok {
					name, _ := serverInfo["name"].(string)
					serverStatus, _ := serverInfo["status"].(string)
					fmt.Printf("  - %s: %s\n", name, serverStatus)
				}
			}
		}
	}

	// Toggle MCP server off
	fmt.Println("\n=== Toggling MCP Server OFF ===")
	if err := client.ToggleMCPServer(ctx, "demo", false); err != nil {
		fmt.Printf("ToggleMCPServer(false) error: %v\n", err)
	} else {
		fmt.Println("MCP server 'demo' disabled")
	}

	// Toggle MCP server on
	fmt.Println("\n=== Toggling MCP Server ON ===")
	if err := client.ToggleMCPServer(ctx, "demo", true); err != nil {
		fmt.Printf("ToggleMCPServer(true) error: %v\n", err)
	} else {
		fmt.Println("MCP server 'demo' enabled")
	}

	// Reconnect MCP server
	fmt.Println("\n=== Reconnecting MCP Server ===")
	if err := client.ReconnectMCPServer(ctx, "demo"); err != nil {
		fmt.Printf("ReconnectMCPServer error: %v\n", err)
	} else {
		fmt.Println("MCP server 'demo' reconnected")
	}

	// Query after control operations
	fmt.Println("\n=== Query After Control Operations ===")
	queryAndConsume(ctx, client, msgChan, "Use the get_time tool to tell me the current time")

	// Demonstrate StopTask (with a fake task ID - will likely fail but shows API usage)
	fmt.Println("\n=== StopTask Demo ===")
	// Note: In real usage, you would get the task ID from a TaskStartedMessage
	taskID := "example-task-id"
	if err := client.StopTask(ctx, taskID); err != nil {
		fmt.Printf("StopTask error (expected for fake ID): %v\n", err)
	}

	fmt.Println("\n=== Example Complete ===")
}

func queryAndConsume(ctx context.Context, client *claude.Client, msgChan <-chan types.Message, prompt string) {
	if err := client.Query(ctx, prompt); err != nil {
		fmt.Printf("Query error: %v\n", err)
		return
	}

	fmt.Printf("Prompt: %s\n", prompt)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context done")
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}

			switch m := msg.(type) {
			case *types.AssistantMessage:
				for _, block := range m.Content {
					if tb, ok := block.(types.TextBlock); ok {
						fmt.Printf("Assistant: %s\n", tb.Text)
					}
				}
			case *types.TaskStartedMessage:
				fmt.Printf("Task Started: %s - %s\n", m.TaskID, m.Description)
			case *types.TaskProgressMessage:
				fmt.Printf("Task Progress: %s (tokens: %d)\n", m.TaskID, m.Usage.TotalTokens)
			case *types.TaskNotificationMessage:
				fmt.Printf("Task Notification: %s - %s\n", m.TaskID, m.Status)
			case *types.ResultMessage:
				cost := "<nil>"
				if m.TotalCostUSD != nil {
					cost = fmt.Sprintf("$%.6f", *m.TotalCostUSD)
				}
				fmt.Printf("Result: IsError=%v, Cost=%s, Turns=%d\n", m.IsError, cost, m.NumTurns)
				// Drain remaining messages
				go func() {
					for range msgChan {
					}
				}()
				return
			}
		}
	}
}
