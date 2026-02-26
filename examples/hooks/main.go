// Example hooks demonstrates hook registration and usage in the Claude Agent SDK for Go.
//
// Hooks allow you to intercept and modify SDK behavior at key points:
// - PreToolUse: Before a tool is executed (can block, modify input, or allow)
// - PostToolUse: After a tool completes successfully (can modify output)
//
// This example shows:
// 1. Registering hooks for PreToolUse and PostToolUse events
// 2. Implementing the HookCallback interface
// 3. Using HookMatcher to filter which tools trigger hooks
// 4. Blocking tools, modifying tool input, and modifying tool output
package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Hook Callback Implementations
// ============================================================================

// LoggingHook is a simple hook that logs tool usage without modifying behavior.
type LoggingHook struct{}

// Execute implements the HookCallback interface.
func (h *LoggingHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	switch i := input.(type) {
	case types.PreToolUseHookInput:
		fmt.Printf("[PreToolUse] Tool: %s, ToolUseID: %s\n", i.ToolName, i.ToolUseID)
	case types.PostToolUseHookInput:
		fmt.Printf("[PostToolUse] Tool: %s completed\n", i.ToolName)
	default:
		// Handle other hook input types
		fmt.Printf("[Hook] Event: %s\n", input.GetHookEventName())
	}
	// Return nil to continue with default behavior
	return nil, nil
}

// BlockingHook demonstrates how to block tool execution.
// This hook prevents execution of tools matching a pattern.
type BlockingHook struct {
	blockedTools map[string]bool
}

// Execute implements the HookCallback interface.
func (h *BlockingHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	preInput, ok := input.(types.PreToolUseHookInput)
	if !ok {
		return nil, nil
	}

	// Check if this tool should be blocked
	if h.blockedTools[preInput.ToolName] {
		fmt.Printf("[BLOCKED] Tool %s is not allowed\n", preInput.ToolName)

		// Block the tool by setting continue=false and providing a reason
		continueVal := false
		reason := fmt.Sprintf("Tool %s is blocked by policy", preInput.ToolName)
		return &types.SyncHookJSONOutput{
			Continue_: &continueVal,
			Reason:    &reason,
		}, nil
	}

	// Allow the tool to proceed
	return nil, nil
}

// InputModifyingHook demonstrates how to modify tool input before execution.
// This example adds a safety check prefix to Bash commands.
type InputModifyingHook struct{}

// Execute implements the HookCallback interface.
func (h *InputModifyingHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	preInput, ok := input.(types.PreToolUseHookInput)
	if !ok {
		return nil, nil
	}

	// Only modify Bash tool commands
	if preInput.ToolName != "Bash" {
		return nil, nil
	}

	// Get the command from tool input
	cmd, ok := preInput.ToolInput["command"].(string)
	if !ok {
		return nil, nil
	}

	// Skip dangerous commands
	dangerousCommands := []string{"rm -rf", "dd", "mkfs", ":(){:|:&};:"}
	for _, dangerous := range dangerousCommands {
		if cmd == dangerous {
			fmt.Printf("[BLOCKED] Dangerous command detected: %s\n", cmd)
			continueVal := false
			reason := "Dangerous command blocked for safety"
			return &types.SyncHookJSONOutput{
				Continue_: &continueVal,
				Reason:    &reason,
			}, nil
		}
	}

	// Add logging prefix to command
	loggingPrefix := "echo '[Executing command]' && "
	modifiedCmd := loggingPrefix + cmd

	fmt.Printf("[MODIFIED] Command: %s -> %s\n", cmd, modifiedCmd)

	// Return modified input
	updatedInput := map[string]interface{}{
		"command": modifiedCmd,
	}

	// Copy other input fields
	for k, v := range preInput.ToolInput {
		if k != "command" {
			updatedInput[k] = v
		}
	}

	return &types.SyncHookJSONOutput{
		HookSpecificOutput: types.PreToolUseHookSpecificOutput{
			HookEventName: "PreToolUse",
			UpdatedInput:  updatedInput,
		},
	}, nil
}

// OutputModifyingHook demonstrates how to modify tool output after execution.
// This example truncates long outputs.
type OutputModifyingHook struct {
	maxOutputLength int
}

