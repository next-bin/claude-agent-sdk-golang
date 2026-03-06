// Example mcp_calculator demonstrates how to create an in-process MCP server with
// calculator tools using the Claude Agent SDK for Go.
//
// Unlike external MCP servers that require separate processes, this server
// runs directly within your Go application, providing better performance
// and simpler deployment.
//
// This example shows:
// 1. Creating tools using the sdkmcp convenience package
// 2. Creating an SDK MCP server with multiple tools
// 3. Handling errors (divide by zero, negative sqrt)
// 4. Running example calculations with the tools
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"log"
	"math"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/examples/internal"
	"github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Calculator Tool Definitions
// ============================================================================

// createAddTool creates the add tool.
func createAddTool() *sdkmcp.SdkMcpTool {
	return sdkmcp.Tool("add", "Add two numbers together", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		result := a + b
		return sdkmcp.TextResult(fmt.Sprintf("%.0f + %.0f = %.0f", a, b, result)), nil
	})
}

// createSubtractTool creates the subtract tool.
func createSubtractTool() *sdkmcp.SdkMcpTool {
	return sdkmcp.Tool("subtract", "Subtract one number from another", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		result := a - b
		return sdkmcp.TextResult(fmt.Sprintf("%.0f - %.0f = %.0f", a, b, result)), nil
	})
}

// createMultiplyTool creates the multiply tool.
func createMultiplyTool() *sdkmcp.SdkMcpTool {
	return sdkmcp.Tool("multiply", "Multiply two numbers", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		result := a * b
		return sdkmcp.TextResult(fmt.Sprintf("%.0f × %.0f = %.0f", a, b, result)), nil
	})
}

// createDivideTool creates the divide tool with error handling for division by zero.
func createDivideTool() *sdkmcp.SdkMcpTool {
	return sdkmcp.Tool("divide", "Divide one number by another", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		if b == 0 {
			return sdkmcp.TextResultWithError("Error: Division by zero is not allowed"), nil
		}
		result := a / b
		return sdkmcp.TextResult(fmt.Sprintf("%.0f ÷ %.0f = %g", a, b, result)), nil
	})
}

// createSqrtTool creates the square root tool with error handling for negative numbers.
func createSqrtTool() *sdkmcp.SdkMcpTool {
	return sdkmcp.Tool("sqrt", "Calculate square root", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"n": map[string]interface{}{"type": "number"},
		},
		"required": []string{"n"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		n, _ := args["n"].(float64)
		if n < 0 {
			return sdkmcp.TextResultWithError(fmt.Sprintf("Error: Cannot calculate square root of negative number %.0f", n)), nil
		}
		result := math.Sqrt(n)
		return sdkmcp.TextResult(fmt.Sprintf("√%.0f = %g", n, result)), nil
	})
}

// createPowerTool creates the power tool.
func createPowerTool() *sdkmcp.SdkMcpTool {
	return sdkmcp.Tool("power", "Raise a number to a power", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"base":     map[string]interface{}{"type": "number"},
			"exponent": map[string]interface{}{"type": "number"},
		},
		"required": []string{"base", "exponent"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		base, _ := args["base"].(float64)
		exponent, _ := args["exponent"].(float64)
		result := math.Pow(base, exponent)
		return sdkmcp.TextResult(fmt.Sprintf("%.0f^%.0f = %g", base, exponent, result)), nil
	})
}

// ============================================================================
// Message Display Helper
// ============================================================================

// displayMessage prints message content in a clean format.
func displayMessage(msg types.Message) {
	switch m := msg.(type) {
	case *types.UserMessage:
		switch content := m.Content.(type) {
		case string:
			fmt.Printf("User: %s\n", content)
		case []types.ContentBlock:
			for _, block := range content {
				switch b := block.(type) {
				case types.TextBlock:
					fmt.Printf("User: %s\n", b.Text)
				case types.ToolResultBlock:
					resultContent := fmt.Sprintf("%v", b.Content)
					if len(resultContent) > 100 {
						resultContent = resultContent[:100] + "..."
					}
					fmt.Printf("Tool Result: %s\n", resultContent)
				}
			}
		}
	case *types.AssistantMessage:
		for _, block := range m.Content {
			switch b := block.(type) {
			case types.TextBlock:
				fmt.Printf("Claude: %s\n", b.Text)
			case types.ToolUseBlock:
				fmt.Printf("Using tool: %s\n", b.Name)
				if b.Input != nil {
					fmt.Printf("  Input: %v\n", b.Input)
				}
			}
		}
	case *types.SystemMessage:
		// Ignore system messages
	case *types.ResultMessage:
		fmt.Println("Result ended")
		if m.TotalCostUSD != nil {
			fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
		}
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	// Create the calculator server with all tools
	calculator := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{
		createAddTool(),
		createSubtractTool(),
		createMultiplyTool(),
		createDivideTool(),
		createSqrtTool(),
		createPowerTool(),
	}, sdkmcp.WithServerVersion("2.0.0"))

	fmt.Println("=== Claude Agent SDK Go - Calculator MCP Server Example ===")
	fmt.Println()
	fmt.Println("Created calculator MCP server with 6 tools:")
	fmt.Println("  - add: Add two numbers together")
	fmt.Println("  - subtract: Subtract one number from another")
	fmt.Println("  - multiply: Multiply two numbers")
	fmt.Println("  - divide: Divide one number by another")
	fmt.Println("  - sqrt: Calculate square root")
	fmt.Println("  - power: Raise a number to a power")
	fmt.Println()

	// Configure Claude to use the calculator server with allowed tools
	// Pre-approve all calculator MCP tools so they can be used without permission prompts
	mode := types.PermissionModeBypassPermissions
	options := &types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"calc": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: calculator,
			},
		},
		AllowedTools: []string{
			"mcp__calc__add",
			"mcp__calc__subtract",
			"mcp__calc__multiply",
			"mcp__calc__divide",
			"mcp__calc__sqrt",
			"mcp__calc__power",
		},
		PermissionMode: &mode,
	}

	// Example prompts to demonstrate calculator usage
	prompts := []string{
		"Calculate 15 + 27",
	}

	// Create a single client for all queries (more efficient than creating new one each time)
	client := claude.NewClientWithOptions(options)
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	for i, prompt := range prompts {
		fmt.Printf("\n%s\n", "==================================================")
		fmt.Printf("Prompt: %s\n", prompt)
		fmt.Printf("%s\n", "==================================================")

		msgChan, err := client.Query(ctx, prompt)
		if err != nil {
			log.Printf("Query failed: %v", err)
			continue
		}

		for msg := range msgChan {
			displayMessage(msg)
		}

		// Print separator between prompts (except last one)
		if i < len(prompts)-1 {
			fmt.Println()
		}
	}

	fmt.Println("\n=== All calculations complete ===")

	// Explicitly close client and exit
	client.Close()
}
