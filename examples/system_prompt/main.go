// Example system_prompt demonstrates system prompt configuration in the Claude Agent SDK for Go.
//
// System prompts allow you to customize Claude's behavior by providing context,
// instructions, or constraints. The SDK supports both simple string prompts
// and preset configurations.
//
// This example shows:
// 1. Using SystemPrompt option as a string for custom instructions
// 2. Using SystemPromptPreset configuration for preset behavior
// 3. Custom system prompts with append for extending the default preset
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

	fmt.Println("=== Claude Agent SDK Go - System Prompt Example ===")
	fmt.Println()

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	// Example 1: Using SystemPrompt as a simple string
	fmt.Println("--- Example 1: SystemPrompt as String ---")
	stringPromptExample(ctx)

	// Example 2: Using SystemPromptPreset configuration
	fmt.Println("\n--- Example 2: SystemPromptPreset Configuration ---")
	presetExample(ctx)

	// Example 3: Custom system prompt with append
	fmt.Println("\n--- Example 3: Custom System Prompt with Append ---")
	presetWithAppendExample(ctx)

	// Example 4: Practical use case - Domain-specific system prompt
	fmt.Println("\n--- Example 4: Domain-Specific System Prompt ---")
	domainSpecificExample(ctx)
}

// stringPromptExample demonstrates using SystemPrompt as a simple string.
// This is the simplest way to provide custom instructions to Claude.
func stringPromptExample(ctx context.Context) {
	// Define a custom system prompt as a string
	customPrompt := `You are a helpful coding assistant specialized in Go programming.
When providing code examples:
- Use idiomatic Go patterns
- Include proper error handling
- Add comments for complex logic
- Follow Go naming conventions`

	// Create client with string system prompt
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String("claude-sonnet-4-20250514"),
		SystemPrompt: customPrompt, // SystemPrompt accepts a string directly
	})
	defer client.Close()

	fmt.Println("Client configured with custom string system prompt")
	fmt.Println("Prompt preview: \"You are a helpful coding assistant specialized in Go...\"")
	_ = ctx // ctx would be used in actual query

	// To actually run a query, uncomment:
	// runQuery(ctx, client, "How do I read a file in Go?")
}

// presetExample demonstrates using SystemPromptPreset for preset behavior.
// The preset uses Claude Code's default system prompt configuration.
func presetExample(ctx context.Context) {
	// Create a SystemPromptPreset configuration
	// The "claude_code" preset uses the standard Claude Code system prompt
	preset := types.SystemPromptPreset{
		Type:   "preset",     // Always "preset" for preset configuration
		Preset: "claude_code", // Use the Claude Code default system prompt
	}

	// Create client with preset system prompt
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String("claude-sonnet-4-20250514"),
		SystemPrompt: preset, // SystemPrompt accepts SystemPromptPreset struct
	})
	defer client.Close()

	fmt.Println("Client configured with SystemPromptPreset")
	fmt.Printf("Preset type: %s, preset: %s\n", preset.Type, preset.Preset)
	_ = ctx

	// To actually run a query, uncomment:
	// runQuery(ctx, client, "What tools do you have available?")
}

// presetWithAppendExample shows how to extend the default preset with custom instructions.
// This allows you to keep the base Claude Code behavior while adding domain-specific guidance.
func presetWithAppendExample(ctx context.Context) {
	// Create a preset with appended custom instructions
	// This keeps the default Claude Code prompt and adds your customizations
	appendText := `

ADDITIONAL INSTRUCTIONS:
- Always explain your reasoning before providing code
- When suggesting file operations, show the expected outcome first
- Prefer standard library packages over third-party dependencies
- Provide alternative solutions when appropriate`

	presetWithAppend := types.SystemPromptPreset{
		Type:   "preset",     // Always "preset" for preset configuration
		Preset: "claude_code", // Use the Claude Code default system prompt
		Append: &appendText,   // Append custom instructions to the preset
	}

	// Create client with preset + append
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String("claude-sonnet-4-20250514"),
		SystemPrompt: presetWithAppend,
	})
	defer client.Close()

	fmt.Println("Client configured with SystemPromptPreset + custom append")
	fmt.Println("Base preset: claude_code")
	fmt.Printf("Append text: %d characters of custom instructions\n", len(appendText))
	_ = ctx

	// To actually run a query, uncomment:
	// runQuery(ctx, client, "Help me implement a REST API endpoint")
}

