package e2e_tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Comprehensive MCP E2E Tests
// ============================================================================

// TestSDKMCPComprehensive tests the complete SDK MCP workflow including:
// - Tool registration and discovery
// - Tool execution with various parameter types
// - Multiple tools on a single server
// - Multiple queries in sequence
// - Error handling
// - MCP status reporting
func TestSDKMCPComprehensive(t *testing.T) {
	SkipIfNoAPIKey(t)

	logger := NewTestLogger(t, "SDKMCPComprehensive")
	logger.Step("Creating SDK MCP server with multiple tools")

	// Create tools with different parameter types
	echoTool := sdkmcp.Tool("echo", "Echo back the input message",
		sdkmcp.Schema(map[string]interface{}{
			"message": sdkmcp.StringProperty("The message to echo"),
		}, []string{"message"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			message, _ := args["message"].(string)
			return sdkmcp.TextResult(fmt.Sprintf("Echo: %s", message)), nil
		})

	addTool := sdkmcp.Tool("add", "Add two numbers",
		sdkmcp.Schema(map[string]interface{}{
			"a": sdkmcp.NumberProperty("First number"),
			"b": sdkmcp.NumberProperty("Second number"),
		}, []string{"a", "b"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			result := a + b
			return sdkmcp.TextResult(fmt.Sprintf("Result: %.2f", result)), nil
		})

	greetTool := sdkmcp.Tool("greet", "Greet a person by name",
		sdkmcp.SimpleSchema(map[string]string{"name": "string"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			name, _ := args["name"].(string)
			return sdkmcp.TextResult(fmt.Sprintf("Hello, %s! Nice to meet you.", name)), nil
		})

	// Create server with all tools
	server := sdkmcp.CreateSdkMcpServer("test-comprehensive", []*sdkmcp.SdkMcpTool{
		echoTool,
		addTool,
		greetTool,
	})

	logger.Step("Creating client with SDK MCP server")
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"test-comprehensive": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools: []string{
			"mcp__test-comprehensive__echo",
			"mcp__test-comprehensive__add",
			"mcp__test-comprehensive__greet",
		},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	// Connect
	bgCtx := context.Background()
	logger.Step("Connecting to Claude")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Test 1: Echo tool
	t.Run("EchoTool", func(t *testing.T) {
		testEchoTool(t, client, msgChan, logger)
	})

	// Test 2: Add tool
	t.Run("AddTool", func(t *testing.T) {
		testAddTool(t, client, msgChan, logger)
	})

	// Test 3: Greet tool
	t.Run("GreetTool", func(t *testing.T) {
		testGreetTool(t, client, msgChan, logger)
	})

	// Test 4: MCP Status
	t.Run("MCPStatus", func(t *testing.T) {
		testMCPStatus(t, client, logger)
	})
}

// testEchoTool tests the echo tool functionality
func testEchoTool(t *testing.T, client *claude.Client, msgChan <-chan types.Message, logger *TestLogger) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	logger.Step("Testing echo tool")
	if err := client.Query(ctx, "Use the echo tool to say 'Hello, MCP World!'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "EchoTool")
	if !foundResult {
		t.Error("Expected to receive a result message")
	}
	if resultMsg != nil && resultMsg.IsError {
		t.Errorf("Result was an error: %v", resultMsg)
	}
	if resultMsg == nil || resultMsg.TotalCostUSD == nil || *resultMsg.TotalCostUSD == 0 {
		t.Error("Expected non-zero cost for echo tool execution")
	}

	logger.Result(foundResult, fmt.Sprintf("Messages: %d, Cost: $%.6f", count, *resultMsg.TotalCostUSD))
}

// testAddTool tests the add tool functionality
func testAddTool(t *testing.T, client *claude.Client, msgChan <-chan types.Message, logger *TestLogger) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	logger.Step("Testing add tool")
	if err := client.Query(ctx, "Use the add tool to calculate 42 + 58"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "AddTool")
	if !foundResult {
		t.Error("Expected to receive a result message")
	}
	if resultMsg != nil && resultMsg.IsError {
		t.Errorf("Result was an error: %v", resultMsg)
	}

	logger.Result(foundResult, fmt.Sprintf("Messages: %d", count))
}

// testGreetTool tests the greet tool functionality
func testGreetTool(t *testing.T, client *claude.Client, msgChan <-chan types.Message, logger *TestLogger) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	logger.Step("Testing greet tool")
	if err := client.Query(ctx, "Use the greet tool to greet 'Claude'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "GreetTool")
	if !foundResult {
		t.Error("Expected to receive a result message")
	}
	if resultMsg != nil && resultMsg.IsError {
		t.Errorf("Result was an error: %v", resultMsg)
	}

	logger.Result(foundResult, fmt.Sprintf("Messages: %d", count))
}

// testMCPStatus tests the MCP status reporting
func testMCPStatus(t *testing.T, client *claude.Client, logger *TestLogger) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Step("Testing MCP status")
	status, err := client.GetMCPStatus(ctx)
	if err != nil {
		t.Fatalf("Failed to get MCP status: %v", err)
	}

	// Verify status structure
	servers, ok := status["mcpServers"].([]interface{})
	if !ok {
		t.Fatalf("Expected mcpServers array in status, got %T", status["mcpServers"])
	}

	// Find our server
	var foundServer map[string]interface{}
	for _, s := range servers {
		if server, ok := s.(map[string]interface{}); ok {
			if name, _ := server["name"].(string); name == "test-comprehensive" {
				foundServer = server
				break
			}
		}
	}

	if foundServer == nil {
		t.Fatal("Expected to find 'test-comprehensive' server in status")
	}

	// Check server status
	serverStatus, _ := foundServer["status"].(string)
	if serverStatus != "connected" {
		t.Errorf("Expected server status 'connected', got '%s'", serverStatus)
	}

	// Check tools
	tools, ok := foundServer["tools"].([]interface{})
	if !ok {
		t.Fatalf("Expected tools array in server status, got %T", foundServer["tools"])
	}

	// Verify we have at least our registered tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			if name, ok := toolMap["name"].(string); ok {
				toolNames[name] = true
			}
		}
	}

	expectedTools := []string{"echo", "add", "greet"}
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool '%s' not found in status", expected)
		}
	}

	logger.Result(true, fmt.Sprintf("Server: test-comprehensive, Status: %s, Tools: %d", serverStatus, len(tools)))
}

