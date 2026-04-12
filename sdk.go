// Package claude provides the Claude Agent SDK for Go.
//
// This SDK enables building AI agents with Claude. It provides both
// one-shot query functionality and interactive session management.
//
// For simple queries, use the Query function:
//
//	result, err := claude.Query(ctx, "Hello, Claude!", nil)
//
// For more control, create a client:
//
//	client := client.New()
//	result, err := client.Query(ctx, "Hello, Claude!")
//
// See the examples directory for more usage examples.
package claude

import (
	"context"

	"github.com/next-bin/claude-agent-sdk-golang/client"
	sdkerrors "github.com/next-bin/claude-agent-sdk-golang/errors"
	"github.com/next-bin/claude-agent-sdk-golang/internal/sessions"
	"github.com/next-bin/claude-agent-sdk-golang/query"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// Query sends a one-shot query to Claude and returns messages through a channel.
//
// This is the simplest way to interact with Claude. Pass a prompt and
// optional configuration, and receive messages through a channel.
//
// Example:
//
//	msgChan, err := claude.Query(ctx, "What is 2+2?", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for msg := range msgChan {
//	    fmt.Printf("%v\n", msg)
//	}
func Query(ctx context.Context, prompt string, opts *types.ClaudeAgentOptions) (<-chan types.Message, error) {
	return query.Query(ctx, prompt, opts)
}

// QueryWithClient sends a query using an existing client.
//
// Use this when you have a pre-configured client with custom options.
//
// Example:
//
//	c := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
//	    Model: types.String(types.ModelSonnet),
//	})
//	msgChan, err := claude.QueryWithClient(ctx, c, "Hello!")
func QueryWithClient(ctx context.Context, c *client.Client, prompt string) (<-chan types.Message, error) {
	return query.QueryWithClient(ctx, c, prompt)
}

// NewClient creates a new Claude client with default options.
//
// Example:
//
//	client := claude.NewClient()
//	defer client.Close()
//	result, err := client.Query(ctx, "Hello, Claude!")
func NewClient() *client.Client {
	return client.New()
}

// NewClientWithOptions creates a new Claude client with custom options.
//
// Example:
//
//	client := claude.NewClientWithOptions(&types.Options{
//	    Model:       types.ModelSonnet,
//	    MaxTokens:   4096,
//	    Temperature: 0.7,
//	})
func NewClientWithOptions(opts *types.Options) *client.Client {
	return client.NewWithOptions(opts)
}

// Re-export types for convenience.
type (
	// Options represents configuration options for the SDK.
	Options = types.Options

	// Message represents a message in a conversation.
	Message = types.Message

	// ContentBlock represents a content block in a message.
	ContentBlock = types.ContentBlock

	// QueryResult represents the result of a query operation.
	QueryResult = types.QueryResult

	// AssistantMessage represents a message from the assistant.
	AssistantMessage = types.AssistantMessage

	// ResultMessage represents the final result of a conversation.
	ResultMessage = types.ResultMessage

	// UserMessage represents a message from the user.
	UserMessage = types.UserMessage

	// SystemMessage represents a system message.
	SystemMessage = types.SystemMessage

	// StreamEvent represents a streaming event.
	StreamEvent = types.StreamEvent

	// ToolUseBlock represents a tool use content block.
	ToolUseBlock = types.ToolUseBlock

	// ToolResultBlock represents a tool result content block.
	ToolResultBlock = types.ToolResultBlock

	// TextBlock represents a text content block.
	TextBlock = types.TextBlock

	// ThinkingBlock represents a thinking content block.
	ThinkingBlock = types.ThinkingBlock

	// Permission types
	PermissionResult      = types.PermissionResult
	PermissionResultAllow = types.PermissionResultAllow
	PermissionResultDeny  = types.PermissionResultDeny
	PermissionUpdate      = types.PermissionUpdate
	ToolPermissionContext = types.ToolPermissionContext

	// Hook types
	HookEvent    = types.HookEvent
	HookMatcher  = types.HookMatcher
	HookInput    = types.HookInput
	HookContext  = types.HookContext
	HookCallback = types.HookCallback

	// MCP types
	McpServerConfig      = types.McpServerConfig
	McpStdioServerConfig = types.McpStdioServerConfig
	McpSSEServerConfig   = types.McpSSEServerConfig
	McpHttpServerConfig  = types.McpHttpServerConfig

	// Agent types
	AgentDefinition = types.AgentDefinition

	// Session types (v0.1.46)
	SDKSessionInfo = types.SDKSessionInfo
	SessionMessage = types.SessionMessage
)

// Client is an alias for the client type.
type Client = client.Client

// Re-export errors for convenience.
var (
	// ErrNoAPIKey is returned when no API key is provided.
	ErrNoAPIKey = sdkerrors.ErrNoAPIKey

	// ErrNotInstalled is returned when the Claude CLI is not installed.
	ErrNotInstalled = sdkerrors.ErrNotInstalled

	// ErrConnectionFailed is returned when connection to Claude CLI fails.
	ErrConnectionFailed = sdkerrors.ErrConnectionFailed

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = sdkerrors.ErrTimeout

	// ErrInterrupted is returned when an operation is interrupted.
	ErrInterrupted = sdkerrors.ErrInterrupted
)

// SDKError represents an error from the SDK.
type SDKError = sdkerrors.SDKError

// CLIError represents an error from the Claude CLI.
type CLIError = sdkerrors.CLIError

// ToolExecutionError represents an error during tool execution.
type ToolExecutionError = sdkerrors.ToolExecutionError

// ============================================================================
// Sessions API (v0.1.46)
// ============================================================================

// ListSessions lists sessions with metadata extracted from stat + head/tail reads.
//
// When directory is provided, returns sessions for that project directory and its
// git worktrees. When empty, returns sessions across all projects.
//
// The limit parameter limits the number of sessions returned (0 means no limit).
// The includeWorktrees parameter controls whether to include git worktree sessions.
//
// Example:
//
//	// List sessions for current directory
//	sessions, err := claude.ListSessions("/path/to/project", 10, true)
//
//	// List all sessions
//	sessions, err := claude.ListSessions("", 0, false)
func ListSessions(directory string, limit int, includeWorktrees bool) ([]types.SDKSessionInfo, error) {
	return sessions.ListSessions(directory, limit, includeWorktrees)
}

// GetSessionMessages reads a session's conversation messages from its JSONL transcript file.
//
// Parses the full JSONL, builds the conversation chain via parentUuid links,
// and returns user/assistant messages in chronological order.
//
// The limit parameter limits the number of messages returned (0 means no limit).
// The offset parameter skips the first N messages.
//
// Example:
//
//	messages, err := claude.GetSessionMessages("550e8400-e29b-41d4-a716-446655440000", "/path/to/project", 10, 0)
func GetSessionMessages(sessionID, directory string, limit, offset int) ([]types.SessionMessage, error) {
	return sessions.GetSessionMessages(sessionID, directory, limit, offset)
}

// GetSessionInfo reads metadata for a single session by ID.
//
// Wraps readSessionLite for one file — no O(n) directory scan.
// Directory resolution matches GetSessionMessages: directory is the project path;
// when omitted, all project directories are searched for the session file.
//
// Returns SDKSessionInfo for the session, or nil if the session file
// is not found, is a sidechain session, or has no extractable summary.
//
// Example:
//
//	// Look up a session in a specific project
//	info := claude.GetSessionInfo("550e8400-e29b-41d4-a716-446655440000", "/path/to/project")
//	if info != nil {
//	    fmt.Println(info.Summary)
//	}
//
//	// Search all projects for a session
//	info := claude.GetSessionInfo("550e8400-e29b-41d4-a716-446655440000", "")
func GetSessionInfo(sessionID, directory string) *types.SDKSessionInfo {
	return sessions.GetSessionInfo(sessionID, directory)
}

// ============================================================================
// Session Mutations API (v0.1.49+)
// ============================================================================

// RenameSession renames a session by appending a custom-title entry.
// list_sessions reads the LAST custom-title from the file tail, so repeated calls are safe.
//
// Example:
//
//	err := claude.RenameSession("550e8400-e29b-41d4-a716-446655440000", "My refactoring session", "/path/to/project")
func RenameSession(sessionID, title, directory string) error {
	return sessions.RenameSession(sessionID, title, directory)
}

// TagSession tags a session. Pass empty string to clear the tag.
// list_sessions reads the LAST tag from the file tail, so repeated calls are safe.
//
// Tags are Unicode-sanitized before storing for CLI filter compatibility.
//
// Example:
//
//	err := claude.TagSession("550e8400-e29b-41d4-a716-446655440000", "experiment", "/path/to/project")
func TagSession(sessionID, tag, directory string) error {
	return sessions.TagSession(sessionID, tag, directory)
}