// domainSpecificExample demonstrates a practical use case for domain-specific prompts.
// This creates a specialized assistant for a specific domain.
func domainSpecificExample(ctx context.Context) {
	// Example 4a: Security-focused assistant with appended instructions
	securityAppend := `

SECURITY GUIDELINES:
- Never suggest code with known vulnerabilities
- Always validate user input
- Use parameterized queries for database operations
- Recommend security best practices proactively
- Flag potential security issues in existing code`

	securityPreset := types.SystemPromptPreset{
		Type:   "preset",
		Preset: "claude_code",
		Append: &securityAppend,
	}

	securityClient := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String("claude-sonnet-4-20250514"),
		SystemPrompt: securityPreset,
	})
	defer securityClient.Close()

	fmt.Println("Security-focused client configured with preset + security guidelines")

	// Example 4b: Documentation-focused assistant using string prompt
	docPrompt := `You are a technical documentation specialist.
Your responsibilities:
- Write clear, concise documentation
- Include code examples with explanations
- Use proper markdown formatting
- Create API documentation with parameter descriptions
- Write user-friendly tutorials and guides

Always structure documentation with:
1. Overview/Introduction
2. Prerequisites
3. Step-by-step instructions
4. Code examples
5. Troubleshooting section`

	docClient := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String("claude-sonnet-4-20250514"),
		SystemPrompt: docPrompt,
	})
	defer docClient.Close()

	fmt.Println("Documentation-focused client configured with custom string prompt")
	_ = ctx

	// Example 4c: Testing-focused assistant
	testingAppend := `

TESTING FOCUS:
- Prioritize test-driven development
- Write comprehensive unit tests
- Include edge case coverage
- Suggest integration tests where appropriate
- Aim for high code coverage
- Use table-driven tests in Go`

	testingPreset := types.SystemPromptPreset{
		Type:   "preset",
		Preset: "claude_code",
		Append: &testingAppend,
	}

	testingClient := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String("claude-sonnet-4-20250514"),
		SystemPrompt: testingPreset,
	})
	defer testingClient.Close()

	fmt.Println("Testing-focused client configured with preset + testing guidelines")
}

// runQuery demonstrates how to execute a query with the configured client.
// This helper function shows the pattern for running queries.
func runQuery(ctx context.Context, client interface {
	Connect(context.Context) error
	Query(context.Context, interface{}, ...string) (<-chan types.Message, error)
	Close() error
}, prompt string) {
	// Connect to Claude
	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	msgChan, err := client.Query(ctx, prompt)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	// Process messages from the channel
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(types.TextBlock); ok {
					fmt.Printf("Assistant: %s\n", textBlock.Text)
				}
			}
		case *types.ResultMessage:
			if m.Result != nil {
				fmt.Printf("Result: %s\n", *m.Result)
			}
			if m.TotalCostUSD != nil {
				fmt.Printf("Session: %s, Cost: $%.4f\n", m.SessionID, *m.TotalCostUSD)
			} else {
				fmt.Printf("Session: %s\n", m.SessionID)
			}
		}
	}
}

// combinedOptionsExample demonstrates combining system prompt with other options.
// This shows how to use system prompts alongside other configuration options.
func combinedOptionsExample(ctx context.Context) {
	appendText := `
Always provide:
1. A brief explanation of your approach
2. The solution/implementation
3. Testing suggestions
4. Potential improvements`

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String("claude-sonnet-4-20250514"),
		SystemPrompt: types.SystemPromptPreset{
			Type:   "preset",
			Preset: "claude_code",
			Append: &appendText,
		},
		// Combine with other options
		PermissionMode: (*types.PermissionMode)(types.String("default")),
		MaxTurns:       types.Int(5),
	})
	defer client.Close()

	// The client now has:
	// - Custom system prompt extending the Claude Code preset
	// - Default permission mode
	// - Maximum 5 conversation turns
	_ = ctx
	fmt.Println("Client configured with combined options")
}