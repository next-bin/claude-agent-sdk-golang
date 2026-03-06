// Example streaming_interactive demonstrates interactive streaming patterns with the Claude Agent SDK for Go.
//
// This example is designed for interactive use and quick experimentation,
// similar to Python's IPython-friendly snippets. Each function is self-contained
// and can be used as a reference for common patterns.
//
// The queries are intentionally simplistic. In reality, a query can be a more
// complex task that Claude SDK uses its agentic capabilities and tools
// (e.g., run bash commands, edit files, search the web, fetch web content) to accomplish.
//
// Usage:
//
//	go run main.go                # List available examples
//	go run main.go basic          # Run basic streaming example
//	go run main.go all            # Run all examples
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/unitsvc/claude-agent-sdk-golang/client"
	"github.com/unitsvc/claude-agent-sdk-golang/examples/internal"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
	fmt.Println("=== Claude Agent SDK Go - Interactive Streaming Examples ===")
	fmt.Println()

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	example := os.Args[1]
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	switch example {
	case "basic":
		basicStreaming(ctx)
	case "multi_turn":
		multiTurnConversation(ctx)
	case "persistent":
		persistentClient(ctx)
	case "interrupt":
		withInterrupt(ctx)
	case "timeout":
		withTimeout(ctx)
	case "interactive":
		interactiveREPL(ctx)
	case "all":
		runAll(ctx)
	default:
		fmt.Printf("Unknown example: %s\n\n", example)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: go run main.go <example>")
	fmt.Println("\nAvailable examples:")
	fmt.Println("  basic      - Basic streaming with client")
	fmt.Println("  multi_turn - Multi-turn conversation")
	fmt.Println("  persistent - Persistent client for multiple questions")
	fmt.Println("  interrupt  - With interrupt capability")
	fmt.Println("  timeout    - Error handling with timeout")
	fmt.Println("  interactive- Interactive REPL mode")
	fmt.Println("  all        - Run all examples")
}

// ============================================================================
// Basic Streaming
// ============================================================================

func basicStreaming(ctx context.Context) {
	fmt.Println("--- Basic Streaming ---")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("User: What is 2+2?")
	msgChan, err := c.Query(ctx, "What is 2+2?")
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
		return
	}

	for msg := range msgChan {
		if am, ok := msg.(*types.AssistantMessage); ok {
			for _, block := range am.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		}
	}
}

// ============================================================================
// Multi-Turn Conversation
// ============================================================================

func multiTurnConversation(ctx context.Context) {
	fmt.Println("--- Multi-Turn Conversation ---")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	sendAndReceive := func(prompt string) {
		fmt.Printf("User: %s\n", prompt)
		msgChan, err := c.Query(ctx, prompt)
		if err != nil {
			fmt.Printf("Query failed: %v\n", err)
			return
		}

		for msg := range msgChan {
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						fmt.Printf("Claude: %s\n", tb.Text)
					}
				}
			}
			if _, ok := msg.(*types.ResultMessage); ok {
				break
			}
		}
	}

	sendAndReceive("Tell me a short joke")
	fmt.Println("\n---")
	fmt.Println()
	sendAndReceive("Now tell me a fun fact")
}

// ============================================================================
// Persistent Client for Multiple Questions
// ============================================================================

