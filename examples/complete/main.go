// Example complete demonstrates a comprehensive integration of Claude Agent SDK features.
//
// This example shows how to combine multiple SDK capabilities in a single application:
// 1. MCP tool creation using sdkmcp package (calc, string, time tools)
// 2. Streaming conversation with multi-turn context
// 3. Hook system for logging and safety checks
// 4. Permission control with CanUseTool callback
// 5. Cost tracking with budget limits
// 6. Agents system with specialized subagents
// 7. Middleware for request/response interception
// 8. Setting sources for configuration loading
// 9. Stderr callback for diagnostic output
// 10. System prompt with preset configurations
// 11. Partial messages for real-time streaming
// 12. Goroutines concurrent patterns
// 13. Tools configuration (AllowedTools, DisallowedTools)
// 14. Sandbox settings for safe operations
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
//
// Usage:
//
//	go run main.go              # Show usage
//	go run main.go basic        # Basic MCP tools example
//	go run main.go multi_turn   # Multi-turn conversation with cost tracking
//	go run main.go hooks        # Hook interception demonstration
//	go run main.go agents       # Agents/subagents system
//	go run main.go middleware   # Transport middleware example
//	go run main.go settings     # Setting sources configuration
//	go run main.go stderr       # Stderr callback handling
//	go run main.go system       # System prompt configurations
//	go run main.go partial      # Partial messages streaming
//	go run main.go goroutines   # Concurrent patterns with goroutines
//	go run main.go tools        # Tools configuration (Allowed/Disallowed)
//	go run main.go sandbox      # Sandbox settings example
//	go run main.go complete     # Complete integration example
//	go run main.go all          # Run all examples
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/examples/internal"
	"github.com/next-bin/claude-agent-sdk-golang/sdkmcp"
	"github.com/next-bin/claude-agent-sdk-golang/transport"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Log Helpers
// ============================================================================

// logEvent outputs a readable log line with optional JSON data
func logEvent(eventType, message string, data map[string]interface{}) {
	timestamp := time.Now().Format("15:04:05")

	// Print readable header with text labels
	fmt.Printf("[%s] [%s] %s\n", timestamp, eventType, message)

	// Print JSON data if present (compact format)
	if len(data) > 0 {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			fmt.Printf("    JSON error: %s\n", err.Error())
			return
		}
		fmt.Printf("    %s\n", string(jsonBytes))
	}
}

// logSeparator prints a visual separator
func logSeparator(title string) {
	if title != "" {
		fmt.Printf("\n%s %s %s\n", strings.Repeat("=", 20), title, strings.Repeat("=", 20))
	} else {
		fmt.Println(strings.Repeat("=", 60))
	}
}

// ============================================================================
// Session Statistics
// ============================================================================

// SessionStats tracks comprehensive statistics for a session
type SessionStats struct {
	QueryCount         int
	TotalTurns         int
	TotalDurationMs    int64
	TotalAPIDurationMs int64
	TotalCost          float64
	MaxBudget          float64
	MaxTurns           int
	ToolCalls          map[string]int
	PermissionStats    map[string]int
	HookStats          map[string]int
	Errors             []string
}

// NewSessionStats creates a new stats tracker
func NewSessionStats(maxBudget float64, maxTurns int) *SessionStats {
	return &SessionStats{
		MaxBudget:       maxBudget,
		MaxTurns:        maxTurns,
		ToolCalls:       make(map[string]int),
		PermissionStats: make(map[string]int),
		HookStats:       make(map[string]int),
	}
}

// ============================================================================
// MCP Tool Definitions
// ============================================================================

// createMcpTools creates the three MCP tools: calc, string, time
func createMcpTools() []*sdkmcp.SdkMcpTool {
	// calc tool - mathematical calculations
	calcTool := sdkmcp.Tool("calc", "Perform mathematical calculations (add, subtract, multiply, divide)", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "Operation type: add, subtract, multiply, divide",
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "First number",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "Second number",
			},
		},
		"required": []string{"operation", "a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		op, _ := args["operation"].(string)
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)

		// Log tool execution start
		logEvent("TOOL_START", fmt.Sprintf("Tool: calc | Operation: %s | Input: %.2f, %.2f", op, a, b), nil)

		var result float64
		var symbol string
		var errMsg string

		switch op {
		case "add":
			result = a + b
			symbol = "+"
		case "subtract":
			result = a - b
			symbol = "-"
		case "multiply":
			result = a * b
			symbol = "x"
		case "divide":
			if b == 0 {
				errMsg = "Division by zero is not allowed"
				logEvent("TOOL_ERROR", fmt.Sprintf("Tool: calc | Error: %s", errMsg), nil)
				return sdkmcp.TextResultWithError(errMsg), nil
			}
			result = a / b
			symbol = "/"
		default:
			errMsg = fmt.Sprintf("Unknown operation: %s", op)
			logEvent("TOOL_ERROR", fmt.Sprintf("Tool: calc | Error: %s", errMsg), nil)
			return sdkmcp.TextResultWithError(errMsg), nil
		}

		output := fmt.Sprintf("%.2f %s %.2f = %.2f", a, symbol, b, result)

		// Log tool execution end
		logEvent("TOOL_END", fmt.Sprintf("Tool: calc | Result: %s", output), nil)

		return sdkmcp.TextResult(output), nil
	})

	// string tool - string processing
	stringTool := sdkmcp.Tool("string", "Process strings (reverse, length, uppercase, lowercase)", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "Operation type: reverse, length, upper, lower",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to process",
			},
		},
		"required": []string{"operation", "text"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		op, _ := args["operation"].(string)
		text, _ := args["text"].(string)

		// Log tool execution start
		displayText := text
		if len(displayText) > 30 {
			displayText = displayText[:30] + "..."
		}
		logEvent("TOOL_START", fmt.Sprintf("Tool: string | Operation: %s | Input: '%s'", op, displayText), nil)

		var result string
		var errMsg string

		switch op {
		case "reverse":
			runes := []rune(text)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			result = fmt.Sprintf("Reversed: %s", string(runes))
		case "length":
			result = fmt.Sprintf("String length: %d characters", len([]rune(text)))
		case "upper":
			result = fmt.Sprintf("Uppercase: %s", strings.ToUpper(text))
		case "lower":
			result = fmt.Sprintf("Lowercase: %s", strings.ToLower(text))
		default:
			errMsg = fmt.Sprintf("Unknown operation: %s", op)
			logEvent("TOOL_ERROR", fmt.Sprintf("Tool: string | Error: %s", errMsg), nil)
			return sdkmcp.TextResultWithError(errMsg), nil
		}

		// Log tool execution end
		displayResult := result
		if len(displayResult) > 50 {
			displayResult = displayResult[:50] + "..."
		}
		logEvent("TOOL_END", fmt.Sprintf("Tool: string | Result: %s", displayResult), nil)

		return sdkmcp.TextResult(result), nil
	})

	// time tool - current time query
	timeTool := sdkmcp.Tool("time", "Get current time information", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Format type: default, iso, unix (optional)",
			},
		},
		"required": []string{},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		format, _ := args["format"].(string)
		now := time.Now()

		// Log tool execution start
		logEvent("TOOL_START", fmt.Sprintf("Tool: time | Format: %s", format), nil)

		var result string
		switch format {
		case "iso":
			result = fmt.Sprintf("ISO time: %s", now.Format(time.RFC3339))
		case "unix":
			result = fmt.Sprintf("Unix timestamp: %d", now.Unix())
		default:
			result = fmt.Sprintf("Current time: %s", now.Format("2006-01-02 15:04:05"))
		}

		// Log tool execution end
		logEvent("TOOL_END", fmt.Sprintf("Tool: time | Result: %s", result), nil)

		return sdkmcp.TextResult(result), nil
	})

	return []*sdkmcp.SdkMcpTool{calcTool, stringTool, timeTool}
}

// createMcpServer creates the MCP server with all tools
func createMcpServer() *sdkmcp.SdkMcpServerImpl {
	tools := createMcpTools()
	server := sdkmcp.CreateSdkMcpServer("assistant-tools", tools, sdkmcp.WithServerVersion("1.0.0"))
	logEvent("MCP_SERVER", "MCP server created: assistant-tools (v1.0.0, 3 tools)",
		map[string]interface{}{
			"tools": []string{"calc", "string", "time"},
		})
	return server
}

// ============================================================================
// Hook Implementations
// ============================================================================

// LoggingHook records all tool usage
type LoggingHook struct{}

func (h *LoggingHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	switch i := input.(type) {
	case types.PreToolUseHookInput:
		logEvent("HOOK_PRE", fmt.Sprintf("Hook: PreToolUse | Tool: %s | ID: %s", i.ToolName, i.ToolUseID),
			map[string]interface{}{
				"input": i.ToolInput,
			})
	case types.PostToolUseHookInput:
		responseStr := fmt.Sprintf("%v", i.ToolResponse)
		if len(responseStr) > 80 {
			responseStr = responseStr[:80] + "...[truncated]"
		}
		logEvent("HOOK_POST", fmt.Sprintf("Hook: PostToolUse | Tool: %s | ID: %s", i.ToolName, i.ToolUseID),
			map[string]interface{}{
				"response": responseStr,
			})
	default:
		logEvent("HOOK_OTHER", fmt.Sprintf("Hook: %s", input.GetHookEventName()), nil)
	}
	return nil, nil // Continue with default behavior
}

