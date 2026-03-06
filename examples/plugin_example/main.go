// Example plugin_example demonstrates SDK plugin configuration in the Claude Agent SDK for Go.
//
// Plugins extend the SDK with custom commands, agents, and MCP servers.
// See: https://docs.claude.com/en/docs/claude-code/plugins
//
// This example shows:
// 1. Loading a local demo plugin
// 2. Verifying plugin loading via system messages
// 3. Configuring multiple plugins
// 4. Using plugins with other SDK options
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/examples/internal"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	fmt.Println("=== Claude Agent SDK Go - Plugin Example ===")
	fmt.Println()

	// Example 1: Load and verify demo plugin
	demoPluginExample(ctx)

	// Example 2: Multiple plugins configuration
	multiplePluginsExample(ctx)

	// Example 3: Plugin with other SDK options
	pluginWithOptionsExample(ctx)

	// Example 4: Plugin structure reference
	pluginStructureExample(ctx)
}

// getDemoPluginPath returns the path to the demo plugin directory.
// In a real application, you would use an absolute path or relative path
// based on your project structure.
func getDemoPluginPath() string {
	// Get the directory of this source file
	_, filename, _, _ := runtime.Caller(0)
	examplesDir := filepath.Dir(filepath.Dir(filename))
	pluginPath := filepath.Join(examplesDir, "plugins", "demo-plugin")
	return pluginPath
}

// demoPluginExample demonstrates loading the demo plugin and verifying it's loaded.
func demoPluginExample(ctx context.Context) {
	fmt.Println("--- Example 1: Load and Verify Demo Plugin ---")
	fmt.Println("Load a local plugin and check that it's properly registered.")
	fmt.Println()

	// Get the path to the demo plugin
	pluginPath := getDemoPluginPath()
	fmt.Printf("Plugin path: %s\n", pluginPath)

	// Configure the client with the demo plugin
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Plugins: []types.SdkPluginConfig{
			{
				Type: "local",
				Path: pluginPath,
			},
		},
		MaxTurns: types.Int(1), // Limit to one turn for quick demo
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Plugin configured successfully!")
	fmt.Println()

	// The plugin provides a /greet command
	fmt.Println("Demo plugin provides:")
	fmt.Println("  - /greet command: A custom greeting command")
	fmt.Println()

	// To verify the plugin is loaded, send a query and check system messages
	fmt.Println("To verify plugin loading, check the SystemMessage with subtype 'init'")
	fmt.Println("The plugins data will contain information about loaded plugins.")
	fmt.Println()

	// Code pattern for verifying plugin loading:
	fmt.Println("// Code pattern for verifying plugins:")
	fmt.Println("// for msg := range msgChan {")
	fmt.Println("//     if sysMsg, ok := msg.(*types.SystemMessage); ok {")
	fmt.Println("//         if sysMsg.Subtype == \"init\" {")
	fmt.Println("//             if plugins, ok := sysMsg.Data[\"plugins\"].([]interface{}); ok {")
	fmt.Println("//                 for _, p := range plugins {")
	fmt.Println("//                     // Print loaded plugin info")
	fmt.Println("//                 }")
	fmt.Println("//             }")
	fmt.Println("//         }")
	fmt.Println("//     }")
	fmt.Println("// }")
	fmt.Println()
}

