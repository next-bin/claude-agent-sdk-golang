package e2e_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// SDK MCP Tools E2E Tests
// ============================================================================

func TestSdkMcpToolExecution(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a simple test tool
	testTool := sdkmcp.Tool("echo", "Echo back the input message", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{"type": "string"},
		},
		"required": []string{"message"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		message, _ := args["message"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Echo: %s", message)), nil
	})

	// Create SDK MCP server
	server := sdkmcp.CreateSdkMcpServer("test", []*sdkmcp.SdkMcpTool{testTool})

	mode := types.PermissionModeBypassPermissions
	// Configure client with the SDK MCP server
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"test": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__test__echo"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	// Connect
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Query
	msgChan, err := client.Query(ctx, "Use the echo tool to say 'Hello World'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Collect messages
	var foundResult bool
	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		if m, ok := msg.(*types.ResultMessage); ok {
			foundResult = true
			if m.IsError {
				t.Errorf("Result was an error: %v", m)
			}
		}
	})

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

func TestSdkMcpMultipleTools(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create multiple tools
	addTool := sdkmcp.Tool("add", "Add two numbers", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return sdkmcp.TextResult(fmt.Sprintf("%.2f", a+b)), nil
	})

	multiplyTool := sdkmcp.Tool("multiply", "Multiply two numbers", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
		"required": []string{"a", "b"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		a, _ := args["a"].(float64)
		b, _ := args["b"].(float64)
		return sdkmcp.TextResult(fmt.Sprintf("%.2f", a*b)), nil
	})

	// Create SDK MCP server with multiple tools
	server := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{addTool, multiplyTool})

	mode := types.PermissionModeBypassPermissions
	// Configure client
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"calc": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__calc__add", "mcp__calc__multiply"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Calculate 5 + 3 and then multiply the result by 2")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		if m, ok := msg.(*types.ResultMessage); ok {
			foundResult = true
			if m.IsError {
				t.Errorf("Result was an error: %v", m)
			}
		}
	})

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

func TestSdkMcpToolWithError(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a tool that returns an error
	failTool := sdkmcp.Tool("fail", "A tool that always fails", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		return sdkmcp.TextResultWithError("This tool always fails"), nil
	})

	server := sdkmcp.CreateSdkMcpServer("failserver", []*sdkmcp.SdkMcpTool{failTool})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"fail": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__fail__fail"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Use the fail tool")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		if _, ok := msg.(*types.ResultMessage); ok {
			foundResult = true
		}
	})

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

func TestSdkMcpImageContent(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a tool that returns image content
	imageTool := sdkmcp.Tool("get_image", "Returns a simple test image", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		// Simple 1x1 PNG image (base64 encoded minimal PNG)
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
			0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
			0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
			0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59,
			0xE7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
			0x44, 0xAE, 0x42, 0x60, 0x82, // IEND chunk
		}
		return sdkmcp.ImageResult(pngData, "image/png"), nil
	})

	server := sdkmcp.CreateSdkMcpServer("imageserver", []*sdkmcp.SdkMcpTool{imageTool})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"img": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__img__get_image"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Use the get_image tool and describe what you see")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		if m, ok := msg.(*types.ResultMessage); ok {
			foundResult = true
			if m.IsError {
				t.Errorf("Result was an error: %v", m)
			}
		}
	})

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

func TestSdkMcpToolWithAnnotations(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a tool with annotations
	readOnlyTool := sdkmcp.Tool("get_info", "Get some information (read-only)", map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		return sdkmcp.TextResult("This is read-only information"), nil
	}, sdkmcp.WithAnnotations(&sdkmcp.ToolAnnotations{
		Title:        "Get Info",
		ReadOnlyHint: true,
	}))

	server := sdkmcp.CreateSdkMcpServer("infoserver", []*sdkmcp.SdkMcpTool{readOnlyTool})

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"info": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		AllowedTools:   []string{"mcp__info__get_info"},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Use the get_info tool")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		if _, ok := msg.(*types.ResultMessage); ok {
			foundResult = true
		}
	})

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

// TestSdkMcpPermissionEnforcement tests that disallowed_tools prevents SDK MCP tool execution.
func TestSdkMcpPermissionEnforcement(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var executions []string

	// Create echo tool
	echoTool := sdkmcp.Tool("echo", "Echo back the input text", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text": map[string]interface{}{"type": "string"},
		},
		"required": []string{"text"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		executions = append(executions, "echo")
		text, _ := args["text"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Echo: %s", text)), nil
	})

	// Create greet tool
	greetTool := sdkmcp.Tool("greet", "Greet a person by name", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []string{"name"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		executions = append(executions, "greet")
		name, _ := args["name"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Hello, %s!", name)), nil
	})

	server := sdkmcp.CreateSdkMcpServer("test", []*sdkmcp.SdkMcpTool{echoTool, greetTool})

	mode := types.PermissionModeBypassPermissions
	// Block echo tool, allow greet
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"test": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		DisallowedTools: []string{"mcp__test__echo"},
		AllowedTools:    []string{"mcp__test__greet"},
		PermissionMode:  &mode,
		MaxTurns:        types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "You must use the mcp__test__greet tool to greet 'Alice'. This is a test of the greet tool, so please call it.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Executions: %v", executions)

	// Check actual function executions
	foundEcho := false
	foundGreet := false
	for _, e := range executions {
		if e == "echo" {
			foundEcho = true
		}
		if e == "greet" {
			foundGreet = true
		}
	}

	if foundEcho {
		t.Error("Disallowed echo tool was executed")
	}
	// Note: greet tool may or may not be called depending on CLI decision
	// The main purpose of this test is to verify the disallowed tool is NOT called
	t.Logf("Greet tool called: %v", foundGreet)
}

// TestSdkMcpWithoutPermissions tests SDK MCP tool behavior without explicit allowed_tools.
func TestSdkMcpWithoutPermissions(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var executions []string

	// Create echo tool
	echoTool := sdkmcp.Tool("echo", "Echo back the input text", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text": map[string]interface{}{"type": "string"},
		},
		"required": []string{"text"},
	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
		executions = append(executions, "echo")
		text, _ := args["text"].(string)
		return sdkmcp.TextResult(fmt.Sprintf("Echo: %s", text)), nil
	})

	server := sdkmcp.CreateSdkMcpServer("noperm", []*sdkmcp.SdkMcpTool{echoTool})

	mode := types.PermissionModeBypassPermissions
	// No allowed_tools specified
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		MCPServers: map[string]types.McpServerConfig{
			"noperm": types.McpSdkServerConfig{
				Type:     "sdk",
				Instance: server,
			},
		},
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Call the mcp__noperm__echo tool")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	consumeMessagesUntilResult(ctx, msgChan)

	t.Logf("Executions: %v", executions)
	// Note: Without allowed_tools, the tool may or may not be called depending on CLI settings
}