// TestSDKMCPErrorHandling tests error handling in SDK MCP tools
func TestSDKMCPErrorHandling(t *testing.T) {
	SkipIfNoAPIKey(t)

	logger := NewTestLogger(t, "SDKMCPErrorHandling")
	logger.Step("Creating SDK MCP server with error-prone tools")

	// Create a tool that returns an error
	errorTool := sdkmcp.Tool("always_error", "A tool that always returns an error",
		sdkmcp.Schema(map[string]interface{}{
			"message": sdkmcp.StringProperty("Error message to return"),
		}, []string{"message"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			message, _ := args["message"].(string)
			return sdkmcp.TextResultWithError(fmt.Sprintf("Tool error: %s", message)), nil
		})

	// Create a tool that panics (tests panic recovery)
	panicTool := sdkmcp.Tool("panic_tool", "A tool that panics",
		sdkmcp.SimpleSchema(map[string]string{}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			panic("intentional panic for testing")
		})

	server := sdkmcp.CreateSdkMcpServer("error-server", []*sdkmcp.SdkMcpTool{
		errorTool,
		panicTool,
	})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"error-server": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools: []string{
			"mcp__error-server__always_error",
			"mcp__error-server__panic_tool",
		},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	bgCtx := context.Background()
	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Test error tool
	t.Run("ErrorTool", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		logger.Step("Testing error tool")
		if err := client.Query(ctx, "Use the always_error tool with message 'test error'"); err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		count, foundResult, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "ErrorTool")
		if !foundResult {
			t.Error("Expected to receive a result message even with tool error")
		}
		logger.Result(foundResult, fmt.Sprintf("Messages: %d", count))
	})
}

