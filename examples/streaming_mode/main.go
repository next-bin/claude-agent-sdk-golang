// Example streaming_mode demonstrates using the Claude SDK Client with Connect() for streaming conversations.
//
// This example shows various patterns for building applications with the Claude SDK Client
// streaming interface in Go.
//
// The queries are intentionally simplistic. In reality, a query can be a more complex task
// that Claude SDK uses its agentic capabilities and tools (e.g., run bash commands, edit files,
// search the web, fetch web content) to accomplish.
//
// Usage:
//
//	go run main.go                # List the examples
//	go run main.go all            # Run all examples
//	go run main.go basic_streaming # Run a specific example
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/unitsvc/claude-agent-sdk-golang/client"
	"github.com/unitsvc/claude-agent-sdk-golang/errors"
	"github.com/unitsvc/claude-agent-sdk-golang/examples/internal"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// displayMessage prints a standardized representation of messages.
// - UserMessage: "User: <content>"
// - AssistantMessage: "Claude: <content>"
// - SystemMessage: ignored
// - ResultMessage: "Result ended" + cost if available
func displayMessage(msg types.Message) {
	switch m := msg.(type) {
	case *types.UserMessage:
		switch content := m.Content.(type) {
		case string:
			fmt.Printf("User: %s\n", content)
		case []types.ContentBlock:
			for _, block := range content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("User: %s\n", tb.Text)
				}
			}
		}
	case *types.AssistantMessage:
		for _, block := range m.Content {
			if tb, ok := block.(types.TextBlock); ok {
				fmt.Printf("Claude: %s\n", tb.Text)
			}
		}
	case *types.SystemMessage:
		// Ignore system messages
	case *types.ResultMessage:
		fmt.Println("Result ended")
		if m.TotalCostUSD != nil {
			fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
		}
	}
}