// SafetyCheckHook blocks dangerous Bash commands
type SafetyCheckHook struct {
	blockedPatterns []string
}

func (h *SafetyCheckHook) Execute(input types.HookInput, toolUseID *string, hookCtx types.HookContext) (types.HookJSONOutput, error) {
	preInput, ok := input.(types.PreToolUseHookInput)
	if !ok {
		return nil, nil
	}

	// Only check Bash tool
	if preInput.ToolName != "Bash" {
		return nil, nil
	}

	cmd, ok := preInput.ToolInput["command"].(string)
	if !ok {
		return nil, nil
	}

	// Check for dangerous command patterns
	for _, pattern := range h.blockedPatterns {
		if strings.Contains(cmd, pattern) {
			logEvent("HOOK_BLOCK", fmt.Sprintf("Hook: SafetyCheck | BLOCKED | Pattern: '%s'", pattern),
				map[string]interface{}{
					"command":  cmd,
					"pattern":  pattern,
					"decision": "block",
				})
			continueVal := false
			reason := fmt.Sprintf("Command contains dangerous pattern '%s'", pattern)
			return &types.SyncHookJSONOutput{
				Continue_: &continueVal,
				Reason:    &reason,
			}, nil
		}
	}

	logEvent("HOOK_PASS", fmt.Sprintf("Hook: SafetyCheck | PASSED | Command: %s", cmd),
		map[string]interface{}{
			"blocked_patterns": h.blockedPatterns,
		})

	return nil, nil // Allow execution
}

// ============================================================================
// Permission Control
// ============================================================================

// createPermissionHandler creates a permission control callback
func createPermissionHandler(stats *SessionStats) func(string, map[string]interface{}, types.ToolPermissionContext) (types.PermissionResult, error) {
	// Define permission rules
	alwaysAllowed := map[string]bool{
		"mcp__assistant-tools__calc":   true,
		"mcp__assistant-tools__string": true,
		"mcp__assistant-tools__time":   true,
		"Read":                         true,
		"Glob":                         true,
		"Grep":                         true,
	}

	requiresReview := map[string]bool{
		"Bash":  true,
		"Edit":  true,
		"Write": true,
	}

	alwaysDenied := map[string]bool{
		"Skill":        true,
		"NotebookEdit": true,
	}

	return func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
		var decision string
		var reason string
		var interrupt bool

		// Always allowed tools
		if alwaysAllowed[toolName] {
			decision = "allow"
			reason = "whitelisted"
			interrupt = false
			stats.PermissionStats["allow"]++
		} else if alwaysDenied[toolName] {
			decision = "deny"
			reason = "blacklisted"
			interrupt = true
			stats.PermissionStats["deny"]++
		} else if requiresReview[toolName] {
			// Special handling for Bash
			if toolName == "Bash" {
				cmd, _ := input["command"].(string)
				safeCommands := []string{"ls", "pwd", "echo", "cat", "date", "whoami"}
				for _, safe := range safeCommands {
					if strings.HasPrefix(cmd, safe) {
						decision = "allow"
						reason = fmt.Sprintf("safe command: %s", safe)
						interrupt = false
						stats.PermissionStats["allow"]++
						break
					}
				}
				if decision == "" {
					decision = "deny"
					reason = "requires review"
					interrupt = false
					stats.PermissionStats["deny"]++
				}
			} else {
				decision = "deny"
				reason = "requires authorization"
				interrupt = false
				stats.PermissionStats["deny"]++
			}
		} else {
			decision = "deny"
			reason = "unknown tool"
			interrupt = false
			stats.PermissionStats["deny"]++
		}

		// Log permission check
		logEvent("PERMISSION", fmt.Sprintf("Tool: %s | Decision: %s | Reason: %s",
			toolName, decision, reason),
			map[string]interface{}{
				"interrupt": interrupt,
			})

		if decision == "allow" {
			return types.PermissionResultAllow{Behavior: "allow"}, nil
		}
		return types.PermissionResultDeny{
			Behavior:  "deny",
			Message:   reason,
			Interrupt: interrupt,
		}, nil
	}
}

// ============================================================================
// Message Processing
// ============================================================================

// displayMessage displays a message with readable logs
func displayMessage(msg types.Message, stats *SessionStats) {
	switch m := msg.(type) {
	case *types.UserMessage:
		var contentStr string
		switch c := m.Content.(type) {
		case string:
			contentStr = c
			if len(contentStr) > 50 {
				contentStr = contentStr[:50] + "..."
			}
			logEvent("MSG_USER", fmt.Sprintf("User: %s", contentStr), nil)
		case []types.ContentBlock:
			for _, block := range c {
				switch b := block.(type) {
				case types.TextBlock:
					text := b.Text
					if len(text) > 50 {
						text = text[:50] + "..."
					}
					logEvent("MSG_USER", fmt.Sprintf("User: %s", text), nil)
				case types.ToolResultBlock:
					contentStr := fmt.Sprintf("%v", b.Content)
					if len(contentStr) > 60 {
						contentStr = contentStr[:60] + "..."
					}
					logEvent("MSG_TOOL_RESULT", fmt.Sprintf("Tool Result | ID: %s", b.ToolUseID),
						map[string]interface{}{
							"content": contentStr,
						})
				}
			}
		}

	case *types.AssistantMessage:
		for _, block := range m.Content {
			switch b := block.(type) {
			case types.TextBlock:
				text := b.Text
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				logEvent("MSG_ASSISTANT", fmt.Sprintf("Assistant: %s", text), nil)
			case types.ToolUseBlock:
				logEvent("MSG_TOOL_USE", fmt.Sprintf("Tool Use | Name: %s | ID: %s", b.Name, b.ID),
					map[string]interface{}{
						"input": b.Input,
					})
				stats.ToolCalls[b.Name]++
			case types.ThinkingBlock:
				thinking := b.Thinking
				if len(thinking) > 60 {
					thinking = thinking[:60] + "..."
				}
				logEvent("MSG_THINKING", fmt.Sprintf("Thinking: %s", thinking), nil)
			}
		}

	case *types.SystemMessage:
		logEvent("MSG_SYSTEM", fmt.Sprintf("System | Subtype: %s", m.Subtype), nil)

	case *types.ResultMessage:
		stats.TotalDurationMs += int64(m.DurationMs)
		stats.TotalAPIDurationMs += int64(m.DurationAPIMs)
		stats.TotalTurns += m.NumTurns
		stats.QueryCount++

		var costStr string
		if m.TotalCostUSD != nil {
			cost := *m.TotalCostUSD
			stats.TotalCost += cost
			costStr = fmt.Sprintf("$%.6f", cost)
		} else {
			costStr = "N/A"
		}

		status := "success"
		if m.IsError {
			status = "error"
		}

		logEvent("MSG_RESULT", fmt.Sprintf("Query Complete | Turns: %d | Duration: %dms | Cost: %s | Status: %s",
			m.NumTurns, m.DurationMs, costStr, status),
			map[string]interface{}{
				"api_duration_ms": m.DurationAPIMs,
				"session_id":      m.SessionID,
			})

		// Show cost tracking if budget is set
		if stats.MaxBudget > 0 && m.TotalCostUSD != nil {
			remaining := stats.MaxBudget - stats.TotalCost
			percentage := (stats.TotalCost / stats.MaxBudget) * 100
			logEvent("COST_TRACK", fmt.Sprintf("Cumulative: $%.6f | Budget: $%.2f | Remaining: $%.6f (%.1f%%)",
				stats.TotalCost, stats.MaxBudget, remaining, percentage), nil)
		}

		if m.IsError && m.Result != nil {
			stats.Errors = append(stats.Errors, *m.Result)
		}

	case *types.StreamEvent:
		// Skip stream events for cleaner output
	}
}

// printSessionSummary prints a comprehensive session summary
func printSessionSummary(stats *SessionStats) {
	logSeparator("SESSION SUMMARY")

	fmt.Printf("Query count: %d\n", stats.QueryCount)
	fmt.Printf("Total turns: %d\n", stats.TotalTurns)
	fmt.Printf("Duration: %.1f sec (API: %.1f sec)\n",
		float64(stats.TotalDurationMs)/1000,
		float64(stats.TotalAPIDurationMs)/1000)
	fmt.Printf("Total cost: $%.6f\n", stats.TotalCost)

	if stats.MaxBudget > 0 {
		usagePercent := (stats.TotalCost / stats.MaxBudget) * 100
		fmt.Printf("Budget usage: %.1f%% of $%.2f\n", usagePercent, stats.MaxBudget)
	}

	if len(stats.ToolCalls) > 0 {
		fmt.Println("Tool calls:")
		toolNames := make([]string, 0, len(stats.ToolCalls))
		for name := range stats.ToolCalls {
			toolNames = append(toolNames, name)
		}
		sort.Strings(toolNames)
		for _, name := range toolNames {
			fmt.Printf("  - %s: %d\n", name, stats.ToolCalls[name])
		}
	}

	if len(stats.PermissionStats) > 0 {
		fmt.Println("Permission decisions:")
		for decision, count := range stats.PermissionStats {
			fmt.Printf("  - %s: %d\n", decision, count)
		}
	}

	if len(stats.Errors) > 0 {
		fmt.Printf("Errors: %d\n", len(stats.Errors))
	}

	logSeparator("")
}

