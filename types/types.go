// Package types defines the core types used throughout the Claude Agent SDK.
//
// This package contains type definitions for messages, tool calls, permissions,
// hooks, MCP server configurations, and other data structures used in the SDK.
package types

import "encoding/json"

// ============================================================================
// Type Aliases and Constants
// ============================================================================

// PermissionMode represents the permission mode for the SDK.
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// SdkBeta represents SDK beta features - see https://docs.anthropic.com/en/api/beta-headers
type SdkBeta string

const (
	SdkBetaContext1M SdkBeta = "context-1m-2025-08-07"
)

// SettingSource represents the source of a setting.
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// ============================================================================
// Agent Definition Types
// ============================================================================

// SystemPromptPreset represents a system prompt preset configuration.
type SystemPromptPreset struct {
	Type   string  `json:"type"`             // Always "preset"
	Preset string  `json:"preset"`           // Always "claude_code"
	Append *string `json:"append,omitempty"` // Optional append text
}

// ToolsPreset represents a tools preset configuration.
type ToolsPreset struct {
	Type   string `json:"type"`   // Always "preset"
	Preset string `json:"preset"` // Always "claude_code"
}

// AgentDefinition represents an agent definition configuration.
type AgentDefinition struct {
	Description string   `json:"description"`
	Prompt      string   `json:"prompt"`
	Tools       []string `json:"tools,omitempty"`
	Model       *string  `json:"model,omitempty"` // "sonnet", "opus", "haiku", or "inherit"
}

// ============================================================================
// Permission Update Types
// ============================================================================

// PermissionUpdateDestination represents where permission updates should be applied.
type PermissionUpdateDestination string

const (
	PermissionUpdateDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionUpdateDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionUpdateDestinationSession         PermissionUpdateDestination = "session"
)

// PermissionBehavior represents the behavior for a permission rule.
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateType represents the type of permission update operation.
type PermissionUpdateType string

const (
	PermissionUpdateTypeAddRules          PermissionUpdateType = "addRules"
	PermissionUpdateTypeReplaceRules      PermissionUpdateType = "replaceRules"
	PermissionUpdateTypeRemoveRules       PermissionUpdateType = "removeRules"
	PermissionUpdateTypeSetMode           PermissionUpdateType = "setMode"
	PermissionUpdateTypeAddDirectories    PermissionUpdateType = "addDirectories"
	PermissionUpdateTypeRemoveDirectories PermissionUpdateType = "removeDirectories"
)

// PermissionRuleValue represents a permission rule value.
type PermissionRuleValue struct {
	ToolName    string  `json:"toolName"`
	RuleContent *string `json:"ruleContent,omitempty"`
}

// PermissionUpdate represents a permission update configuration.
type PermissionUpdate struct {
	Type        PermissionUpdateType         `json:"type"`
	Rules       []PermissionRuleValue        `json:"rules,omitempty"`
	Behavior    *PermissionBehavior          `json:"behavior,omitempty"`
	Mode        *PermissionMode              `json:"mode,omitempty"`
	Directories []string                     `json:"directories,omitempty"`
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// ToDict converts PermissionUpdate to a map for JSON serialization.
func (p *PermissionUpdate) ToDict() map[string]interface{} {
	result := map[string]interface{}{
		"type": p.Type,
	}

	if p.Destination != nil {
		result["destination"] = *p.Destination
	}

	switch p.Type {
	case PermissionUpdateTypeAddRules, PermissionUpdateTypeReplaceRules, PermissionUpdateTypeRemoveRules:
		if p.Rules != nil {
			rules := make([]map[string]interface{}, len(p.Rules))
			for i, rule := range p.Rules {
				ruleMap := map[string]interface{}{
					"toolName": rule.ToolName,
				}
				if rule.RuleContent != nil {
					ruleMap["ruleContent"] = *rule.RuleContent
				}
				rules[i] = ruleMap
			}
			result["rules"] = rules
		}
		if p.Behavior != nil {
			result["behavior"] = *p.Behavior
		}
	case PermissionUpdateTypeSetMode:
		if p.Mode != nil {
			result["mode"] = *p.Mode
		}
	case PermissionUpdateTypeAddDirectories, PermissionUpdateTypeRemoveDirectories:
		if p.Directories != nil {
			result["directories"] = p.Directories
		}
	}

	return result
}

// ============================================================================
// Tool Permission Types
// ============================================================================

// ToolPermissionContext provides context information for tool permission callbacks.
type ToolPermissionContext struct {
	Signal      interface{}        `json:"signal,omitempty"`      // Future: abort signal support
	Suggestions []PermissionUpdate `json:"suggestions,omitempty"` // Permission suggestions from CLI
}

// PermissionResultAllow represents an allow permission result.
type PermissionResultAllow struct {
	Behavior           string                 `json:"behavior"` // Always "allow"
	UpdatedInput       map[string]interface{} `json:"updatedInput,omitempty"`
	UpdatedPermissions []PermissionUpdate     `json:"updatedPermissions,omitempty"`
}

// PermissionResultDeny represents a deny permission result.
type PermissionResultDeny struct {
	Behavior  string `json:"behavior"` // Always "deny"
	Message   string `json:"message"`
	Interrupt bool   `json:"interrupt"`
}

// PermissionResult is a union type of PermissionResultAllow and PermissionResultDeny.
// In Go, we use an interface and check the Behavior field to determine the type.
type PermissionResult interface {
	GetBehavior() string
}

// GetBehavior returns the behavior for PermissionResultAllow.
func (p PermissionResultAllow) GetBehavior() string { return p.Behavior }

// GetBehavior returns the behavior for PermissionResultDeny.
func (p PermissionResultDeny) GetBehavior() string { return p.Behavior }

// ============================================================================
// Hook Types
// ============================================================================

// HookEvent represents the type of hook event.
type HookEvent string

const (
	HookEventPreToolUse         HookEvent = "PreToolUse"
	HookEventPostToolUse        HookEvent = "PostToolUse"
	HookEventPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookEventUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookEventStop               HookEvent = "Stop"
	HookEventSubagentStop       HookEvent = "SubagentStop"
	HookEventPreCompact         HookEvent = "PreCompact"
	HookEventNotification       HookEvent = "Notification"
	HookEventSubagentStart      HookEvent = "SubagentStart"
	HookEventPermissionRequest  HookEvent = "PermissionRequest"
	HookEventSessionStart       HookEvent = "SessionStart"
	HookEventSessionEnd         HookEvent = "SessionEnd"
)

// BaseHookInput contains base hook input fields present across many hook events.
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	CWD            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
}

