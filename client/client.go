// Package client provides the Claude SDK client for interactive sessions.
//
// The client package handles communication with the Claude CLI and provides
// an interface for sending queries and receiving responses.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/unitsvc/claude-agent-sdk-golang/errors"
	"github.com/unitsvc/claude-agent-sdk-golang/internal/messageparser"
	"github.com/unitsvc/claude-agent-sdk-golang/internal/query"
	"github.com/unitsvc/claude-agent-sdk-golang/internal/transport"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// Client represents a Claude SDK client for bidirectional, interactive conversations.
//
// This client provides full control over the conversation flow with support
// for streaming, interrupts, and dynamic message sending.
type Client struct {
	options         *types.ClaudeAgentOptions
	customTransport query.Transport
	transport       query.Transport
	query           *query.Query
	connected       bool
}

// New creates a new Claude SDK client with default options.
func New() *Client {
	return NewWithOptions(nil)
}

// NewWithOptions creates a new Claude SDK client with custom options.
func NewWithOptions(opts *types.ClaudeAgentOptions) *Client {
	if opts == nil {
		opts = &types.ClaudeAgentOptions{}
	}
	return &Client{
		options: opts,
	}
}

// Options returns the current client options.
func (c *Client) Options() *types.ClaudeAgentOptions {
	return c.options
}

// Connect establishes a connection to Claude with an optional prompt.
//
// For interactive use without an initial prompt, call Connect() with no arguments.
// For one-shot queries, provide a prompt string or channel.
func (c *Client) Connect(ctx context.Context, prompt ...interface{}) error {
	if c.connected {
		return nil
	}

	// Set entrypoint environment variable
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go-client")

	options := c.options

	// Validate and configure permission settings
	if c.options.CanUseTool != nil {
		// canUseTool and permission_prompt_tool_name are mutually exclusive
		if c.options.PermissionPromptToolName != nil && *c.options.PermissionPromptToolName != "" {
			return fmt.Errorf("can_use_tool callback cannot be used with permission_prompt_tool_name. Please use one or the other")
		}

		// Automatically set permission_prompt_tool_name to "stdio" for control protocol
		optsCopy := *c.options
		stdio := "stdio"
		optsCopy.PermissionPromptToolName = &stdio
		options = &optsCopy
	}

	// Use provided custom transport or create subprocess transport
	if c.customTransport != nil {
		c.transport = c.customTransport
	} else {
		var promptVal interface{}
		if len(prompt) > 0 {
			promptVal = prompt[0]
		}
		t, err := transport.NewSubprocessCLITransport(promptVal, options)
		if err != nil {
			return err
		}
		// Connect the transport before using it
		if err := t.Connect(ctx); err != nil {
			return err
		}
		c.transport = t
	}

	// Extract SDK MCP servers from options
	sdkMcpServers := make(map[string]query.McpServer)
	if c.options.MCPServers != nil {
		if servers, ok := c.options.MCPServers.(map[string]interface{}); ok {
			for name, config := range servers {
				if cfg, ok := config.(map[string]interface{}); ok {
					if cfg["type"] == "sdk" {
						if instance, exists := cfg["instance"]; exists {
							if srv, ok := instance.(query.McpServer); ok {
								sdkMcpServers[name] = srv
							}
						}
					}
				}
			}
		}
	}

	// Calculate initialize timeout from environment variable
	initializeTimeout := 60 * time.Second
	if timeoutStr := os.Getenv("CLAUDE_CODE_STREAM_CLOSE_TIMEOUT"); timeoutStr != "" {
		var timeoutMs int
		if _, err := fmt.Sscanf(timeoutStr, "%d", &timeoutMs); err == nil {
			d := time.Duration(timeoutMs) * time.Millisecond
			if d < 60*time.Second {
				d = 60 * time.Second
			}
			initializeTimeout = d
		}
	}

	// Convert agents to dict format
	var agentsDict map[string]map[string]interface{}
	if c.options.Agents != nil {
		agentsDict = make(map[string]map[string]interface{})
		for name, agentDef := range c.options.Agents {
			agentsDict[name] = map[string]interface{}{
				"description": agentDef.Description,
				"prompt":      agentDef.Prompt,
			}
			if agentDef.Tools != nil {
				agentsDict[name]["tools"] = agentDef.Tools
			}
			if agentDef.Model != nil {
				agentsDict[name]["model"] = *agentDef.Model
			}
		}
	}

	// Convert hooks to internal format
	var hooks map[string][]query.HookMatcher
	if c.options.Hooks != nil {
		hooks = c.convertHooksToInternalFormat(c.options.Hooks)
	}

	// Convert canUseTool callback
	var canUseToolCb query.CanUseToolCallback
	if c.options.CanUseTool != nil {
		canUseToolCb = func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
			return c.options.CanUseTool(toolName, input, context)
		}
	}

	// Create Query to handle control protocol
	c.query = query.NewQuery(
		c.transport,
		true, // ClaudeSDKClient always uses streaming mode
		canUseToolCb,
		hooks,
		sdkMcpServers,
		initializeTimeout,
		agentsDict,
	)

	// Start reading messages and initialize
	if err := c.query.Start(ctx); err != nil {
		return err
	}
	if _, err := c.query.Initialize(ctx); err != nil {
		return err
	}

	// Handle prompt input - send user message after initialize (matching Python SDK behavior)
	if len(prompt) > 0 {
		switch p := prompt[0].(type) {
		case string:
			// For string prompts, write user message to stdin after initialize
			userMessage := map[string]interface{}{
				"type":               "user",
				"session_id":         "",
				"message":            map[string]interface{}{"role": "user", "content": p},
				"parent_tool_use_id": nil,
			}
			data, err := json.Marshal(userMessage)
			if err != nil {
				return fmt.Errorf("failed to marshal user message: %w", err)
			}
			if err := c.transport.Write(ctx, string(data)+"\n"); err != nil {
				return fmt.Errorf("failed to write user message: %w", err)
			}
			// End input after sending the prompt
			if err := c.transport.EndInput(ctx); err != nil {
				return fmt.Errorf("failed to end input: %w", err)
			}
		case chan map[string]interface{}:
			// For channel prompts, stream input in background
			go func() {
				for msg := range p {
					data, _ := json.Marshal(msg)
					c.transport.Write(ctx, string(data)+"\n")
				}
			}()
		}
	}

	c.connected = true
	return nil
}

