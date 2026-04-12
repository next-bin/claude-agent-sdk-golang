// Package errors defines error types for the Claude Agent SDK.
//
// This package provides structured error handling for SDK operations.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common error cases.
var (
	// ErrNoAPIKey is returned when no API key is provided.
	ErrNoAPIKey = errors.New("no API key provided")

	// ErrNotInstalled is returned when the Claude CLI is not installed.
	ErrNotInstalled = errors.New("claude CLI not installed")

	// ErrConnectionFailed is returned when connection to Claude CLI fails.
	ErrConnectionFailed = errors.New("failed to connect to Claude CLI")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("operation timed out")

	// ErrInterrupted is returned when an operation is interrupted.
	ErrInterrupted = errors.New("operation interrupted")
)

// ClaudeSDKError is the base error type for all Claude SDK errors.
// It provides a type, message, and optional cause for error wrapping.
type ClaudeSDKError struct {
	Type    string
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *ClaudeSDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *ClaudeSDKError) Unwrap() error {
	return e.Cause
}

// NewClaudeSDKError creates a new ClaudeSDKError.
func NewClaudeSDKError(errType, message string, cause error) *ClaudeSDKError {
	return &ClaudeSDKError{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// CLIConnectionError is returned when unable to connect to Claude Code.
type CLIConnectionError struct {
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *CLIConnectionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("CLI connection error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("CLI connection error: %s", e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *CLIConnectionError) Unwrap() error {
	return e.Cause
}

// NewCLIConnectionError creates a new CLIConnectionError.
func NewCLIConnectionError(message string, cause error) *CLIConnectionError {
	return &CLIConnectionError{
		Message: message,
		Cause:   cause,
	}
}

// CLINotFoundError is returned when Claude Code is not found or not installed.
type CLINotFoundError struct {
	Message string
	CLIPath string
}

// Error implements the error interface.
func (e *CLINotFoundError) Error() string {
	if e.CLIPath != "" {
		return fmt.Sprintf("Claude Code not found: %s", e.CLIPath)
	}
	return "Claude Code not found"
}

// NewCLINotFoundError creates a new CLINotFoundError.
// If cliPath is provided, it will be included in the error message.
func NewCLINotFoundError(message string, cliPath string) *CLINotFoundError {
	if message == "" {
		message = "Claude Code not found"
	}
	return &CLINotFoundError{
		Message: message,
		CLIPath: cliPath,
	}
}

// ProcessError is returned when the CLI process fails.
// It includes exit code and stderr output for debugging.
type ProcessError struct {
	Message  string
	ExitCode *int
	Stderr   string
}

// Error implements the error interface.
func (e *ProcessError) Error() string {
	msg := e.Message
	if e.ExitCode != nil {
		msg = fmt.Sprintf("%s (exit code: %d)", msg, *e.ExitCode)
	}
	if e.Stderr != "" {
		msg = fmt.Sprintf("%s\nError output: %s", msg, e.Stderr)
	}
	return msg
}

// NewProcessError creates a new ProcessError.
// exitCode and stderr are optional and can be nil/empty.
func NewProcessError(message string, exitCode *int, stderr string) *ProcessError {
	return &ProcessError{
		Message:  message,
		ExitCode: exitCode,
		Stderr:   stderr,
	}
}

// CLIJSONDecodeError is returned when unable to decode JSON from CLI output.
type CLIJSONDecodeError struct {
	Line          string
	OriginalError error
}

// Error implements the error interface.
func (e *CLIJSONDecodeError) Error() string {
	// Truncate line to 100 characters like Python version
	line := e.Line
	if len(line) > 100 {
		line = line[:100]
	}
	return fmt.Sprintf("Failed to decode JSON: %s...", line)
}

// Unwrap returns the underlying cause of the error.
func (e *CLIJSONDecodeError) Unwrap() error {
	return e.OriginalError
}

// NewCLIJSONDecodeError creates a new CLIJSONDecodeError.
func NewCLIJSONDecodeError(line string, originalError error) *CLIJSONDecodeError {
	return &CLIJSONDecodeError{
		Line:          line,
		OriginalError: originalError,
	}
}

// MessageParseError is returned when unable to parse a message from CLI output.
type MessageParseError struct {
	Message string
	Data    map[string]any
}

// Error implements the error interface.
func (e *MessageParseError) Error() string {
	return e.Message
}

// NewMessageParseError creates a new MessageParseError.
// data is optional and can be nil.
func NewMessageParseError(message string, data map[string]any) *MessageParseError {
	return &MessageParseError{
		Message: message,
		Data:    data,
	}
}

// SDKError represents an error from the SDK.
// Deprecated: Use ClaudeSDKError instead.
type SDKError = ClaudeSDKError

// CLIError represents an error from the Claude CLI.
type CLIError struct {
	ExitCode int
	Message  string
	Stderr   string
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("CLI error (exit code %d): %s\nstderr: %s", e.ExitCode, e.Message, e.Stderr)
	}
	return fmt.Sprintf("CLI error (exit code %d): %s", e.ExitCode, e.Message)
}

// NewCLIError creates a new CLI error.
func NewCLIError(exitCode int, message, stderr string) *CLIError {
	return &CLIError{
		ExitCode: exitCode,
		Message:  message,
		Stderr:   stderr,
	}
}

// ToolExecutionError represents an error during tool execution.
type ToolExecutionError struct {
	ToolName string
	Message  string
	Cause    error
}

// Error implements the error interface.
func (e *ToolExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool execution error (%s): %s: %v", e.ToolName, e.Message, e.Cause)
	}
	return fmt.Sprintf("tool execution error (%s): %s", e.ToolName, e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *ToolExecutionError) Unwrap() error {
	return e.Cause
}

// NewToolExecutionError creates a new tool execution error.
func NewToolExecutionError(toolName, message string, cause error) *ToolExecutionError {
	return &ToolExecutionError{
		ToolName: toolName,
		Message:  message,
		Cause:    cause,
	}
}

// IsTimeout checks if an error is a timeout error.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsInterrupted checks if an error is an interruption error.
func IsInterrupted(err error) bool {
	return errors.Is(err, ErrInterrupted)
}
