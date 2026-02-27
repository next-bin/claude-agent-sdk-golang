package sdkmcp

import (
	"context"
	"testing"
)

// ============================================================================
// Content Block Tests
// ============================================================================

func TestTextContentGetType(t *testing.T) {
	tc := &TextContent{Type: "text", Text: "hello"}
	if tc.GetType() != "text" {
		t.Errorf("Expected type 'text', got '%s'", tc.GetType())
	}
}

func TestImageContentGetType(t *testing.T) {
	ic := &ImageContent{Type: "image", Data: "base64data", MimeType: "image/png"}
	if ic.GetType() != "image" {
		t.Errorf("Expected type 'image', got '%s'", ic.GetType())
	}
}

// ============================================================================
// Tool Result Tests
// ============================================================================

func TestTextResult(t *testing.T) {
	result := TextResult("hello world")
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(result.Content))
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	tc, ok := result.Content[0].(*TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	if tc.Text != "hello world" {
		t.Errorf("Expected text 'hello world', got '%s'", tc.Text)
	}
}

func TestTextResultWithError(t *testing.T) {
	result := TextResultWithError("something went wrong")
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(result.Content))
	}
	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	tc, ok := result.Content[0].(*TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}
	if tc.Text != "something went wrong" {
		t.Errorf("Expected text 'something went wrong', got '%s'", tc.Text)
	}
}

func TestImageResult(t *testing.T) {
	imageData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	result := ImageResult(imageData, "image/png")
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(result.Content))
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
	ic, ok := result.Content[0].(*ImageContent)
	if !ok {
		t.Fatal("Expected ImageContent")
	}
	if ic.MimeType != "image/png" {
		t.Errorf("Expected mimeType 'image/png', got '%s'", ic.MimeType)
	}
	if ic.Data == "" {
		t.Error("Expected base64-encoded data")
	}
}

// ============================================================================
// Tool Tests
// ============================================================================

func TestToolCreation(t *testing.T) {
	handler := func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("test"), nil
	}

	tool := Tool("test_tool", "A test tool", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{"type": "string"},
		},
	}, handler)

	if tool.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", tool.Description)
	}
	if tool.handler == nil {
		t.Error("Expected handler to be set")
	}
	if tool.Annotations != nil {
		t.Error("Expected annotations to be nil by default")
	}
}

func TestToolWithAnnotations(t *testing.T) {
	handler := func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("test"), nil
	}

	annotations := &ToolAnnotations{
		Title:        "Test Tool",
		ReadOnlyHint: true,
	}

	tool := Tool("test_tool", "A test tool", map[string]interface{}{}, handler, WithAnnotations(annotations))

	if tool.Annotations == nil {
		t.Fatal("Expected annotations to be set")
	}
	if tool.Annotations.Title != "Test Tool" {
		t.Errorf("Expected title 'Test Tool', got '%s'", tool.Annotations.Title)
	}
	if !tool.Annotations.ReadOnlyHint {
		t.Error("Expected ReadOnlyHint to be true")
	}
}

func TestToolGetHandler(t *testing.T) {
	handler := func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("test"), nil
	}

	tool := Tool("test_tool", "A test tool", map[string]interface{}{}, handler)

	if tool.GetHandler() == nil {
		t.Error("Expected GetHandler to return non-nil handler")
	}
}

// ============================================================================
// Server Creation Tests
// ============================================================================

func TestCreateSdkMcpServer(t *testing.T) {
	tool1 := Tool("add", "Add numbers", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("result"), nil
	})
	tool2 := Tool("subtract", "Subtract numbers", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("result"), nil
	})

	server := CreateSdkMcpServer("calculator", []*SdkMcpTool{tool1, tool2})

	if server.Name() != "calculator" {
		t.Errorf("Expected name 'calculator', got '%s'", server.Name())
	}
	if server.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", server.Version())
	}
	if len(server.GetTools()) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(server.GetTools()))
	}
}