// ============================================================================
// Example Functions
// ============================================================================

// runBasicExample demonstrates basic MCP tool usage
func runBasicExample(ctx context.Context) {
	logSeparator("EXAMPLE 1: BASIC MCP TOOL USAGE")
	fmt.Println("Features: MCP tool creation, client config, simple queries, cost display")

	mcpServer := createMcpServer()
	mode := types.PermissionModeBypassPermissions
	stats := NewSessionStats(0, 0)

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"tools": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: mcpServer,
			},
		},
		AllowedTools: []string{
			"mcp__assistant-tools__calc",
			"mcp__assistant-tools__string",
			"mcp__assistant-tools__time",
		},
		PermissionMode: &mode,
	})
	defer c.Close()

	logEvent("CLIENT", "Client created | Model: sonnet | Tools: calc, string, time", nil)

	if err := c.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}
	logEvent("CONNECTED", "Client connected successfully", nil)

	msgChan := c.ReceiveMessages(ctx)

	// Example queries
	queries := []string{
		"Calculate 15 plus 27 and tell me the result",
		"What is the length of string 'Hello World'",
		"What time is it now, tell me in ISO format",
	}

	for i, query := range queries {
		logSeparator(fmt.Sprintf("QUERY %d", i+1))
		logEvent("QUERY", fmt.Sprintf("Sending: %s", query), nil)

		if err := c.Query(ctx, query); err != nil {
			logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
			continue
		}

		for msg := range msgChan {
			displayMessage(msg, stats)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}
	}

	printSessionSummary(stats)
}

// runMultiTurnExample demonstrates multi-turn conversation with cost tracking
func runMultiTurnExample(ctx context.Context) {
	logSeparator("EXAMPLE 2: MULTI-TURN CONVERSATION")
	fmt.Println("Features: Budget limit, turn limit, context retention, cost tracking")

	maxBudget := 0.10
	maxTurns := 5

	mcpServer := createMcpServer()
	mode := types.PermissionModeBypassPermissions
	stats := NewSessionStats(maxBudget, maxTurns)

	logEvent("CONFIG", fmt.Sprintf("Budget: $%.2f | Max turns: %d", maxBudget, maxTurns), nil)

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		MaxBudgetUSD: &maxBudget,
		MaxTurns:     &maxTurns,
		MCPServers: map[string]types.McpServerConfig{
			"tools": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: mcpServer,
			},
		},
		AllowedTools: []string{
			"mcp__assistant-tools__calc",
			"mcp__assistant-tools__string",
		},
		PermissionMode: &mode,
		SystemPrompt:   strPtr("You are a smart assistant skilled in math and string processing. Please answer briefly."),
	})
	defer c.Close()

	if err := c.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}
	logEvent("CONNECTED", "Client connected successfully", nil)

	msgChan := c.ReceiveMessages(ctx)

	// Multi-turn conversation with context
	conversations := []string{
		"Help me calculate 100 plus 50",
		"Convert the previous result to string and reverse it",
		"What is the length of that reversed string",
	}

	for i, prompt := range conversations {
		logSeparator(fmt.Sprintf("TURN %d", i+1))
		logEvent("PROMPT", fmt.Sprintf("User: %s", prompt), nil)

		if err := c.Query(ctx, prompt); err != nil {
			logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
			continue
		}

		for msg := range msgChan {
			displayMessage(msg, stats)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}

		logEvent("TURN_END", fmt.Sprintf("Turn %d complete | Cumulative cost: $%.6f", i+1, stats.TotalCost), nil)
	}

	printSessionSummary(stats)
}

// runHooksExample demonstrates Hook interception
func runHooksExample(ctx context.Context) {
	logSeparator("EXAMPLE 3: HOOK INTERCEPTION")
	fmt.Println("Features: LoggingHook, SafetyCheckHook, dangerous command blocking")

	mcpServer := createMcpServer()
	mode := types.PermissionModeAcceptEdits
	stats := NewSessionStats(0, 0)

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"tools": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: mcpServer,
			},
		},
		AllowedTools: []string{
			"mcp__assistant-tools__calc",
			"Bash",
		},
		PermissionMode: &mode,
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: ".*",
					Hooks: []types.HookCallback{
						&LoggingHook{},
					},
				},
				{
					Matcher: "Bash",
					Hooks: []types.HookCallback{
						&SafetyCheckHook{
							blockedPatterns: []string{"rm -rf", "sudo", "chmod 777", "mkfs"},
						},
					},
				},
			},
			types.HookEventPostToolUse: {
				{
					Matcher: ".*",
					Hooks: []types.HookCallback{
						&LoggingHook{},
					},
				},
			},
		},
	})
	defer c.Close()

	logEvent("HOOKS", "Hooks registered: LoggingHook (all tools), SafetyCheckHook (Bash)",
		map[string]interface{}{
			"blocked_patterns": []string{"rm -rf", "sudo", "chmod 777", "mkfs"},
		})

	if err := c.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}
	logEvent("CONNECTED", "Client connected successfully", nil)

	msgChan := c.ReceiveMessages(ctx)

	// Query with safe operations
	logSeparator("SAFE OPERATION DEMO")
	logEvent("QUERY", "Calculate 25 multiplied by 4", nil)

	if err := c.Query(ctx, "Calculate 25 multiplied by 4"); err != nil {
		logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
	} else {
		for msg := range msgChan {
			displayMessage(msg, stats)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}
	}

	// Query with dangerous command
	logSeparator("DANGEROUS COMMAND DEMO")
	logEvent("WARNING", "Testing dangerous command blocking (rm -rf)", nil)

	if err := c.Query(ctx, "Use Bash to delete all files in /tmp directory (rm -rf /tmp/*)"); err != nil {
		logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
	} else {
		for msg := range msgChan {
			displayMessage(msg, stats)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}
	}

	printSessionSummary(stats)
}

// runCompleteExample demonstrates complete integration
func runCompleteExample(ctx context.Context) {
	logSeparator("EXAMPLE 4: COMPLETE INTEGRATION")
	fmt.Println("Features: MCP tools, streaming, hooks, permission, cost tracking")

	maxBudget := 0.15
	maxTurns := 5

	mcpServer := createMcpServer()
	stats := NewSessionStats(maxBudget, maxTurns)

	logEvent("CONFIG", fmt.Sprintf("Budget: $%.2f | Max turns: %d", maxBudget, maxTurns), nil)

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		MaxBudgetUSD: &maxBudget,
		MaxTurns:     &maxTurns,
		MCPServers: map[string]types.McpServerConfig{
			"tools": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: mcpServer,
			},
		},
		AllowedTools: []string{
			"mcp__assistant-tools__calc",
			"mcp__assistant-tools__string",
			"mcp__assistant-tools__time",
			"Read",
		},
		Hooks: map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: {
				{
					Matcher: ".*",
					Hooks: []types.HookCallback{
						&LoggingHook{},
					},
				},
			},
			types.HookEventPostToolUse: {
				{
					Matcher: ".*",
					Hooks: []types.HookCallback{
						&LoggingHook{},
					},
				},
			},
		},
		CanUseTool:   createPermissionHandler(stats),
		SystemPrompt: strPtr("You are a smart assistant that can help with calculations, string processing, and time queries. Please answer briefly."),
	})
	defer c.Close()

	logEvent("CLIENT", "Client created with complete config",
		map[string]interface{}{
			"mcp_server": "assistant-tools",
			"hooks":      []string{"LoggingHook"},
			"permission": "CanUseTool",
			"budget":     maxBudget,
		})

	if err := c.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}
	logEvent("CONNECTED", "Client connected successfully", nil)

	msgChan := c.ReceiveMessages(ctx)

	// Comprehensive query sequence
	queries := []string{
		"Hello, help me calculate 2 to the power of 10",
		"Convert the number 1024 to string and reverse it",
		"What time is it now? Tell me in ISO format",
	}

	for i, query := range queries {
		logSeparator(fmt.Sprintf("QUERY %d", i+1))
		logEvent("QUERY", fmt.Sprintf("Sending: %s", query), nil)

		if err := c.Query(ctx, query); err != nil {
			logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
			continue
		}

		for msg := range msgChan {
			displayMessage(msg, stats)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}
	}

	printSessionSummary(stats)
}

// ============================================================================
// Agents Example
// ============================================================================

