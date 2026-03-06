// Example budget demonstrates using MaxBudgetUSD and MaxTurns options with cost tracking.
//
// This example shows how to:
// 1. Use MaxBudgetUSD option to limit the cost of a query
// 2. Use MaxTurns option to limit the number of agent turns
// 3. Track budget and cost from ResultMessage
//
// Usage:
//
//	go run main.go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/unitsvc/claude-agent-sdk-golang/client"
	"github.com/unitsvc/claude-agent-sdk-golang/examples/internal"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	fmt.Println("=== Budget and Turn Limiting Example ===")
	fmt.Println()

	// Example 1: Using MaxBudgetUSD to limit cost
	fmt.Println("--- Example 1: MaxBudgetUSD ---")
	runWithBudgetLimit(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 2: Using MaxTurns to limit agent turns
	fmt.Println("--- Example 2: MaxTurns ---")
	runWithTurnLimit(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 3: Using both MaxBudgetUSD and MaxTurns together
	fmt.Println("--- Example 3: Combined Budget and Turn Limits ---")
	runWithCombinedLimits(ctx)

	fmt.Println()
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println()

	// Example 4: Tracking cost over multiple queries
	fmt.Println("--- Example 4: Cost Tracking Over Session ---")
	runSessionWithCostTracking(ctx)
}

// runWithBudgetLimit demonstrates using MaxBudgetUSD option.
func runWithBudgetLimit(ctx context.Context) {
	// Set a maximum budget of $0.05 for this query
	maxBudget := 0.05

	options := &types.ClaudeAgentOptions{
		MaxBudgetUSD: &maxBudget,
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Printf("Max Budget set to: $%.2f\n", maxBudget)
	fmt.Println("Query: What is the capital of France?")

	if err := c.Query(ctx, "What is the capital of France?"); err != nil {
		handleError(err)
		return
	}

	processMessages(msgChan)
}

// runWithTurnLimit demonstrates using MaxTurns option.
func runWithTurnLimit(ctx context.Context) {
	// Limit the agent to a maximum of 2 turns
	maxTurns := 2

	options := &types.ClaudeAgentOptions{
		MaxTurns: &maxTurns,
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Printf("Max Turns set to: %d\n", maxTurns)
	fmt.Println("Query: What is 2+2?")

	if err := c.Query(ctx, "What is 2+2?"); err != nil {
		handleError(err)
		return
	}

	processMessages(msgChan)
}

// runWithCombinedLimits demonstrates using both MaxBudgetUSD and MaxTurns together.
func runWithCombinedLimits(ctx context.Context) {
	// Set both budget and turn limits
	maxBudget := 0.10
	maxTurns := 3

	options := &types.ClaudeAgentOptions{
		MaxBudgetUSD: &maxBudget,
		MaxTurns:     &maxTurns,
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Printf("Max Budget: $%.2f, Max Turns: %d\n", maxBudget, maxTurns)
	fmt.Println("Query: Tell me a short joke about programming.")

	if err := c.Query(ctx, "Tell me a short joke about programming."); err != nil {
		handleError(err)
		return
	}

	processMessages(msgChan)
}

// runSessionWithCostTracking demonstrates tracking costs over multiple queries.
func runSessionWithCostTracking(ctx context.Context) {
	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		handleError(err)
		return
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	queries := []string{
		"What is 5 + 7?",
		"What is the capital of Japan?",
		"Name three primary colors.",
	}

	totalCost := 0.0
	totalTurns := 0

	for i, query := range queries {
		fmt.Printf("\nQuery %d: %s\n", i+1, query)

		if err := c.Query(ctx, query); err != nil {
			handleError(err)
			continue
		}

		turnCost, turns := processMessagesWithCost(msgChan)
		totalCost += turnCost
		totalTurns += turns
	}

	fmt.Println()
	fmt.Println("=== Session Summary ===")
	fmt.Printf("Total Cost: $%.6f\n", totalCost)
	fmt.Printf("Total Turns: %d\n", totalTurns)
}

// processMessages handles incoming messages and prints the result.
func processMessages(msgChan <-chan types.Message) {
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		case *types.ResultMessage:
			printResultMessage(m)
			return
		}
	}
}

// processMessagesWithCost handles messages and returns cost information.
func processMessagesWithCost(msgChan <-chan types.Message) (cost float64, turns int) {
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		case *types.ResultMessage:
			printResultMessage(m)
			if m.TotalCostUSD != nil {
				cost = *m.TotalCostUSD
			}
			turns = m.NumTurns
			return
		}
	}
	return 0, 0
}

// printResultMessage displays the ResultMessage details including cost.
func printResultMessage(m *types.ResultMessage) {
	fmt.Println("\n--- Result ---")
	fmt.Printf("Duration: %d ms (API: %d ms)\n", m.DurationMs, m.DurationAPIMs)
	fmt.Printf("Turns: %d\n", m.NumTurns)
	fmt.Printf("Session ID: %s\n", m.SessionID)

	if m.TotalCostUSD != nil {
		fmt.Printf("Total Cost: $%.6f\n", *m.TotalCostUSD)
	} else {
		fmt.Println("Total Cost: N/A")
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