// convertHooksToInternalFormat converts HookMatcher format to internal Query format.
func (c *Client) convertHooksToInternalFormat(hooks map[types.HookEvent][]types.HookMatcher) map[string][]query.HookMatcher {
	internalHooks := make(map[string][]query.HookMatcher)
	for event, matchers := range hooks {
		internalHooks[string(event)] = make([]query.HookMatcher, 0, len(matchers))
		for _, matcher := range matchers {
			// Convert hook callbacks - wrap interface to function type
			var callbacks []query.HookCallbackFunc
			for _, cb := range matcher.Hooks {
				// Create a wrapper that adapts the HookCallback interface to HookCallbackFunc
				callback := cb // capture loop variable
				wrapper := func(ctx context.Context, input interface{}, toolUseID *string, context types.HookContext) (map[string]interface{}, error) {
					// Convert input to HookInput based on the input map
					hookInput := convertToHookInput(input)
					result, err := callback.Execute(hookInput, toolUseID, context)
					if err != nil {
						return nil, err
					}
					// Convert HookJSONOutput to map[string]interface{}
					return hookOutputToMap(result), nil
				}
				callbacks = append(callbacks, wrapper)
			}
			internalMatcher := query.HookMatcher{
				Matcher: matcher.Matcher,
				Hooks:   callbacks,
			}
			if matcher.Timeout != nil {
				internalMatcher.Timeout = matcher.Timeout
			}
			internalHooks[string(event)] = append(internalHooks[string(event)], internalMatcher)
		}
	}
	return internalHooks
}

// convertToHookInput converts a map input to the appropriate HookInput type.
func convertToHookInput(input interface{}) types.HookInput {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil
	}

	hookEventName, _ := inputMap["hook_event_name"].(string)

	switch types.HookEvent(hookEventName) {
	case types.HookEventPreToolUse:
		return parsePreToolUseHookInput(inputMap)
	case types.HookEventPostToolUse:
		return parsePostToolUseHookInput(inputMap)
	case types.HookEventPostToolUseFailure:
		return parsePostToolUseFailureHookInput(inputMap)
	case types.HookEventUserPromptSubmit:
		return parseUserPromptSubmitHookInput(inputMap)
	default:
		// Return a generic input with just the event name
		return &genericHookInput{hookEventName: hookEventName, data: inputMap}
	}
}

