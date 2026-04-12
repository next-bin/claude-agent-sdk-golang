// Example quick_start demonstrates basic usage of the Claude Agent SDK for Go.
//
// This example shows:
// 1. Basic one-shot query
// 2. Query with custom options
// 3. Query with tools
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/examples/internal"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func main() {
	// Create a context that cancels on Ctrl+C
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	// Run examples based on command line argument
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "basic":
			basicExample(ctx)
		case "options":
			withOptionsExample(ctx)
		case "tools":
			withToolsExample(ctx)
		case "all":
			basicExample(ctx)
			withOptionsExample(ctx)
			withToolsExample(ctx)
		default:
			fmt.Println("Usage: go run main.go [basic|options|tools|all]")
			fmt.Println("\nExamples:")
			fmt.Println("  basic   - Simple question")
			fmt.Println("  options - Custom options (system prompt, max turns)")
			fmt.Println("  tools   - Using tools (Read, Write)")
			fmt.Println("  all     - Run all examples")
		}
		return
	}

	// Default: run all examples
	basicExample(ctx)
	withOptionsExample(ctx)
	withToolsExample(ctx)
}

// basicExample demonstrates a simple one-shot query.
func basicExample(ctx context.Context) {
	fmt.Println("=== Basic Example ===")

	msgChan, err := claude.Query(ctx, "What is 2 + 2?", nil)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		}
	}
	fmt.Println()
}

// withOptionsExample demonstrates query with custom options.
func withOptionsExample(ctx context.Context) {
	fmt.Println("=== With Options Example ===")

	options := &types.ClaudeAgentOptions{
		SystemPrompt: "You are a helpful assistant that explains things simply.",
		MaxTurns:     types.Int(1),
	}

	msgChan, err := claude.Query(ctx, "Explain what Go is in one sentence.", options)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		}
	}
	fmt.Println()
}

// withToolsExample demonstrates using tools with the query.
func withToolsExample(ctx context.Context) {
	fmt.Println("=== With Tools Example ===")

	options := &types.ClaudeAgentOptions{
		AllowedTools: []string{"Read", "Write"},
		SystemPrompt: "You are a helpful file assistant.",
	}

	msgChan, err := claude.Query(ctx, "Create a file called hello.txt with 'Hello, World!' in it", options)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		case *types.ResultMessage:
			if m.TotalCostUSD != nil {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}
	fmt.Println()
}
