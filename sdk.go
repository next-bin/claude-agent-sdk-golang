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

	"github.com/unitsvc/claude-agent-sdk-golang/client"
	sdkerrors "github.com/unitsvc/claude-agent-sdk-golang/errors"
	"github.com/unitsvc/claude-agent-sdk-golang/query"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
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
//	    Model: types.String("claude-sonnet-4-20250514"),
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
//	    Model:       "claude-sonnet-4-20250514",
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
)

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