func TestCreateSdkMcpServerWithVersion(t *testing.T) {
	tool := Tool("test", "Test tool", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("result"), nil
	})

	server := CreateSdkMcpServer("test", []*SdkMcpTool{tool}, WithServerVersion("2.0.0"))

	if server.Version() != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", server.Version())
	}
}

func TestCreateSdkMcpServerEmptyTools(t *testing.T) {
	server := CreateSdkMcpServer("empty", []*SdkMcpTool{})

	if server.Name() != "empty" {
		t.Errorf("Expected name 'empty', got '%s'", server.Name())
	}
	if len(server.GetTools()) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(server.GetTools()))
	}
}

// ============================================================================
// MCP Protocol Tests
// ============================================================================

func TestServerHandleInitialize(t *testing.T) {
	tool := Tool("test", "Test tool", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("result"), nil
	})
	server := CreateSdkMcpServer("test", []*SdkMcpTool{tool}, WithServerVersion("1.0.0"))

	ctx := context.Background()
	result, err := server.HandleRequest(ctx, "initialize", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	protocolVersion, ok := result["protocolVersion"].(string)
	if !ok || protocolVersion != "2024-11-05" {
		t.Errorf("Expected protocolVersion '2024-11-05', got '%v'", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected serverInfo to be a map")
	}
	if serverInfo["name"] != "test" {
		t.Errorf("Expected name 'test', got '%s'", serverInfo["name"])
	}
	if serverInfo["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", serverInfo["version"])
	}
}

func TestServerHandleToolsList(t *testing.T) {
	tool := Tool("add", "Add two numbers", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number"},
			"b": map[string]interface{}{"type": "number"},
		},
	}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("result"), nil
	})
	server := CreateSdkMcpServer("calculator", []*SdkMcpTool{tool})

	ctx := context.Background()
	result, err := server.HandleRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected tools to be a slice")
	}
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}
	if tools[0]["name"] != "add" {
		t.Errorf("Expected name 'add', got '%s'", tools[0]["name"])
	}
	if tools[0]["description"] != "Add two numbers" {
		t.Errorf("Expected description 'Add two numbers', got '%s'", tools[0]["description"])
	}
}

func TestServerHandleToolsListWithAnnotations(t *testing.T) {
	annotations := &ToolAnnotations{
		Title:        "Add Tool",
		ReadOnlyHint: true,
	}
	tool := Tool("add", "Add two numbers", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("result"), nil
	}, WithAnnotations(annotations))
	server := CreateSdkMcpServer("calculator", []*SdkMcpTool{tool})

	ctx := context.Background()
	result, err := server.HandleRequest(ctx, "tools/list", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	tools := result["tools"].([]map[string]interface{})
	toolAnnotations, ok := tools[0]["annotations"].(*ToolAnnotations)
	if !ok {
		t.Fatal("Expected annotations to be ToolAnnotations")
	}
	if toolAnnotations.Title != "Add Tool" {
		t.Errorf("Expected title 'Add Tool', got '%s'", toolAnnotations.Title)
	}
}

func TestServerHandleToolsCall(t *testing.T) {
	tool := Tool("add", "Add two numbers", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		// Simulate adding (arguments are provided in the test)
		return TextResult("5.00"), nil
	})
	server := CreateSdkMcpServer("calculator", []*SdkMcpTool{tool})

	ctx := context.Background()
	result, err := server.HandleRequest(ctx, "tools/call", map[string]interface{}{
		"name": "add",
		"arguments": map[string]interface{}{
			"a": 2.0,
			"b": 3.0,
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected content to be a slice")
	}
	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}
	if content[0]["type"] != "text" {
		t.Errorf("Expected type 'text', got '%s'", content[0]["type"])
	}
	if content[0]["text"] != "5.00" {
		t.Errorf("Expected text '5.00', got '%s'", content[0]["text"])
	}
}

