// Example plugin_example demonstrates SDK plugin configuration in the Claude Agent SDK for Go.
//
// Plugins extend the SDK with custom commands, agents, and MCP servers.
// See: https://docs.claude.com/en/docs/claude-code/plugins
//
// This example shows:
// 1. Configuring local plugins
// 2. Plugin path configuration
// 3. Using plugins with the SDK
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Claude Agent SDK Go - Plugin Example ===")
	fmt.Println()

	// Example 1: Local plugin configuration
	localPluginExample(ctx)

	// Example 2: Multiple plugins
	multiplePluginsExample(ctx)

	// Example 3: Plugin with other options
	pluginWithOptionsExample(ctx)
}

// localPluginExample demonstrates configuring a local plugin.
func localPluginExample(ctx context.Context) {
	fmt.Println("--- Example 1: Local Plugin Configuration ---")
	fmt.Println("Plugins extend the SDK with custom functionality.")
	fmt.Println()

	// Configure a local plugin
	// Plugins are typically directories containing:
	// - commands/ directory with slash commands
	// - agents/ directory with custom agents
	// - MCP server configurations
	pluginPath := "/path/to/your/plugin"

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Plugins: []types.SdkPluginConfig{
			{
				Type: "local",
				Path: pluginPath,
			},
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Plugin configuration:")
	fmt.Printf("  - Type: local\n")
	fmt.Printf("  - Path: %s\n", pluginPath)
	fmt.Println()
	fmt.Println("Plugin structure expected:")
	fmt.Println("  your-plugin/")
	fmt.Println("  ├── commands/")
	fmt.Println("  │   └── my-command.md    # Custom slash command")
	fmt.Println("  ├── agents/")
	fmt.Println("  │   └── my-agent.md       # Custom agent definition")
	fmt.Println("  └── mcp-servers/")
	fmt.Println("      └── server.json      # MCP server configuration")
	fmt.Println()
}

// multiplePluginsExample demonstrates configuring multiple plugins.
func multiplePluginsExample(ctx context.Context) {
	fmt.Println("--- Example 2: Multiple Plugins ---")
	fmt.Println("You can load multiple plugins for different functionalities.")
	fmt.Println()

	// Configure multiple plugins
	// Each plugin can provide different commands, agents, or MCP servers
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Plugins: []types.SdkPluginConfig{
			{
				Type: "local",
				Path: "/path/to/code-review-plugin",
			},
			{
				Type: "local",
				Path: "/path/to/documentation-plugin",
			},
			{
				Type: "local",
				Path: "/path/to/testing-plugin",
			},
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Multiple plugins configured:")
	fmt.Println("  1. code-review-plugin - Code review commands and agents")
	fmt.Println("  2. documentation-plugin - Documentation generators")
	fmt.Println("  3. testing-plugin - Test generation and analysis")
	fmt.Println()

	// Example plugin structures
	fmt.Println("Example: code-review-plugin structure:")
	fmt.Println("  code-review-plugin/")
	fmt.Println("  ├── commands/")
	fmt.Println("  │   ├── review.md         # /review command")
	fmt.Println("  │   ├── lint.md           # /lint command")
	fmt.Println("  │   └── security-scan.md  # /security-scan command")
	fmt.Println("  └── agents/")
	fmt.Println("      └── reviewer.md        # Code review agent")
	fmt.Println()
}

// pluginWithOptionsExample demonstrates plugins combined with other SDK options.
func pluginWithOptionsExample(ctx context.Context) {
	fmt.Println("--- Example 3: Plugin with Other Options ---")
	fmt.Println("Plugins work alongside other SDK configurations.")
	fmt.Println()

	// Combine plugins with other options
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		// Plugin configuration
		Plugins: []types.SdkPluginConfig{
			{
				Type: "local",
				Path: "/path/to/custom-tools-plugin",
			},
		},

		// Model selection
		Model: types.String("claude-sonnet-4-20250514"),

		// Custom system prompt
		SystemPrompt: "You are a helpful coding assistant with access to custom plugins.",

		// Allowed tools (can include plugin tools)
		AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},

		// Permission mode for automated operations
		PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Combined configuration:")
	fmt.Println("  - Plugin: custom-tools-plugin")
	fmt.Println("  - Model: claude-sonnet-4-20250514")
	fmt.Println("  - Custom system prompt")
	fmt.Println("  - Permission mode: acceptEdits")
	fmt.Println()

	// How plugins extend functionality
	fmt.Println("Plugin capabilities:")
	fmt.Println("  - Add custom slash commands (e.g., /my-custom-command)")
	fmt.Println("  - Define specialized agents")
	fmt.Println("  - Configure MCP servers")
	fmt.Println("  - Extend tool capabilities")
	fmt.Println()
}

// Example plugin command file content:
// File: commands/my-command.md
const exampleCommandContent = `
# My Custom Command

A custom slash command that does something useful.

## Usage

/my-command [options]

## Arguments

- options: Optional arguments for the command

## Example

/my-command --verbose

## Implementation

This command will:
1. Read the specified file or directory
2. Process the content
3. Generate output
`

// Example plugin agent file content:
// File: agents/my-agent.md
const exampleAgentContent = `
# My Custom Agent

A specialized agent for specific tasks.

## Description

This agent handles:
- Task type 1
- Task type 2

## Prompt

You are a specialized agent. Your task is to:
1. Analyze the input
2. Process according to specific rules
3. Generate appropriate output

## Tools

- Read
- Write
- Bash
- Glob

## Model

sonnet
`