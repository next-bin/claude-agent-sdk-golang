// Package sdkmcp provides convenience functions for creating in-process MCP servers.
//
// This package simplifies the creation of MCP (Model Context Protocol) servers
// that run within your Go application. Unlike external MCP servers that require
// separate processes, SDK MCP servers run directly in your application's process,
// providing better performance and simpler deployment.
//
// Example:
//
//	// Create a simple calculator tool using schema helpers
//	addTool := sdkmcp.Tool("add", "Add two numbers",
//	    sdkmcp.Schema(map[string]interface{}{
//	        "a": sdkmcp.NumberProperty("First number"),
//	        "b": sdkmcp.NumberProperty("Second number"),
//	    }, []string{"a", "b"}),
//	    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
//	        a, _ := args["a"].(float64)
//	        b, _ := args["b"].(float64)
//	        return sdkmcp.TextResult(fmt.Sprintf("Result: %.2f", a+b)), nil
//	    })
//
//	// Or use SimpleSchema for even more concise syntax:
//	addTool := sdkmcp.Tool("add", "Add two numbers",
//	    sdkmcp.SimpleSchema(map[string]string{"a": "number", "b": "number"}),
//	    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
//	        a, _ := args["a"].(float64)
//	        b, _ := args["b"].(float64)
//	        return sdkmcp.TextResult(fmt.Sprintf("Result: %.2f", a+b)), nil
//	    })
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
// Schema Helpers
// ============================================================================

// StringProperty creates a JSON Schema property definition for a string type.
//
// This is a convenience function for defining string parameters in tool schemas.
// It returns a map that can be used as a property value in Schema() or directly
// in a manually constructed input schema.
//
// Parameters:
//   - description: A human-readable description of the parameter
//
// Example:
//
//	properties := map[string]interface{}{
//	    "name": sdkmcp.StringProperty("The user's name"),
//	}
func StringProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

// NumberProperty creates a JSON Schema property definition for a number type.
//
// Number accepts both integers and floating-point values. Use IntegerProperty
// if you need to restrict to whole numbers only.
//
// Parameters:
//   - description: A human-readable description of the parameter
//
// Example:
//
//	properties := map[string]interface{}{
//	    "amount": sdkmcp.NumberProperty("The payment amount in dollars"),
//	}
func NumberProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "number",
		"description": description,
	}
}

// IntegerProperty creates a JSON Schema property definition for an integer type.
//
// Integer only accepts whole numbers. Use NumberProperty if you need to accept
// floating-point values.
//
// Parameters:
//   - description: A human-readable description of the parameter
//
// Example:
//
//	properties := map[string]interface{}{
//	    "count": sdkmcp.IntegerProperty("Number of items to retrieve"),
//	}
func IntegerProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
	}
}

// BooleanProperty creates a JSON Schema property definition for a boolean type.
//
// Parameters:
//   - description: A human-readable description of the parameter
//
// Example:
//
//	properties := map[string]interface{}{
//	    "verbose": sdkmcp.BooleanProperty("Whether to output detailed logs"),
//	}
func BooleanProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

// ArrayProperty creates a JSON Schema property definition for an array type.
//
// Arrays in JSON Schema require an "items" schema that defines the type of
// each element in the array.
//
// Parameters:
//   - items: The schema for items in the array (use property helpers like StringProperty)
//   - description: A human-readable description of the parameter
//
// Example:
//
//	properties := map[string]interface{}{
//	    "tags": sdkmcp.ArrayProperty(
//	        sdkmcp.StringProperty("A tag name"),
//	        "List of tags to apply",
//	    ),
//	    "scores": sdkmcp.ArrayProperty(
//	        sdkmcp.NumberProperty("A score value"),
//	        "List of numeric scores",
//	    ),
//	}
func ArrayProperty(items map[string]interface{}, description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"items":       items,
		"description": description,
	}
}