// Execute implements the HookCallback interface.
func (h *OutputModifyingHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	postInput, ok := input.(types.PostToolUseHookInput)
	if !ok {
		return nil, nil
	}

	// Get the tool response
	response := postInput.ToolResponse
	responseStr := fmt.Sprintf("%v", response)

	// Truncate if too long
	if len(responseStr) > h.maxOutputLength {
		truncated := responseStr[:h.maxOutputLength] + "...[truncated]"
		fmt.Printf("[MODIFIED OUTPUT] Truncated output for tool %s (original length: %d)\n",
			postInput.ToolName, len(responseStr))

		return &types.SyncHookJSONOutput{
			HookSpecificOutput: types.PostToolUseHookSpecificOutput{
				HookEventName:        "PostToolUse",
				UpdatedMCPToolOutput: truncated,
			},
		}, nil
	}

	return nil, nil
}

// PermissionHook demonstrates using permission decisions.
// This hook can allow or deny tools based on custom logic.
type PermissionHook struct {
	allowedTools map[string]bool
}

// Execute implements the HookCallback interface.
func (h *PermissionHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	preInput, ok := input.(types.PreToolUseHookInput)
	if !ok {
		return nil, nil
	}

	decision := "allow"
	reason := ""

	if !h.allowedTools[preInput.ToolName] {
		decision = "deny"
		reason = fmt.Sprintf("Tool %s is not in the allowed list", preInput.ToolName)
		fmt.Printf("[DENIED] Tool: %s - %s\n", preInput.ToolName, reason)
	} else {
		fmt.Printf("[ALLOWED] Tool: %s\n", preInput.ToolName)
	}

	return &types.SyncHookJSONOutput{
		HookSpecificOutput: types.PreToolUseHookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       &decision,
			PermissionDecisionReason: &reason,
		},
	}, nil
}

// ContextAddingHook demonstrates adding additional context to hooks.
type ContextAddingHook struct{}

// Execute implements the HookCallback interface.
func (h *ContextAddingHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	additionalContext := "This is additional context provided by the hook"

	switch input.(type) {
	case types.PreToolUseHookInput:
		return &types.SyncHookJSONOutput{
			HookSpecificOutput: types.PreToolUseHookSpecificOutput{
				HookEventName:     "PreToolUse",
				AdditionalContext: &additionalContext,
			},
		}, nil
	case types.PostToolUseHookInput:
		return &types.SyncHookJSONOutput{
			HookSpecificOutput: types.PostToolUseHookSpecificOutput{
				HookEventName:     "PostToolUse",
				AdditionalContext: &additionalContext,
			},
		}, nil
	}

	return nil, nil
}

// AsyncHook demonstrates async hook output for long-running operations.
type AsyncHook struct {
	timeoutSeconds int
}

// Execute implements the HookCallback interface.
func (h *AsyncHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	fmt.Printf("[ASYNC] Deferring hook execution with %d second timeout\n", h.timeoutSeconds)

	return types.AsyncHookJSONOutput{
		Async_:       true,
		AsyncTimeout: &h.timeoutSeconds,
	}, nil
}

// ============================================================================
// Main Example
// ============================================================================

func main() {
	fmt.Println("=== Claude Agent SDK Go - Hooks Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates hook usage patterns:")
	fmt.Println("1. Logging hooks - observe tool usage")
	fmt.Println("2. Blocking hooks - prevent tool execution")
	fmt.Println("3. Input modification hooks - change tool inputs")
	fmt.Println("4. Output modification hooks - change tool outputs")
	fmt.Println("5. Permission hooks - allow/deny decisions")
	fmt.Println()

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	// Example 1: Create client with logging hooks
	fmt.Println("--- Example 1: Logging Hooks ---")
	loggingClient := createClientWithLoggingHooks()
	defer loggingClient.Close()
	fmt.Println("Client created with logging hooks for Bash and Read tools")
	fmt.Println()

	// Example 2: Create client with blocking hooks
	fmt.Println("--- Example 2: Blocking Hooks ---")
	blockingClient := createClientWithBlockingHooks()
	defer blockingClient.Close()
	fmt.Println("Client created with blocking hooks for dangerous tools")
	fmt.Println()

	// Example 3: Create client with input modification hooks
	fmt.Println("--- Example 3: Input Modification Hooks ---")
	modifyingClient := createClientWithInputModificationHooks()
	defer modifyingClient.Close()
	fmt.Println("Client created with input modification hooks for Bash commands")
	fmt.Println()

	// Example 4: Create client with permission hooks
	fmt.Println("--- Example 4: Permission Hooks ---")
	permissionClient := createClientWithPermissionHooks()
	defer permissionClient.Close()
	fmt.Println("Client created with permission hooks (allowing only Read and Bash)")
	fmt.Println()

	// Example 5: Create client with multiple hooks per event
	fmt.Println("--- Example 5: Multiple Hooks per Event ---")
	multiHookClient := createClientWithMultipleHooks()
	defer multiHookClient.Close()
	fmt.Println("Client created with multiple hooks chained together")
	fmt.Println()

	// Example 6: Create client with async hooks
	fmt.Println("--- Example 6: Async Hooks ---")
	asyncHookClient := createClientWithAsyncHooks()
	defer asyncHookClient.Close()
	fmt.Println("Client created with async hooks for deferred execution")
	fmt.Println()

	// Example 7: Create client with output modification hooks
	fmt.Println("--- Example 7: Output Modification Hooks ---")
	outputClient := createClientWithOutputModificationHooks()
	defer outputClient.Close()
	fmt.Println("Client created with output modification hooks")
	fmt.Println()

	// To actually run queries, uncomment the following:
	// ctx := context.Background()
	// runExampleQuery(ctx, loggingClient)
}