// exampleBasicStreaming demonstrates basic streaming with the client.
func exampleBasicStreaming(ctx context.Context) error {
	fmt.Println("=== Basic Streaming Example ===")

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("User: What is 2+2?")
	if err := c.Query(ctx, "What is 2+2?"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Receive complete response
	for msg := range msgChan {
		displayMessage(msg)
		// Stop after receiving result message
		if _, isResult := msg.(*types.ResultMessage); isResult {
			break
		}
	}

	fmt.Println()
	return nil
}

// exampleMultiTurnConversation demonstrates multi-turn conversations.
func exampleMultiTurnConversation(ctx context.Context) error {
	fmt.Println("=== Multi-Turn Conversation Example ===")

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	// First turn
	fmt.Println("User: What's the capital of France?")
	if err := c.Query(ctx, "What's the capital of France?"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	for msg := range msgChan {
		displayMessage(msg)
		if _, isResult := msg.(*types.ResultMessage); isResult {
			break
		}
	}

	// Second turn - follow-up
	fmt.Println("\nUser: What's the population of that city?")
	if err := c.Query(ctx, "What's the population of that city?"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	for msg := range msgChan {
		displayMessage(msg)
		if _, isResult := msg.(*types.ResultMessage); isResult {
			break
		}
	}

	fmt.Println()
	return nil
}

// exampleConcurrentResponses demonstrates handling responses while sending new messages.
func exampleConcurrentResponses(ctx context.Context) error {
	fmt.Println("=== Concurrent Send/Receive Example ===")

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Channel to signal when to stop receiving
	stopChan := make(chan struct{})
	var receiveErr error

	// Background goroutine to continuously receive messages
	go func() {
		msgChan := c.ReceiveMessages(ctx)
		for {
			select {
			case <-stopChan:
				return
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				displayMessage(msg)
			}
		}
	}()

	// Send multiple messages with delays
	questions := []string{
		"What is 2 + 2?",
		"What is the square root of 144?",
		"What is 10% of 80?",
	}

	for i, question := range questions {
		fmt.Printf("\nUser: %s\n", question)
		if err := c.Query(ctx, question); err != nil {
			receiveErr = fmt.Errorf("failed to query: %w", err)
			break
		}

		// Wait between messages (except for the last one)
		if i < len(questions)-1 {
			time.Sleep(3 * time.Second)
		}
	}

	// Give time for final responses
	time.Sleep(2 * time.Second)
	close(stopChan)

	fmt.Println()
	return receiveErr
}

// exampleWithInterrupt demonstrates interrupt capability.
func exampleWithInterrupt(ctx context.Context) error {
	fmt.Println("=== Interrupt Example ===")
	fmt.Println("IMPORTANT: Interrupts require active message consumption.")

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	// Start a long-running task
	fmt.Println("\nUser: Count from 1 to 100 slowly")
	if err := c.Query(ctx, "Count from 1 to 100 slowly, with a brief pause between each number"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Create a context with timeout for the interrupt
	interruptCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Consume messages until timeout
	messageCount := 0
consumeLoop:
	for {
		select {
		case <-interruptCtx.Done():
			fmt.Println("\n[After 2 seconds, sending interrupt...]")
			break consumeLoop
		case msg, ok := <-msgChan:
			if !ok {
				break consumeLoop
			}
			messageCount++
			displayMessage(msg)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break consumeLoop
			}
		}
	}

	// Send interrupt
	if err := c.Interrupt(ctx); err != nil {
		fmt.Printf("Interrupt failed: %v\n", err)
	}

	// Send new instruction after interrupt
	fmt.Println("\nUser: Never mind, just tell me a quick joke")
	if err := c.Query(ctx, "Never mind, just tell me a quick joke"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Get the joke
	for msg := range msgChan {
		displayMessage(msg)
		if _, isResult := msg.(*types.ResultMessage); isResult {
			break
		}
	}

	fmt.Println()
	return nil
}

// exampleManualMessageHandling demonstrates manually handling message stream for custom logic.
func exampleManualMessageHandling(ctx context.Context) error {
	fmt.Println("=== Manual Message Handling Example ===")

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	if err := c.Query(ctx, "List 5 programming languages and their main use cases"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Manually process messages with custom logic
	languagesFound := []string{}
	targetLanguages := []string{"Python", "JavaScript", "Java", "C++", "Go", "Rust", "Ruby"}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
					// Custom logic: extract language names
					for _, lang := range targetLanguages {
						if strings.Contains(tb.Text, lang) && !contains(languagesFound, lang) {
							languagesFound = append(languagesFound, lang)
							fmt.Printf("Found language: %s\n", lang)
						}
					}
				}
			}
		case *types.ResultMessage:
			displayMessage(msg)
			fmt.Printf("Total languages mentioned: %d\n", len(languagesFound))
			break
		}
	}

	fmt.Println()
	return nil
}

// helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// exampleWithOptions demonstrates using ClaudeAgentOptions to configure the client.
func exampleWithOptions(ctx context.Context) error {
	fmt.Println("=== Custom Options Example ===")

	// Configure options
	options := &types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		AllowedTools: []string{"Read", "Write"},
		SystemPrompt: strPtr("You are a helpful coding assistant."),
	}

	c := client.NewWithOptions(options)
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("User: Create a simple hello.txt file with a greeting message")
	if err := c.Query(ctx, "Create a simple hello.txt file with a greeting message"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	toolUses := []string{}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			displayMessage(msg)
			for _, block := range m.Content {
				if tub, ok := block.(types.ToolUseBlock); ok {
					toolUses = append(toolUses, tub.Name)
				}
			}
		case *types.ResultMessage:
			displayMessage(msg)
			break
		}
	}

	if len(toolUses) > 0 {
		fmt.Printf("Tools used: %s\n", strings.Join(toolUses, ", "))
	}

	fmt.Println()
	return nil
}

