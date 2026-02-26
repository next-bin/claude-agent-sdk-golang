// Example mcp_calculator demonstrates how to configure and use MCP (Model Context Protocol) servers
// with the Claude Agent SDK for Go.
//
// This example shows:
// 1. Configuring an MCP server using McpStdioServerConfig
// 2. Using MCPServers option in ClaudeAgentOptions
// 3. How the agent can use MCP tools provided by the server
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
// - MCP calculator server available (e.g., mcp-server-calculator or similar)
//
// The MCP server must be installed and available on your system. For this example,
// you can use any MCP server that provides calculator-like tools. Common options include:
// - mcp-server-calculator (npm install -g @modelcontextprotocol/server-calculator)
// - Any custom MCP server that implements the MCP protocol
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
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

	// Example 1: Configure MCP server using McpStdioServerConfig
	// This is the most common way to configure an MCP server that communicates via stdio.
	//
	// McpStdioServerConfig has the following fields:
	// - Type: Optional, defaults to "stdio" if not specified
	// - Command: The command to run the MCP server (required)
	// - Args: Optional command-line arguments to pass to the server
	// - Env: Optional environment variables to set for the server process
	calculatorServer := types.McpStdioServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@anthropic-ai/mcp-server-calculator"},
		Env: map[string]string{
			"NODE_OPTIONS": "--max-old-space-size=512",
		},
	}

	// Example 2: Create a map of MCP servers
	// Multiple MCP servers can be configured, each with a unique name
	mcpServers := map[string]types.McpServerConfig{
		"calculator": calculatorServer,
	}

	// Example 3: Configure ClaudeAgentOptions with MCP servers
	// The MCPServers field accepts the map we created above
	options := &types.ClaudeAgentOptions{
		Model: types.String("claude-sonnet-4-20250514"),

		// MCPServers configures one or more MCP servers that Claude can use
		// The agent will automatically discover and use tools provided by these servers
		MCPServers: mcpServers,

		// Optional: Configure permission mode
		// "bypassPermissions" is useful for automation but use with caution
		// PermissionMode: (*types.PermissionMode)(types.String("bypassPermissions")),
	}

	// Create a client with the configured options
	client := claude.NewClientWithOptions(options)
	defer client.Close()

	// Connect to Claude with the MCP servers configured
	fmt.Println("Connecting to Claude with MCP calculator server...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected successfully!")

	// Get MCP server status to verify the server is connected
	fmt.Println("\nChecking MCP server status...")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		log.Printf("Warning: Could not get MCP status: %v", err)
	} else {
		fmt.Printf("MCP Status: %+v\n", status)
	}

	// Example 4: Send a query that uses the MCP calculator tools
	// The agent will automatically discover and use the calculator tools
	// from the MCP server when appropriate
	fmt.Println("\nSending query to Claude...")
	fmt.Println("Query: 'Calculate 123 * 456 using the calculator tool'")

	msgChan, err := client.Query(ctx, "Calculate 123 * 456 using the calculator tool. Show me the result.")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Process messages from Claude
	fmt.Println("\n--- Response from Claude ---")
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			// Handle assistant messages
			fmt.Println("\n[Assistant Message]")
			for _, block := range m.Content {
				switch b := block.(type) {
				case types.TextBlock:
					fmt.Printf("Text: %s\n", b.Text)
				case types.ToolUseBlock:
					fmt.Printf("Tool Use: %s (ID: %s)\n", b.Name, b.ID)
					fmt.Printf("  Input: %+v\n", b.Input)
				case types.ToolResultBlock:
					fmt.Printf("Tool Result (ToolUseID: %s): %v\n", b.ToolUseID, b.Content)
				case types.ThinkingBlock:
					fmt.Printf("Thinking: %s...\n", truncate(b.Thinking, 100))
				default:
					fmt.Printf("Block type: %T\n", block)
				}
			}
		case *types.ResultMessage:
			// Final result message
			fmt.Println("\n[Result Message]")
			if m.Result != nil {
				fmt.Printf("Result: %s\n", *m.Result)
			}
			fmt.Printf("Duration: %dms (API: %dms)\n", m.DurationMs, m.DurationAPIMs)
			fmt.Printf("Turns: %d, Session: %s\n", m.NumTurns, m.SessionID)
			if m.TotalCostUSD != nil {
				fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
			}
			if m.IsError {
				fmt.Println("Error: true")
			}
		case *types.SystemMessage:
			// System message (e.g., MCP server status)
			fmt.Printf("\n[System Message] Subtype: %s\n", m.Subtype)
			for k, v := range m.Data {
				if k != "type" && k != "subtype" {
					fmt.Printf("  %s: %v\n", k, v)
				}
			}
		case *types.UserMessage:
			// User message echoed back
			fmt.Printf("\n[User Message] Content: %v\n", m.Content)
		case *types.StreamEvent:
			// Stream event for partial messages
			fmt.Printf("\n[Stream Event] UUID: %s\n", m.UUID)
		default:
			fmt.Printf("\n[Unknown Message Type] %T\n", msg)
		}
	}

	fmt.Println("\n--- Example Complete ---")

	// Example 5: Alternative configuration with multiple MCP servers
	fmt.Println("\n--- Alternative Configuration Example ---")
	exampleMultipleServers()
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// exampleMultipleServers demonstrates how to configure multiple MCP servers
func exampleMultipleServers() {
	// You can configure multiple MCP servers in a single client
	multipleServers := map[string]types.McpServerConfig{
		"calculator": types.McpStdioServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@anthropic-ai/mcp-server-calculator"},
		},
		"filesystem": types.McpStdioServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@anthropic-ai/mcp-server-filesystem", "/tmp"},
		},
	}

	options := &types.ClaudeAgentOptions{
		Model:      types.String("claude-sonnet-4-20250514"),
		MCPServers: multipleServers,
	}

	fmt.Printf("Options with multiple MCP servers: %+v\n", options.MCPServers)

	// Note: When using multiple MCP servers, Claude can use tools from any
	// of the configured servers. Tool names are typically prefixed with the
	// server name (e.g., "calculator_add", "filesystem_read_file")
}

// Example of using McpSSEServerConfig for SSE-based MCP servers (HTTP-based)
// Uncomment and modify as needed:
//
// func exampleSSEServer() {
// 	// For MCP servers that communicate via Server-Sent Events (SSE)
// 	sseServer := types.McpSSEServerConfig{
// 		Type: "sse",
// 		URL:  "http://localhost:8080/mcp",
// 	}
//
// 	mcpServers := map[string]types.McpServerConfig{
// 		"remote-calculator": sseServer,
// 	}
//
// 	options := &types.ClaudeAgentOptions{
// 		Model:      types.String("claude-sonnet-4-20250514"),
// 		MCPServers: mcpServers,
// 	}
//
// 	client := claude.NewClientWithOptions(options)
// 	defer client.Close()
// 	// ... rest of client usage
// }