// hookOutputToMap converts HookJSONOutput to map[string]interface{}.
func hookOutputToMap(output types.HookJSONOutput) map[string]interface{} {
	result := make(map[string]interface{})

	switch o := output.(type) {
	case types.AsyncHookJSONOutput:
		result["async"] = o.Async_
		if o.AsyncTimeout != nil {
			result["asyncTimeout"] = *o.AsyncTimeout
		}
	case types.SyncHookJSONOutput:
		if o.Continue_ != nil {
			result["continue"] = *o.Continue_
		}
		if o.SuppressOutput != nil {
			result["suppressOutput"] = *o.SuppressOutput
		}
		if o.StopReason != nil {
			result["stopReason"] = *o.StopReason
		}
		if o.Decision != nil {
			result["decision"] = *o.Decision
		}
		if o.SystemMessage != nil {
			result["systemMessage"] = *o.SystemMessage
		}
		if o.Reason != nil {
			result["reason"] = *o.Reason
		}
		if o.HookSpecificOutput != nil {
			result["hookSpecificOutput"] = hookSpecificOutputToMap(o.HookSpecificOutput)
		}
	}

	return result
}

// hookSpecificOutputToMap converts HookSpecificOutput to map[string]interface{}.
func hookSpecificOutputToMap(output types.HookSpecificOutput) map[string]interface{} {
	result := make(map[string]interface{})

	switch o := output.(type) {
	case types.PreToolUseHookSpecificOutput:
		if o.PermissionDecision != nil {
			result["permissionDecision"] = *o.PermissionDecision
		}
		if o.PermissionDecisionReason != nil {
			result["permissionDecisionReason"] = *o.PermissionDecisionReason
		}
		if o.UpdatedInput != nil {
			result["updatedInput"] = o.UpdatedInput
		}
		if o.AdditionalContext != nil {
			result["additionalContext"] = *o.AdditionalContext
		}
	case types.PostToolUseHookSpecificOutput:
		if o.AdditionalContext != nil {
			result["additionalContext"] = *o.AdditionalContext
		}
		if o.UpdatedMCPToolOutput != nil {
			result["updatedMCPToolOutput"] = o.UpdatedMCPToolOutput
		}
	case types.PostToolUseFailureHookSpecificOutput:
		if o.AdditionalContext != nil {
			result["additionalContext"] = *o.AdditionalContext
		}
	case types.UserPromptSubmitHookSpecificOutput:
		if o.AdditionalContext != nil {
			result["additionalContext"] = *o.AdditionalContext
		}
	}

	return result
}

// genericHookInput is a fallback for unrecognized hook input types.
type genericHookInput struct {
	hookEventName string
	data          map[string]interface{}
}

func (g *genericHookInput) GetHookEventName() string { return g.hookEventName }

// parsePreToolUseHookInput parses a map into PreToolUseHookInput.
func parsePreToolUseHookInput(m map[string]interface{}) types.PreToolUseHookInput {
	input := types.PreToolUseHookInput{
		HookEventName: "PreToolUse",
	}
	if toolName, ok := m["tool_name"].(string); ok {
		input.ToolName = toolName
	}
	if toolInput, ok := m["tool_input"].(map[string]interface{}); ok {
		input.ToolInput = toolInput
	}
	if toolUseID, ok := m["tool_use_id"].(string); ok {
		input.ToolUseID = toolUseID
	}
	// Parse BaseHookInput fields
	if sessionID, ok := m["session_id"].(string); ok {
		input.SessionID = sessionID
	}
	if transcriptPath, ok := m["transcript_path"].(string); ok {
		input.TranscriptPath = transcriptPath
	}
	if cwd, ok := m["cwd"].(string); ok {
		input.CWD = cwd
	}
	if permMode, ok := m["permission_mode"].(string); ok {
		input.PermissionMode = &permMode
	}
	return input
}

// parsePostToolUseHookInput parses a map into PostToolUseHookInput.
func parsePostToolUseHookInput(m map[string]interface{}) types.PostToolUseHookInput {
	input := types.PostToolUseHookInput{
		HookEventName: "PostToolUse",
	}
	if toolName, ok := m["tool_name"].(string); ok {
		input.ToolName = toolName
	}
	if toolInput, ok := m["tool_input"].(map[string]interface{}); ok {
		input.ToolInput = toolInput
	}
	if toolResponse, ok := m["tool_response"]; ok {
		input.ToolResponse = toolResponse
	}
	if toolUseID, ok := m["tool_use_id"].(string); ok {
		input.ToolUseID = toolUseID
	}
	return input
}

// parsePostToolUseFailureHookInput parses a map into PostToolUseFailureHookInput.
func parsePostToolUseFailureHookInput(m map[string]interface{}) types.PostToolUseFailureHookInput {
	input := types.PostToolUseFailureHookInput{
		HookEventName: "PostToolUseFailure",
	}
	if toolName, ok := m["tool_name"].(string); ok {
		input.ToolName = toolName
	}
	if toolInput, ok := m["tool_input"].(map[string]interface{}); ok {
		input.ToolInput = toolInput
	}
	if errMsg, ok := m["error"].(string); ok {
		input.Error = errMsg
	}
	return input
}