// exampleBashCommand demonstrates tool use blocks when running bash commands.
func exampleBashCommand(ctx context.Context) error {
	fmt.Println("=== Bash Command Example ===")

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("User: Run a bash echo command")
	if err := c.Query(ctx, "Run a bash echo command that says 'Hello from bash!'"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Track all message types received
	messageTypes := []string{}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.UserMessage:
			messageTypes = append(messageTypes, "UserMessage")
			// User messages can contain tool results
			switch content := m.Content.(type) {
			case []types.ContentBlock:
				for _, block := range content {
					switch b := block.(type) {
					case types.TextBlock:
						fmt.Printf("User: %s\n", b.Text)
					case types.ToolResultBlock:
						contentStr := fmt.Sprintf("%v", b.Content)
						if len(contentStr) > 100 {
							contentStr = contentStr[:100] + "..."
						}
						fmt.Printf("Tool Result (id: %s): %s\n", b.ToolUseID, contentStr)
					}
				}
			}

		case *types.AssistantMessage:
			messageTypes = append(messageTypes, "AssistantMessage")
			// Assistant messages can contain tool use blocks
			for _, block := range m.Content {
				switch b := block.(type) {
				case types.TextBlock:
					fmt.Printf("Claude: %s\n", b.Text)
				case types.ToolUseBlock:
					messageTypes = append(messageTypes, "ToolUseBlock")
					fmt.Printf("Tool Use: %s (id: %s)\n", b.Name, b.ID)
					if b.Name == "Bash" {
						if cmd, ok := b.Input["command"].(string); ok {
							fmt.Printf("  Command: %s\n", cmd)
						}
					}
				}
			}

		case *types.ResultMessage:
			messageTypes = append(messageTypes, "ResultMessage")
			fmt.Println("Result ended")
			if m.TotalCostUSD != nil {
				fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
			}
			break
		}
	}

	fmt.Printf("\nMessage types received: %s\n", strings.Join(unique(messageTypes), ", "))
	fmt.Println()
	return nil
}

// helper to get unique strings
func unique(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

// exampleControlProtocol demonstrates server info and interrupt capabilities.
func exampleControlProtocol(ctx context.Context) error {
	fmt.Println("=== Control Protocol Example ===")
	fmt.Println("Shows server info retrieval and interrupt capability")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// 1. Get server initialization info
	fmt.Println("1. Getting server info...")
	serverInfo := c.GetServerInfo()

	if serverInfo != nil {
		fmt.Println("Server info retrieved successfully!")
		if commands, ok := serverInfo["commands"].([]interface{}); ok {
			fmt.Printf("  - Available commands: %d\n", len(commands))
		}
		if outputStyle, ok := serverInfo["output_style"].(string); ok {
			fmt.Printf("  - Output style: %s\n", outputStyle)
		}

		// Show available output styles if present
		if styles, ok := serverInfo["available_output_styles"].([]interface{}); ok && len(styles) > 0 {
			styleStrs := make([]string, len(styles))
			for i, s := range styles {
				styleStrs[i] = fmt.Sprintf("%v", s)
			}
			fmt.Printf("  - Available output styles: %s\n", strings.Join(styleStrs, ", "))
		}

		// Show a few example commands
		if commands, ok := serverInfo["commands"].([]interface{}); ok && len(commands) > 0 {
			fmt.Println("  - Example commands:")
			for i, cmd := range commands {
				if i >= 5 {
					break
				}
				if cmdMap, ok := cmd.(map[string]interface{}); ok {
					if name, ok := cmdMap["name"].(string); ok {
						fmt.Printf("    - %s\n", name)
					}
				}
			}
		}
	} else {
		fmt.Println("No server info available (may not be in streaming mode)")
	}

	fmt.Println("\n2. Testing interrupt capability...")

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	// Start a long-running task
	fmt.Println("User: Count from 1 to 20 slowly")
	if err := c.Query(ctx, "Count from 1 to 20 slowly, pausing between each number"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Create a context with timeout for the interrupt
	interruptCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Consume messages until timeout
consumeLoop2:
	for {
		select {
		case <-interruptCtx.Done():
			fmt.Println("\n[Sending interrupt after 2 seconds...]")
			break consumeLoop2
		case msg, ok := <-msgChan:
			if !ok {
				break consumeLoop2
			}
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						// Print first 50 chars to show progress
						text := tb.Text
						if len(text) > 50 {
							text = text[:50] + "..."
						}
						fmt.Printf("Claude: %s\n", text)
					}
				}
			}
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break consumeLoop2
			}
		}
	}

	// Send interrupt
	if err := c.Interrupt(ctx); err != nil {
		fmt.Printf("Interrupt failed: %v\n", err)
	} else {
		fmt.Println("Interrupt sent successfully")
	}

	// Send new query after interrupt
	fmt.Println("\nUser: Just say 'Hello!'")
	if err := c.Query(ctx, "Just say 'Hello!'"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	for msg := range msgChan {
		if am, ok := msg.(*types.AssistantMessage); ok {
			for _, block := range am.Content {
				if tb, ok := block.(types.TextBlock); ok {
					fmt.Printf("Claude: %s\n", tb.Text)
				}
			}
		}
		if _, isResult := msg.(*types.ResultMessage); isResult {
			break
		}
	}

	fmt.Println()
	return nil
}