// multiplePluginsExample demonstrates configuring multiple plugins.
func multiplePluginsExample(ctx context.Context) {
	fmt.Println("--- Example 2: Multiple Plugins ---")
	fmt.Println("You can load multiple plugins for different functionalities.")
	fmt.Println()

	// Configure multiple plugins
	// Each plugin can provide different commands, agents, or MCP servers
	pluginPath := getDemoPluginPath()

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Plugins: []types.SdkPluginConfig{
			{
				Type: "local",
				Path: pluginPath,
			},
			{
				Type: "local",
				Path: "/path/to/another-plugin",
			},
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Multiple plugins can be configured:")
	fmt.Println("  1. demo-plugin - Provides /greet command")
	fmt.Println("  2. another-plugin - Additional functionality")
	fmt.Println()

	// Example: Plugin configuration options
	fmt.Println("Plugin configuration options:")
	fmt.Println("  - Type: \"local\" for local plugin directories")
	fmt.Println("  - Path: Absolute or relative path to plugin directory")
	fmt.Println()
}

// pluginWithOptionsExample demonstrates plugins combined with other SDK options.
func pluginWithOptionsExample(ctx context.Context) {
	fmt.Println("--- Example 3: Plugin with Other Options ---")
	fmt.Println("Plugins work alongside other SDK configurations.")
	fmt.Println()

	pluginPath := getDemoPluginPath()

	// Combine plugins with other options
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		// Plugin configuration
		Plugins: []types.SdkPluginConfig{
			{
				Type: "local",
				Path: pluginPath,
			},
		},

		// Model selection
		Model: types.String(types.ModelSonnet),

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
	fmt.Println("  - Plugin: demo-plugin")
	fmt.Println("  - Model: claude-sonnet-4-20250514")
	fmt.Println("  - Custom system prompt")
	fmt.Println("  - Permission mode: acceptEdits")
	fmt.Println()

	// How plugins extend functionality
	fmt.Println("Plugin capabilities:")
	fmt.Println("  - Add custom slash commands (e.g., /greet)")
	fmt.Println("  - Define specialized agents")
	fmt.Println("  - Configure MCP servers")
	fmt.Println("  - Extend tool capabilities")
	fmt.Println()
}

// pluginStructureExample shows the expected structure of a plugin.
func pluginStructureExample(ctx context.Context) {
	fmt.Println("--- Example 4: Plugin Structure Reference ---")
	fmt.Println("A plugin directory should follow this structure:")
	fmt.Println()

	fmt.Println("demo-plugin/")
	fmt.Println("├── .claude-plugin/")
	fmt.Println("│   └── plugin.json       # Plugin metadata (required)")
	fmt.Println("└── commands/")
	fmt.Println("    └── greet.md          # Custom slash command /greet")
	fmt.Println()

	fmt.Println("plugin.json format:")
	fmt.Println(`{
  "name": "demo-plugin",
  "description": "A demo plugin showing how to extend Claude Code",
  "version": "1.0.0",
  "author": {
    "name": "Your Name"
  }
}`)
	fmt.Println()

	fmt.Println("commands/greet.md format:")
	fmt.Println(`# Greet Command

This is a custom greeting command from the demo plugin.

When the user runs this command, greet them warmly and explain
that this message came from a custom plugin loaded via the SDK.`)
	fmt.Println()

	fmt.Println("Optional plugin directories:")
	fmt.Println("  agents/         - Custom agent definitions (.md files)")
	fmt.Println("  mcp-servers/    - MCP server configurations (.json files)")
	fmt.Println("  skills/         - Custom skills (.md files)")
	fmt.Println("  hooks/          - Hook scripts")
	fmt.Println()
}

// runQueryWithPlugin demonstrates running a query with plugin verification.
// This function is provided as reference for actual usage.
func runQueryWithPlugin(ctx context.Context, client *claude.Client) {
	fmt.Println("Running query to verify plugin loading...")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(ctx)

	if err := client.Query(ctx, "Hello!"); err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.SystemMessage:
			if m.Subtype == "init" {
				fmt.Println("System initialized!")
				if plugins, ok := m.Data["plugins"].([]interface{}); ok {
					fmt.Printf("Plugins loaded: %d\n", len(plugins))
					for _, p := range plugins {
						if plugin, ok := p.(map[string]interface{}); ok {
							if name, ok := plugin["name"].(string); ok {
								fmt.Printf("  - %s\n", name)
							}
						}
					}
				}
			}
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if text, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", text.Text)
				}
			}
		case *types.ResultMessage:
			fmt.Println("Query completed!")
			if m.TotalCostUSD != nil {
				fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
			}
		}
	}
}