// TestMultipleSDKMCPServersComprehensive tests multiple SDK MCP servers working together
func TestMultipleSDKMCPServersComprehensive(t *testing.T) {
	SkipIfNoAPIKey(t)

	logger := NewTestLogger(t, "MultipleSDKMCPServers")
	logger.Step("Creating multiple SDK MCP servers")

	// Server 1: Calculator tools
	calcAdd := sdkmcp.Tool("calc_add", "Add numbers",
		sdkmcp.Schema(map[string]interface{}{
			"numbers": sdkmcp.ArrayProperty(sdkmcp.NumberProperty("A number"), "Numbers to add"),
		}, []string{"numbers"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			numbers, _ := args["numbers"].([]interface{})
			var sum float64
			for _, n := range numbers {
				if f, ok := n.(float64); ok {
					sum += f
				}
			}
			return sdkmcp.TextResult(fmt.Sprintf("Sum: %.2f", sum)), nil
		})

	calcMultiply := sdkmcp.Tool("calc_multiply", "Multiply numbers",
		sdkmcp.Schema(map[string]interface{}{
			"numbers": sdkmcp.ArrayProperty(sdkmcp.NumberProperty("A number"), "Numbers to multiply"),
		}, []string{"numbers"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			numbers, _ := args["numbers"].([]interface{})
			product := 1.0
			for _, n := range numbers {
				if f, ok := n.(float64); ok {
					product *= f
				}
			}
			return sdkmcp.TextResult(fmt.Sprintf("Product: %.2f", product)), nil
		})

	calcServer := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{calcAdd, calcMultiply})

	// Server 2: String tools
	stringUpper := sdkmcp.Tool("string_upper", "Convert string to uppercase",
		sdkmcp.Schema(map[string]interface{}{
			"text": sdkmcp.StringProperty("Text to convert"),
		}, []string{"text"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			text, _ := args["text"].(string)
			return sdkmcp.TextResult(strings.ToUpper(text)), nil
		})

	stringLower := sdkmcp.Tool("string_lower", "Convert string to lowercase",
		sdkmcp.Schema(map[string]interface{}{
			"text": sdkmcp.StringProperty("Text to convert"),
		}, []string{"text"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			text, _ := args["text"].(string)
			return sdkmcp.TextResult(strings.ToLower(text)), nil
		})

	stringServer := sdkmcp.CreateSdkMcpServer("stringtools", []*sdkmcp.SdkMcpTool{stringUpper, stringLower})

	// Server 3: Info tools
	infoServer := sdkmcp.CreateSdkMcpServer("info", []*sdkmcp.SdkMcpTool{
		sdkmcp.Tool("get_time", "Get current time", sdkmcp.Schema(map[string]interface{}{}, []string{}),
			func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
				return sdkmcp.TextResult(time.Now().Format(time.RFC3339)), nil
			}),
	})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"calculator": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: calcServer,
			},
			"stringtools": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: stringServer,
			},
			"info": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: infoServer,
			},
		},
		AllowedTools: []string{
			"mcp__calculator__calc_add",
			"mcp__calculator__calc_multiply",
			"mcp__stringtools__string_upper",
			"mcp__stringtools__string_lower",
			"mcp__info__get_time",
		},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	bgCtx := context.Background()
	logger.Step("Connecting with 3 MCP servers")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Test using tools from different servers
	t.Run("CrossServerTools", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		logger.Step("Testing tools from multiple servers")
		// Ask Claude to use tools from different servers
		if err := client.Query(ctx, "First use calc_add to add [10, 20, 30], then use string_upper to convert 'hello world' to uppercase"); err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "CrossServerTools")
		if !foundResult {
			t.Error("Expected to receive a result message")
		}
		if resultMsg != nil && resultMsg.IsError {
			t.Errorf("Result was an error: %v", resultMsg)
		}
		logger.Result(foundResult, fmt.Sprintf("Messages: %d", count))
	})

	// Verify MCP status shows all servers
	t.Run("Status", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		status, err := client.GetMCPStatus(ctx)
		if err != nil {
			t.Fatalf("Failed to get MCP status: %v", err)
		}

		servers, _ := status["mcpServers"].([]interface{})
		serverNames := make(map[string]bool)
		for _, s := range servers {
			if server, ok := s.(map[string]interface{}); ok {
				if name, ok := server["name"].(string); ok {
					serverNames[name] = true
				}
			}
		}

		expectedServers := []string{"calculator", "stringtools", "info"}
		for _, expected := range expectedServers {
			if !serverNames[expected] {
				t.Errorf("Expected server '%s' not found in status", expected)
			}
		}

		logger.Result(true, fmt.Sprintf("Found %d servers in status", len(serverNames)))
	})
}

