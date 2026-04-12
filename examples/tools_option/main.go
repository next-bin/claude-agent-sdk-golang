// Example tools_option demonstrates using Tools, AllowedTools, and DisallowedTools options
// with the Claude Agent SDK for Go.
//
// This example shows:
// 1. Using Tools option with ToolsPreset for preset tool configurations
// 2. Using AllowedTools option to restrict available tools to a specific list
// 3. Using DisallowedTools option to block specific tools
//
// Tool control is useful for:
// - Limiting agent capabilities for safety
// - Creating specialized agents with focused toolsets
// - Preventing access to potentially dangerous operations
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/next-bin/claude-agent-sdk-golang/client"
	"github.com/next-bin/claude-agent-sdk-golang/examples/internal"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	fmt.Println("=== Claude Agent SDK Go - Tools Option Example ===")
	fmt.Println()

	// Example 1: Using Tools option with ToolsPreset
	fmt.Println("--- Example 1: Tools with ToolsPreset ---")
	demoToolsPreset(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 2: Using AllowedTools to restrict available tools
	fmt.Println("--- Example 2: AllowedTools ---")
	demoAllowedTools(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 3: Using DisallowedTools to block specific tools
	fmt.Println("--- Example 3: DisallowedTools ---")
	demoDisallowedTools(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 4: Combining tool restrictions
	fmt.Println("--- Example 4: Combining Tool Restrictions ---")
	demoCombinedTools(ctx)
}

// demoToolsPreset demonstrates using the Tools option with a ToolsPreset.
// ToolsPreset provides a predefined set of tools configured by Claude Code.
func demoToolsPreset(ctx context.Context) {
	// ToolsPreset uses the Claude Code preset tool configuration
	// This provides the standard set of tools available in Claude Code
	toolsPreset := types.ToolsPreset{
		Type:   "preset",
		Preset: "claude_code",
	}

	options := &types.ClaudeAgentOptions{
		Tools: toolsPreset,
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("Configured with ToolsPreset (claude_code)")
	fmt.Println("This provides the standard Claude Code tool set including:")
	fmt.Println("  - Read, Write, Edit (file operations)")
	fmt.Println("  - Bash (command execution)")
	fmt.Println("  - Glob, Grep (search tools)")
	fmt.Println("  - And other Claude Code tools")
	fmt.Println()
	fmt.Println("Query: What tools do you have available?")

	if err := c.Query(ctx, "What tools do you have available? List them briefly."); err != nil {
		handleError(err)
		return
	}

	processMessages(msgChan)
}

// demoAllowedTools demonstrates restricting available tools using AllowedTools.
// Only the tools listed in AllowedTools will be accessible to the agent.
func demoAllowedTools(ctx context.Context) {
	// Restrict agent to only use Read and Glob tools
	// The agent cannot use Bash, Write, Edit, or any other tools
	allowedTools := []string{
		"Read", // Allow reading files
		"Glob", // Allow searching for files by pattern
		"Grep", // Allow searching file contents
	}

	options := &types.ClaudeAgentOptions{
		AllowedTools: allowedTools,
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("Configured with AllowedTools restriction:")
	fmt.Printf("  Allowed tools: %v\n", allowedTools)
	fmt.Println("  The agent can ONLY use these tools")
	fmt.Println("  Tools like Bash, Write, Edit are NOT available")
	fmt.Println()
	fmt.Println("Query: List the files in the current directory")

	if err := c.Query(ctx, "List the Go files in the current directory using Glob."); err != nil {
		handleError(err)
		return
	}

	processMessages(msgChan)
}

// demoDisallowedTools demonstrates blocking specific tools using DisallowedTools.
// All tools are available EXCEPT those listed in DisallowedTools.
func demoDisallowedTools(ctx context.Context) {
	// Block access to file writing and command execution
	// The agent can read files but cannot modify them or run commands
	disallowedTools := []string{
		"Write", // Block file writing
		"Edit",  // Block file editing
		"Bash",  // Block command execution
	}

	options := &types.ClaudeAgentOptions{
		DisallowedTools: disallowedTools,
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("Configured with DisallowedTools restriction:")
	fmt.Printf("  Disallowed tools: %v\n", disallowedTools)
	fmt.Println("  All other tools are available")
	fmt.Println("  The agent can READ files but cannot MODIFY or EXECUTE commands")
	fmt.Println()
	fmt.Println("Query: Read the go.mod file and explain its contents")

	if err := c.Query(ctx, "Read the go.mod file and explain its contents briefly."); err != nil {
		handleError(err)
		return
	}

	processMessages(msgChan)
}

// demoCombinedTools demonstrates combining different tool restriction options.
// This example shows read-only access by combining tools preset with disallowed tools.
func demoCombinedTools(ctx context.Context) {
	// Start with the Claude Code preset and block write/execute operations
	// This creates a read-only agent configuration
	options := &types.ClaudeAgentOptions{
		Tools: types.ToolsPreset{
			Type:   "preset",
			Preset: "claude_code",
		},
		// Block tools that can modify the system
		DisallowedTools: []string{
			"Write",
			"Edit",
			"Bash",
		},
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("Configured with combined tool restrictions:")
	fmt.Println("  Base: ToolsPreset (claude_code)")
	fmt.Println("  Additional: DisallowedTools (Write, Edit, Bash)")
	fmt.Println()
	fmt.Println("Result: Read-only agent that can search and read files")
	fmt.Println("but cannot modify anything or run shell commands")
	fmt.Println()
	fmt.Println("Query: Search for and summarize the package documentation")

	if err := c.Query(ctx, "Use Grep to find the package documentation comment in types.go and summarize it."); err != nil {
		handleError(err)
		return
	}

	processMessages(msgChan)
}

// processMessages handles incoming messages and prints the result.
func processMessages(msgChan <-chan types.Message) {
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case types.TextBlock:
					fmt.Printf("Claude: %s\n", b.Text)
				case types.ToolUseBlock:
					fmt.Printf("Tool Use: %s\n", b.Name)
				}
			}
		case *types.ResultMessage:
			printResultMessage(m)
			return
		}
	}
}

// printResultMessage displays the ResultMessage details.
func printResultMessage(m *types.ResultMessage) {
	fmt.Println("\n--- Result ---")
	fmt.Printf("Duration: %d ms (API: %d ms)\n", m.DurationMs, m.DurationAPIMs)
	fmt.Printf("Turns: %d\n", m.NumTurns)
	fmt.Printf("Session ID: %s\n", m.SessionID)

	if m.TotalCostUSD != nil {
		fmt.Printf("Total Cost: $%.6f\n", *m.TotalCostUSD)
	}

	if m.IsError {
		fmt.Println("Status: Error")
		if m.Result != nil {
			fmt.Printf("Error: %s\n", *m.Result)
		}
	} else {
		fmt.Println("Status: Success")
	}
}

// handleError prints error information with helpful suggestions.
func handleError(err error) {
	fmt.Printf("Error: %v\n", err)
	fmt.Println("\nPlease ensure Claude CLI is installed and authenticated:")
	fmt.Println("  npm install -g @anthropic-ai/claude-code")
	fmt.Println("  claude login")
	os.Exit(1)
}