// ObjectProperty creates a JSON Schema property definition for a nested object type.
//
// Use this for properties that themselves contain structured data. The properties
// parameter defines the schema for the nested object's fields, and required lists
// which fields must be present.
//
// Parameters:
//   - properties: Map of property names to their schema definitions
//   - required: List of property names that are required within this object
//   - description: A human-readable description of the parameter
//
// Example:
//
//	properties := map[string]interface{}{
//	    "address": sdkmcp.ObjectProperty(
//	        map[string]interface{}{
//	            "street":  sdkmcp.StringProperty("Street address"),
//	            "city":    sdkmcp.StringProperty("City name"),
//	            "zipcode": sdkmcp.StringProperty("Postal code"),
//	        },
//	        []string{"street", "city"},
//	        "User's address information",
//	    ),
//	}
func ObjectProperty(properties map[string]interface{}, required []string, description string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        "object",
		"properties":  properties,
		"description": description,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// Schema creates a complete JSON Schema for tool input parameters.
//
// This is the main builder function for creating tool input schemas. It wraps
// the property definitions in the required object structure with type "object"
// and specifies which properties are required.
//
// Parameters:
//   - properties: Map of parameter names to their schema definitions
//   - required: List of parameter names that must be provided
//
// Example:
//
//	// Simple tool with two required number parameters
//	schema := sdkmcp.Schema(map[string]interface{}{
//	    "a": sdkmcp.NumberProperty("First number"),
//	    "b": sdkmcp.NumberProperty("Second number"),
//	}, []string{"a", "b"})
//
//	// Tool with optional parameters
//	schema := sdkmcp.Schema(map[string]interface{}{
//	    "query":    sdkmcp.StringProperty("Search query"),
//	    "limit":    sdkmcp.IntegerProperty("Max results (optional)"),
//	    "verbose":  sdkmcp.BooleanProperty("Enable verbose output (optional)"),
//	}, []string{"query"})
func Schema(properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// SimpleSchema creates a JSON Schema from a simple type map.
//
// This function provides a concise way to define schemas when you don't need
// descriptions for individual properties. It's inspired by standard SDK's
// type-to-schema conversion.
//
// Supported type strings:
//   - "string": String type
//   - "number": Number type (accepts integers and floats)
//   - "integer": Integer type (whole numbers only)
//   - "boolean": Boolean type
//   - "string[]": Array of strings
//   - "number[]": Array of numbers
//   - "integer[]": Array of integers
//   - "boolean[]": Array of booleans
//
// All properties are marked as required by default.
//
// Example:
//
//	// Simple schema with multiple types
//	schema := sdkmcp.SimpleSchema(map[string]string{
//	    "name":    "string",
//	    "age":     "integer",
//	    "active":  "boolean",
//	    "score":   "number",
//	    "tags":    "string[]",
//	    "scores":  "number[]",
//	})
//
//	// Equivalent to:
//	schema := sdkmcp.Schema(map[string]interface{}{
//	    "name":   sdkmcp.StringProperty(""),
//	    "age":    sdkmcp.IntegerProperty(""),
//	    "active": sdkmcp.BooleanProperty(""),
//	    "score":  sdkmcp.NumberProperty(""),
//	    "tags":   sdkmcp.ArrayProperty(sdkmcp.StringProperty(""), ""),
//	    "scores": sdkmcp.ArrayProperty(sdkmcp.NumberProperty(""), ""),
//	}, []string{"name", "age", "active", "score", "tags", "scores"})
func SimpleSchema(types map[string]string) map[string]interface{} {
	properties := make(map[string]interface{}, len(types))
	required := make([]string, 0, len(types))

	for name, typeStr := range types {
		properties[name] = simplePropertyFromType(typeStr)
		required = append(required, name)
	}

	return Schema(properties, required)
}

// simplePropertyFromType creates a property schema from a type string.
// It handles both basic types and array types (e.g., "string[]").
func simplePropertyFromType(typeStr string) map[string]interface{} {
	// Check for array type (e.g., "string[]", "number[]")
	if len(typeStr) > 2 && typeStr[len(typeStr)-2:] == "[]" {
		baseType := typeStr[:len(typeStr)-2]
		var items map[string]interface{}
		switch baseType {
		case "string":
			items = map[string]interface{}{"type": "string"}
		case "number":
			items = map[string]interface{}{"type": "number"}
		case "integer":
			items = map[string]interface{}{"type": "integer"}
		case "boolean":
			items = map[string]interface{}{"type": "boolean"}
		default:
			// Unknown array type, default to string items
			items = map[string]interface{}{"type": "string"}
		}
		return map[string]interface{}{
			"type":  "array",
			"items": items,
		}
	}

	// Handle basic types
	var propType string
	switch typeStr {
	case "string", "number", "integer", "boolean":
		propType = typeStr
	default:
		// Unknown type, default to string
		propType = "string"
	}

	return map[string]interface{}{"type": propType}
}

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
// Example using schema helpers (recommended):
//
//	addTool := sdkmcp.Tool("add", "Add two numbers",
//	    sdkmcp.Schema(map[string]interface{}{
//	        "a": sdkmcp.NumberProperty("First number"),
//	        "b": sdkmcp.NumberProperty("Second number"),
//	    }, []string{"a", "b"}),
//	    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
//	        a, _ := args["a"].(float64)
//	        b, _ := args["b"].(float64)
//	        return sdkmcp.TextResult(fmt.Sprintf("%.2f", a+b)), nil
//	    })
//
// Example using SimpleSchema (most concise):
//
//	addTool := sdkmcp.Tool("add", "Add two numbers",
//	    sdkmcp.SimpleSchema(map[string]string{"a": "number", "b": "number"}),
//	    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
//	        a, _ := args["a"].(float64)
//	        b, _ := args["b"].(float64)
//	        return sdkmcp.TextResult(fmt.Sprintf("%.2f", a+b)), nil
//	    })
//
// Example with manual JSON Schema (verbose but flexible):
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

	// MCP 2025-11-25 Resources support
	resources         []*SdkMcpResource
	resourceMap       map[string]*SdkMcpResource
	resourceTemplates []*SdkMcpResourceTemplate

	// MCP 2025-11-25 Prompts support
	prompts   []*SdkMcpPrompt
	promptMap map[string]*SdkMcpPrompt

	// MCP Logging support
	logLevel LogLevel
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
		name:        name,
		version:     "1.0.0",
		tools:       tools,
		toolMap:     make(map[string]*SdkMcpTool),
		resourceMap: make(map[string]*SdkMcpResource),
		promptMap:   make(map[string]*SdkMcpPrompt),
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
	case "resources/list":
		return s.handleResourcesList(ctx, params)
	case "resources/read":
		return s.handleResourcesRead(ctx, params)
	case "resources/templates/list":
		return s.handleResourceTemplatesList(ctx, params)
	case "prompts/list":
		return s.handlePromptsList(ctx, params)
	case "prompts/get":
		return s.handlePromptsGet(ctx, params)
	case "notifications/initialized":
		// No response needed for notifications
		return map[string]interface{}{}, nil
	case "logging/setLevel":
		return s.handleLoggingSetLevel(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

// handleInitialize handles the initialize request.
func (s *SdkMcpServerImpl) handleInitialize(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	capabilities := map[string]interface{}{
		"tools": map[string]interface{}{},
	}

	// Add resources capability if resources are defined
	if len(s.resources) > 0 || len(s.resourceTemplates) > 0 {
		resources := map[string]interface{}{}
		if len(s.resourceTemplates) > 0 {
			resources["templates"] = map[string]interface{}{}
		}
		capabilities["resources"] = resources
	}

	// Add Prompts capability if prompts are defined
	if len(s.prompts) > 0 {
		capabilities["prompts"] = map[string]interface{}{
			"listChanged": false,
		}
	}

	// Add Logging capability
	capabilities["logging"] = map[string]interface{}{}

	return map[string]interface{}{
		"protocolVersion": "2025-11-25",
		"capabilities":    capabilities,
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

// ============================================================================
// MCP Resources Handlers (2025-11-25 Specification)
// ============================================================================

// handleResourcesList returns the list of available resources.
func (s *SdkMcpServerImpl) handleResourcesList(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if len(s.resources) == 0 {
		return map[string]interface{}{"resources": []interface{}{}}, nil
	}

	resources := make([]map[string]interface{}, len(s.resources))
	for i, r := range s.resources {
		resources[i] = map[string]interface{}{
			"uri":         r.URI,
			"name":        r.Name,
			"description": r.Description,
			"mimeType":    r.MimeType,
		}
	}
	return map[string]interface{}{"resources": resources}, nil
}

// handleResourcesRead returns the content of a specific resource.
func (s *SdkMcpServerImpl) handleResourcesRead(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	uri, ok := params["uri"].(string)
	if !ok {
		return nil, fmt.Errorf("missing resource URI")
	}

	resource, exists := s.resourceMap[uri]
	if !exists {
		return nil, fmt.Errorf("unknown resource: %s", uri)
	}

	contents := []map[string]interface{}{}
	if resource.Text != "" {
		contents = append(contents, map[string]interface{}{
			"type":     "text",
			"text":     resource.Text,
			"mimeType": resource.MimeType,
		})
	}
	if resource.Blob != "" {
		contents = append(contents, map[string]interface{}{
			"type":     "blob",
			"blob":     resource.Blob,
			"mimeType": resource.MimeType,
		})
	}

	return map[string]interface{}{
		"contents": contents,
	}, nil
}

// handleResourceTemplatesList returns the list of resource templates.
func (s *SdkMcpServerImpl) handleResourceTemplatesList(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if len(s.resourceTemplates) == 0 {
		return map[string]interface{}{"resourceTemplates": []interface{}{}}, nil
	}

	templates := make([]map[string]interface{}, len(s.resourceTemplates))
	for i, t := range s.resourceTemplates {
		templates[i] = map[string]interface{}{
			"uriTemplate": t.URITemplate,
			"name":        t.Name,
			"description": t.Description,
			"mimeType":    t.MimeType,
		}
	}
	return map[string]interface{}{"resourceTemplates": templates}, nil
}

// ============================================================================
// MCP Prompts Handlers (2025-11-25 Specification)
// ============================================================================

// handlePromptsList returns the list of available prompts.
func (s *SdkMcpServerImpl) handlePromptsList(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	if len(s.prompts) == 0 {
		return map[string]interface{}{"prompts": []interface{}{}}, nil
	}

	prompts := make([]map[string]interface{}, len(s.prompts))
	for i, p := range s.prompts {
		prompts[i] = map[string]interface{}{
			"name":        p.Name,
			"description": p.Description,
			"arguments":   p.Arguments,
		}
	}
	return map[string]interface{}{"prompts": prompts}, nil
}

// handlePromptsGet returns a prompt template with resolved arguments.
func (s *SdkMcpServerImpl) handlePromptsGet(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	name, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing prompt name")
	}

	prompt, exists := s.promptMap[name]
	if !exists {
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}

	// Apply arguments to prompt messages
	args, _ := params["arguments"].(map[string]interface{})
	messages := make([]map[string]interface{}, len(prompt.Messages))
	for i, msg := range prompt.Messages {
		text := msg.Content.Text
		// Simple variable substitution
		for k, v := range args {
			text = replaceVar(text, k, v)
		}
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": map[string]interface{}{"type": "text", "text": text},
		}
	}

	return map[string]interface{}{
		"description": prompt.Description,
		"messages":    messages,
	}, nil
}

// replaceVar performs simple variable substitution.
func replaceVar(text, key string, value interface{}) string {
	template := "{{" + key + "}}"
	return textReplaceAll(text, template, fmt.Sprintf("%v", value))
}

// textReplaceAll replaces all occurrences of old in text with new.
func textReplaceAll(text, old, new string) string {
	result := ""
	for i := 0; i < len(text); {
		if i+len(old) <= len(text) && text[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(text[i])
			i++
		}
	}
	return result
}

// ============================================================================
// MCP Logging Handler (2025-11-25 Specification)
// ============================================================================

// handleLoggingSetLevel sets the server's minimum log level.
func (s *SdkMcpServerImpl) handleLoggingSetLevel(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	level, ok := params["level"].(string)
	if !ok {
		return nil, fmt.Errorf("missing log level")
	}

	s.logLevel = LogLevel(level)
	return map[string]interface{}{}, nil
}

// ============================================================================
// MCP Resources (2025-11-25 Specification)
// ============================================================================

// SdkMcpResource represents an MCP resource that provides context/data.
type SdkMcpResource struct {
	// URI is the unique identifier for the resource.
	URI string `json:"uri"`
	// Name is a human-readable name for the resource.
	Name string `json:"name"`
	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
	// MimeType is the MIME type of the resource content.
	MimeType string `json:"mimeType,omitempty"`
	// Text content for text-based resources.
	Text string `json:"text,omitempty"`
	// Blob content (base64-encoded) for binary resources.
	Blob string `json:"blob,omitempty"`
}

// SdkMcpResourceTemplate represents a URI template for dynamic resources.
type SdkMcpResourceTemplate struct {
	// URITemplate is a URI template for matching resource requests.
	URITemplate string `json:"uriTemplate"`
	// Name is a human-readable name for the template.
	Name string `json:"name"`
	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
	// MimeType is the MIME type for resources matching this template.
	MimeType string `json:"mimeType,omitempty"`
}

// Resource creates a new SdkMcpResource with the given configuration.
func Resource(uri, name, mimeType, content string) *SdkMcpResource {
	return &SdkMcpResource{
		URI:      uri,
		Name:     name,
		MimeType: mimeType,
		Text:     content,
	}
}

// BinaryResource creates a new SdkMcpResource with binary content (base64-encoded).
func BinaryResource(uri, name, mimeType, blob string) *SdkMcpResource {
	return &SdkMcpResource{
		URI:      uri,
		Name:     name,
		MimeType: mimeType,
		Blob:     blob,
	}
}

// ResourceTemplate creates a new SdkMcpResourceTemplate.
func ResourceTemplate(uriTemplate, name, mimeType string) *SdkMcpResourceTemplate {
	return &SdkMcpResourceTemplate{
		URITemplate: uriTemplate,
		Name:        name,
		MimeType:    mimeType,
	}
}

// WithResources adds resources to the server.
func WithResources(resources ...*SdkMcpResource) ServerOption {
	return func(s *SdkMcpServerImpl) {
		for _, r := range resources {
			s.resources = append(s.resources, r)
			s.resourceMap[r.URI] = r
		}
	}
}

// WithResourceTemplates adds resource templates to the server.
func WithResourceTemplates(templates ...*SdkMcpResourceTemplate) ServerOption {
	return func(s *SdkMcpServerImpl) {
		s.resourceTemplates = append(s.resourceTemplates, templates...)
	}
}

// ============================================================================
// MCP Prompts (2025-11-25 Specification)
// ============================================================================

// SdkMcpPrompt represents an MCP prompt template.
type SdkMcpPrompt struct {
	// Name is the unique identifier for the prompt.
	Name string `json:"name"`
	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
	// Arguments defines the parameters the prompt accepts.
	Arguments []PromptArgument `json:"arguments,omitempty"`
	// Messages contains the prompt messages to return.
	Messages []PromptMessage `json:"messages,omitempty"`
}

// PromptArgument defines an argument that a prompt accepts.
type PromptArgument struct {
	// Name is the argument name.
	Name string `json:"name"`
	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
	// Required indicates whether the argument is required.
	Required bool `json:"required,omitempty"`
}

// PromptMessage represents a message in a prompt template.
type PromptMessage struct {
	// Role is the message role (user, assistant, system).
	Role string `json:"role"`
	// Content contains the message content.
	Content PromptContent `json:"content"`
}

// PromptContent represents the content of a prompt message.
type PromptContent struct {
	// Type is the content type (text, image, resource).
	Type string `json:"type"`
	// Text is the text content (for type="text").
	Text string `json:"text,omitempty"`
}

// Prompt creates a new SdkMcpPrompt with the given configuration.
func Prompt(name, description string) *SdkMcpPrompt {
	return &SdkMcpPrompt{
		Name:        name,
		Description: description,
	}
}

// WithArgument adds an argument to the prompt.
func (p *SdkMcpPrompt) WithArgument(name, description string, required bool) *SdkMcpPrompt {
	p.Arguments = append(p.Arguments, PromptArgument{
		Name:        name,
		Description: description,
		Required:    required,
	})
	return p
}

// WithMessage adds a message to the prompt.
func (p *SdkMcpPrompt) WithMessage(role, text string) *SdkMcpPrompt {
	p.Messages = append(p.Messages, PromptMessage{
		Role:    role,
		Content: PromptContent{Type: "text", Text: text},
	})
	return p
}

// WithPrompts adds prompts to the server.
func WithPrompts(prompts ...*SdkMcpPrompt) ServerOption {
	return func(s *SdkMcpServerImpl) {
		for _, p := range prompts {
			s.prompts = append(s.prompts, p)
			s.promptMap[p.Name] = p
		}
	}
}

// ============================================================================
// MCP Logging (2025-11-25 Specification)
// ============================================================================

// LogLevel represents the severity of a log message.
type LogLevel string

const (
	LogLevelDebug     LogLevel = "debug"
	LogLevelInfo      LogLevel = "info"
	LogLevelNotice    LogLevel = "notice"
	LogLevelWarning   LogLevel = "warning"
	LogLevelError     LogLevel = "error"
	LogLevelCritical  LogLevel = "critical"
	LogLevelAlert     LogLevel = "alert"
	LogLevelEmergency LogLevel = "emergency"
	LogLevelFatal     LogLevel = "fatal"
)

// WithLogLevel sets the server's log level.
func WithLogLevel(level LogLevel) ServerOption {
	return func(s *SdkMcpServerImpl) {
		s.logLevel = level
	}
}

// ============================================================================
// Server Creation (Updated)
// ============================================================================
