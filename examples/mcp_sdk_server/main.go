// Example mcp_sdk_server demonstrates creating an in-process MCP server with custom tools.
//
// This example shows how to:
// 1. Implement the McpServer interface for custom tools
// 2. Create calculator tools (add, subtract, multiply, divide, sqrt, power)
// 3. Configure the SDK client with an in-process MCP server
// 4. Handle MCP protocol requests
//
// Unlike external MCP servers that require separate processes, SDK MCP servers
// run directly within your Go application, providing better performance
// and simpler deployment.
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Calculator MCP Server Implementation
// ============================================================================

// CalculatorServer implements the McpServer interface for calculator operations.
type CalculatorServer struct {
	name    string
	version string
	tools   []ToolDefinition
}

// ToolDefinition defines a tool's metadata and handler.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     func(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error)
}

// NewCalculatorServer creates a new calculator MCP server.
func NewCalculatorServer() *CalculatorServer {
	return &CalculatorServer{
		name:    "calculator",
		version: "2.0.0",
		tools: []ToolDefinition{
			{
				Name:        "add",
				Description: "Add two numbers together",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number"},
						"b": map[string]interface{}{"type": "number"},
					},
					"required": []string{"a", "b"},
				},
				Handler: handleAdd,
			},
			{
				Name:        "subtract",
				Description: "Subtract one number from another",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number"},
						"b": map[string]interface{}{"type": "number"},
					},
					"required": []string{"a", "b"},
				},
				Handler: handleSubtract,
			},
			{
				Name:        "multiply",
				Description: "Multiply two numbers together",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number"},
						"b": map[string]interface{}{"type": "number"},
					},
					"required": []string{"a", "b"},
				},
				Handler: handleMultiply,
			},
			{
				Name:        "divide",
				Description: "Divide one number by another",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"a": map[string]interface{}{"type": "number"},
						"b": map[string]interface{}{"type": "number"},
					},
					"required": []string{"a", "b"},
				},
				Handler: handleDivide,
			},
			{
				Name:        "sqrt",
				Description: "Calculate the square root of a number",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"n": map[string]interface{}{"type": "number"},
					},
					"required": []string{"n"},
				},
				Handler: handleSqrt,
			},
			{
				Name:        "power",
				Description: "Raise a number to a power",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"base":     map[string]interface{}{"type": "number"},
						"exponent": map[string]interface{}{"type": "number"},
					},
					"required": []string{"base", "exponent"},
				},
				Handler: handlePower,
			},
		},
	}
}

// Name returns the server name.
func (s *CalculatorServer) Name() string {
	return s.name
}

// Version returns the server version.
func (s *CalculatorServer) Version() string {
	return s.version
}

// HandleRequest handles MCP protocol requests.
func (s *CalculatorServer) HandleRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	switch method {
	case "tools/list":
		return s.handleToolsList(ctx, params)
	case "tools/call":
		return s.handleToolsCall(ctx, params)
	case "initialize":
		return s.handleInitialize(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

// handleInitialize handles the initialize request.
func (s *CalculatorServer) handleInitialize(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    s.name,
			"version": s.version,
		},
	}, nil
}

// handleToolsList returns the list of available tools.
func (s *CalculatorServer) handleToolsList(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	tools := make([]map[string]interface{}, len(s.tools))
	for i, tool := range s.tools {
		tools[i] = map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
	}
	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall executes a tool call.
func (s *CalculatorServer) handleToolsCall(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	toolName, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tool name")
	}

	// Find the tool
	var tool *ToolDefinition
	for i := range s.tools {
		if s.tools[i].Name == toolName {
			tool = &s.tools[i]
			break
		}
	}
	if tool == nil {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	// Get arguments
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = make(map[string]interface{})
	}

	// Call the handler
	result, err := tool.Handler(ctx, args)
	if err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Error: %v", err),
				},
			},
			"isError": true,
		}, nil
	}

	return result, nil
}

// ============================================================================
// Tool Handlers
// ============================================================================

func handleAdd(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)
	result := a + b
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result)},
		},
	}, nil
}

func handleSubtract(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)
	result := a - b
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("%.2f - %.2f = %.2f", a, b, result)},
		},
	}, nil
}

func handleMultiply(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)
	result := a * b
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("%.2f × %.2f = %.2f", a, b, result)},
		},
	}, nil
}

func handleDivide(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)
	if b == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "Error: Division by zero is not allowed"},
			},
			"isError": true,
		}, nil
	}
	result := a / b
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("%.2f ÷ %.2f = %.2f", a, b, result)},
		},
	}, nil
}

func handleSqrt(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	n, _ := args["n"].(float64)
	if n < 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Error: Cannot calculate square root of negative number %.2f", n)},
			},
			"isError": true,
		}, nil
	}
	result := math.Sqrt(n)
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("√%.2f = %.2f", n, result)},
		},
	}, nil
}

func handlePower(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	base, _ := args["base"].(float64)
	exponent, _ := args["exponent"].(float64)
	result := math.Pow(base, exponent)
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": fmt.Sprintf("%.2f^%.2f = %.2f", base, exponent, result)},
		},
	}, nil
}

