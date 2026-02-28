// Example setting_sources demonstrates configuration source management in the Claude Agent SDK for Go.
//
// Settings can be loaded from different sources: user settings, project settings,
// or local settings. This example shows how to control which settings sources are used.
//
// This example shows:
// 1. Using default settings sources
// 2. Specifying which settings sources to load
// 3. Understanding settings priority
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

	fmt.Println("=== Claude Agent SDK Go - Setting Sources Example ===")
	fmt.Println()

	// Example 1: Default behavior (all sources)
	defaultSourcesExample(ctx)

	// Example 2: User settings only
	userSettingsOnlyExample(ctx)

	// Example 3: Project settings only
	projectSettingsOnlyExample(ctx)

	// Example 4: Multiple specific sources
	multipleSourcesExample(ctx)
}

// defaultSourcesExample shows the default behavior where all settings are loaded.
func defaultSourcesExample(ctx context.Context) {
	fmt.Println("--- Example 1: Default Settings Sources ---")
	fmt.Println("By default, settings are loaded from all sources.")
	fmt.Println()

	// Default behavior - all settings sources are used
	// Priority order (later overrides earlier):
	// 1. User settings (~/.claude/settings.json)
	// 2. Project settings (.claude/settings.json in project root)
	// 3. Local settings (.claude/settings.local.json)
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Default settings sources loaded (in priority order):")
	fmt.Println("  1. User settings: ~/.claude/settings.json")
	fmt.Println("  2. Project settings: .claude/settings.json")
	fmt.Println("  3. Local settings: .claude/settings.local.json")
	fmt.Println()
	fmt.Println("Settings priority: Local > Project > User")
	fmt.Println("  - Local settings override project settings")
	fmt.Println("  - Project settings override user settings")
	fmt.Println()
}

// userSettingsOnlyExample demonstrates using only user-level settings.
func userSettingsOnlyExample(ctx context.Context) {
	fmt.Println("--- Example 2: User Settings Only ---")
	fmt.Println("Load settings only from user-level configuration.")
	fmt.Println()

	// Only use user settings - ignore project and local settings
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(types.ModelSonnet),
		SettingSources: []types.SettingSource{types.SettingSourceUser},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Settings sources loaded:")
	fmt.Println("  ✓ User settings: ~/.claude/settings.json")
	fmt.Println("  ✗ Project settings: (ignored)")
	fmt.Println("  ✗ Local settings: (ignored)")
	fmt.Println()
	fmt.Println("Use case: Consistent behavior across all projects")
	fmt.Println("  - Same keybindings, aliases, and preferences everywhere")
	fmt.Println("  - Ignores project-specific configurations")
	fmt.Println()
}

// projectSettingsOnlyExample demonstrates using only project-level settings.
func projectSettingsOnlyExample(ctx context.Context) {
	fmt.Println("--- Example 3: Project Settings Only ---")
	fmt.Println("Load settings only from project-level configuration.")
	fmt.Println()

	// Only use project settings - ignore user and local settings
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(types.ModelSonnet),
		SettingSources: []types.SettingSource{types.SettingSourceProject},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Settings sources loaded:")
	fmt.Println("  ✗ User settings: (ignored)")
	fmt.Println("  ✓ Project settings: .claude/settings.json")
	fmt.Println("  ✗ Local settings: (ignored)")
	fmt.Println()
	fmt.Println("Use case: Team-shared configuration")
	fmt.Println("  - Shareable settings via version control")
	fmt.Println("  - Consistent behavior for all team members")
	fmt.Println("  - Project-specific tools, MCP servers, etc.")
	fmt.Println()
}

// multipleSourcesExample demonstrates selecting specific sources.
func multipleSourcesExample(ctx context.Context) {
	fmt.Println("--- Example 4: Multiple Specific Sources ---")
	fmt.Println("Combine user and project settings, but ignore local.")
	fmt.Println()

	// Use user and project settings, but ignore local settings
	// This is useful for CI/CD or team environments
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		SettingSources: []types.SettingSource{
			types.SettingSourceUser,
			types.SettingSourceProject,
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Settings sources loaded (in priority order):")
	fmt.Println("  1. User settings: ~/.claude/settings.json")
	fmt.Println("  2. Project settings: .claude/settings.json")
	fmt.Println("  ✗ Local settings: (ignored)")
	fmt.Println()
	fmt.Println("Use case: CI/CD or team environments")
	fmt.Println("  - Use shared project configuration")
	fmt.Println("  - Respect user preferences")
	fmt.Println("  - Ignore machine-specific local settings")
	fmt.Println()

	// Another example: local and project only
	fmt.Println("Alternative: Project + Local only")
	fmt.Println()

	client2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		SettingSources: []types.SettingSource{
			types.SettingSourceProject,
			types.SettingSourceLocal,
		},
	})
	defer client2.Close()

	if err := client2.Connect(ctx); err != nil {
		log.Printf("Failed to connect client2: %v", err)
		return
	}

	fmt.Println("Settings sources loaded (in priority order):")
	fmt.Println("  ✗ User settings: (ignored)")
	fmt.Println("  1. Project settings: .claude/settings.json")
	fmt.Println("  2. Local settings: .claude/settings.local.json")
	fmt.Println()

	// Summary table
	fmt.Println("=== Setting Sources Summary ===")
	fmt.Println()
	fmt.Println("Source         | Location                        | Use Case")
	fmt.Println("---------------|--------------------------------|------------------------------------")
	fmt.Println("User           | ~/.claude/settings.json         | Personal preferences")
	fmt.Println("Project       | .claude/settings.json           | Team-shared configuration")
	fmt.Println("Local          | .claude/settings.local.json    | Machine-specific overrides")
	fmt.Println()
	fmt.Println("Priority Order (lowest to highest):")
	fmt.Println("  User → Project → Local")
	fmt.Println()
	fmt.Println("Available constants in Go:")
	fmt.Println("  types.SettingSourceUser    = \"user\"")
	fmt.Println("  types.SettingSourceProject = \"project\"")
	fmt.Println("  types.SettingSourceLocal   = \"local\"")
	fmt.Println()
}