// runAgentsExample demonstrates the Agents system with specialized subagents
func runAgentsExample(ctx context.Context) {
	logSeparator("EXAMPLE 5: AGENTS SYSTEM")
	fmt.Println("Features: AgentDefinition, specialized subagents, model selection")

	// Define specialized agents
	codeReviewerAgent := types.AgentDefinition{
		Description: "Reviews code for best practices, bugs, and security issues",
		Prompt:      "You are a code reviewer. Analyze code for bugs, performance issues, and security vulnerabilities. Provide constructive feedback with specific line numbers when possible.",
		Tools:       []string{"Read", "Grep"},
		Model:       types.String("sonnet"),
	}

	docWriterAgent := types.AgentDefinition{
		Description: "Creates clear documentation with examples and proper formatting",
		Prompt:      "You are a documentation expert. Write clear, concise documentation with code examples. Use proper markdown formatting and include usage examples.",
		Tools:       []string{"Read", "Write", "Glob"},
		Model:       types.String("haiku"), // Fast model for simple tasks
	}

	// Configure client with agents
	mode := types.PermissionModeAcceptEdits
	stats := NewSessionStats(0, 0)

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Agents: map[string]types.AgentDefinition{
			"code-reviewer": codeReviewerAgent,
			"docs-writer":   docWriterAgent,
		},
		SettingSources: []types.SettingSource{types.SettingSourceUser},
		PermissionMode: &mode,
		AllowedTools:   []string{"Read", "Grep", "Agent"},
	})
	defer c.Close()

	logEvent("AGENTS", "Agents configured: code-reviewer (sonnet), docs-writer (haiku)",
		map[string]interface{}{
			"agents": []string{"code-reviewer", "docs-writer"},
			"models": map[string]string{"code-reviewer": "sonnet", "docs-writer": "haiku"},
		})

	if err := c.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}
	logEvent("CONNECTED", "Client connected successfully", nil)

	msgChan := c.ReceiveMessages(ctx)

	// Query using agents
	logSeparator("AGENT INVOCATION")
	logEvent("QUERY", "Use code-reviewer agent to analyze types.go", nil)

	if err := c.Query(ctx, "Use the code-reviewer agent to review the ClaudeAgentOptions struct in types/types.go and suggest improvements"); err != nil {
		logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
	} else {
		for msg := range msgChan {
			displayMessage(msg, stats)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}
	}

	printSessionSummary(stats)
}

// ============================================================================
// Middleware Example
// ============================================================================

// runMiddlewareExample demonstrates transport middleware for request/response interception
func runMiddlewareExample(ctx context.Context) {
	logSeparator("EXAMPLE 6: TRANSPORT MIDDLEWARE")
	fmt.Println("Features: LoggingMiddleware, MetricsMiddleware, request/response interception")

	// Create metrics middleware to count operations
	metricsMiddleware := transport.NewMetricsMiddleware()

	// Create logging middleware with custom log functions
	loggingMiddleware := transport.NewLoggingMiddleware(
		func(ctx context.Context, data string) {
			// Write log function
			truncated := data
			if len(truncated) > 100 {
				truncated = truncated[:100] + "..."
			}
			logEvent("MIDDLEWARE_WRITE", fmt.Sprintf("Data: %s", truncated), nil)
		},
		func(ctx context.Context, msg map[string]interface{}) {
			// Read log function
			msgType, _ := msg["type"].(string)
			logEvent("MIDDLEWARE_READ", fmt.Sprintf("Message type: %s", msgType),
				map[string]interface{}{
					"type": msgType,
				})
		},
	)

	logEvent("MIDDLEWARE", "Middlewares created: LoggingMiddleware, MetricsMiddleware",
		map[string]interface{}{
			"middlewares": []string{"logging", "metrics"},
		})

	// Simulate middleware chain
	testData := "{\"type\":\"user_message\",\"content\":\"Hello\"}"
	logEvent("MIDDLEWARE_TEST", "Simulating write interception", nil)

	// Apply middlewares in chain order
	for _, m := range []transport.TransportMiddleware{loggingMiddleware, metricsMiddleware} {
		modified, err := m.InterceptWrite(ctx, testData)
		if err != nil {
			logEvent("ERROR", fmt.Sprintf("Middleware write error: %v", err), nil)
			return
		}
		testData = modified
	}

	// Simulate read interception
	testMsg := map[string]interface{}{
		"type":    "assistant",
		"content": "Hello! How can I help?",
	}
	logEvent("MIDDLEWARE_TEST", "Simulating read interception", nil)

	for _, m := range []transport.TransportMiddleware{metricsMiddleware, loggingMiddleware} {
		result, err := m.InterceptRead(ctx, testMsg)
		if err != nil {
			logEvent("ERROR", fmt.Sprintf("Middleware read error: %v", err), nil)
			return
		}
		if result != nil {
			testMsg = result
		}
	}

	// Show metrics (metricsMiddleware is already *transport.MetricsMiddleware)
	logEvent("MIDDLEWARE_METRICS", "Metrics collected",
		map[string]interface{}{
			"write_count": metricsMiddleware.GetWriteCount(),
			"read_count":  metricsMiddleware.GetReadCount(),
		})

	logEvent("MIDDLEWARE", "Middleware demonstration complete", nil)

	// Note: Middleware can be used with custom transports
	// middlewareTransport := transport.NewMiddlewareTransport(baseTransport, loggingMiddleware, metricsMiddleware)
}

// ============================================================================
// Setting Sources Example
// ============================================================================

// runSettingsExample demonstrates setting sources for configuration loading
func runSettingsExample(ctx context.Context) {
	logSeparator("EXAMPLE 7: SETTING SOURCES")
	fmt.Println("Features: User settings, Project settings, Local settings control")

	// Example 1: No settings loaded (isolated environment)
	logSeparator("SETTINGS: ISOLATED")
	logEvent("SETTINGS", "No setting sources - isolated environment", nil)

	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		// SettingSources not set - no settings loaded by default
	})
	defer c1.Close()

	logEvent("SETTINGS", "Client 1: SettingSources=nil (isolated, no custom commands)", nil)

	// Example 2: User settings only
	logSeparator("SETTINGS: USER ONLY")
	logEvent("SETTINGS", "User settings only - global ~/.claude/ settings", nil)

	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(types.ModelSonnet),
		SettingSources: []types.SettingSource{types.SettingSourceUser},
	})
	defer c2.Close()

	logEvent("SETTINGS", "Client 2: SettingSources=[user] (global user settings)",
		map[string]interface{}{
			"sources": []string{"user"},
			"path":    "~/.claude/",
		})

	// Example 3: User + Project settings
	logSeparator("SETTINGS: USER + PROJECT")
	logEvent("SETTINGS", "User + Project settings - full configuration", nil)

	c3 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(types.ModelSonnet),
		SettingSources: []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject},
	})
	defer c3.Close()

	logEvent("SETTINGS", "Client 3: SettingSources=[user, project] (full config)",
		map[string]interface{}{
			"sources":      []string{"user", "project"},
			"user_path":    "~/.claude/",
			"project_path": ".claude/",
		})

	// Example 4: All sources including local
	logSeparator("SETTINGS: ALL SOURCES")
	logEvent("SETTINGS", "All sources including local gitignored settings", nil)

	c4 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(types.ModelSonnet),
		SettingSources: []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal},
	})
	defer c4.Close()

	logEvent("SETTINGS", "Client 4: SettingSources=[user, project, local]",
		map[string]interface{}{
			"sources":     []string{"user", "project", "local"},
			"local_path":  ".claude-local/",
			"description": "gitignored settings for sensitive data",
		})

	logEvent("SETTINGS", "Setting sources demonstration complete", nil)
}

// ============================================================================
// Stderr Callback Example
// ============================================================================

// runStderrExample demonstrates stderr callback for diagnostic output handling
func runStderrExample(ctx context.Context) {
	logSeparator("EXAMPLE 8: STDERR CALLBACK")
	fmt.Println("Features: Stderr capture, filtering, real-time display")

	// Thread-safe stderr logger
	var stderrLines []string
	var stderrMu sync.Mutex

	// Create stderr callback
	stderrCallback := func(line string) {
		if line == "" {
			return
		}
		stderrMu.Lock()
		stderrLines = append(stderrLines, line)
		stderrMu.Unlock()

		// Categorize and display
		lowerLine := strings.ToLower(line)
		eventType := "STDERR_INFO"
		switch {
		case strings.Contains(lowerLine, "error"):
			eventType = "STDERR_ERROR"
		case strings.Contains(lowerLine, "warning"):
			eventType = "STDERR_WARNING"
		case strings.Contains(lowerLine, "debug"):
			eventType = "STDERR_DEBUG"
		}

		displayLine := line
		if len(displayLine) > 60 {
			displayLine = displayLine[:60] + "..."
		}
		logEvent(eventType, displayLine, nil)
	}

	// Configure client with stderr callback
	mode := types.PermissionModeBypassPermissions
	stats := NewSessionStats(0, 0)

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(types.ModelSonnet),
		Stderr:         stderrCallback,
		PermissionMode: &mode,
	})
	defer c.Close()

	logEvent("STDERR", "Stderr callback configured for diagnostic capture",
		map[string]interface{}{
			"categories": []string{"error", "warning", "debug", "info"},
		})

	if err := c.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}
	logEvent("CONNECTED", "Client connected with stderr callback", nil)

	msgChan := c.ReceiveMessages(ctx)

	logSeparator("STDERR TEST")
	logEvent("QUERY", "Simple query to capture stderr output", nil)

	if err := c.Query(ctx, "What is 2 + 2?"); err != nil {
		logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
	} else {
		for msg := range msgChan {
			displayMessage(msg, stats)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}
	}

	// Show captured stderr summary
	stderrMu.Lock()
	lineCount := len(stderrLines)
	stderrMu.Unlock()

	logEvent("STDERR_SUMMARY", fmt.Sprintf("Captured %d stderr lines", lineCount), nil)

	printSessionSummary(stats)
}