// ============================================================================
// Main Examples
// ============================================================================

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	fmt.Println("=== Claude Agent SDK Go - MCP SDK Server Example ===")
	fmt.Println()

	// Example 1: Basic calculator server usage
	fmt.Println("--- Example 1: Calculator Server ---")
	basicCalculatorExample(ctx)

	fmt.Println()

	// Example 2: Multiple SDK MCP servers
	fmt.Println("--- Example 2: Multiple SDK MCP Servers ---")
	multipleServersExample(ctx)
}

// basicCalculatorExample demonstrates the basic calculator server.
func basicCalculatorExample(ctx context.Context) {
	// Create the calculator server
	calculator := NewCalculatorServer()

	fmt.Printf("Created calculator MCP server: %s v%s\n", calculator.Name(), calculator.Version())
	fmt.Println("Available tools:")
	for _, tool := range calculator.tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// Configure the SDK client with the calculator server
	// Using McpSdkServerConfig to wrap our server
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"calc": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: calculator,
			},
		},
		// Pre-approve all calculator MCP tools so they can be used without permission prompts
		AllowedTools: []string{
			"mcp__calc__add",
			"mcp__calc__subtract",
			"mcp__calc__multiply",
			"mcp__calc__divide",
			"mcp__calc__sqrt",
			"mcp__calc__power",
		},
	})
	defer client.Close()

	fmt.Println("SDK client configured with MCP calculator server")
	fmt.Println("Tools are prefixed with 'mcp__calc__' (e.g., 'mcp__calc__add')")
	fmt.Println()

	// Example prompts that could be used with this server
	fmt.Println("Example prompts to use with this server:")
	fmt.Println("  - 'Calculate 15 + 27'")
	fmt.Println("  - 'What is 100 divided by 7?'")
	fmt.Println("  - 'Calculate the square root of 144'")
	fmt.Println("  - 'What is 2 raised to the power of 8?'")
	fmt.Println("  - 'Calculate (12 + 8) * 3 - 10'")
	fmt.Println()

	// To actually run a query, uncomment the following:
	/*
		if err := client.Connect(ctx); err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		// Create message channel once and reuse for all queries
		msgChan := client.ReceiveMessages(ctx)

		if err := client.Query(ctx, "Calculate 15 + 27"); err != nil {
			log.Fatalf("Query failed: %v", err)
		}

		for msg := range msgChan {
			// Process messages...
			fmt.Printf("Message: %v\n", msg)
		}
	*/
}

// multipleServersExample demonstrates using multiple SDK MCP servers.
func multipleServersExample(ctx context.Context) {
	// Create multiple SDK MCP servers
	calculator := NewCalculatorServer()

	// You could create additional servers here
	// weatherServer := NewWeatherServer()
	// databaseServer := NewDatabaseServer()

	fmt.Println("Multiple SDK MCP servers can be configured:")
	fmt.Println()

	// Configure with multiple servers
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		MCPServers: map[string]types.McpServerConfig{
			"calc": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: calculator,
			},
			// Additional servers would be added here
			// "weather": types.McpSdkServerConfig{
			//     Type:     "sdk",
			//     Instance: weatherServer,
			// },
		},
	})
	defer client.Close()

	fmt.Println("Multiple servers configuration:")
	fmt.Println("  1. calc - Calculator tools (add, subtract, multiply, divide, sqrt, power)")
	fmt.Println("  2. [Additional servers would be listed here]")
	fmt.Println()

	// Benefits of SDK MCP servers
	fmt.Println("Benefits of in-process SDK MCP servers:")
	fmt.Println("  - No external process management")
	fmt.Println("  - Direct function calls (no IPC overhead)")
	fmt.Println("  - Easier debugging and testing")
	fmt.Println("  - Simpler deployment")
	fmt.Println("  - Full control over tool implementations")
	fmt.Println()
}

// ============================================================================
// Additional Server Types (Examples)
// ============================================================================

// WeatherServer example (not implemented - for reference only)
type WeatherServer struct {
	name    string
	version string
}

// NewWeatherServer creates a new weather MCP server.
func NewWeatherServer() *WeatherServer {
	return &WeatherServer{
		name:    "weather",
		version: "1.0.0",
	}
}

func (s *WeatherServer) Name() string    { return s.name }
func (s *WeatherServer) Version() string { return s.version }

func (s *WeatherServer) HandleRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	// Implementation would handle tools/list and tools/call for weather tools
	return nil, fmt.Errorf("not implemented")
}

// DatabaseServer example (not implemented - for reference only)
type DatabaseServer struct {
	name    string
	version string
}

// NewDatabaseServer creates a new database MCP server.
func NewDatabaseServer() *DatabaseServer {
	return &DatabaseServer{
		name:    "database",
		version: "1.0.0",
	}
}

func (s *DatabaseServer) Name() string    { return s.name }
func (s *DatabaseServer) Version() string { return s.version }

func (s *DatabaseServer) HandleRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	// Implementation would handle tools/list and tools/call for database tools
	return nil, fmt.Errorf("not implemented")
}