// exampleErrorHandling demonstrates proper error handling.
func exampleErrorHandling(ctx context.Context) error {
	fmt.Println("=== Error Handling Example ===")

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	// Send a message that will take time to process
	fmt.Println("User: Run a bash sleep command for 60 seconds not in the background")
	if err := c.Query(ctx, "Run a bash sleep command for 60 seconds not in the background"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// Try to receive response with a short timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	messages := []types.Message{}
messageLoop:
	for {
		select {
		case <-timeoutCtx.Done():
			fmt.Println("\nResponse timeout after 10 seconds - demonstrating graceful handling")
			fmt.Printf("Received %d messages before timeout\n", len(messages))
			break messageLoop
		case msg, ok := <-msgChan:
			if !ok {
				break messageLoop
			}
			messages = append(messages, msg)
			if am, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range am.Content {
					if tb, ok := block.(types.TextBlock); ok {
						text := tb.Text
						if len(text) > 50 {
							text = text[:50] + "..."
						}
						fmt.Printf("Claude: %s\n", text)
					}
				}
			} else if _, isResult := msg.(*types.ResultMessage); isResult {
				displayMessage(msg)
				break messageLoop
			}
		}
	}

	fmt.Println()
	return nil
}

// exampleMessageTypes demonstrates processing different message types.
func exampleMessageTypes(ctx context.Context) error {
	fmt.Println("=== Message Types Example ===")
	fmt.Println("Demonstrates all message types: ResultMessage, AssistantMessage, UserMessage, etc.")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	fmt.Println("User: Tell me a short joke")
	if err := c.Query(ctx, "Tell me a short joke"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.UserMessage:
			fmt.Printf("[UserMessage] ")
			switch content := m.Content.(type) {
			case string:
				fmt.Printf("Content: %s\n", content)
			case []types.ContentBlock:
				fmt.Printf("Content blocks: %d\n", len(content))
				for i, block := range content {
					fmt.Printf("  Block %d: %T\n", i, block)
				}
			}
			if m.UUID != nil {
				fmt.Printf("  UUID: %s\n", *m.UUID)
			}

		case *types.AssistantMessage:
			fmt.Printf("[AssistantMessage] Model: %s\n", m.Model)
			for i, block := range m.Content {
				switch b := block.(type) {
				case types.TextBlock:
					fmt.Printf("  TextBlock %d: %s\n", i, b.Text)
				case types.ThinkingBlock:
					fmt.Printf("  ThinkingBlock %d: %s...\n", i, truncate(b.Thinking, 50))
				case types.ToolUseBlock:
					fmt.Printf("  ToolUseBlock %d: name=%s id=%s\n", i, b.Name, b.ID)
				case types.ToolResultBlock:
					fmt.Printf("  ToolResultBlock %d: tool_use_id=%s\n", i, b.ToolUseID)
				default:
					fmt.Printf("  Block %d: %T\n", i, block)
				}
			}
			if m.Error != nil {
				fmt.Printf("  Error: %s\n", *m.Error)
			}

		case *types.SystemMessage:
			fmt.Printf("[SystemMessage] Subtype: %s\n", m.Subtype)
			for k, v := range m.Data {
				fmt.Printf("  %s: %v\n", k, v)
			}

		case *types.ResultMessage:
			fmt.Printf("[ResultMessage]\n")
			fmt.Printf("  Subtype: %s\n", m.Subtype)
			fmt.Printf("  Duration: %d ms (API: %d ms)\n", m.DurationMs, m.DurationAPIMs)
			fmt.Printf("  Turns: %d\n", m.NumTurns)
			fmt.Printf("  Session ID: %s\n", m.SessionID)
			fmt.Printf("  Is Error: %v\n", m.IsError)
			if m.TotalCostUSD != nil {
				fmt.Printf("  Total Cost: $%.6f\n", *m.TotalCostUSD)
			}
			if m.Result != nil {
				fmt.Printf("  Result: %s\n", *m.Result)
			}
			// Break after result message
			break

		case *types.StreamEvent:
			fmt.Printf("[StreamEvent] Session: %s, UUID: %s\n", m.SessionID, m.UUID)

		default:
			fmt.Printf("[Unknown] Type: %T\n", msg)
		}
		fmt.Println()
	}

	fmt.Println()
	return nil
}