// ============================================================================
// System Prompt Example
// ============================================================================

// runSystemPromptExample demonstrates system prompt configurations
func runSystemPromptExample(ctx context.Context) {
	logSeparator("EXAMPLE 9: SYSTEM PROMPT")
	fmt.Println("Features: String prompt, SystemPromptPreset, preset with append")

	// Example 1: String system prompt
	logSeparator("SYSTEM_PROMPT: STRING")
	customPrompt := "You are a helpful coding assistant specialized in Go. Always use idiomatic Go patterns and include proper error handling."

	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		SystemPrompt: customPrompt,
	})
	defer c1.Close()

	logEvent("SYSTEM_PROMPT", "Client with string system prompt",
		map[string]interface{}{
			"type":    "string",
			"preview": customPrompt[:50] + "...",
		})

	// Example 2: SystemPromptPreset (Claude Code default)
	logSeparator("SYSTEM_PROMPT: PRESET")
	preset := types.SystemPromptPreset{
		Type:   "preset",
		Preset: "claude_code",
	}

	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		SystemPrompt: preset,
	})
	defer c2.Close()

	logEvent("SYSTEM_PROMPT", "Client with SystemPromptPreset",
		map[string]interface{}{
			"type":   "preset",
			"preset": "claude_code",
		})

	// Example 3: Preset with append (extend default behavior)
	logSeparator("SYSTEM_PROMPT: PRESET + APPEND")
	appendText := `
ADDITIONAL INSTRUCTIONS:
- Always explain your reasoning before providing code
- Prefer standard library packages over third-party dependencies
- Include comments for complex logic`

	presetWithAppend := types.SystemPromptPreset{
		Type:   "preset",
		Preset: "claude_code",
		Append: &appendText,
	}

	c3 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		SystemPrompt: presetWithAppend,
	})
	defer c3.Close()

	logEvent("SYSTEM_PROMPT", "Client with preset + custom append",
		map[string]interface{}{
			"type":       "preset+append",
			"preset":     "claude_code",
			"append_len": len(appendText),
		})

	// Example 4: Domain-specific prompt (security-focused)
	logSeparator("SYSTEM_PROMPT: DOMAIN-SPECIFIC")
	securityPrompt := types.SystemPromptPreset{
		Type:   "preset",
		Preset: "claude_code",
		Append: strPtr(`
SECURITY GUIDELINES:
- Never suggest code with known vulnerabilities
- Always validate user input
- Use parameterized queries for database operations
- Flag potential security issues proactively`),
	}

	c4 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		SystemPrompt: securityPrompt,
	})
	defer c4.Close()

	logEvent("SYSTEM_PROMPT", "Client with security-focused prompt",
		map[string]interface{}{
			"domain":     "security",
			"has_append": true,
		})

	logEvent("SYSTEM_PROMPT", "System prompt configurations demonstrated", nil)
}

// ============================================================================
// Partial Messages Example
// ============================================================================

// runPartialMessagesExample demonstrates partial/streaming messages for real-time display
func runPartialMessagesExample(ctx context.Context) {
	logSeparator("EXAMPLE 10: PARTIAL MESSAGES")
	fmt.Println("Features: IncludePartialMessages, StreamEvent handling, real-time text accumulation")

	// Enable partial messages for streaming
	mode := types.PermissionModeBypassPermissions
	stats := NewSessionStats(0, 0)

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:                  types.String(types.ModelSonnet),
		IncludePartialMessages: true, // Enable real-time streaming
		PermissionMode:         &mode,
	})
	defer c.Close()

	logEvent("PARTIAL", "Client configured with IncludePartialMessages=true",
		map[string]interface{}{
			"streaming_enabled": true,
		})

	if err := c.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}
	logEvent("CONNECTED", "Client connected with streaming enabled", nil)

	msgChan := c.ReceiveMessages(ctx)

	// Track streaming text
	streamTexts := make(map[string]string)
	var streamMu sync.Mutex

	logSeparator("PARTIAL TEST")
	logEvent("QUERY", "Query with real-time streaming", nil)

	if err := c.Query(ctx, "Write a short poem about coding in Go"); err != nil {
		logEvent("ERROR", fmt.Sprintf("Query failed: %v", err), nil)
	} else {
		for msg := range msgChan {
			switch m := msg.(type) {
			case *types.StreamEvent:
				// Handle stream events for real-time display
				streamMu.Lock()
				// m.Event is already map[string]interface{}
				if delta, ok := m.Event["delta"].(map[string]interface{}); ok {
					if text, ok := delta["text"].(string); ok {
						streamTexts[m.UUID] += text
						// Show partial progress
						if len(streamTexts[m.UUID])%20 == 0 || len(streamTexts[m.UUID]) < 20 {
							logEvent("STREAM_PARTIAL", fmt.Sprintf("UUID: %s | Accumulated: %d chars",
								m.UUID[:12], len(streamTexts[m.UUID])), nil)
						}
					}
				}
				streamMu.Unlock()
			default:
				displayMessage(msg, stats)
			}

			if _, isResult := msg.(*types.ResultMessage); isResult {
				break
			}
		}
	}

	// Show final accumulated text
	streamMu.Lock()
	for uuid, text := range streamTexts {
		if len(text) > 50 {
			text = text[:50] + "..."
		}
		logEvent("STREAM_FINAL", fmt.Sprintf("UUID: %s | Final text: %s", uuid[:12], text), nil)
	}
	streamMu.Unlock()

	logEvent("PARTIAL", "Partial messages streaming demonstration complete", nil)

	printSessionSummary(stats)
}

// ============================================================================
// Goroutines Example
// ============================================================================

// runGoroutinesExample demonstrates concurrent patterns with goroutines
func runGoroutinesExample(ctx context.Context) {
	logSeparator("EXAMPLE 11: GOROUTINES CONCURRENT")
	fmt.Println("Features: Background processing, concurrent queries, context cancellation")

	// Example 1: Background message processing
	logSeparator("GOROUTINES: BACKGROUND")
	logEvent("GOROUTINES", "Demonstrating background message processing", nil)

	// Use WaitGroup for synchronization
	var wg sync.WaitGroup
	results := make(chan string, 10)

	// Background goroutine to process results
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range results {
			logEvent("BACKGROUND", fmt.Sprintf("Processed: %s", result), nil)
		}
	}()

	// Create client for background processing
	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c1.Close()

	if err := c1.Connect(ctx); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
	} else {
		msgChan := c1.ReceiveMessages(ctx)

		queries := []string{"What is 1 + 1?", "What is 2 + 2?", "What is 3 + 3?"}
		for i, query := range queries {
			logEvent("GOROUTINES_QUERY", fmt.Sprintf("Query %d: %s", i+1, query), nil)

			if err := c1.Query(ctx, query); err != nil {
				results <- fmt.Sprintf("Error: %v", err)
				continue
			}

			for msg := range msgChan {
				if am, ok := msg.(*types.AssistantMessage); ok {
					for _, block := range am.Content {
						if tb, ok := block.(types.TextBlock); ok {
							text := tb.Text
							if len(text) > 40 {
								text = text[:40] + "..."
							}
							results <- fmt.Sprintf("Q%d: %s", i+1, text)
							logEvent("GOROUTINES_RESULT", text, nil)
						}
					}
				}
				if _, isResult := msg.(*types.ResultMessage); isResult {
					break
				}
			}
		}
	}

	close(results)
	wg.Wait()
	logEvent("GOROUTINES", "Background processing complete", nil)

	// Example 2: Context cancellation demonstration
	logSeparator("GOROUTINES: CANCELLATION")
	logEvent("GOROUTINES", "Demonstrating context cancellation", nil)

	// Create cancellable context
	ctxCancel, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c2.Close()

	if err := c2.Connect(ctxCancel); err != nil {
		logEvent("ERROR", fmt.Sprintf("Connection failed: %v", err), nil)
		return
	}

	logEvent("GOROUTINES", "Context with 10s timeout created",
		map[string]interface{}{
			"timeout_seconds": 10,
		})

	logEvent("GOROUTINES", "Goroutines concurrent patterns demonstrated", nil)
}

// ============================================================================
// Tools Configuration Example
// ============================================================================