func persistentClient(ctx context.Context) {
	fmt.Println("--- Persistent Client ---")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	getResponse := func() {
		msgChan := c.ReceiveMessages(ctx)
		for {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				if am, ok := msg.(*types.AssistantMessage); ok {
					for _, block := range am.Content {
						if tb, ok := block.(types.TextBlock); ok {
							fmt.Printf("Claude: %s\n", tb.Text)
						}
					}
				}
				if _, ok := msg.(*types.ResultMessage); ok {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}

	// First question
	fmt.Println("User: What's 2+2?")
	c.Query(ctx, "What's 2+2?")
	getResponse()

	fmt.Println()

	// Second question
	fmt.Println("User: What's 10*10?")
	c.Query(ctx, "What's 10*10?")
	getResponse()
}

// ============================================================================
// With Interrupt Capability
// ============================================================================

func withInterrupt(ctx context.Context) {
	fmt.Println("--- With Interrupt Capability ---")
	fmt.Println("IMPORTANT: Interrupts require active message consumption.")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("--- Sending initial message ---")
	fmt.Println()
	fmt.Println("User: Count from 1 to 100, with a brief pause between each number")

	msgChan, err := c.Query(ctx, "Count from 1 to 100, with a brief pause between each number. Do NOT use bash sleep, just count quickly.")
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
		return
	}

	// Consume messages with a timeout for interrupt
	messagesReceived := 0
	interruptTimeout := time.After(3 * time.Second)

consumeLoop:
	for {
		select {
		case <-interruptTimeout:
			fmt.Println()
			fmt.Println("--- Sending interrupt after 3 seconds ---")
			fmt.Println()
			c.Interrupt(ctx)
			break consumeLoop
		case msg, ok := <-msgChan:
			if !ok {
				break consumeLoop
			}
			messagesReceived++
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						// Truncate long responses
						text := tb.Text
						if len(text) > 100 {
							text = text[:100] + "..."
						}
						fmt.Printf("Claude: %s\n", text)
					}
				}
			}
			if _, ok := msg.(*types.ResultMessage); ok {
				break consumeLoop
			}
		}
	}

	// Send a new message after interrupt
	fmt.Println()
	fmt.Println("--- After interrupt, sending new message ---")
	fmt.Println()
	fmt.Println("User: Just say 'Hello! I was interrupted.'")

	msgChan, err = c.Query(ctx, "Just say 'Hello! I was interrupted.'")
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
		return
	}

	for msg := range msgChan {
		if am, ok := msg.(*types.AssistantMessage); ok {
			for _, block := range am.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		}
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}
}

// ============================================================================
// Error Handling with Timeout
// ============================================================================

func withTimeout(ctx context.Context) {
	fmt.Println("--- Error Handling with Timeout ---")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	fmt.Println("User: Run a bash sleep command for 60 seconds")

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msgChan, err := c.Query(ctx, "Run a bash sleep command for 60 seconds")
	if err != nil {
		fmt.Printf("Query failed: %v\n", err)
		return
	}

	messages := []types.Message{}

consumeWithTimeout:
	for {
		select {
		case <-timeoutCtx.Done():
			fmt.Println("\nRequest timed out after 5 seconds")
			fmt.Printf("Received %d messages before timeout\n", len(messages))
			break consumeWithTimeout
		case msg, ok := <-msgChan:
			if !ok {
				break consumeWithTimeout
			}
			messages = append(messages, msg)
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						fmt.Printf("Claude: %s\n", tb.Text)
					}
				}
			}
			if _, ok := msg.(*types.ResultMessage); ok {
				break consumeWithTimeout
			}
		}
	}
}

// ============================================================================
// Interactive REPL Mode
// ============================================================================

func interactiveREPL(ctx context.Context) {
	fmt.Println("--- Interactive REPL Mode ---")
	fmt.Println("Type your messages and press Enter to send.")
	fmt.Println("Type 'quit' or 'exit' to stop.")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		msgChan, err := c.Query(ctx, input)
		if err != nil {
			fmt.Printf("Query failed: %v\n", err)
			continue
		}

		fmt.Print("Claude: ")
		for msg := range msgChan {
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						fmt.Print(tb.Text)
					}
				}
			}
			if _, ok := msg.(*types.ResultMessage); ok {
				fmt.Println()
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

// ============================================================================
// Run All Examples
// ============================================================================

func runAll(ctx context.Context) {
	examples := []struct {
		name string
		fn   func(context.Context)
	}{
		{"basic", basicStreaming},
		{"multi_turn", multiTurnConversation},
		{"persistent", persistentClient},
		{"timeout", withTimeout},
	}

	for _, ex := range examples {
		fmt.Printf("Running: %s\n", ex.name)
		ex.fn(ctx)
		fmt.Println("\n" + strings.Repeat("-", 50) + "\n")
	}
}
