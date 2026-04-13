// Example setting_sources demonstrates how to control which settings are loaded.
//
// Setting sources determine where Claude Code loads configurations from:
//   - "user": Global user settings (~/.claude/)
//   - "project": Project-level settings (.claude/ in project)
//   - "local": Local gitignored settings (.claude-local/)
//
// When setting_sources is not provided (nil), NO settings are loaded by default.
// This creates an isolated environment. To load settings, explicitly specify
// which sources to use.
//
// By controlling which sources are loaded, you can:
//   - Create isolated environments with no custom settings (default)
//   - Load only user settings, excluding project-specific configurations
//   - Combine multiple sources as needed
//
// Usage:
//
//	go run main.go         - List the examples
//	go run main.go all     - Run all examples
//	go run main.go default - Run a specific example
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/examples/internal"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// extractSlashCommands extracts slash command names from server info.
func extractSlashCommands(serverInfo map[string]interface{}) []string {
	if serverInfo == nil {
		return nil
	}
	if commands, ok := serverInfo["commands"].([]interface{}); ok {
		result := make([]string, 0, len(commands))
		for _, cmd := range commands {
			if cmdMap, ok := cmd.(map[string]interface{}); ok {
				if name, ok := cmdMap["name"].(string); ok {
					result = append(result, name)
				}
			}
		}
		return result
	}
	return nil
}

// hasCommand checks if a list of commands contains a specific command.
func hasCommand(commands []string, name string) bool {
	for _, cmd := range commands {
		if cmd == name {
			return true
		}
	}
	return false
}

func exampleDefault(ctx context.Context) error {
	fmt.Println("=== Default Behavior Example ===")
	fmt.Println("Setting sources: None (default)")
	fmt.Println("Expected: No custom slash commands will be available")

	options := &types.ClaudeAgentOptions{}

	c := claude.NewClientWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	if err := c.Query(ctx, "What is 2 + 2?"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Check the server info for available commands
	serverInfo := c.GetServerInfo()
	commands := extractSlashCommands(serverInfo)

	fmt.Printf("Available slash commands: %v\n", commands)
	if hasCommand(commands, "commit") {
		fmt.Println("X /commit is available (unexpected)")
	} else {
		fmt.Println("✓ /commit is NOT available (expected - no settings loaded)")
	}

	// Drain remaining messages
	for msg := range msgChan {
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	fmt.Println()
	return nil
}

func exampleUserOnly(ctx context.Context) error {
	fmt.Println("=== User Settings Only Example ===")
	fmt.Println("Setting sources: ['user']")
	fmt.Println("Expected: Project slash commands (like /commit) will NOT be available")

	options := &types.ClaudeAgentOptions{
		SettingSources: []types.SettingSource{types.SettingSourceUser},
	}

	c := claude.NewClientWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	if err := c.Query(ctx, "What is 2 + 2?"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Check the server info for available commands
	serverInfo := c.GetServerInfo()
	commands := extractSlashCommands(serverInfo)

	fmt.Printf("Available slash commands: %v\n", commands)
	if hasCommand(commands, "commit") {
		fmt.Println("X /commit is available (unexpected)")
	} else {
		fmt.Println("✓ /commit is NOT available (expected)")
	}

	// Drain remaining messages
	for msg := range msgChan {
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	fmt.Println()
	return nil
}

func exampleProjectAndUser(ctx context.Context) error {
	fmt.Println("=== Project + User Settings Example ===")
	fmt.Println("Setting sources: ['user', 'project']")
	fmt.Println("Expected: Project slash commands (like /commit) WILL be available")

	options := &types.ClaudeAgentOptions{
		SettingSources: []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject},
	}

	c := claude.NewClientWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	if err := c.Query(ctx, "What is 2 + 2?"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Check the server info for available commands
	serverInfo := c.GetServerInfo()
	commands := extractSlashCommands(serverInfo)

	fmt.Printf("Available slash commands: %v\n", commands)
	if hasCommand(commands, "commit") {
		fmt.Println("✓ /commit is available (expected)")
	} else {
		fmt.Println("X /commit is NOT available (unexpected)")
	}

	// Drain remaining messages
	for msg := range msgChan {
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	fmt.Println()
	return nil
}

func printUsage() {
	fmt.Println("Usage: go run main.go <example_name>")
	fmt.Println("\nAvailable examples:")
	fmt.Println("  all              - Run all examples")
	fmt.Println("  default          - Default behavior (no settings)")
	fmt.Println("  user_only        - Load only user settings")
	fmt.Println("  project_and_user - Load both project and user settings")
}

func main() {
	examples := map[string]func(context.Context) error{
		"default":          exampleDefault,
		"user_only":        exampleUserOnly,
		"project_and_user": exampleProjectAndUser,
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	exampleName := os.Args[1]

	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	fmt.Println("Starting Claude SDK Setting Sources Examples...")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	if exampleName == "all" {
		for name, fn := range examples {
			if err := fn(ctx); err != nil {
				fmt.Printf("Error in %s: %v\n", name, err)
			}
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println()
		}
	} else if fn, ok := examples[exampleName]; ok {
		if err := fn(ctx); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Error: Unknown example '%s'\n\n", exampleName)
		printUsage()
		os.Exit(1)
	}
}