// runToolsConfigExample demonstrates tools configuration (Allowed/Disallowed)
func runToolsConfigExample(ctx context.Context) {
	logSeparator("EXAMPLE 12: TOOLS CONFIGURATION")
	fmt.Println("Features: AllowedTools, DisallowedTools, ToolsPreset")

	// Example 1: AllowedTools - restrict to specific tools
	logSeparator("TOOLS: ALLOWED ONLY")
	logEvent("TOOLS", "AllowedTools - restrict to specific tools", nil)

	allowedTools := []string{"Read", "Glob", "Grep"}

	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		AllowedTools: allowedTools,
	})
	defer c1.Close()

	logEvent("TOOLS", "Client with AllowedTools restriction",
		map[string]interface{}{
			"allowed":     allowedTools,
			"description": "Only these tools are available",
		})

	// Example 2: DisallowedTools - block specific tools
	logSeparator("TOOLS: DISALLOWED")
	logEvent("TOOLS", "DisallowedTools - block specific tools", nil)

	disallowedTools := []string{"Write", "Edit", "Bash"}

	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:           types.String(types.ModelSonnet),
		DisallowedTools: disallowedTools,
	})
	defer c2.Close()

	logEvent("TOOLS", "Client with DisallowedTools restriction",
		map[string]interface{}{
			"disallowed":  disallowedTools,
			"description": "These tools are blocked, others available",
		})

	// Example 3: ToolsPreset - use Claude Code preset
	logSeparator("TOOLS: PRESET")
	logEvent("TOOLS", "ToolsPreset - Claude Code preset toolset", nil)

	toolsPreset := types.ToolsPreset{
		Type:   "preset",
		Preset: "claude_code",
	}

	c3 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Tools: toolsPreset,
	})
	defer c3.Close()

	logEvent("TOOLS", "Client with ToolsPreset",
		map[string]interface{}{
			"type":   "preset",
			"preset": "claude_code",
			"tools":  "standard Claude Code tools",
		})

	// Example 4: Combined - preset + disallowed (read-only mode)
	logSeparator("TOOLS: READ-ONLY")
	logEvent("TOOLS", "Combined: ToolsPreset + DisallowedTools (read-only)", nil)

	c4 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Tools: types.ToolsPreset{
			Type:   "preset",
			Preset: "claude_code",
		},
		DisallowedTools: []string{"Write", "Edit", "Bash"},
	})
	defer c4.Close()

	logEvent("TOOLS", "Client in read-only mode",
		map[string]interface{}{
			"mode":        "read-only",
			"disallowed":  []string{"Write", "Edit", "Bash"},
			"description": "Can read/search but cannot modify",
		})

	logEvent("TOOLS", "Tools configuration demonstrated", nil)
}

// ============================================================================
// Sandbox Example
// ============================================================================

// runSandboxExample demonstrates sandbox settings for safe operations
func runSandboxExample(ctx context.Context) {
	logSeparator("EXAMPLE 13: SANDBOX SETTINGS")
	fmt.Println("Features: Sandbox enabled, AutoAllowBashIfSandboxed, safe command execution")

	// Example 1: Sandbox enabled with auto-allow Bash
	logSeparator("SANDBOX: ENABLED")
	sandboxEnabled := true
	autoAllowBash := true

	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Sandbox: &types.SandboxSettings{
			Enabled:                  &sandboxEnabled,
			AutoAllowBashIfSandboxed: &autoAllowBash,
		},
	})
	defer c1.Close()

	logEvent("SANDBOX", "Client with sandbox enabled",
		map[string]interface{}{
			"enabled":         sandboxEnabled,
			"auto_allow_bash": autoAllowBash,
			"description":     "Bash commands allowed automatically in sandbox",
		})

	// Example 2: Sandbox disabled
	logSeparator("SANDBOX: DISABLED")
	sandboxDisabled := false

	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Sandbox: &types.SandboxSettings{
			Enabled: &sandboxDisabled,
		},
	})
	defer c2.Close()

	logEvent("SANDBOX", "Client with sandbox disabled",
		map[string]interface{}{
			"enabled":     sandboxDisabled,
			"description": "No sandbox restrictions",
		})

	// Example 3: Sandbox with agents
	logSeparator("SANDBOX: WITH AGENTS")
	workspaceAgent := types.AgentDefinition{
		Description: "A workspace manager with sandboxed file access",
		Prompt:      "You are a workspace manager. Work safely within the sandboxed environment.",
		Tools:       []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
		Model:       types.String("sonnet"),
	}

	c3 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Sandbox: &types.SandboxSettings{
			Enabled:                  &sandboxEnabled,
			AutoAllowBashIfSandboxed: &autoAllowBash,
		},
		Agents: map[string]types.AgentDefinition{
			"workspace-manager": workspaceAgent,
		},
	})
	defer c3.Close()

	logEvent("SANDBOX", "Client with sandbox + agent",
		map[string]interface{}{
			"sandbox_enabled": sandboxEnabled,
			"agent":           "workspace-manager",
			"agent_tools":     []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
		})

	logEvent("SANDBOX", "Sandbox settings demonstrated", nil)
}

// ============================================================================
// Plugins Example
// ============================================================================

// runPluginsExample demonstrates plugin configuration
func runPluginsExample(ctx context.Context) {
	logSeparator("EXAMPLE 14: PLUGINS")
	fmt.Println("Features: SdkPluginConfig, local plugins, custom commands")

	// Example 1: Local plugin configuration
	logSeparator("PLUGINS: LOCAL")
	logEvent("PLUGINS", "Local plugin loading", nil)

	// Configure client with local plugin
	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Plugins: []types.SdkPluginConfig{
			{
				Type: "local",
				Path: "/path/to/demo-plugin", // In real usage, use actual plugin path
			},
		},
	})
	defer c1.Close()

	logEvent("PLUGINS", "Client with local plugin",
		map[string]interface{}{
			"type":     "local",
			"path":     "/path/to/demo-plugin",
			"provides": []string{"custom commands", "agents", "MCP servers"},
		})

	// Example 2: Multiple plugins
	logSeparator("PLUGINS: MULTIPLE")
	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Plugins: []types.SdkPluginConfig{
			{Type: "local", Path: "/path/to/plugin-1"},
			{Type: "local", Path: "/path/to/plugin-2"},
		},
		AllowedTools: []string{"Read", "Write", "Agent"},
	})
	defer c2.Close()

	logEvent("PLUGINS", "Client with multiple plugins",
		map[string]interface{}{
			"plugins":       []string{"plugin-1", "plugin-2"},
			"allowed_tools": []string{"Read", "Write", "Agent"},
		})

	// Example 3: Plugin structure reference
	logSeparator("PLUGINS: STRUCTURE")
	logEvent("PLUGINS", "Plugin directory structure",
		map[string]interface{}{
			"structure": map[string]string{
				".claude-plugin/plugin.json": "required - plugin metadata",
				"commands/*.md":              "custom slash commands",
				"agents/*.md":                "custom agent definitions",
				"mcp-servers/*.json":         "MCP server configs",
				"skills/*.md":                "custom skills",
				"hooks/*":                    "hook scripts",
			},
		})

	logEvent("PLUGINS", "Plugins demonstrated", nil)
}

// ============================================================================
// Task Messages Example
// ============================================================================

// runTaskMessagesExample demonstrates task message handling
func runTaskMessagesExample(ctx context.Context) {
	logSeparator("EXAMPLE 15: TASK MESSAGES")
	fmt.Println("Features: TaskStartedMessage, TaskProgressMessage, TaskNotificationMessage")

	// Task messages track subagent/task execution
	logEvent("TASK_MESSAGES", "Task messages track subagent execution",
		map[string]interface{}{
			"types": []string{"TaskStartedMessage", "TaskProgressMessage", "TaskNotificationMessage"},
		})

	// Example: TaskStartedMessage structure
	logSeparator("TASK_MESSAGES: STARTED")
	logEvent("TASK_STARTED", "TaskStartedMessage fields",
		map[string]interface{}{
			"task_id":     "unique task identifier",
			"description": "task description",
			"task_type":   "optional task type",
			"agent_id":    "optional agent ID",
		})

	// Example: TaskProgressMessage structure
	logSeparator("TASK_MESSAGES: PROGRESS")
	logEvent("TASK_PROGRESS", "TaskProgressMessage fields",
		map[string]interface{}{
			"task_id":        "task identifier",
			"usage":          "usage statistics",
			"total_tokens":   "token count",
			"tool_uses":      "tool use count",
			"duration_ms":    "execution duration",
			"last_tool_name": "last tool used",
		})

	// Example: TaskNotificationMessage structure
	logSeparator("TASK_MESSAGES: NOTIFICATION")
	logEvent("TASK_NOTIFICATION", "TaskNotificationMessage fields",
		map[string]interface{}{
			"task_id": "task identifier",
			"status":  "completed/failed/stopped",
			"summary": "task summary",
			"usage":   "final usage stats",
		})

	// Status values explanation
	logEvent("TASK_STATUS", "Task notification status values",
		map[string]interface{}{
			"completed": "task finished successfully",
			"failed":    "task encountered error",
			"stopped":   "task was stopped externally",
		})

	logEvent("TASK_MESSAGES", "Task messages demonstrated", nil)
}

// ============================================================================
// MCP Control Example
// ============================================================================