// TestSDKMCPWithComplexSchema tests SDK MCP with complex JSON schema
func TestSDKMCPWithComplexSchema(t *testing.T) {
	SkipIfNoAPIKey(t)

	logger := NewTestLogger(t, "SDKMCPComplexSchema")
	logger.Step("Creating SDK MCP server with complex schema tools")

	// Tool with nested object schema
	addressTool := sdkmcp.Tool("format_address", "Format an address",
		sdkmcp.Schema(map[string]interface{}{
			"street":  sdkmcp.StringProperty("Street address"),
			"city":    sdkmcp.StringProperty("City name"),
			"zipcode": sdkmcp.StringProperty("Postal code"),
			"country": sdkmcp.StringProperty("Country name"),
		}, []string{"street", "city"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			street, _ := args["street"].(string)
			city, _ := args["city"].(string)
			zipcode, _ := args["zipcode"].(string)
			country, _ := args["country"].(string)

			result := fmt.Sprintf("%s, %s", street, city)
			if zipcode != "" {
				result += fmt.Sprintf(" %s", zipcode)
			}
			if country != "" {
				result += fmt.Sprintf(", %s", country)
			}
			return sdkmcp.TextResult(result), nil
		})

	// Tool with array of objects
	processItemsTool := sdkmcp.Tool("process_items", "Process a list of items",
		sdkmcp.Schema(map[string]interface{}{
			"items": sdkmcp.ArrayProperty(
				sdkmcp.ObjectProperty(map[string]interface{}{
					"name":     sdkmcp.StringProperty("Item name"),
					"quantity": sdkmcp.IntegerProperty("Item quantity"),
				}, []string{"name"}, "An item"),
				"List of items to process"),
		}, []string{"items"}),
		func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
			items, _ := args["items"].([]interface{})
			var result strings.Builder
			result.WriteString("Processed items:\n")
			for i, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					name, _ := itemMap["name"].(string)
					qty, _ := itemMap["quantity"].(float64)
					result.WriteString(fmt.Sprintf("  %d. %s (qty: %.0f)\n", i+1, name, qty))
				}
			}
			return sdkmcp.TextResult(result.String()), nil
		})

	server := sdkmcp.CreateSdkMcpServer("complex-schema", []*sdkmcp.SdkMcpTool{
		addressTool,
		processItemsTool,
	})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"complex-schema": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools: []string{
			"mcp__complex-schema__format_address",
			"mcp__complex-schema__process_items",
		},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	bgCtx := context.Background()
	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Test address formatting
	t.Run("FormatAddress", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		logger.Step("Testing format_address with complex schema")
		if err := client.Query(ctx, "Use format_address with street '123 Main St', city 'San Francisco', zipcode '94102', country 'USA'"); err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "FormatAddress")
		if !foundResult {
			t.Error("Expected to receive a result message")
		}
		if resultMsg != nil && resultMsg.IsError {
			t.Errorf("Result was an error: %v", resultMsg)
		}
		logger.Result(foundResult, fmt.Sprintf("Messages: %d", count))
	})

	// Test items processing
	t.Run("ProcessItems", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		logger.Step("Testing process_items with array of objects")
		if err := client.Query(ctx, "Use process_items with items: [{name: 'Apple', quantity: 5}, {name: 'Banana', quantity: 3}]"); err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "ProcessItems")
		if !foundResult {
			t.Error("Expected to receive a result message")
		}
		if resultMsg != nil && resultMsg.IsError {
			t.Errorf("Result was an error: %v", resultMsg)
		}
		logger.Result(foundResult, fmt.Sprintf("Messages: %d", count))
	})
}
