// Example include_partial_messages demonstrates handling partial/streaming messages in the Claude Agent SDK for Go.
//
// This example shows:
// 1. Using IncludePartialMessages option for real-time streaming
// 2. Processing StreamEvent messages
// 3. Handling partial assistant content
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/examples/internal"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	fmt.Println("=== Claude Agent SDK Go - Include Partial Messages Example ===")
	fmt.Println()

	// Example 1: Enable partial messages for streaming
	streamingExample(ctx)

	// Example 2: Process stream events
	streamEventExample(ctx)

	// Example 3: Real-time text accumulation
	textAccumulationExample(ctx)
}

// streamingExample demonstrates enabling partial messages for real-time streaming.
func streamingExample(ctx context.Context) {
	fmt.Println("--- Example 1: Enable Partial Messages ---")
	fmt.Println("With IncludePartialMessages enabled, you receive real-time content chunks.")
	fmt.Println()

	// Enable partial messages for streaming
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		IncludePartialMessages: true,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Client configured with IncludePartialMessages: true")
	fmt.Println("This enables real-time content streaming via StreamEvent messages.")
	fmt.Println()
}

// streamEventExample demonstrates processing StreamEvent messages.
func streamEventExample(ctx context.Context) {
	fmt.Println("--- Example 2: Process Stream Events ---")
	fmt.Println("StreamEvent messages contain partial content during generation.")
	fmt.Println()

	// Example of processing stream events
	fmt.Println("Stream events contain:")
	fmt.Println("  - UUID: unique identifier for the stream")
	fmt.Println("  - SessionID: session identifier")
	fmt.Println("  - Event: raw Anthropic API stream event data")
	fmt.Println("  - ParentToolUseID: parent tool use ID for subagent streams")
	fmt.Println()

	// Code pattern for processing stream events:
	// See the processing pattern below (in comments to avoid go vet warnings):
	fmt.Println("// Processing pattern:")
	fmt.Println("// for msg := range msgChan {")
	fmt.Println("//     switch m := msg.(type) {")
	fmt.Println("//     case *types.StreamEvent:")
	fmt.Println("//         // Process stream event using m.UUID, m.SessionID, m.Event")
	fmt.Println("//     case *types.AssistantMessage:")
	fmt.Println("//         // Process complete assistant message")
	fmt.Println("//         for _, block := range m.Content {")
	fmt.Println("//             if text, ok := block.(types.TextBlock); ok {")
	fmt.Println("//                 // Handle text block")
	fmt.Println("//             }")
	fmt.Println("//         }")
	fmt.Println("//     case *types.ResultMessage:")
	fmt.Println("//         // Query complete - check m.TotalCostUSD")
	fmt.Println("//     }")
	fmt.Println("// }")
	fmt.Println()
}

// textAccumulationExample demonstrates real-time text accumulation.
func textAccumulationExample(ctx context.Context) {
	fmt.Println("--- Example 3: Real-Time Text Accumulation ---")
	fmt.Println("Accumulate streaming text for real-time display.")
	fmt.Println()

	// Pattern for accumulating streaming text
	fmt.Println("// Text accumulation pattern:")
	fmt.Println("// streams := make(map[string]string)")
	fmt.Println("//")
	fmt.Println("// for msg := range msgChan {")
	fmt.Println("//     switch m := msg.(type) {")
	fmt.Println("//     case *types.StreamEvent:")
	fmt.Println("//         if eventMap, ok := m.Event.(map[string]interface{}); ok {")
	fmt.Println("//             if delta, ok := eventMap[\"delta\"].(map[string]interface{}); ok {")
	fmt.Println("//                 if text, ok := delta[\"text\"].(string); ok {")
	fmt.Println("//                     streams[m.UUID] += text")
	fmt.Println("//                     // Real-time display")
	fmt.Println("//                 }")
	fmt.Println("//             }")
	fmt.Println("//         }")
	fmt.Println("//     case *types.ResultMessage:")
	fmt.Println("//         if m.TotalCostUSD != nil {")
	fmt.Println("//             // Display final cost")
	fmt.Println("//         }")
	fmt.Println("//         delete(streams, m.UUID)")
	fmt.Println("//     }")
	fmt.Println("// }")
	fmt.Println()

	// Show the benefit of partial messages
	fmt.Println("Benefits of IncludePartialMessages:")
	fmt.Println("  1. Real-time user feedback during long responses")
	fmt.Println("  2. Ability to show typing indicators")
	fmt.Println("  3. Progressive rendering of content")
	fmt.Println("  4. Better UX for interactive applications")
	fmt.Println()

	// Example with full configuration
	fmt.Println("Full configuration example:")
	fmt.Println("// client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{")
	fmt.Println("//     Model:                   types.String(\"claude-sonnet-4-20250514\"),")
	fmt.Println("//     IncludePartialMessages: true,  // Enable streaming")
	fmt.Println("//     MaxTurns:               types.Int(10),")
	fmt.Println("// })")
	fmt.Println("//")
	fmt.Println("// if err := client.Connect(ctx); err != nil {")
	fmt.Println("//     log.Fatal(err)")
	fmt.Println("// }")
	fmt.Println("// defer client.Close()")
	fmt.Println()
}