// PreToolUseHookInput represents input data for PreToolUse hook events.
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // Always "PreToolUse"
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolUseID     string                 `json:"tool_use_id"`
}

// PostToolUseHookInput represents input data for PostToolUse hook events.
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // Always "PostToolUse"
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolResponse  interface{}            `json:"tool_response"`
	ToolUseID     string                 `json:"tool_use_id"`
}

// PostToolUseFailureHookInput represents input data for PostToolUseFailure hook events.
type PostToolUseFailureHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // Always "PostToolUseFailure"
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolUseID     string                 `json:"tool_use_id"`
	Error         string                 `json:"error"`
	IsInterrupt   *bool                  `json:"is_interrupt,omitempty"`
}

// UserPromptSubmitHookInput represents input data for UserPromptSubmit hook events.
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // Always "UserPromptSubmit"
	Prompt        string `json:"prompt"`
}

// StopHookInput represents input data for Stop hook events.
type StopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // Always "Stop"
	StopHookActive bool   `json:"stop_hook_active"`
}

// SubagentStopHookInput represents input data for SubagentStop hook events.
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName       string `json:"hook_event_name"` // Always "SubagentStop"
	StopHookActive      bool   `json:"stop_hook_active"`
	AgentID             string `json:"agent_id"`
	AgentTranscriptPath string `json:"agent_transcript_path"`
	AgentType           string `json:"agent_type"`
}

// PreCompactHookInput represents input data for PreCompact hook events.
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      string  `json:"hook_event_name"` // Always "PreCompact"
	Trigger            string  `json:"trigger"`         // "manual" or "auto"
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// NotificationHookInput represents input data for Notification hook events.
type NotificationHookInput struct {
	BaseHookInput
	HookEventName    string  `json:"hook_event_name"` // Always "Notification"
	Message          string  `json:"message"`
	Title            *string `json:"title,omitempty"`
	NotificationType string  `json:"notification_type"`
}

// SubagentStartHookInput represents input data for SubagentStart hook events.
type SubagentStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // Always "SubagentStart"
	AgentID       string `json:"agent_id"`
	AgentType     string `json:"agent_type"`
}

// PermissionRequestHookInput represents input data for PermissionRequest hook events.
type PermissionRequestHookInput struct {
	BaseHookInput
	HookEventName         string                 `json:"hook_event_name"` // Always "PermissionRequest"
	ToolName              string                 `json:"tool_name"`
	ToolInput             map[string]interface{} `json:"tool_input"`
	PermissionSuggestions []interface{}          `json:"permission_suggestions,omitempty"`
}

// SessionStartHookInput represents input data for SessionStart hook events.
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // Always "SessionStart"
}

// SessionEndHookInput represents input data for SessionEnd hook events.
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // Always "SessionEnd"
}