// createClientWithLoggingHooks creates a client with simple logging hooks.
func createClientWithLoggingHooks() interface{ Close() error } {
	return claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: "Bash|Read|Write", // Match multiple tools using regex
					Hooks: []types.HookCallback{
						&LoggingHook{},
					},
				},
			},
			types.HookEventPostToolUse: {
				{
					Matcher: "Bash|Read|Write",
					Hooks: []types.HookCallback{
						&LoggingHook{},
					},
				},
			},
		},
	})
}

// createClientWithBlockingHooks creates a client that blocks certain tools.
func createClientWithBlockingHooks() interface{ Close() error } {
	return claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks: []types.HookCallback{
						&BlockingHook{
							blockedTools: map[string]bool{
								"Bash": true, // Block Bash tool
							},
						},
					},
				},
			},
		},
	})
}

// createClientWithInputModificationHooks creates a client that modifies tool input.
func createClientWithInputModificationHooks() interface{ Close() error } {
	return claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks: []types.HookCallback{
						&InputModifyingHook{},
					},
				},
			},
		},
	})
}

// createClientWithOutputModificationHooks creates a client that modifies tool output.
func createClientWithOutputModificationHooks() interface{ Close() error } {
	return claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPostToolUse: {
				{
					Matcher: "Bash|Read", // Match Bash and Read tools
					Hooks: []types.HookCallback{
						&OutputModifyingHook{
							maxOutputLength: 1000,
						},
					},
				},
			},
		},
	})
}

// createClientWithPermissionHooks creates a client with permission-based hooks.
func createClientWithPermissionHooks() interface{ Close() error } {
	return claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: ".*", // Match all tools using regex
					Hooks: []types.HookCallback{
						&PermissionHook{
							allowedTools: map[string]bool{
								"Read":  true,
								"Bash":  true,
								"Write": false,
								"Edit":  false,
							},
						},
					},
				},
			},
		},
	})
}

// createClientWithMultipleHooks creates a client with multiple hooks per event.
// Hooks are executed in order they are registered.
func createClientWithMultipleHooks() interface{ Close() error } {
	return claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: "Bash",
					Hooks: []types.HookCallback{
						&LoggingHook{},        // First: log the tool use
						&InputModifyingHook{}, // Second: modify the input
					},
				},
			},
			types.HookEventPostToolUse: {
				{
					Matcher: "Bash",
					Hooks: []types.HookCallback{
						&LoggingHook{}, // First: log completion
						&OutputModifyingHook{
							maxOutputLength: 500,
						}, // Second: modify output if needed
					},
				},
			},
		},
	})
}

// createClientWithAsyncHooks creates a client with async hooks.
func createClientWithAsyncHooks() interface{ Close() error } {
	return claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: ".*",
					Hooks: []types.HookCallback{
						&AsyncHook{
							timeoutSeconds: 30,
						},
					},
					Timeout: float64Ptr(60), // Hook timeout in seconds
				},
			},
		},
	})
}

// float64Ptr returns a pointer to a float64.
func float64Ptr(f float64) *float64 {
	return &f
}

// runExampleQuery demonstrates how to run a query with hooks enabled.
func runExampleQuery(ctx context.Context, client interface {
	Connect(context.Context) error
	Query(context.Context, interface{}, ...string) (<-chan types.Message, error)
	Close() error
}) {
	// Connect to Claude
	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	// Send a query
	msgChan, err := client.Query(ctx, "List the files in the current directory using Bash")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	// Process messages
	for msg := range msgChan {
		fmt.Printf("Message: %v\n", msg)
	}
}
