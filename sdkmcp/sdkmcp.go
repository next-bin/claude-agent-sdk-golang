// Package sdkmcp provides convenience functions for creating in-process MCP servers.
//
// This package simplifies the creation of MCP (Model Context Protocol) servers
// that run within your Go application. Unlike external MCP servers that require
// separate processes, SDK MCP servers run directly in your application's process,
// providing better performance and simpler deployment.
//
// Example:
//
//	// Create a simple calculator tool
//	addTool := sdkmcp.Tool("add", "Add two numbers", map[string]interface{}{
//	    "type": "object",
//	    "properties": map[string]interface{}{
//	        "a": map[string]interface{}{"type": "number"},
//	        "b": map[string]interface{}{"type": "number"},
//	    },
//	    "required": []string{"a", "b"},
//	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
//	    a, _ := args["a"].(float64)
//	    b, _ := args["b"].(float64)
//	    return sdkmcp.TextResult(fmt.Sprintf("Result: %.2f", a+b)), nil
//	})
//
//	// Create the server
//	server := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{addTool})
//
//	// Use with Claude SDK client
//	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
//	    MCPServers: map[string]types.McpServerConfig{
//	        "calc": types.McpSdkServerConfig{
//	            Type:     "sdk",
//	            Instance: server,
//	        },
//	    },
//	})
package sdkmcp

import (
	"context"
	"encoding/base64"
	"fmt"
)

// ============================================================================
// Content Block Types
// ============================================================================

// ContentBlock represents content in a tool result.
// This interface is implemented by TextContent and ImageContent.
type ContentBlock interface {
	GetType() string
}

// TextContent represents a text content block in a tool result.
type TextContent struct {
	// Type is always "text".
	Type string `json:"type"`
	// Text is the text content.
	Text string `json:"text"`
}

// GetType returns the content type.
func (t *TextContent) GetType() string {
	return "text"
}

// ImageContent represents an image content block in a tool result.
type ImageContent struct {
	// Type is always "image".
	Type string `json:"type"`
	// Data is the base64-encoded image data.
	Data string `json:"data"`
	// MimeType is the MIME type of the image (e.g., "image/png", "image/jpeg").
	MimeType string `json:"mimeType"`
}

// GetType returns the content type.
func (i *ImageContent) GetType() string {
	return "image"
}

// ============================================================================
// Tool Result
// ============================================================================

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	// Content is the list of content blocks in the result.
	Content []ContentBlock `json:"content"`
	// IsError indicates whether the result represents an error.
	IsError bool `json:"isError,omitempty"`
}

// TextResult creates a ToolResult with a single text content block.
// This is a convenience function for the common case of returning text.
func TextResult(text string) *ToolResult {
	return &ToolResult{
		Content: []ContentBlock{&TextContent{Type: "text", Text: text}},
	}
}

// TextResultWithError creates a ToolResult with an error text content block.
func TextResultWithError(text string) *ToolResult {
	return &ToolResult{
		Content: []ContentBlock{&TextContent{Type: "text", Text: text}},
		IsError: true,
	}
}

// ImageResult creates a ToolResult with an image content block.
// The data should be the raw image bytes; it will be base64-encoded.
func ImageResult(data []byte, mimeType string) *ToolResult {
	return &ToolResult{
		Content: []ContentBlock{
			&ImageContent{
				Type:     "image",
				Data:     base64.StdEncoding.EncodeToString(data),
				MimeType: mimeType,
			},
		},
	}
}

// ============================================================================
// Tool Annotations
// ============================================================================