func TestServerHandleToolsCallWithImageResult(t *testing.T) {
	imageData := []byte{0x89, 0x50, 0x4E, 0x47}
	tool := Tool("get_image", "Get an image", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return ImageResult(imageData, "image/png"), nil
	})
	server := CreateSdkMcpServer("image_server", []*SdkMcpTool{tool})

	ctx := context.Background()
	result, err := server.HandleRequest(ctx, "tools/call", map[string]interface{}{
		"name":      "get_image",
		"arguments": map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content := result["content"].([]map[string]interface{})
	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}
	if content[0]["type"] != "image" {
		t.Errorf("Expected type 'image', got '%s'", content[0]["type"])
	}
	if content[0]["mimeType"] != "image/png" {
		t.Errorf("Expected mimeType 'image/png', got '%s'", content[0]["mimeType"])
	}
}

func TestServerHandleToolsCallUnknownTool(t *testing.T) {
	server := CreateSdkMcpServer("test", []*SdkMcpTool{})

	ctx := context.Background()
	_, err := server.HandleRequest(ctx, "tools/call", map[string]interface{}{
		"name":      "unknown_tool",
		"arguments": map[string]interface{}{},
	})
	if err == nil {
		t.Error("Expected error for unknown tool")
	}
}

func TestServerHandleToolsCallMissingName(t *testing.T) {
	tool := Tool("test", "Test tool", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("result"), nil
	})
	server := CreateSdkMcpServer("test", []*SdkMcpTool{tool})

	ctx := context.Background()
	_, err := server.HandleRequest(ctx, "tools/call", map[string]interface{}{
		"arguments": map[string]interface{}{},
	})
	if err == nil {
		t.Error("Expected error for missing tool name")
	}
}

func TestServerHandleToolsCallHandlerError(t *testing.T) {
	tool := Tool("fail", "A failing tool", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResultWithError("handler failed"), nil
	})
	server := CreateSdkMcpServer("test", []*SdkMcpTool{tool})

	ctx := context.Background()
	result, err := server.HandleRequest(ctx, "tools/call", map[string]interface{}{
		"name":      "fail",
		"arguments": map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	content := result["content"].([]map[string]interface{})
	if content[0]["text"] != "handler failed" {
		t.Errorf("Expected text 'handler failed', got '%s'", content[0]["text"])
	}
	isError, ok := result["isError"].(bool)
	if !ok || !isError {
		t.Error("Expected isError to be true")
	}
}

func TestServerHandleUnsupportedMethod(t *testing.T) {
	server := CreateSdkMcpServer("test", []*SdkMcpTool{})

	ctx := context.Background()
	_, err := server.HandleRequest(ctx, "unsupported/method", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for unsupported method")
	}
}

func TestServerHandleNotificationsInitialized(t *testing.T) {
	server := CreateSdkMcpServer("test", []*SdkMcpTool{})

	ctx := context.Background()
	result, err := server.HandleRequest(ctx, "notifications/initialized", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

// ============================================================================
// Multiple Tools Test
// ============================================================================

func TestMultipleTools(t *testing.T) {
	addTool := Tool("add", "Add numbers", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("added"), nil
	})
	subTool := Tool("subtract", "Subtract numbers", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("subtracted"), nil
	})
	mulTool := Tool("multiply", "Multiply numbers", map[string]interface{}{}, func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		return TextResult("multiplied"), nil
	})

	server := CreateSdkMcpServer("calculator", []*SdkMcpTool{addTool, subTool, mulTool})

	// Test tools/list
	ctx := context.Background()
	result, _ := server.HandleRequest(ctx, "tools/list", map[string]interface{}{})
	tools := result["tools"].([]map[string]interface{})
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	// Test each tool
	for _, toolName := range []string{"add", "subtract", "multiply"} {
		result, err := server.HandleRequest(ctx, "tools/call", map[string]interface{}{
			"name":      toolName,
			"arguments": map[string]interface{}{},
		})
		if err != nil {
			t.Errorf("Unexpected error for tool '%s': %v", toolName, err)
		}
		content := result["content"].([]map[string]interface{})
		if content[0]["text"] == "" {
			t.Errorf("Expected non-empty result for tool '%s'", toolName)
		}
	}
}
