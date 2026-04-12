// Example goroutines demonstrates concurrent patterns using goroutines and channels
// with the Claude Agent SDK for Go.
//
// This example shows Go-specific patterns for:
// 1. Multi-turn conversations with goroutines
// 2. Background message processing
// 3. Concurrent query handling
// 4. Context cancellation and timeouts
//
// Similar to Python's trio example but using Go's native concurrency primitives.
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/next-bin/claude-agent-sdk-golang/client"
	"github.com/next-bin/claude-agent-sdk-golang/examples/internal"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	fmt.Println("=== Claude Agent SDK Go - Goroutines & Channels Patterns ===")
	fmt.Println()

	// Example 1: Multi-turn conversation
	fmt.Println("--- Example 1: Multi-Turn Conversation ---")
	multiTurnConversation(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 2: Background message processing
	fmt.Println("--- Example 2: Background Message Processing ---")
	backgroundProcessing(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 3: Concurrent queries
	fmt.Println("--- Example 3: Concurrent Queries ---")
	concurrentQueries(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 4: Context cancellation
	fmt.Println("--- Example 4: Context Cancellation ---")
	contextCancellation(ctx)
}

// ============================================================================
// Multi-Turn Conversation
// ============================================================================

func multiTurnConversation(ctx context.Context) {
	// Create client
	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("=== Multi-turn Conversation ===")
	fmt.Println()

	// Helper to display messages
	displayMessage := func(msg types.Message) {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		case *types.ResultMessage:
			if m.TotalCostUSD != nil {
				fmt.Printf("Result: Turn cost $%.6f\n", *m.TotalCostUSD)
			}
		}
	}

	// First turn: Simple math question
	fmt.Println("User: What's 15 + 27?")
	if err := c.Query(ctx, "What's 15 + 27?"); err != nil {
		fmt.Printf("Query failed: %v\n", err)
		return
	}
	for msg := range msgChan {
		displayMessage(msg)
	}
	fmt.Println()

	// Second turn: Follow-up calculation
	fmt.Println("User: Now multiply that result by 2")
	if err := c.Query(ctx, "Now multiply that result by 2"); err != nil {
		fmt.Printf("Query failed: %v\n", err)
		return
	}
	for msg := range msgChan {
		displayMessage(msg)
	}
	fmt.Println()

	// Third turn: One more operation
	fmt.Println("User: Divide that by 7 and round to 2 decimal places")
	if err := c.Query(ctx, "Divide that by 7 and round to 2 decimal places"); err != nil {
		fmt.Printf("Query failed: %v\n", err)
		return
	}
	for msg := range msgChan {
		displayMessage(msg)
	}

	fmt.Println("\nConversation complete!")
}

// ============================================================================
// Background Message Processing
// ============================================================================

func backgroundProcessing(ctx context.Context) {
	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	// Channel for message results
	results := make(chan string)
	var wg sync.WaitGroup

	// Background goroutine to process messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range results {
			fmt.Printf("[Background] Processed: %s\n", result)
		}
	}()

	// Send multiple queries
	queries := []string{
		"What is 2 + 2?",
		"What is the capital of France?",
		"Name a primary color.",
	}

	for i, query := range queries {
		fmt.Printf("\nQuery %d: %s\n", i+1, query)

		if err := c.Query(ctx, query); err != nil {
			fmt.Printf("Query failed: %v\n", err)
			continue
		}

		for msg := range msgChan {
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						// Send result to background processor
						results <- fmt.Sprintf("Q%d: %s", i+1, truncate(tb.Text, 50))
						fmt.Printf("Claude: %s\n", truncate(tb.Text, 100))
					}
				}
			}
		}
	}

	close(results)
	wg.Wait()
	fmt.Println("\nBackground processing complete!")
}

// ============================================================================
// Concurrent Queries
// ============================================================================

func concurrentQueries(ctx context.Context) {
	// Create multiple clients for concurrent queries
	queries := []struct {
		id   int
		text string
	}{
		{1, "What is 1 + 1?"},
		{2, "What is 2 + 2?"},
		{3, "What is 3 + 3?"},
	}

	var wg sync.WaitGroup
	results := make(chan string, len(queries))

	// Launch concurrent queries
	for _, q := range queries {
		wg.Add(1)
		go func(id int, text string) {
			defer wg.Done()

			// Each goroutine gets its own client
			c := client.New()
			defer c.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := c.Connect(ctx); err != nil {
				results <- fmt.Sprintf("[Query %d] Failed: %v", id, err)
				return
			}

			msgChan := c.ReceiveMessages(ctx)
			if err := c.Query(ctx, text); err != nil {
				results <- fmt.Sprintf("[Query %d] Query failed: %v", id, err)
				return
			}

			for msg := range msgChan {
				if am, ok := msg.(*types.AssistantMessage); ok {
					for _, block := range am.Content {
						if tb, ok := block.(types.TextBlock); ok {
							results <- fmt.Sprintf("[Query %d] Result: %s", id, truncate(tb.Text, 50))
						}
					}
				}
			}
		}(q.id, q.text)
	}

	// Wait for all queries to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results as they come in
	fmt.Println("Concurrent queries running...")
	for result := range results {
		fmt.Println(result)
	}

	fmt.Println("\nAll concurrent queries complete!")
}

// ============================================================================
// Context Cancellation
// ============================================================================

func contextCancellation(ctx context.Context) {
	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Println("Starting a long-running query...")
	fmt.Println("Will cancel after 5 seconds.")
	fmt.Println()

	// Channel to signal when query is done
	done := make(chan struct{})

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	go func() {
		defer close(done)

		if err := c.Query(ctx, "Count from 1 to 100. Do not use bash sleep, just count quickly."); err != nil {
			fmt.Printf("Query failed: %v\n", err)
			return
		}

		messageCount := 0
		for msg := range msgChan {
			messageCount++
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						// Show truncated output
						fmt.Printf("[Msg %d] %s\n", messageCount, truncate(tb.Text, 60))
					}
				}
			}
		}
	}()

	// Wait 5 seconds then cancel
	time.Sleep(5 * time.Second)
	fmt.Println("\nCancelling context...")
	cancel()

	// Wait for query to finish
	<-done
	fmt.Println("Query cancelled via context!")

	// Show that client is still usable after cancellation
	fmt.Println("\nSending a new query after cancellation...")
	if err := c.Query(context.Background(), "Just say 'Hello after cancellation!'"); err != nil {
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
// Helper Functions
// ============================================================================

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