// helper to truncate strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// helper to get string pointer
func strPtr(s string) *string {
	return &s
}

// exampleWithReceiveResponse demonstrates using the ReceiveResponse helper method.
func exampleWithReceiveResponse(ctx context.Context) error {
	fmt.Println("=== ReceiveResponse Helper Example ===")
	fmt.Println("Using ReceiveResponse() for simpler single-response workflows")
	fmt.Println()

	c := client.New()
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := c.ReceiveMessages(ctx)

	// Connect with an initial prompt
	fmt.Println("User: What is the capital of Italy?")
	if err := c.Query(ctx, "What is the capital of Italy?"); err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// ReceiveResponse receives messages until and including a ResultMessage
	responseChan := c.ReceiveResponse(ctx)
	go func() {
		// Forward messages from Query to be processed
		for msg := range msgChan {
			// Process messages in the main goroutine
			_ = msg
		}
	}()

	for msg := range responseChan {
		displayMessage(msg)
	}

	fmt.Println()
	return nil
}

func printUsage() {
	fmt.Println("Usage: go run main.go <example_name>")
	fmt.Println("\nAvailable examples:")
	fmt.Println("  all                    - Run all examples")
	fmt.Println("  basic_streaming        - Basic streaming with client")
	fmt.Println("  multi_turn             - Multi-turn conversation")
	fmt.Println("  concurrent             - Concurrent send/receive")
	fmt.Println("  interrupt              - Interrupt capability demo")
	fmt.Println("  manual_handling        - Manual message handling")
	fmt.Println("  with_options           - Custom options configuration")
	fmt.Println("  bash_command           - Tool use blocks demo")
	fmt.Println("  control_protocol       - Server info and interrupt")
	fmt.Println("  error_handling         - Error handling demo")
	fmt.Println("  message_types          - All message types demo")
	fmt.Println("  receive_response       - ReceiveResponse helper demo")
}

func main() {
	examples := map[string]func(context.Context) error{
		"basic_streaming":  exampleBasicStreaming,
		"multi_turn":       exampleMultiTurnConversation,
		"concurrent":       exampleConcurrentResponses,
		"interrupt":        exampleWithInterrupt,
		"manual_handling":  exampleManualMessageHandling,
		"with_options":     exampleWithOptions,
		"bash_command":     exampleBashCommand,
		"control_protocol": exampleControlProtocol,
		"error_handling":   exampleErrorHandling,
		"message_types":    exampleMessageTypes,
		"receive_response": exampleWithReceiveResponse,
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	exampleName := os.Args[1]

	// Create a context that cancels on Ctrl+C
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	if exampleName == "all" {
		// Run all examples
		for name, fn := range examples {
			fmt.Printf("Running: %s\n", name)
			if err := fn(ctx); err != nil {
				// Check for CLI connection error
				if errors.IsTimeout(err) || errors.IsInterrupted(err) {
					fmt.Printf("Error in %s: %v\n", name, err)
					continue
				}
				// For CLI connection errors, print a helpful message
				if _, ok := err.(*errors.CLIConnectionError); ok {
					fmt.Printf("Connection error: %v\n", err)
					fmt.Println("\nPlease ensure Claude CLI is installed and authenticated:")
					fmt.Println("  npm install -g @anthropic-ai/claude-code")
					fmt.Println("  claude login")
					os.Exit(1)
				}
				fmt.Printf("Error in %s: %v\n", name, err)
			}
			fmt.Println(strings.Repeat("-", 50))
			fmt.Println()
		}
	} else if fn, ok := examples[exampleName]; ok {
		// Run specific example
		if err := fn(ctx); err != nil {
			if _, ok := err.(*errors.CLIConnectionError); ok {
				fmt.Printf("Connection error: %v\n", err)
				fmt.Println("\nPlease ensure Claude CLI is installed and authenticated:")
				fmt.Println("  npm install -g @anthropic-ai/claude-code")
				fmt.Println("  claude login")
				os.Exit(1)
			}
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Error: Unknown example '%s'\n\n", exampleName)
		printUsage()
		os.Exit(1)
	}
}