// ToolAnnotations provides hints about tool behavior to the model.
//
// These annotations help Claude understand the nature of tools:
// - ReadOnlyHint: Tool doesn't modify state (e.g., read-only queries)
// - DestructiveHint: Tool may perform destructive operations (e.g., delete files)
// - IdempotentHint: Running the tool multiple times has the same effect
// - OpenWorldHint: Tool interacts with external systems (e.g., web APIs)
type ToolAnnotations struct {
	// Title is a human-readable title for the tool.
	Title string `json:"title,omitempty"`
	// ReadOnlyHint indicates the tool only reads data, doesn't modify it.
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`
	// DestructiveHint indicates the tool may perform destructive operations.
	DestructiveHint bool `json:"destructiveHint,omitempty"`
	// IdempotentHint indicates the tool is idempotent.
	IdempotentHint bool `json:"idempotentHint,omitempty"`
	// OpenWorldHint indicates the tool interacts with external systems.
	OpenWorldHint bool `json:"openWorldHint,omitempty"`
}

// ============================================================================
// Tool Handler
// ============================================================================

// ToolHandler is the function signature for tool handlers.
//
// The handler receives a context for cancellation and a map of arguments
// parsed from the tool call. It should return a ToolResult or an error.
//
// Example:
//
//	func handleAdd(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
//	    a, _ := args["a"].(float64)
//	    b, _ := args["b"].(float64)
//	    return sdkmcp.TextResult(fmt.Sprintf("%.2f", a+b)), nil
//	}
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*ToolResult, error)

// ============================================================================
// SDK MCP Tool
// ============================================================================

// SdkMcpTool represents a tool definition for SDK MCP servers.
//
// Use the Tool() function to create tool definitions with a fluent API.
type SdkMcpTool struct {
	// Name is the unique identifier for the tool.
	// This is what Claude will use to reference the tool in function calls.
	Name string `json:"name"`
	// Description is a human-readable description of what the tool does.
	// This helps Claude understand when to use the tool.
	Description string `json:"description"`
	// InputSchema is the JSON Schema defining the tool's input parameters.
	// Example: {"type": "object", "properties": {"a": {"type": "number"}}}
	InputSchema map[string]interface{} `json:"inputSchema"`
	// Handler is the function that executes the tool logic.
	handler ToolHandler
	// Annotations provides hints about tool behavior.
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// GetHandler returns the tool's handler function.
func (t *SdkMcpTool) GetHandler() ToolHandler {
	return t.handler
}

// ============================================================================
// Tool Creation
// ============================================================================

// ToolOption is a function that modifies an SdkMcpTool during creation.
type ToolOption func(*SdkMcpTool)

// WithAnnotations sets the tool annotations.
func WithAnnotations(annotations *ToolAnnotations) ToolOption {
	return func(t *SdkMcpTool) {
		t.Annotations = annotations
	}
}

// Tool creates a new SdkMcpTool with the given configuration.
//
// Parameters:
//   - name: Unique identifier for the tool (e.g., "add", "search")
//   - description: Human-readable description of what the tool does
//   - inputSchema: JSON Schema defining the tool's input parameters
//   - handler: Function that executes the tool logic
//   - opts: Optional configuration (e.g., WithAnnotations)
//
// Example:
//
//	addTool := sdkmcp.Tool("add", "Add two numbers", map[string]interface{}{
//	    "type": "object",
//	    "properties": map[string]interface{}{
//	        "a": map[string]interface{}{"type": "number"},
//	        "b": map[string]interface{}{"type": "number"},
//	    },
//	    "required": []string{"a", "b"},
//	}, func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
//	    a, _ := args["a"].(float64)
//	    b, _ := args["b"].(float64)
//	    return sdkmcp.TextResult(fmt.Sprintf("%.2f", a+b)), nil
//	})
func Tool(name, description string, inputSchema map[string]interface{}, handler ToolHandler, opts ...ToolOption) *SdkMcpTool {
	t := &SdkMcpTool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		handler:     handler,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// ============================================================================
// SDK MCP Server
// ============================================================================

// SdkMcpServerImpl implements the McpServer interface for in-process MCP servers.
type SdkMcpServerImpl struct {
	name    string
	version string
	tools   []*SdkMcpTool
	// toolMap provides quick lookup by name
	toolMap map[string]*SdkMcpTool
}

// ServerOption is a function that modifies an SdkMcpServerImpl during creation.
type ServerOption func(*SdkMcpServerImpl)

// WithServerVersion sets the server version.
func WithServerVersion(version string) ServerOption {
	return func(s *SdkMcpServerImpl) {
		s.version = version
	}
}

// CreateSdkMcpServer creates an in-process MCP server from a list of tools.
//
// This function creates a server that runs within your Go application,
// providing better performance than external MCP servers by avoiding
// inter-process communication overhead.
//
// Parameters:
//   - name: Unique identifier for the server (used in tool names: mcp__<name>__<tool>)
//   - tools: List of tool definitions created with Tool()
//   - opts: Optional configuration (e.g., WithServerVersion())
//
// Returns an SdkMcpServerImpl that implements the query.McpServer interface.
//
// Example:
//
//	server := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{
//	    addTool,
//	    subtractTool,
//	}, sdkmcp.WithServerVersion("2.0.0"))
func CreateSdkMcpServer(name string, tools []*SdkMcpTool, opts ...ServerOption) *SdkMcpServerImpl {
	s := &SdkMcpServerImpl{
		name:    name,
		version: "1.0.0",
		tools:   tools,
		toolMap: make(map[string]*SdkMcpTool),
	}

	// Build tool map
	for _, tool := range tools {
		s.toolMap[tool.Name] = tool
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Name returns the server name.
func (s *SdkMcpServerImpl) Name() string {
	return s.name
}

// Version returns the server version.
func (s *SdkMcpServerImpl) Version() string {
	return s.version
}

// HandleRequest handles an MCP protocol request.
// This implements the query.McpServer interface.
func (s *SdkMcpServerImpl) HandleRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	switch method {
	case "initialize":
		return s.handleInitialize(ctx, params)
	case "tools/list":
		return s.handleToolsList(ctx, params)
	case "tools/call":
		return s.handleToolsCall(ctx, params)
	case "notifications/initialized":
		// No response needed for notifications
		return map[string]interface{}{}, nil
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

// handleInitialize handles the initialize request.
func (s *SdkMcpServerImpl) handleInitialize(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
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
func (s *SdkMcpServerImpl) handleToolsList(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	tools := make([]map[string]interface{}, len(s.tools))
	for i, tool := range s.tools {
		toolDef := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
		if tool.Annotations != nil {
			toolDef["annotations"] = tool.Annotations
		}
		tools[i] = toolDef
	}
	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// handleToolsCall executes a tool call.
func (s *SdkMcpServerImpl) handleToolsCall(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	toolName, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tool name")
	}

	// Find the tool
	tool, exists := s.toolMap[toolName]
	if !exists {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	// Get arguments
	args, _ := params["arguments"].(map[string]interface{})
	if args == nil {
		args = make(map[string]interface{})
	}

	// Call the handler
	result, err := tool.handler(ctx, args)
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

	// Convert result to MCP format
	content := make([]map[string]interface{}, len(result.Content))
	for i, block := range result.Content {
		switch b := block.(type) {
		case *TextContent:
			content[i] = map[string]interface{}{
				"type": "text",
				"text": b.Text,
			}
		case *ImageContent:
			content[i] = map[string]interface{}{
				"type":     "image",
				"data":     b.Data,
				"mimeType": b.MimeType,
			}
		default:
			content[i] = map[string]interface{}{
				"type": "text",
				"text": fmt.Sprintf("%v", block),
			}
		}
	}

	response := map[string]interface{}{
		"content": content,
	}
	if result.IsError {
		response["isError"] = true
	}

	return response, nil
}

// GetTools returns the list of tools defined for this server.
func (s *SdkMcpServerImpl) GetTools() []*SdkMcpTool {
	return s.tools
}