// runMcpControlExample demonstrates MCP server control operations
func runMcpControlExample(ctx context.Context) {
	logSeparator("EXAMPLE 16: MCP CONTROL")
	fmt.Println("Features: GetMCPStatus, ToggleMCPServer, ReconnectMCPServer, StopTask")

	// Create MCP server for demo
	mcpServer := createMcpServer()
	mode := types.PermissionModeBypassPermissions

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"tools": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: mcpServer,
			},
		},
		AllowedTools:   []string{"mcp__assistant-tools__calc"},
		PermissionMode: &mode,
	})
	defer c.Close()

	logEvent("MCP_CONTROL", "Client with MCP server for control demo",
		map[string]interface{}{
			"server": "assistant-tools",
			"tools":  []string{"calc", "string", "time"},
		})

	// MCP control methods explanation
	logSeparator("MCP_CONTROL: METHODS")
	logEvent("MCP_CONTROL", "Available MCP control methods",
		map[string]interface{}{
			"methods": []string{
				"GetMCPStatus(ctx) - get server status",
				"ToggleMCPServer(ctx, name, enabled) - enable/disable server",
				"ReconnectMCPServer(ctx, name) - reconnect disconnected server",
				"StopTask(ctx, taskID) - stop running task",
			},
		})

	// Example usage patterns (without actual execution)
	logSeparator("MCP_CONTROL: USAGE")
	logEvent("MCP_CONTROL", "GetMCPStatus usage pattern", nil)
	fmt.Println("  status, err := client.GetMCPStatus(ctx)")
	fmt.Println("  // status contains 'mcpServers' with server info")

	logEvent("MCP_CONTROL", "ToggleMCPServer usage pattern", nil)
	fmt.Println("  err := client.ToggleMCPServer(ctx, \"demo\", false) // disable")
	fmt.Println("  err := client.ToggleMCPServer(ctx, \"demo\", true)  // enable")

	logEvent("MCP_CONTROL", "ReconnectMCPServer usage pattern", nil)
	fmt.Println("  err := client.ReconnectMCPServer(ctx, \"demo\")")

	logEvent("MCP_CONTROL", "StopTask usage pattern", nil)
	fmt.Println("  err := client.StopTask(ctx, \"task-123\")")

	logEvent("MCP_CONTROL", "MCP control demonstrated", nil)
}

// ============================================================================
// Interactive Streaming Example
// ============================================================================

// runInteractiveExample demonstrates interactive streaming patterns
func runInteractiveExample(ctx context.Context) {
	logSeparator("EXAMPLE 17: INTERACTIVE STREAMING")
	fmt.Println("Features: Interrupt, timeout, REPL mode patterns")

	// Example 1: Interrupt capability
	logSeparator("INTERACTIVE: INTERRUPT")
	logEvent("INTERACTIVE", "Interrupt capability for stopping queries",
		map[string]interface{}{
			"method":      "client.Interrupt(ctx)",
			"requirement": "must consume messages actively",
		})

	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
	})
	defer c1.Close()

	logEvent("INTERACTIVE", "Interrupt pattern",
		map[string]interface{}{
			"pattern": "send query -> consume messages -> call Interrupt() -> send new query",
		})

	// Example 2: Timeout handling
	logSeparator("INTERACTIVE: TIMEOUT")
	logEvent("INTERACTIVE", "Timeout pattern with context",
		map[string]interface{}{
			"pattern": "context.WithTimeout(ctx, 5*time.Second)",
			"usage":   "limit query execution time",
		})

	// Example 3: REPL mode pattern
	logSeparator("INTERACTIVE: REPL")
	logEvent("INTERACTIVE", "REPL mode for continuous interaction",
		map[string]interface{}{
			"features": []string{"user input loop", "message consumption", "exit handling"},
		})

	// Pattern explanation
	logEvent("REPL_PATTERN", "REPL implementation pattern",
		map[string]interface{}{
			"steps": []string{
				"1. Create client and connect",
				"2. Get message channel with ReceiveMessages(ctx)",
				"3. Loop: read user input, Query(ctx), consume messages",
				"4. Handle 'quit'/'exit' to break loop",
			},
		})

	logEvent("INTERACTIVE", "Interactive streaming demonstrated", nil)
}

// ============================================================================
// Skills Configuration Example
// ============================================================================

// runSkillsExample demonstrates Skills configuration for agents
func runSkillsExample(ctx context.Context) {
	logSeparator("EXAMPLE 18: SKILLS CONFIGURATION")
	fmt.Println("Features: AgentDefinition.Skills, skill loading")

	// Skills are defined in AgentDefinition
	logSeparator("SKILLS: DEFINITION")
	agentWithSkills := types.AgentDefinition{
		Description: "An agent with specific skills",
		Prompt:      "You are a specialized agent with coding skills.",
		Tools:       []string{"Read", "Write", "Edit"},
		Skills:      []string{"coding", "testing", "debugging"}, // Skills field
		Model:       types.String("sonnet"),
	}

	logEvent("SKILLS", "AgentDefinition with Skills",
		map[string]interface{}{
			"skills": agentWithSkills.Skills,
			"tools":  agentWithSkills.Tools,
			"model":  "sonnet",
		})

	// Configure client with skilled agent
	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Agents: map[string]types.AgentDefinition{
			"skilled-agent": agentWithSkills,
		},
	})
	defer c.Close()

	logEvent("SKILLS", "Client with skilled agent",
		map[string]interface{}{
			"agent":  "skilled-agent",
			"skills": []string{"coding", "testing", "debugging"},
		})

	// Skills vs Tools explanation
	logSeparator("SKILLS: VS TOOLS")
	logEvent("SKILLS", "Skills vs Tools distinction",
		map[string]interface{}{
			"skills": "predefined capabilities/behaviors loaded from plugin",
			"tools":  "explicit tool names agent can use",
		})

	logEvent("SKILLS", "Skills configuration demonstrated", nil)
}

// ============================================================================
// Thinking Configuration Example
// ============================================================================

// runThinkingExample demonstrates ThinkingConfig and Effort settings
func runThinkingExample(ctx context.Context) {
	logSeparator("EXAMPLE 19: THINKING CONFIGURATION")
	fmt.Println("Features: ThinkingConfig, MaxThinkingTokens, Effort")

	// Example 1: ThinkingConfig
	logSeparator("THINKING: CONFIG")
	thinkingConfig := types.ThinkingConfigEnabled{
		Type:         "enabled",
		BudgetTokens: 10000,
	}

	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:    types.String(types.ModelSonnet),
		Thinking: thinkingConfig,
	})
	defer c1.Close()

	logEvent("THINKING", "Client with ThinkingConfig",
		map[string]interface{}{
			"type":          "enabled",
			"budget_tokens": 10000,
		})

	// Example 2: MaxThinkingTokens (deprecated)
	logSeparator("THINKING: MAX TOKENS")
	maxTokens := 5000
	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:             types.String(types.ModelSonnet),
		MaxThinkingTokens: &maxTokens,
	})
	defer c2.Close()

	logEvent("THINKING", "Client with MaxThinkingTokens (deprecated)",
		map[string]interface{}{
			"max_thinking_tokens": maxTokens,
			"note":                "use ThinkingConfig instead",
		})

	// Example 3: Effort level
	logSeparator("THINKING: EFFORT")
	effort := "high"
	c3 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:  types.String(types.ModelSonnet),
		Effort: &effort,
	})
	defer c3.Close()

	logEvent("THINKING", "Client with Effort level",
		map[string]interface{}{
			"effort":  "high",
			"options": []string{"low", "medium", "high", "max"},
		})

	logEvent("THINKING", "Thinking configuration demonstrated", nil)
}

// ============================================================================
// Output Format Example
// ============================================================================

// runOutputFormatExample demonstrates OutputFormat for structured outputs
func runOutputFormatExample(ctx context.Context) {
	logSeparator("EXAMPLE 20: OUTPUT FORMAT")
	fmt.Println("Features: JSON schema output, structured response format")

	// Example: JSON schema output format
	logSeparator("OUTPUT_FORMAT: JSON SCHEMA")
	outputFormat := map[string]interface{}{
		"type": "json_schema",
		"schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"answer":     map[string]string{"type": "string"},
				"confidence": map[string]string{"type": "number"},
			},
			"required": []string{"answer", "confidence"},
		},
	}

	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		OutputFormat: outputFormat,
	})
	defer c.Close()

	logEvent("OUTPUT_FORMAT", "Client with JSON schema output format",
		map[string]interface{}{
			"type":              "json_schema",
			"schema_properties": []string{"answer", "confidence"},
		})

	logEvent("OUTPUT_FORMAT", "Output format configuration demonstrated", nil)
}

// ============================================================================
// Advanced Options Example
// ============================================================================

