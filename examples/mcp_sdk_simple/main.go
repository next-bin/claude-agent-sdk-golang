// Example mcp_sdk_simple demonstrates using the sdkmcp convenience package.
//
// This example shows how to:
// 1. Create tools using sdkmcp.Tool() with simple handler functions
// 2. Create an SDK MCP server using sdkmcp.CreateSdkMcpServer()
// 3. Return text and image results using sdkmcp.TextResult() and sdkmcp.ImageResult()
//
// Compare this with examples/mcp_sdk_server to see how much simpler
// the sdkmcp package makes MCP server creation.
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	fmt.Println("=== Claude Agent SDK Go - Simple SDK MCP Server Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates the sdkmcp convenience package")
	fmt.Println("which simplifies creating in-process MCP servers.")
	fmt.Println()

	// Create tools using sdkmcp.Tool()
	addTool := sdkmcp.Tool("add", "Add two numbers together", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return sdkmcp.TextResult(fmt.Sprintf("%.2f + %.2f = %.2f", a, b, a+b)), nil
	})

	multiplyTool := sdkmcp.Tool("multiply", "Multiply two numbers", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return sdkmcp.TextResult(fmt.Sprintf("%.2f × %.2f = %.2f", a, b, a*b)), nil
	})

	greetTool := sdkmcp.Tool("greet", "Greet a person by name", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []string{"name"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		name, _ := args["name"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Hello, %s! Nice to meet you!", name)), nil
	})

	// Create the SDK MCP server with multiple tools
	server := sdkmcp.CreateSdkMcpServer("simple-calc", []*sdkmcp.SdkMcpTool{
		addTool,
		multiplyTool,
		greetTool,
	})

	fmt.Println("Created SDK MCP server 'simple-calc' with 3 tools:")
	fmt.Println("  - add: Add two numbers")
	fmt.Println("  - multiply: Multiply two numbers")
	fmt.Println("  - greet: Greet a person")
	fmt.Println()

	// Configure the SDK client with the SDK MCP server
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"calc": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools: []string{
			"mcp__calc__add",
			"mcp__calc__multiply",
			"mcp__calc__greet",
		},
		PermissionMode: &mode,
	})
	defer client.Close()

	fmt.Println("SDK client configured with MCP server")
	fmt.Println()

	// Connect to Claude
	fmt.Println("Connecting to Claude...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Sending query: 'Calculate 15 + 27, then multiply the result by 3'")
	fmt.Println()

	// Query Claude
	msgChan, err := client.Query(ctx, "Calculate 15 + 27, then multiply the result by 3. Show me the calculations.")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Process messages
	fmt.Println("=== Response ===")
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, content := range m.Content {
				if text, ok := content.(types.TextBlock); ok {
					fmt.Print(text.Text)
				}
			}
		case *types.ResultMessage:
			fmt.Println()
			fmt.Println()
			if m.IsError {
				fmt.Printf("Error: %v\n", m.Result)
			} else {
				fmt.Println("=== Session Complete ===")
				if m.TotalCostUSD != nil {
					fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
				}
				fmt.Printf("Duration: %dms\n", m.DurationMs)
				fmt.Printf("Turns: %d\n", m.NumTurns)
			}
		}
	}
}