// HookInput is a union type for all hook input types.
// Use the HookEventName field to determine the concrete type.
type HookInput interface {
	GetHookEventName() string
}

// GetHookEventName returns the hook event name for PreToolUseHookInput.
func (h PreToolUseHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for PostToolUseHookInput.
func (h PostToolUseHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for PostToolUseFailureHookInput.
func (h PostToolUseFailureHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for UserPromptSubmitHookInput.
func (h UserPromptSubmitHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for StopHookInput.
func (h StopHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for SubagentStopHookInput.
func (h SubagentStopHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for PreCompactHookInput.
func (h PreCompactHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for NotificationHookInput.
func (h NotificationHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for SubagentStartHookInput.
func (h SubagentStartHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for PermissionRequestHookInput.
func (h PermissionRequestHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for SessionStartHookInput.
func (h SessionStartHookInput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name for SessionEndHookInput.
func (h SessionEndHookInput) GetHookEventName() string { return h.HookEventName }

// ============================================================================
// Hook Output Types
// ============================================================================

// PreToolUseHookSpecificOutput represents hook-specific output for PreToolUse events.
type PreToolUseHookSpecificOutput struct {
	HookEventName            string                 `json:"hookEventName"`                // Always "PreToolUse"
	PermissionDecision       *string                `json:"permissionDecision,omitempty"` // "allow", "deny", or "ask"
	PermissionDecisionReason *string                `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             map[string]interface{} `json:"updatedInput,omitempty"`
	AdditionalContext        *string                `json:"additionalContext,omitempty"`
}

// PostToolUseHookSpecificOutput represents hook-specific output for PostToolUse events.
type PostToolUseHookSpecificOutput struct {
	HookEventName        string      `json:"hookEventName"` // Always "PostToolUse"
	AdditionalContext    *string     `json:"additionalContext,omitempty"`
	UpdatedMCPToolOutput interface{} `json:"updatedMCPToolOutput,omitempty"`
}

// PostToolUseFailureHookSpecificOutput represents hook-specific output for PostToolUseFailure events.
type PostToolUseFailureHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // Always "PostToolUseFailure"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// UserPromptSubmitHookSpecificOutput represents hook-specific output for UserPromptSubmit events.
type UserPromptSubmitHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // Always "UserPromptSubmit"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// SessionStartHookSpecificOutput represents hook-specific output for SessionStart events.
type SessionStartHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // Always "SessionStart"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// NotificationHookSpecificOutput represents hook-specific output for Notification events.
type NotificationHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // Always "Notification"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// SubagentStartHookSpecificOutput represents hook-specific output for SubagentStart events.
type SubagentStartHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // Always "SubagentStart"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// PermissionRequestHookSpecificOutput represents hook-specific output for PermissionRequest events.
type PermissionRequestHookSpecificOutput struct {
	HookEventName string                 `json:"hookEventName"` // Always "PermissionRequest"
	Decision      map[string]interface{} `json:"decision"`
}

// SessionEndHookSpecificOutput represents hook-specific output for SessionEnd events.
type SessionEndHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // Always "SessionEnd"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// HookSpecificOutput is a union type for all hook-specific output types.
type HookSpecificOutput interface {
	GetHookEventName() string
}

// GetHookEventName returns the hook event name.
func (h PreToolUseHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h PostToolUseHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h PostToolUseFailureHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h UserPromptSubmitHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h SessionStartHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h NotificationHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h SubagentStartHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h PermissionRequestHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// GetHookEventName returns the hook event name.
func (h SessionEndHookSpecificOutput) GetHookEventName() string { return h.HookEventName }

// ============================================================================
// Hook JSON Output Types
// ============================================================================

// AsyncHookJSONOutput represents async hook output that defers hook execution.
// See https://docs.anthropic.com/en/docs/claude-code/hooks#advanced%3A-json-output
type AsyncHookJSONOutput struct {
	Async_       bool `json:"async"` // Using Async_ to match Python's async_
	AsyncTimeout *int `json:"asyncTimeout,omitempty"`
}

// SyncHookJSONOutput represents synchronous hook output with control and decision fields.
//
// Common Control Fields:
//   - Continue_: SuppressOutput, StopReason: Control execution flow
//   - Decision, SystemMessage, Reason: Provide feedback and decisions
//   - HookSpecificOutput: Event-specific controls
//
// See https://docs.anthropic.com/en/docs/claude-code/hooks#advanced%3A-json-output
type SyncHookJSONOutput struct {
	// Common control fields
	Continue_      *bool   `json:"continue,omitempty"` // Using Continue_ to match Python's continue_
	SuppressOutput *bool   `json:"suppressOutput,omitempty"`
	StopReason     *string `json:"stopReason,omitempty"`

	// Decision fields
	Decision      *string `json:"decision,omitempty"` // Only "block" is meaningful
	SystemMessage *string `json:"systemMessage,omitempty"`
	Reason        *string `json:"reason,omitempty"`

	// Hook-specific outputs
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// HookJSONOutput is a union of AsyncHookJSONOutput and SyncHookJSONOutput.
type HookJSONOutput interface {
	IsAsync() bool
}

// IsAsync returns true for AsyncHookJSONOutput.
func (h AsyncHookJSONOutput) IsAsync() bool { return h.Async_ }

// IsAsync returns false for SyncHookJSONOutput.
func (h SyncHookJSONOutput) IsAsync() bool { return false }

// HookContext provides context information for hook callbacks.
type HookContext struct {
	Signal interface{} `json:"signal"` // Future: abort signal support, currently always nil
}

// HookMatcher represents hook matcher configuration.
// See https://docs.anthropic.com/en/docs/claude-code/hooks#structure for the
// expected string value. For example, for PreToolUse, the matcher can be
// a tool name like "Bash" or a combination of tool names like "Write|MultiEdit|Edit".
type HookMatcher struct {
	Matcher string         `json:"matcher,omitempty"`
	Hooks   []HookCallback `json:"hooks,omitempty"`
	Timeout *float64       `json:"timeout,omitempty"` // Timeout in seconds (default: 60)
}

// HookCallback is the function signature for hook callbacks.
// Note: In Go, this would typically be a function type, but we define it as
// an interface for flexibility. Users should implement this interface.
type HookCallback interface {
	Execute(input HookInput, toolUseID *string, context HookContext) (HookJSONOutput, error)
}

// ============================================================================
// MCP Server Configuration Types
// ============================================================================

// McpStdioServerConfig represents MCP stdio server configuration.
type McpStdioServerConfig struct {
	Type    string            `json:"type,omitempty"` // Optional for backwards compatibility, defaults to "stdio"
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// McpSSEServerConfig represents MCP SSE server configuration.
type McpSSEServerConfig struct {
	Type    string            `json:"type"` // Always "sse"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpHttpServerConfig represents MCP HTTP server configuration.
type McpHttpServerConfig struct {
	Type    string            `json:"type"` // Always "http"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpSdkServerConfig represents SDK MCP server configuration.
type McpSdkServerConfig struct {
	Type     string      `json:"type"` // Always "sdk"
	Name     string      `json:"name"`
	Instance interface{} `json:"instance"` // McpServer instance
}

// McpServerConfig is a union type for all MCP server configuration types.
type McpServerConfig interface {
	GetType() string
}

// GetType returns the server type.
func (c McpStdioServerConfig) GetType() string {
	if c.Type == "" {
		return "stdio"
	}
	return c.Type
}

// GetType returns the server type.
func (c McpSSEServerConfig) GetType() string { return c.Type }

// GetType returns the server type.
func (c McpHttpServerConfig) GetType() string { return c.Type }

// GetType returns the server type.
func (c McpSdkServerConfig) GetType() string { return c.Type }

// ============================================================================
// SDK Plugin Configuration
// ============================================================================

// SdkPluginConfig represents SDK plugin configuration.
// Currently only local plugins are supported via the 'local' type.
type SdkPluginConfig struct {
	Type string `json:"type"` // Always "local"
	Path string `json:"path"`
}

// ============================================================================
// Sandbox Configuration Types
// ============================================================================

// SandboxNetworkConfig represents network configuration for sandbox.
type SandboxNetworkConfig struct {
	AllowUnixSockets    []string `json:"allowUnixSockets,omitempty"`    // Unix socket paths accessible in sandbox (e.g., SSH agents)
	AllowAllUnixSockets *bool    `json:"allowAllUnixSockets,omitempty"` // Allow all Unix sockets (less secure)
	AllowLocalBinding   *bool    `json:"allowLocalBinding,omitempty"`   // Allow binding to localhost ports (macOS only)
	HTTPProxyPort       *int     `json:"httpProxyPort,omitempty"`       // HTTP proxy port if bringing your own proxy
	SOCKSProxyPort      *int     `json:"socksProxyPort,omitempty"`      // SOCKS5 proxy port if bringing your own proxy
}

// SandboxIgnoreViolations represents violations to ignore in sandbox.
type SandboxIgnoreViolations struct {
	File    []string `json:"file,omitempty"`    // File paths for which violations should be ignored
	Network []string `json:"network,omitempty"` // Network hosts for which violations should be ignored
}

// SandboxSettings controls how Claude Code sandboxes bash commands for filesystem
// and network isolation.
//
// Important: Filesystem and network restrictions are configured via permission
// rules, not via these sandbox settings:
//   - Filesystem read restrictions: Use Read deny rules
//   - Filesystem write restrictions: Use Edit allow/deny rules
//   - Network restrictions: Use WebFetch allow/deny rules
type SandboxSettings struct {
	Enabled                   *bool                    `json:"enabled,omitempty"`                   // Enable bash sandboxing (macOS/Linux only). Default: false
	AutoAllowBashIfSandboxed  *bool                    `json:"autoAllowBashIfSandboxed,omitempty"`  // Auto-approve bash commands when sandboxed. Default: true
	ExcludedCommands          []string                 `json:"excludedCommands,omitempty"`          // Commands that should run outside the sandbox (e.g., ["git", "docker"])
	AllowUnsandboxedCommands  *bool                    `json:"allowUnsandboxedCommands,omitempty"`  // Allow commands to bypass sandbox. Default: true
	Network                   *SandboxNetworkConfig    `json:"network,omitempty"`                   // Network configuration for sandbox
	IgnoreViolations          *SandboxIgnoreViolations `json:"ignoreViolations,omitempty"`          // Violations to ignore
	EnableWeakerNestedSandbox *bool                    `json:"enableWeakerNestedSandbox,omitempty"` // Enable weaker sandbox for unprivileged Docker (Linux only). Default: false
}

// ============================================================================
// Content Block Types
// ============================================================================

// TextBlock represents a text content block.
type TextBlock struct {
	Type string `json:"type"` // Always "text"
	Text string `json:"text"`
}

// ThinkingBlock represents a thinking content block.
type ThinkingBlock struct {
	Type      string `json:"type"` // Always "thinking"
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

// ToolUseBlock represents a tool use content block.
type ToolUseBlock struct {
	Type  string                 `json:"type"` // Always "tool_use"
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResultBlock represents a tool result content block.
type ToolResultBlock struct {
	Type      string      `json:"type"` // Always "tool_result"`
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content,omitempty"` // string, []ContentBlock, or nil
	IsError   *bool       `json:"is_error,omitempty"`
}

// ContentBlock is a union type for all content block types.
// Use the Type field to determine the concrete type.
type ContentBlock interface {
	GetType() string
}

// GetType returns the block type.
func (b TextBlock) GetType() string { return b.Type }

// GetType returns the block type.
func (b ThinkingBlock) GetType() string { return b.Type }

// GetType returns the block type.
func (b ToolUseBlock) GetType() string { return b.Type }

// GetType returns the block type.
func (b ToolResultBlock) GetType() string { return b.Type }

// UnmarshalContentBlock unmarshals JSON into the appropriate ContentBlock type.
func UnmarshalContentBlock(data []byte) (ContentBlock, error) {
	var raw struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	switch raw.Type {
	case "text":
		var block TextBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		block.Type = "text"
		return block, nil
	case "thinking":
		var block ThinkingBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		block.Type = "thinking"
		return block, nil
	case "tool_use":
		var block ToolUseBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		block.Type = "tool_use"
		return block, nil
	case "tool_result":
		var block ToolResultBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		block.Type = "tool_result"
		return block, nil
	default:
		// Return a generic map for unknown types
		var block map[string]interface{}
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, err
		}
		return GenericContentBlock{Data: block}, nil
	}
}

// GenericContentBlock represents an unknown content block type.
type GenericContentBlock struct {
	Data map[string]interface{}
}

// GetType returns the block type.
func (b GenericContentBlock) GetType() string {
	if t, ok := b.Data["type"].(string); ok {
		return t
	}
	return ""
}

// ============================================================================
// Message Types
// ============================================================================

// AssistantMessageError represents error types for assistant messages.
type AssistantMessageError string

const (
	AssistantMessageErrorAuthenticationFailed AssistantMessageError = "authentication_failed"
	AssistantMessageErrorBillingError         AssistantMessageError = "billing_error"
	AssistantMessageErrorRateLimit            AssistantMessageError = "rate_limit"
	AssistantMessageErrorInvalidRequest       AssistantMessageError = "invalid_request"
	AssistantMessageErrorServerError          AssistantMessageError = "server_error"
	AssistantMessageErrorUnknown              AssistantMessageError = "unknown"
)

// UserMessage represents a user message.
type UserMessage struct {
	Content         interface{}            `json:"content"` // string or []ContentBlock
	UUID            *string                `json:"uuid,omitempty"`
	ParentToolUseID *string                `json:"parent_tool_use_id,omitempty"`
	ToolUseResult   map[string]interface{} `json:"tool_use_result,omitempty"`
}

// AssistantMessage represents an assistant message with content blocks.
type AssistantMessage struct {
	Content         []ContentBlock         `json:"content"`
	Model           string                 `json:"model"`
	ParentToolUseID *string                `json:"parent_tool_use_id,omitempty"`
	Error           *AssistantMessageError `json:"error,omitempty"`
}

// SystemMessage represents a system message with metadata.
type SystemMessage struct {
	Subtype string                 `json:"subtype"`
	Data    map[string]interface{} `json:"data"`
}

// ResultMessage represents a result message with cost and usage information.
type ResultMessage struct {
	Subtype          string                 `json:"subtype"`
	DurationMs       int                    `json:"duration_ms"`
	DurationAPIMs    int                    `json:"duration_api_ms"`
	IsError          bool                   `json:"is_error"`
	NumTurns         int                    `json:"num_turns"`
	SessionID        string                 `json:"session_id"`
	TotalCostUSD     *float64               `json:"total_cost_usd,omitempty"`
	Usage            map[string]interface{} `json:"usage,omitempty"`
	Result           *string                `json:"result,omitempty"`
	StructuredOutput interface{}            `json:"structured_output,omitempty"`
}

// StreamEvent represents a stream event for partial message updates during streaming.
type StreamEvent struct {
	UUID            string                 `json:"uuid"`
	SessionID       string                 `json:"session_id"`
	Event           map[string]interface{} `json:"event"` // The raw Anthropic API stream event
	ParentToolUseID *string                `json:"parent_tool_use_id,omitempty"`
}

// Message is a union type for all message types.
type Message interface {
	GetSessionID() string
}

// GetSessionID returns empty for UserMessage (doesn't have session ID).
func (m UserMessage) GetSessionID() string { return "" }

// GetSessionID returns empty for AssistantMessage (doesn't have session ID).
func (m AssistantMessage) GetSessionID() string { return "" }

// GetSessionID returns empty for SystemMessage (doesn't have session ID).
func (m SystemMessage) GetSessionID() string { return "" }

// GetSessionID returns the session ID for ResultMessage.
func (m ResultMessage) GetSessionID() string { return m.SessionID }

// GetSessionID returns the session ID for StreamEvent.
func (m StreamEvent) GetSessionID() string { return m.SessionID }

// ============================================================================
// Thinking Configuration Types
// ============================================================================

// ThinkingConfigAdaptive represents adaptive thinking configuration.
type ThinkingConfigAdaptive struct {
	Type string `json:"type"` // Always "adaptive"
}

// ThinkingConfigEnabled represents enabled thinking configuration with budget.
type ThinkingConfigEnabled struct {
	Type         string `json:"type"` // Always "enabled"
	BudgetTokens int    `json:"budget_tokens"`
}

// ThinkingConfigDisabled represents disabled thinking configuration.
type ThinkingConfigDisabled struct {
	Type string `json:"type"` // Always "disabled"
}

// ThinkingConfig is a union type for all thinking configuration types.
type ThinkingConfig interface {
	GetType() string
}

// GetType returns the config type.
func (c ThinkingConfigAdaptive) GetType() string { return c.Type }

// GetType returns the config type.
func (c ThinkingConfigEnabled) GetType() string { return c.Type }

// GetType returns the config type.
func (c ThinkingConfigDisabled) GetType() string { return c.Type }

// ============================================================================
// Claude Agent Options
// ============================================================================

// ClaudeAgentOptions represents query options for Claude SDK.
type ClaudeAgentOptions struct {
	// Tools configuration
	Tools        interface{} `json:"tools,omitempty"` // []string or ToolsPreset
	AllowedTools []string    `json:"allowed_tools,omitempty"`

	// System configuration
	SystemPrompt interface{} `json:"system_prompt,omitempty"` // string or SystemPromptPreset

	// MCP configuration
	MCPServers interface{} `json:"mcp_servers,omitempty"` // map[string]McpServerConfig, string, or Path

	// Permission configuration
	PermissionMode *PermissionMode `json:"permission_mode,omitempty"`

	// Conversation configuration
	ContinueConversation bool     `json:"continue_conversation,omitempty"`
	Resume               *string  `json:"resume,omitempty"`
	MaxTurns             *int     `json:"max_turns,omitempty"`
	MaxBudgetUSD         *float64 `json:"max_budget_usd,omitempty"`

	// Tool restrictions
	DisallowedTools []string `json:"disallowed_tools,omitempty"`

	// Model configuration
	Model         *string `json:"model,omitempty"`
	FallbackModel *string `json:"fallback_model,omitempty"`

	// Beta features - see https://docs.anthropic.com/en/api/beta-headers
	Betas []SdkBeta `json:"betas,omitempty"`

	// Permission callback configuration
	PermissionPromptToolName *string `json:"permission_prompt_tool_name,omitempty"`

	// Path configuration
	CWD      interface{} `json:"cwd,omitempty"`      // string or Path
	CLIPath  interface{} `json:"cli_path,omitempty"` // string or Path
	Settings *string     `json:"settings,omitempty"`

	// Directory configuration
	AddDirs []interface{} `json:"add_dirs,omitempty"` // []string or []Path

	// Environment configuration
	Env map[string]string `json:"env,omitempty"`

	// Extra CLI arguments
	ExtraArgs map[string]interface{} `json:"extra_args,omitempty"`

	// Buffer configuration
	MaxBufferSize *int `json:"max_buffer_size,omitempty"`

	// Debug output (deprecated)
	DebugStderr interface{} `json:"debug_stderr,omitempty"`

	// Stderr callback
	Stderr func(string) `json:"-"` // Not serialized to JSON

	// Tool permission callback
	CanUseTool func(string, map[string]interface{}, ToolPermissionContext) (PermissionResult, error) `json:"-"` // Not serialized to JSON

	// Hook configurations
	Hooks map[HookEvent][]HookMatcher `json:"hooks,omitempty"`

	// User configuration
	User *string `json:"user,omitempty"`

	// Partial message streaming support
	IncludePartialMessages bool `json:"include_partial_messages,omitempty"`

	// Session forking
	ForkSession bool `json:"fork_session,omitempty"`

	// Agent definitions for custom agents
	Agents map[string]AgentDefinition `json:"agents,omitempty"`

	// Setting sources to load (user, project, local)
	SettingSources []SettingSource `json:"setting_sources,omitempty"`

	// Sandbox configuration for bash command isolation
	Sandbox *SandboxSettings `json:"sandbox,omitempty"`

	// Plugin configurations for custom plugins
	Plugins []SdkPluginConfig `json:"plugins,omitempty"`

	// Max tokens for thinking blocks (deprecated: use Thinking instead)
	MaxThinkingTokens *int `json:"max_thinking_tokens,omitempty"`

	// Controls extended thinking behavior. Takes precedence over MaxThinkingTokens.
	Thinking ThinkingConfig `json:"thinking,omitempty"`

	// Effort level for thinking depth
	Effort *string `json:"effort,omitempty"` // "low", "medium", "high", or "max"

	// Output format for structured outputs (matches Messages API structure)
	// Example: {"type": "json_schema", "schema": {"type": "object", "properties": {...}}}
	OutputFormat map[string]interface{} `json:"output_format,omitempty"`

	// Enable file checkpointing to track file changes during the session
	EnableFileCheckpointing bool `json:"enable_file_checkpointing,omitempty"`
}

// ============================================================================
// SDK Control Protocol Types
// ============================================================================

// SDKControlInterruptRequest represents an interrupt control request.
type SDKControlInterruptRequest struct {
	Subtype string `json:"subtype"` // Always "interrupt"
}

// SDKControlPermissionRequest represents a permission request.
type SDKControlPermissionRequest struct {
	Subtype               string                 `json:"subtype"` // Always "can_use_tool"
	ToolName              string                 `json:"tool_name"`
	Input                 map[string]interface{} `json:"input"`
	PermissionSuggestions []interface{}          `json:"permission_suggestions,omitempty"`
	BlockedPath           *string                `json:"blocked_path,omitempty"`
}

// SDKControlInitializeRequest represents an initialize control request.
type SDKControlInitializeRequest struct {
	Subtype string                            `json:"subtype"` // Always "initialize"
	Hooks   map[HookEvent]interface{}         `json:"hooks,omitempty"`
	Agents  map[string]map[string]interface{} `json:"agents,omitempty"`
}

// SDKControlSetPermissionModeRequest represents a set permission mode request.
type SDKControlSetPermissionModeRequest struct {
	Subtype string `json:"subtype"` // Always "set_permission_mode"
	Mode    string `json:"mode"`
}

// SDKHookCallbackRequest represents a hook callback request.
type SDKHookCallbackRequest struct {
	Subtype    string      `json:"subtype"` // Always "hook_callback"
	CallbackID string      `json:"callback_id"`
	Input      interface{} `json:"input"`
	ToolUseID  *string     `json:"tool_use_id,omitempty"`
}

// SDKControlMcpMessageRequest represents an MCP message request.
type SDKControlMcpMessageRequest struct {
	Subtype    string      `json:"subtype"` // Always "mcp_message"
	ServerName string      `json:"server_name"`
	Message    interface{} `json:"message"`
}

// SDKControlRewindFilesRequest represents a rewind files request.
type SDKControlRewindFilesRequest struct {
	Subtype       string `json:"subtype"` // Always "rewind_files"
	UserMessageID string `json:"user_message_id"`
}

// SDKControlRequest represents a control request from the CLI.
type SDKControlRequest struct {
	Type      string      `json:"type"` // Always "control_request"
	RequestID string      `json:"request_id"`
	Request   interface{} `json:"request"` // One of the SDKControl*Request types
}

// ControlResponse represents a successful control response.
type ControlResponse struct {
	Subtype   string                 `json:"subtype"` // Always "success"
	RequestID string                 `json:"request_id"`
	Response  map[string]interface{} `json:"response,omitempty"`
}

// ControlErrorResponse represents an error control response.
type ControlErrorResponse struct {
	Subtype   string `json:"subtype"` // Always "error"
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
}

// SDKControlResponse represents a control response to the CLI.
type SDKControlResponse struct {
	Type     string      `json:"type"`     // Always "control_response"
	Response interface{} `json:"response"` // ControlResponse or ControlErrorResponse
}

// UnmarshalSDKControlRequest unmarshals JSON into the appropriate SDKControlRequest type.
func UnmarshalSDKControlRequest(data []byte) (*SDKControlRequest, error) {
	var raw struct {
		Type      string          `json:"type"`
		RequestID string          `json:"request_id"`
		Request   json.RawMessage `json:"request"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var request interface{}
	var rawRequest struct {
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(raw.Request, &rawRequest); err != nil {
		return nil, err
	}

	switch rawRequest.Subtype {
	case "interrupt":
		var req SDKControlInterruptRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return nil, err
		}
		request = req
	case "can_use_tool":
		var req SDKControlPermissionRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return nil, err
		}
		request = req
	case "initialize":
		var req SDKControlInitializeRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return nil, err
		}
		request = req
	case "set_permission_mode":
		var req SDKControlSetPermissionModeRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return nil, err
		}
		request = req
	case "hook_callback":
		var req SDKHookCallbackRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return nil, err
		}
		request = req
	case "mcp_message":
		var req SDKControlMcpMessageRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return nil, err
		}
		request = req
	case "rewind_files":
		var req SDKControlRewindFilesRequest
		if err := json.Unmarshal(raw.Request, &req); err != nil {
			return nil, err
		}
		request = req
	default:
		// Keep as raw message for unknown types
		request = raw.Request
	}

	return &SDKControlRequest{
		Type:      raw.Type,
		RequestID: raw.RequestID,
		Request:   request,
	}, nil
}

// ============================================================================
// Utility Functions
// ============================================================================

// String returns a pointer to a string.
func String(s string) *string {
	return &s
}

// Int returns a pointer to an int.
func Int(i int) *int {
	return &i
}

// Float64 returns a pointer to a float64.
func Float64(f float64) *float64 {
	return &f
}

// Bool returns a pointer to a bool.
func Bool(b bool) *bool {
	return &b
}

// PermissionMode returns a pointer to a PermissionMode.
func PermissionModePtr(p PermissionMode) *PermissionMode {
	return &p
}

// PermissionBehavior returns a pointer to a PermissionBehavior.
func PermissionBehaviorPtr(b PermissionBehavior) *PermissionBehavior {
	return &b
}

// PermissionUpdateDestinationPtr returns a pointer to a PermissionUpdateDestination.
func PermissionUpdateDestinationPtr(d PermissionUpdateDestination) *PermissionUpdateDestination {
	return &d
}

// ============================================================================
// Type Aliases for SDK Compatibility
// ============================================================================

// Options is an alias for ClaudeAgentOptions for backward compatibility.
type Options = ClaudeAgentOptions

// QueryResult represents the result of a query operation.
type QueryResult struct {
	// Result is the final result text from the conversation.
	Result string `json:"result,omitempty"`

	// SessionID is the unique identifier for this session.
	SessionID string `json:"session_id,omitempty"`

	// CostUSD is the total cost in USD.
	CostUSD float64 `json:"cost_usd,omitempty"`

	// DurationMs is the total duration in milliseconds.
	DurationMs int `json:"duration_ms,omitempty"`

	// DurationAPIMs is the API duration in milliseconds.
	DurationAPIMs int `json:"duration_api_ms,omitempty"`

	// NumTurns is the number of turns in the conversation.
	NumTurns int `json:"num_turns,omitempty"`

	// IsError indicates whether the result is an error.
	IsError bool `json:"is_error,omitempty"`

	// Subtype indicates the result subtype.
	Subtype string `json:"subtype,omitempty"`

	// Usage contains token usage information.
	Usage map[string]interface{} `json:"usage,omitempty"`

	// StructuredOutput contains structured output if requested.
	StructuredOutput interface{} `json:"structured_output,omitempty"`

	// TotalCostUSD is the total cost in USD (alias for CostUSD for compatibility).
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
}
