// Example: Task Messages Handling
//
// This example demonstrates how to handle task messages:
// - TaskStartedMessage: Emitted when a task/subagent starts
// - TaskProgressMessage: Emitted during task execution
// - TaskNotificationMessage: Emitted when task completes, fails, or is stopped
//
// Run with: go run examples/task_messages/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Configure client
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(types.ModelSonnet),
		PermissionMode: &mode,
		MaxTurns:       types.Int(10),
	})
	defer client.Close()

	// Connect
	fmt.Println("=== Connecting to Claude CLI ===")
	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected successfully!")

	// Track task statistics
	tasks := make(map[string]*TaskInfo)

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(ctx)

	// Make a query that might trigger subagent usage
	fmt.Println("\n=== Running Query ===")
	fmt.Println("Prompt: List all .go files in the current directory")

	if err := client.Query(ctx, "List all .go files in the current directory using Bash tool"); err != nil {
		fmt.Fprintf(os.Stderr, "Query error: %v\n", err)
		os.Exit(1)
	}

	// Process messages
	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nContext done (timeout)")
			goto summary
		case msg, ok := <-msgChan:
			if !ok {
				fmt.Println("\nChannel closed")
				goto summary
			}

			switch m := msg.(type) {
			case *types.AssistantMessage:
				handleAssistantMessage(m)

			case *types.TaskStartedMessage:
				handleTaskStarted(m, tasks)

			case *types.TaskProgressMessage:
				handleTaskProgress(m, tasks)

			case *types.TaskNotificationMessage:
				handleTaskNotification(m, tasks)

			case *types.ResultMessage:
				handleResultMessage(m)
				// Drain remaining messages in background
				go func() {
					for range msgChan {
					}
				}()
				goto summary
			}
		}
	}

summary:
	// Print task summary
	fmt.Println("\n=== Task Summary ===")
	if len(tasks) == 0 {
		fmt.Println("No tasks were started during this query")
	} else {
		for id, info := range tasks {
			fmt.Printf("\nTask %s:\n", id)
			fmt.Printf("  Description: %s\n", info.Description)
			fmt.Printf("  Status: %s\n", info.Status)
			if info.TotalTokens > 0 {
				fmt.Printf("  Total Tokens: %d\n", info.TotalTokens)
				fmt.Printf("  Tool Uses: %d\n", info.ToolUses)
				fmt.Printf("  Duration: %dms\n", info.DurationMs)
			}
		}
	}

	fmt.Println("\n=== Example Complete ===")
}

// TaskInfo holds tracking information for a task
type TaskInfo struct {
	Description  string
	Status       string
	TotalTokens  int
	ToolUses     int
	DurationMs   int
	LastProgress time.Time
}

func handleAssistantMessage(m *types.AssistantMessage) {
	for _, block := range m.Content {
		switch b := block.(type) {
		case types.TextBlock:
			fmt.Printf("\nAssistant: %s\n", truncate(b.Text, 200))
		case types.ToolUseBlock:
			fmt.Printf("\nTool Use: %s\n", b.Name)
		case types.ThinkingBlock:
			fmt.Printf("\nThinking: %s\n", truncate(b.Thinking, 100))
		}
	}
}

func handleTaskStarted(m *types.TaskStartedMessage, tasks map[string]*TaskInfo) {
	fmt.Printf("\n🚀 Task Started: %s\n", m.TaskID)
	fmt.Printf("   Description: %s\n", m.Description)
	if m.TaskType != nil {
		fmt.Printf("   Type: %s\n", *m.TaskType)
	}

	tasks[m.TaskID] = &TaskInfo{
		Description:  m.Description,
		Status:       "started",
		LastProgress: time.Now(),
	}
}

func handleTaskProgress(m *types.TaskProgressMessage, tasks map[string]*TaskInfo) {
	fmt.Printf("\n⏳ Task Progress: %s\n", m.TaskID)
	fmt.Printf("   Tokens: %d, Tools: %d, Duration: %dms\n",
		m.Usage.TotalTokens, m.Usage.ToolUses, m.Usage.DurationMs)
	if m.LastToolName != nil {
		fmt.Printf("   Last Tool: %s\n", *m.LastToolName)
	}

	if info, ok := tasks[m.TaskID]; ok {
		info.TotalTokens = m.Usage.TotalTokens
		info.ToolUses = m.Usage.ToolUses
		info.DurationMs = m.Usage.DurationMs
		info.LastProgress = time.Now()
	}
}

func handleTaskNotification(m *types.TaskNotificationMessage, tasks map[string]*TaskInfo) {
	emoji := "✅"
	if m.Status == types.TaskNotificationStatusFailed {
		emoji = "❌"
	} else if m.Status == types.TaskNotificationStatusStopped {
		emoji = "⏹️"
	}

	fmt.Printf("\n%s Task Notification: %s\n", emoji, m.TaskID)
	fmt.Printf("   Status: %s\n", m.Status)
	fmt.Printf("   Summary: %s\n", truncate(m.Summary, 200))

	if info, ok := tasks[m.TaskID]; ok {
		info.Status = string(m.Status)
		if m.Usage != nil {
			info.TotalTokens = m.Usage.TotalTokens
			info.ToolUses = m.Usage.ToolUses
			info.DurationMs = m.Usage.DurationMs
		}
	}
}

func handleResultMessage(m *types.ResultMessage) {
	fmt.Println("\n=== Result ===")
	fmt.Printf("SessionID: %s\n", m.SessionID)
	fmt.Printf("Turns: %d\n", m.NumTurns)
	fmt.Printf("IsError: %v\n", m.IsError)

	if m.StopReason != nil {
		fmt.Printf("StopReason: %s\n", *m.StopReason)
	}

	if m.TotalCostUSD != nil {
		fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
	}

	if m.Usage != nil {
		if tokens, ok := m.Usage["total_tokens"].(float64); ok {
			fmt.Printf("Total Tokens: %.0f\n", tokens)
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