// runAdvancedOptionsExample demonstrates advanced client options
func runAdvancedOptionsExample(ctx context.Context) {
	logSeparator("EXAMPLE 21: ADVANCED OPTIONS")
	fmt.Println("Features: ForkSession, ContinueConversation, Resume, Betas, FileCheckpointing")

	// Example 1: ForkSession
	logSeparator("ADVANCED: FORK SESSION")
	c1 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:       types.String(types.ModelSonnet),
		ForkSession: true,
	})
	defer c1.Close()

	logEvent("ADVANCED", "ForkSession enabled",
		map[string]interface{}{
			"fork_session": true,
			"description":  "create independent session copy",
		})

	// Example 2: ContinueConversation
	logSeparator("ADVANCED: CONTINUE CONVERSATION")
	c2 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:                types.String(types.ModelSonnet),
		ContinueConversation: true,
	})
	defer c2.Close()

	logEvent("ADVANCED", "ContinueConversation enabled",
		map[string]interface{}{
			"continue_conversation": true,
			"description":           "continue from previous session",
		})

	// Example 3: Resume session
	logSeparator("ADVANCED: RESUME")
	sessionID := "previous-session-id"
	c3 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:  types.String(types.ModelSonnet),
		Resume: &sessionID,
	})
	defer c3.Close()

	logEvent("ADVANCED", "Resume session",
		map[string]interface{}{
			"resume":      sessionID,
			"description": "resume specific session",
		})

	// Example 4: Betas
	logSeparator("ADVANCED: BETAS")
	c4 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Betas: []types.SdkBeta{"token-efficient-tools", "max-tokens"},
	})
	defer c4.Close()

	logEvent("ADVANCED", "Beta features enabled",
		map[string]interface{}{
			"betas": []string{"token-efficient-tools", "max-tokens"},
		})

	// Example 5: FileCheckpointing
	logSeparator("ADVANCED: FILE CHECKPOINTING")
	c5 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:                   types.String(types.ModelSonnet),
		EnableFileCheckpointing: true,
	})
	defer c5.Close()

	logEvent("ADVANCED", "File checkpointing enabled",
		map[string]interface{}{
			"enable_file_checkpointing": true,
			"description":               "track file changes during session",
		})

	// Example 6: ExcludeDynamicSections
	logSeparator("ADVANCED: EXCLUDE DYNAMIC SECTIONS")
	presetWithExclude := types.SystemPromptPreset{
		Type:                   "preset",
		Preset:                 "claude_code",
		ExcludeDynamicSections: boolPtr(true),
	}

	c6 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(types.ModelSonnet),
		SystemPrompt: presetWithExclude,
	})
	defer c6.Close()

	logEvent("ADVANCED", "ExcludeDynamicSections enabled",
		map[string]interface{}{
			"exclude_dynamic_sections": true,
			"description":              "strip per-user dynamic sections from system prompt",
		})

	// Example 7: FallbackModel
	logSeparator("ADVANCED: FALLBACK MODEL")
	c7 := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:         types.String(types.ModelSonnet),
		FallbackModel: types.String(types.ModelHaiku),
	})
	defer c7.Close()

	logEvent("ADVANCED", "Fallback model configured",
		map[string]interface{}{
			"primary_model":  "sonnet",
			"fallback_model": "haiku",
		})

	logEvent("ADVANCED", "Advanced options demonstrated", nil)
}

// ============================================================================
// Additional Hook Types Example
// ============================================================================

// runAdvancedHooksExample demonstrates additional hook types
func runAdvancedHooksExample(ctx context.Context) {
	logSeparator("EXAMPLE 22: ADVANCED HOOKS")
	fmt.Println("Features: Additional hook types beyond PreToolUse/PostToolUse")

	// Available hook events
	logEvent("HOOKS_ADVANCED", "All available HookEvent types",
		map[string]interface{}{
			"events": []string{
				"HookEventPreToolUse",
				"HookEventPostToolUse",
				"HookEventPostToolUseFailure",
				"HookEventUserPromptSubmit",
				"HookEventNotification",
				"HookEventSubagentStart",
				"HookEventSubagentStop",
			},
		})

	// Hook input types explanation
	logSeparator("HOOKS_ADVANCED: INPUT TYPES")
	logEvent("HOOK_INPUT", "PreToolUseHookInput fields",
		map[string]interface{}{
			"fields": []string{"ToolName", "ToolInput", "ToolUseID", "AgentID", "AgentType"},
		})

	logEvent("HOOK_INPUT", "PostToolUseHookInput fields",
		map[string]interface{}{
			"fields": []string{"ToolName", "ToolInput", "ToolOutput", "ToolUseID", "AgentID", "AgentType"},
		})

	logEvent("HOOK_INPUT", "PostToolUseFailureHookInput fields",
		map[string]interface{}{
			"fields": []string{"ToolName", "ToolInput", "ToolUseID", "IsInterrupt", "AgentID", "AgentType"},
		})

	logEvent("HOOK_INPUT", "UserPromptSubmitHookInput fields",
		map[string]interface{}{
			"fields": []string{"Prompt", "CustomInstructions"},
		})

	logEvent("HOOK_INPUT", "NotificationHookInput fields",
		map[string]interface{}{
			"fields": []string{"Message", "Title", "NotificationType"},
		})

	logEvent("HOOK_INPUT", "SubagentStartHookInput fields",
		map[string]interface{}{
			"fields": []string{"AgentID", "AgentName", "AgentDescription", "Prompt"},
		})

	logEvent("HOOKS_ADVANCED", "Advanced hooks demonstrated", nil)
}

// ============================================================================
// Helper Functions
// ============================================================================

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func printUsage() {
	logSeparator("USAGE")
	fmt.Println("Usage: go run main.go <example_name>")
	fmt.Println()
	fmt.Println("Available examples:")
	fmt.Println("  basic          - Basic MCP tool usage")
	fmt.Println("  multi_turn     - Multi-turn conversation with cost tracking")
	fmt.Println("  hooks          - Hook interception demonstration")
	fmt.Println("  agents         - Agents/subagents system")
	fmt.Println("  middleware     - Transport middleware (logging, metrics)")
	fmt.Println("  settings       - Setting sources configuration")
	fmt.Println("  stderr         - Stderr callback handling")
	fmt.Println("  system         - System prompt configurations")
	fmt.Println("  partial        - Partial messages streaming")
	fmt.Println("  goroutines     - Concurrent patterns with goroutines")
	fmt.Println("  tools          - Tools configuration (Allowed/Disallowed)")
	fmt.Println("  sandbox        - Sandbox settings for safe operations")
	fmt.Println("  plugins        - Plugin configuration")
	fmt.Println("  task_messages  - Task message handling")
	fmt.Println("  mcp_control    - MCP server control operations")
	fmt.Println("  interactive    - Interactive streaming patterns")
	fmt.Println("  skills         - Skills configuration for agents")
	fmt.Println("  thinking       - ThinkingConfig and Effort settings")
	fmt.Println("  output_format  - JSON schema output format")
	fmt.Println("  advanced       - Advanced options (ForkSession, Resume, Betas)")
	fmt.Println("  advanced_hooks - Additional hook types")
	fmt.Println("  complete       - Complete feature integration")
	fmt.Println("  all            - Run all examples")
	fmt.Println()
	fmt.Println("Prerequisites:")
	fmt.Println("  1. Install Claude CLI: npm install -g @anthropic-ai/claude-code")
	fmt.Println("  2. Login authentication: claude login")
	logSeparator("")
}

// ============================================================================
// Main Function
// ============================================================================

func main() {
	examples := map[string]func(context.Context){
		"basic":          runBasicExample,
		"multi_turn":     runMultiTurnExample,
		"hooks":          runHooksExample,
		"agents":         runAgentsExample,
		"middleware":     runMiddlewareExample,
		"settings":       runSettingsExample,
		"stderr":         runStderrExample,
		"system":         runSystemPromptExample,
		"partial":        runPartialMessagesExample,
		"goroutines":     runGoroutinesExample,
		"tools":          runToolsConfigExample,
		"sandbox":        runSandboxExample,
		"plugins":        runPluginsExample,
		"task_messages":  runTaskMessagesExample,
		"mcp_control":    runMcpControlExample,
		"interactive":    runInteractiveExample,
		"skills":         runSkillsExample,
		"thinking":       runThinkingExample,
		"output_format":  runOutputFormatExample,
		"advanced":       runAdvancedOptionsExample,
		"advanced_hooks": runAdvancedHooksExample,
		"complete":       runCompleteExample,
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	exampleName := os.Args[1]

	// Create context with signal handling
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	logSeparator("CLAUDE AGENT SDK - COMPLETE FEATURE DEMO")
	fmt.Println("Program started")
	fmt.Println("Features: MCP, Streaming, Hooks, Permission, Cost, Agents, Middleware,")
	fmt.Println("          Settings, Stderr, SystemPrompt, Partial, Goroutines, Tools,")
	fmt.Println("          Sandbox, Plugins, TaskMessages, MCPControl, Interactive,")
	fmt.Println("          Skills, Thinking, OutputFormat, Advanced, Hooks")

	if exampleName == "all" {
		// Run all examples
		for _, fn := range examples {
			fn(ctx)
		}
	} else if fn, ok := examples[exampleName]; ok {
		// Run specific example
		fn(ctx)
	} else {
		logEvent("ERROR", fmt.Sprintf("Unknown example: '%s'", exampleName),
			map[string]interface{}{
				"available": []string{"basic", "multi_turn", "hooks", "agents", "middleware",
					"settings", "stderr", "system", "partial", "goroutines", "tools",
					"sandbox", "plugins", "task_messages", "mcp_control", "interactive",
					"skills", "thinking", "output_format", "advanced", "advanced_hooks",
					"complete", "all"},
			})
		printUsage()
		os.Exit(1)
	}

	logSeparator("PROGRAM END")
	fmt.Println("Program completed")
}
