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
	"sync"
	"time"

	"github.com/next-bin/claude-agent-sdk-golang/errors"
	"github.com/next-bin/claude-agent-sdk-golang/internal/messageparser"
	"github.com/next-bin/claude-agent-sdk-golang/internal/queryimpl"
	"github.com/next-bin/claude-agent-sdk-golang/internal/transportimpl"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// Client represents a Claude SDK client for bidirectional, interactive conversations.
// It provides full control over the conversation flow with support for streaming,
// interrupts, and dynamic message sending. For simple one-shot queries, consider
// using the Query function instead.
//
// Client supports bidirectional message passing, maintains conversation state,
// allows interactive follow-ups, and provides interrupt capabilities. It is
// suitable for building chat interfaces, interactive debugging sessions,
// multi-turn conversations, and real-time applications.
type Client struct {
	options         *types.ClaudeAgentOptions
	customTransport queryimpl.Transport
	transport       queryimpl.Transport
	query           *queryimpl.Query
	connected       bool

	// Cached message channel for fan-out to multiple subscribers
	messageChan     <-chan types.Message
	messageChanOnce bool

	// Goroutine management for channel prompts
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// New creates a new Claude SDK client with default options.
//
// Returns a client ready for connection. Use Connect() to establish
// a connection before sending messages.
//
// Example:
//
//	c := client.New()
//	defer c.Close()
//	err := c.Connect(ctx, "Hello, Claude!")
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
// It initializes the Claude CLI subprocess and prepares for message exchange.
//
// For interactive use without an initial prompt, call Connect with no arguments.
// For one-shot queries, provide a prompt string or a channel of message dictionaries.
//
// The prompt parameter can be:
//   - A string for a single message
//   - A channel of map[string]interface{} for streaming messages
//
// Connect returns an error if the connection fails or if permission settings
// are invalid (e.g., CanUseTool and PermissionPromptToolName are both set).
func (c *Client) Connect(ctx context.Context, prompt ...interface{}) error {
	if c.connected {
		return nil
	}

	options := c.options

	// Validate and configure permission settings
	if c.options.CanUseTool != nil {
		// canUseTool callback requires streaming mode (channel prompt, not string)
		if len(prompt) > 0 {
			if _, ok := prompt[0].(string); ok {
				return fmt.Errorf("can_use_tool callback requires streaming mode. Please provide prompt as a channel instead of a string")
			}
		}

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
		t, err := transportimpl.NewSubprocessCLITransport(promptVal, options)
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
	sdkMcpServers := make(map[string]queryimpl.McpServer)
	if c.options.MCPServers != nil {
		// Try map[string]types.McpServerConfig first (typed map)
		if servers, ok := c.options.MCPServers.(map[string]types.McpServerConfig); ok {
			for name, config := range servers {
				if sdkConfig, ok := config.(types.McpSdkServerConfig); ok {
					if srv, ok := sdkConfig.Instance.(queryimpl.McpServer); ok {
						sdkMcpServers[name] = srv
					}
				}
			}
		} else if servers, ok := c.options.MCPServers.(map[string]interface{}); ok {
			// Try map[string]interface{} (untyped map)
			for name, config := range servers {
				if cfg, ok := config.(map[string]interface{}); ok {
					if cfg["type"] == "sdk" {
						if instance, exists := cfg["instance"]; exists {
							if srv, ok := instance.(queryimpl.McpServer); ok {
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

	// Extract excludeDynamicSections from system prompt preset
	var excludeDynamicSections *bool
	if preset, ok := c.options.SystemPrompt.(types.SystemPromptPreset); ok && preset.ExcludeDynamicSections != nil {
		excludeDynamicSections = preset.ExcludeDynamicSections
	}

	// Convert hooks to internal format
	var hooks map[string][]queryimpl.HookMatcher
	if c.options.Hooks != nil {
		hooks = c.convertHooksToInternalFormat(c.options.Hooks)
	}

	// Convert canUseTool callback
	var canUseToolCb queryimpl.CanUseToolCallback
	if c.options.CanUseTool != nil {
		canUseToolCb = func(ctx context.Context, toolName string, input map[string]interface{}, context types.ToolPermissionContext) (types.PermissionResult, error) {
			return c.options.CanUseTool(toolName, input, context)
		}
	}

	// Create Query to handle control protocol
	c.query = queryimpl.NewQuery(
		c.transport,
		true, // ClaudeSDKClient always uses streaming mode
		canUseToolCb,
		hooks,
		sdkMcpServers,
		initializeTimeout,
		agentsDict,
		excludeDynamicSections,
	)

	// Start reading messages and initialize
	if err := c.query.Start(ctx); err != nil {
		return err
	}
	if _, err := c.query.Initialize(ctx); err != nil {
		return err
	}

	// Handle prompt input - send user message after initialize (matching upstream SDK behavior)
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
			// For channel prompts, stream input in background with proper lifecycle management
			// Use the provided context to ensure cancellation propagates correctly
			streamCtx, streamCancel := context.WithCancel(ctx)
			c.cancel = streamCancel
			c.wg.Add(1)
			go func() {
				defer c.wg.Done()
				for {
					select {
					case <-streamCtx.Done():
						return
					case msg, ok := <-p:
						if !ok {
							return
						}
						data, err := json.Marshal(msg)
						if err != nil {
							continue // Skip invalid messages
						}
						if err := c.transport.Write(streamCtx, string(data)+"\n"); err != nil {
							return // Exit on write error
						}
					}
				}
			}()
		}
	}

	c.connected = true
	return nil
}

// convertHooksToInternalFormat converts HookMatcher format to internal Query format.
func (c *Client) convertHooksToInternalFormat(hooks map[types.HookEvent][]types.HookMatcher) map[string][]queryimpl.HookMatcher {
	internalHooks := make(map[string][]queryimpl.HookMatcher)
	for event, matchers := range hooks {
		internalHooks[string(event)] = make([]queryimpl.HookMatcher, 0, len(matchers))
		for _, matcher := range matchers {
			// Convert hook callbacks - wrap interface to function type
			var callbacks []queryimpl.HookCallbackFunc
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
			internalMatcher := queryimpl.HookMatcher{
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
	case types.HookEventPermissionRequest:
		return parsePermissionRequestHookInput(inputMap)
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
	// Parse agent context fields (present when hook fires from sub-agent)
	if agentID, ok := m["agent_id"].(string); ok {
		input.AgentID = &agentID
	}
	if agentType, ok := m["agent_type"].(string); ok {
		input.AgentType = &agentType
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
	// Parse agent context fields (present when hook fires from sub-agent)
	if agentID, ok := m["agent_id"].(string); ok {
		input.AgentID = &agentID
	}
	if agentType, ok := m["agent_type"].(string); ok {
		input.AgentType = &agentType
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
	if toolUseID, ok := m["tool_use_id"].(string); ok {
		input.ToolUseID = toolUseID
	}
	if errMsg, ok := m["error"].(string); ok {
		input.Error = errMsg
	}
	// Parse agent context fields (present when hook fires from sub-agent)
	if agentID, ok := m["agent_id"].(string); ok {
		input.AgentID = &agentID
	}
	if agentType, ok := m["agent_type"].(string); ok {
		input.AgentType = &agentType
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

// parsePermissionRequestHookInput parses a map into PermissionRequestHookInput.
func parsePermissionRequestHookInput(m map[string]interface{}) types.PermissionRequestHookInput {
	input := types.PermissionRequestHookInput{
		HookEventName: "PermissionRequest",
	}
	if toolName, ok := m["tool_name"].(string); ok {
		input.ToolName = toolName
	}
	if toolInput, ok := m["tool_input"].(map[string]interface{}); ok {
		input.ToolInput = toolInput
	}
	if suggestions, ok := m["permission_suggestions"].([]interface{}); ok {
		input.PermissionSuggestions = suggestions
	}
	// Parse agent context fields (present when hook fires from sub-agent)
	if agentID, ok := m["agent_id"].(string); ok {
		input.AgentID = &agentID
	}
	if agentType, ok := m["agent_type"].(string); ok {
		input.AgentType = &agentType
	}
	return input
}

// ReceiveMessages returns a channel for receiving all messages from Claude.
// The channel yields Message types including AssistantMessage, UserMessage,
// SystemMessage, and ResultMessage. The channel is closed when the context
// is cancelled or the query completes.
//
// ReceiveMessages returns nil if the query is not initialized.
//
// Important: Call ReceiveMessages once and reuse the returned channel for
// multiple queries. Each call to ReceiveMessages returns the same underlying
// channel to avoid message distribution issues when multiple goroutines read
// from the same source.
func (c *Client) ReceiveMessages(ctx context.Context) <-chan types.Message {
	// Return cached channel if available
	if c.messageChanOnce && c.messageChan != nil {
		return c.messageChan
	}

	output := make(chan types.Message)
	c.messageChan = output
	c.messageChanOnce = true

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

// Query sends a new request in streaming mode. The prompt can be a string
// message or a channel of message dictionaries. The optional sessionID
// parameter specifies the session identifier; if not provided, it defaults to "default".
//
// For channel prompts, each message should have the structure:
//
//	map[string]interface{}{
//	    "type": "user",
//	    "message": map[string]interface{}{"role": "user", "content": "..."},
//	    "session_id": "...",
//	}
//
// Query returns an error if the client is not connected.
//
// Important: To receive responses, call ReceiveMessages() once before or after
// Query and reuse the returned channel for all queries. This matches the upstream
// SDK pattern where query() only sends messages and receive_messages() is called
// separately.
func (c *Client) Query(ctx context.Context, prompt interface{}, sessionID ...string) error {
	if !c.connected || c.query == nil || c.transport == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
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
		return c.transport.Write(ctx, string(data)+"\n")
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

	return nil
}

// Interrupt sends an interrupt signal to the running conversation.
// It only works in streaming mode. Interrupt returns an error if the client
// is not connected.
func (c *Client) Interrupt(ctx context.Context) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.Interrupt(ctx)
}

// SetPermissionMode changes the permission mode during a conversation.
// It only works in streaming mode.
//
// The mode parameter specifies the permission mode to set. Valid options are:
//   - "default": CLI prompts for dangerous tools
//   - "acceptEdits": auto-accept file edits
//   - "bypassPermissions": allow all tools (use with caution)
//
// SetPermissionMode returns an error if the client is not connected.
func (c *Client) SetPermissionMode(ctx context.Context, mode string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.SetPermissionMode(ctx, mode)
}

// SetModel changes the AI model during a conversation.
// It only works in streaming mode.
//
// The model parameter specifies the model to use. Examples include:
//   - "claude-opus-4-6" (latest flagship, most intelligent)
//   - "claude-sonnet-4-6" (best balance of speed and intelligence)
//   - "claude-haiku-4-5-20251001" (fastest)
//
// You can also use the model constants defined in the types package:
//   - types.ModelClaudeOpus46
//   - types.ModelClaudeSonnet46
//   - types.ModelClaudeHaiku45
//
// SetModel returns an error if the client is not connected.
func (c *Client) SetModel(ctx context.Context, model string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.SetModel(ctx, model)
}

// RewindFiles rewinds tracked files to their state at a specific user message.
// It only works in streaming mode.
//
// Requirements:
//   - EnableFileCheckpointing must be true to track file changes
//   - ExtraArgs must include "replay-user-messages" to receive UserMessage
//     objects with UUID in the response stream
//
// The userMessageID parameter is the UUID of the user message to rewind to.
// This should be the UUID field from a UserMessage received during the conversation.
//
// RewindFiles returns an error if the client is not connected.
func (c *Client) RewindFiles(ctx context.Context, userMessageID string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.RewindFiles(ctx, userMessageID)
}

// ReconnectMCPServer reconnects a disconnected or failed MCP server.
// It only works in streaming mode.
//
// Use this to retry connecting to an MCP server that failed to connect
// or was disconnected. Returns an exception if the reconnection fails.
//
// The serverName parameter is the name of the MCP server to reconnect.
//
// ReconnectMCPServer returns an error if the client is not connected.
func (c *Client) ReconnectMCPServer(ctx context.Context, serverName string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.ReconnectMCPServer(ctx, serverName)
}

// ToggleMCPServer enables or disables an MCP server.
// It only works in streaming mode.
//
// Disabling a server disconnects it and removes its tools from the
// available tool set. Enabling a server reconnects it and makes its
// tools available again. Returns an exception on failure.
//
// The serverName parameter is the name of the MCP server to toggle.
// The enabled parameter is true to enable the server, false to disable it.
//
// ToggleMCPServer returns an error if the client is not connected.
func (c *Client) ToggleMCPServer(ctx context.Context, serverName string, enabled bool) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.ToggleMCPServer(ctx, serverName, enabled)
}

// StopTask stops a running task.
// It only works in streaming mode.
//
// After this resolves, a task_notification system message with
// status 'stopped' will be emitted by the CLI in the message stream.
//
// The taskID parameter is the task ID from task_notification events.
//
// StopTask returns an error if the client is not connected.
func (c *Client) StopTask(ctx context.Context, taskID string) error {
	if !c.connected || c.query == nil {
		return errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.StopTask(ctx, taskID)
}

// GetMCPStatus returns the current MCP server connection status.
// It only works in streaming mode.
//
// GetMCPStatus queries the Claude Code CLI for the live connection status
// of all configured MCP servers.
//
// The returned map contains an "mcpServers" key with a list of server status
// objects, each having:
//   - "name": server name (string)
//   - "status": connection status ("connected", "pending", "failed",
//     "needs-auth", or "disabled")
//
// GetMCPStatus returns an error if the client is not connected.
func (c *Client) GetMCPStatus(ctx context.Context) (map[string]interface{}, error) {
	if !c.connected || c.query == nil {
		return nil, errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.GetMCPStatus(ctx)
}

// GetContextUsage returns a breakdown of current context window usage by category.
//
// Returns the same data shown by the /context command in the CLI,
// including token counts per category, total usage, and detailed
// breakdowns of MCP tools, memory files, and agents.
//
// GetContextUsage returns an error if the client is not connected.
func (c *Client) GetContextUsage(ctx context.Context) (types.ContextUsageResponse, error) {
	if !c.connected || c.query == nil {
		return types.ContextUsageResponse{}, errors.NewCLIConnectionError("Not connected. Call Connect() first.", nil)
	}
	return c.query.GetContextUsage(ctx)
}

// GetServerInfo returns server initialization information including:
//   - Available commands (slash commands, system commands, etc.)
//   - Current and available output styles
//   - Server capabilities
//
// GetServerInfo returns nil if not in streaming mode or if the query is not initialized.
func (c *Client) GetServerInfo() map[string]interface{} {
	if c.query == nil {
		return nil
	}
	return c.query.GetInitializationResult()
}

// ReceiveResponse receives messages until and including a ResultMessage.
// It is a convenience method over ReceiveMessages for single-response workflows.
//
// ReceiveResponse yields all messages in sequence and automatically terminates
// after yielding a ResultMessage (which indicates the response is complete).
// The ResultMessage is included in the yielded messages. If no ResultMessage
// is received, the iterator continues indefinitely.
//
// ReceiveResponse returns a channel of Message types (UserMessage, AssistantMessage,
// SystemMessage, or ResultMessage).
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

// Disconnect closes the connection to Claude and releases associated resources.
// It is safe to call Disconnect multiple times; subsequent calls are no-ops.
func (c *Client) Disconnect(ctx context.Context) error {
	if !c.connected {
		return nil
	}

	// Cancel any running goroutines
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	// Wait for goroutines to finish
	c.wg.Wait()

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
// It is an alias for Disconnect for convenience and idiomatically matches
// the io.Closer interface.
func (c *Client) Close() error {
	return c.Disconnect(context.Background())
}
