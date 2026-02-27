// Example quick_start demonstrates basic usage of the Claude Agent SDK for Go.
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

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	// Simple one-shot query
	msgChan, err := claude.Query(ctx, "What is 2+2? Please provide just the number.", nil)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	for msg := range msgChan {
		fmt.Printf("Message: %v\n", msg)
	}

	// Using a client with custom options
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String("sonnet"),
	})
	defer client.Close()

	// Connect the client before using it
	if err := client.Connect(ctx, "Tell me a short joke."); err != nil {
		log.Fatalf("Connect failed: %v", err)
	}

	msgChan2 := client.ReceiveMessages(ctx)

	for msg := range msgChan2 {
		fmt.Printf("Joke: %v\n", msg)
	}
}