// parseUserPromptSubmitHookInput parses a map into UserPromptSubmitHookInput.
func parseUserPromptSubmitHookInput(m map[string]interface{}) types.UserPromptSubmitHookInput {
	input := types.UserPromptSubmitHookInput{
		HookEventName: "UserPromptSubmit",
	}
	if prompt, ok := m["prompt"].(string); ok {
		input.Prompt = prompt
	}
	return input
}

// ReceiveMessages returns a channel for receiving all messages from Claude.
func (c *Client) ReceiveMessages(ctx context.Context) <-chan types.Message {
	output := make(chan types.Message)

	go func() {
		defer close(output)
		if c.query == nil {
			return
		}

		msgChan := c.query.ReceiveMessages(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-msgChan:
				if !ok {
					return
				}
				message, err := messageparser.ParseMessage(data)
				if err != nil {
					continue
				}
				if message != nil {
					output <- message
				}
			}
		}
	}()

	return output
}

// Query sends a new request in streaming mode.
//
// prompt can be a string or a channel of message dictionaries.
// sessionID is an optional session identifier for the conversation.
func (c *Client) Query(ctx context.Context, prompt interface{}, sessionID ...string) (<-chan types.Message, error) {
	if !c.connected || c.query == nil || c.transport == nil {
		return nil, errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}

	sid := "default"
	if len(sessionID) > 0 && sessionID[0] != "" {
		sid = sessionID[0]
	}

	// Handle string prompts
	if str, ok := prompt.(string); ok {
		message := map[string]interface{}{
			"type":               "user",
			"message":            map[string]interface{}{"role": "user", "content": str},
			"parent_tool_use_id": nil,
			"session_id":         sid,
		}
		data, _ := json.Marshal(message)
		if err := c.transport.Write(ctx, string(data)+"\n"); err != nil {
			return nil, err
		}
		return c.ReceiveMessages(ctx), nil
	}

	// Handle channel prompts
	if ch, ok := prompt.(chan map[string]interface{}); ok {
		go func() {
			for msg := range ch {
				if _, exists := msg["session_id"]; !exists {
					msg["session_id"] = sid
				}
				data, _ := json.Marshal(msg)
				c.transport.Write(ctx, string(data)+"\n")
			}
		}()
	}

	return c.ReceiveMessages(ctx), nil
}

// Interrupt sends an interrupt signal (only works with streaming mode).
func (c *Client) Interrupt(ctx context.Context) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.Interrupt(ctx)
}

// SetPermissionMode changes permission mode during conversation.
//
// Valid modes: "default", "acceptEdits", "plan", "bypassPermissions"
func (c *Client) SetPermissionMode(ctx context.Context, mode string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.SetPermissionMode(ctx, mode)
}

// SetModel changes the AI model during conversation.
func (c *Client) SetModel(ctx context.Context, model string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.SetModel(ctx, model)
}

// RewindFiles rewinds tracked files to their state at a specific user message.
//
// Requires enable_file_checkpointing=True to track file changes.
func (c *Client) RewindFiles(ctx context.Context, userMessageID string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.RewindFiles(ctx, userMessageID)
}

// GetMCPStatus gets current MCP server connection status.
func (c *Client) GetMCPStatus(ctx context.Context) (map[string]interface{}, error) {
	if !c.connected || c.query == nil {
		return nil, errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.GetMCPStatus(ctx)
}

// GetServerInfo returns server initialization info.
func (c *Client) GetServerInfo() map[string]interface{} {
	if c.query == nil {
		return nil
	}
	return c.query.GetInitializationResult()
}

// ReceiveResponse receives messages until and including a ResultMessage.
//
// This is a convenience method over ReceiveMessages for single-response workflows.
func (c *Client) ReceiveResponse(ctx context.Context) <-chan types.Message {
	output := make(chan types.Message)

	go func() {
		defer close(output)
		msgChan := c.ReceiveMessages(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				output <- msg
				if _, isResult := msg.(*types.ResultMessage); isResult {
					return
				}
			}
		}
	}()

	return output
}

// Disconnect closes the connection to Claude.
func (c *Client) Disconnect(ctx context.Context) error {
	if !c.connected {
		return nil
	}

	var err error
	if c.query != nil {
		err = c.query.Close(ctx)
		c.query = nil
	}
	c.transport = nil
	c.connected = false
	return err
}

// Close releases any resources held by the client.
// Alias for Disconnect for convenience.
func (c *Client) Close() error {
	return c.Disconnect(context.Background())
}
