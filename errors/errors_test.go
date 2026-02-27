package errors

import (
	"errors"
	"testing"
)

// ============================================================================
// Sentinel Error Tests
// ============================================================================

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNoAPIKey", ErrNoAPIKey, "no API key provided"},
		{"ErrNotInstalled", ErrNotInstalled, "Claude CLI not installed"},
		{"ErrConnectionFailed", ErrConnectionFailed, "failed to connect to Claude CLI"},
		{"ErrTimeout", ErrTimeout, "operation timed out"},
		{"ErrInterrupted", ErrInterrupted, "operation interrupted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, tt.err.Error())
			}
		})
	}
}

// ============================================================================
// ClaudeSDKError Tests
// ============================================================================

func TestClaudeSDKError(t *testing.T) {
	err := NewClaudeSDKError("test_type", "test message", nil)
	if err.Type != "test_type" {
		t.Errorf("Expected type 'test_type', got %q", err.Type)
	}
	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got %q", err.Message)
	}
	if err.Error() != "test_type: test message" {
		t.Errorf("Expected 'test_type: test message', got %q", err.Error())
	}
}

func TestClaudeSDKErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewClaudeSDKError("test_type", "test message", cause)
	if err.Error() != "test_type: test message: underlying error" {
		t.Errorf("Expected 'test_type: test message: underlying error', got %q", err.Error())
	}
}

func TestClaudeSDKErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewClaudeSDKError("test_type", "test message", cause)
	unwrapped := err.Unwrap()
	if unwrapped == nil {
		t.Error("Expected non-nil cause")
	}
	if unwrapped.Error() != "underlying error" {
		t.Errorf("Expected 'underlying error', got %q", unwrapped.Error())
	}
}

func TestClaudeSDKErrorUnwrapNil(t *testing.T) {
	err := NewClaudeSDKError("test_type", "test message", nil)
	if err.Unwrap() != nil {
		t.Error("Expected nil cause")
	}
}

// ============================================================================
// CLIConnectionError Tests
// ============================================================================

func TestCLIConnectionError(t *testing.T) {
	err := NewCLIConnectionError("connection failed", nil)
	if err.Message != "connection failed" {
		t.Errorf("Expected message 'connection failed', got %q", err.Message)
	}
	if err.Error() != "CLI connection error: connection failed" {
		t.Errorf("Expected 'CLI connection error: connection failed', got %q", err.Error())
	}
}

func TestCLIConnectionErrorWithCause(t *testing.T) {
	cause := errors.New("network error")
	err := NewCLIConnectionError("connection failed", cause)
	if err.Error() != "CLI connection error: connection failed: network error" {
		t.Errorf("Expected 'CLI connection error: connection failed: network error', got %q", err.Error())
	}
}

func TestCLIConnectionErrorUnwrap(t *testing.T) {
	cause := errors.New("network error")
	err := NewCLIConnectionError("connection failed", cause)
	if err.Unwrap() == nil {
		t.Error("Expected non-nil cause")
	}
}

// ============================================================================
// CLINotFoundError Tests
// ============================================================================

func TestCLINotFoundError(t *testing.T) {
	err := NewCLINotFoundError("Claude not found", "/usr/bin/claude")
	if err.Message != "Claude not found" {
		t.Errorf("Expected message 'Claude not found', got %q", err.Message)
	}
	if err.CLIPath != "/usr/bin/claude" {
		t.Errorf("Expected CLIPath '/usr/bin/claude', got %q", err.CLIPath)
	}
}

func TestCLINotFoundErrorEmptyPath(t *testing.T) {
	err := NewCLINotFoundError("", "")
	if err.Error() != "Claude Code not found" {
		t.Errorf("Expected 'Claude Code not found', got %q", err.Error())
	}
}

func TestCLINotFoundErrorWithCLIPath(t *testing.T) {
	err := NewCLINotFoundError("not found", "/path/to/claude")
	if err.Error() != "Claude Code not found: /path/to/claude" {
		t.Errorf("Expected 'Claude Code not found: /path/to/claude', got %q", err.Error())
	}
}

func TestCLINotFoundErrorEmptyMessage(t *testing.T) {
	err := NewCLINotFoundError("", "/path/to/claude")
	if err.Message != "Claude Code not found" {
		t.Errorf("Expected default message, got %q", err.Message)
	}
}

// ============================================================================
// ProcessError Tests
// ============================================================================

func TestProcessError(t *testing.T) {
	err := NewProcessError("process failed", nil, "")
	if err.Message != "process failed" {
		t.Errorf("Expected message 'process failed', got %q", err.Message)
	}
	if err.Error() != "process failed" {
		t.Errorf("Expected 'process failed', got %q", err.Error())
	}
}

func TestProcessErrorWithExitCode(t *testing.T) {
	exitCode := 1
	err := NewProcessError("process failed", &exitCode, "")
	if err.Error() != "process failed (exit code: 1)" {
		t.Errorf("Expected 'process failed (exit code: 1)', got %q", err.Error())
	}
}

func TestProcessErrorWithStderr(t *testing.T) {
	err := NewProcessError("process failed", nil, "error output")
	if err.Error() != "process failed\nError output: error output" {
		t.Errorf("Expected 'process failed\\nError output: error output', got %q", err.Error())
	}
}

func TestProcessErrorWithAll(t *testing.T) {
	exitCode := 127
	err := NewProcessError("process failed", &exitCode, "command not found")
	expected := "process failed (exit code: 127)\nError output: command not found"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

// ============================================================================
// CLIJSONDecodeError Tests
// ============================================================================

func TestCLIJSONDecodeError(t *testing.T) {
	err := NewCLIJSONDecodeError("invalid json", errors.New("parse error"))
	if err.Line != "invalid json" {
		t.Errorf("Expected line 'invalid json', got %q", err.Line)
	}
	if err.Error() != "Failed to decode JSON: invalid json..." {
		t.Errorf("Expected 'Failed to decode JSON: invalid json...', got %q", err.Error())
	}
}

func TestCLIJSONDecodeErrorLongLine(t *testing.T) {
	longLine := ""
	for i := 0; i < 200; i++ {
		longLine += "x"
	}
	err := NewCLIJSONDecodeError(longLine, nil)
	// Should truncate to 100 chars
	expected := "Failed to decode JSON: " + longLine[:100] + "..."
	if err.Error() != expected {
		t.Errorf("Expected truncated message, got %q", err.Error())
	}
}

func TestCLIJSONDecodeErrorUnwrap(t *testing.T) {
	cause := errors.New("parse error")
	err := NewCLIJSONDecodeError("invalid json", cause)
	if err.Unwrap() == nil {
		t.Error("Expected non-nil cause")
	}
}

// ============================================================================
// MessageParseError Tests
// ============================================================================

func TestMessageParseError(t *testing.T) {
	data := map[string]any{"key": "value"}
	err := NewMessageParseError("parse failed", data)
	if err.Message != "parse failed" {
		t.Errorf("Expected message 'parse failed', got %q", err.Message)
	}
	if err.Data["key"] != "value" {
		t.Errorf("Expected data[key] = 'value', got %v", err.Data["key"])
	}
}

func TestMessageParseErrorNilData(t *testing.T) {
	err := NewMessageParseError("parse failed", nil)
	if err.Data != nil {
		t.Error("Expected nil data")
	}
	if err.Error() != "parse failed" {
		t.Errorf("Expected 'parse failed', got %q", err.Error())
	}
}

// ============================================================================
// CLIError Tests
// ============================================================================

func TestCLIError(t *testing.T) {
	err := NewCLIError(1, "error occurred", "")
	if err.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", err.ExitCode)
	}
	if err.Error() != "CLI error (exit code 1): error occurred" {
		t.Errorf("Expected 'CLI error (exit code 1): error occurred', got %q", err.Error())
	}
}

func TestCLIErrorWithStderr(t *testing.T) {
	err := NewCLIError(1, "error occurred", "stderr output")
	expected := "CLI error (exit code 1): error occurred\nstderr: stderr output"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

// ============================================================================
// ToolExecutionError Tests
// ============================================================================

func TestToolExecutionError(t *testing.T) {
	err := NewToolExecutionError("test_tool", "execution failed", nil)
	if err.ToolName != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got %q", err.ToolName)
	}
	if err.Error() != "tool execution error (test_tool): execution failed" {
		t.Errorf("Expected 'tool execution error (test_tool): execution failed', got %q", err.Error())
	}
}

func TestToolExecutionErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewToolExecutionError("test_tool", "execution failed", cause)
	if err.Error() != "tool execution error (test_tool): execution failed: underlying error" {
		t.Errorf("Expected 'tool execution error (test_tool): execution failed: underlying error', got %q", err.Error())
	}
}

func TestToolExecutionErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewToolExecutionError("test_tool", "execution failed", cause)
	if err.Unwrap() == nil {
		t.Error("Expected non-nil cause")
	}
}

// ============================================================================
// IsTimeout and IsInterrupted Tests
// ============================================================================

func TestIsTimeout(t *testing.T) {
	if !IsTimeout(ErrTimeout) {
		t.Error("Expected IsTimeout(ErrTimeout) to be true")
	}
	if IsTimeout(ErrInterrupted) {
		t.Error("Expected IsTimeout(ErrInterrupted) to be false")
	}
	if IsTimeout(errors.New("some error")) {
		t.Error("Expected IsTimeout(some error) to be false")
	}
}

func TestIsInterrupted(t *testing.T) {
	if !IsInterrupted(ErrInterrupted) {
		t.Error("Expected IsInterrupted(ErrInterrupted) to be true")
	}
	if IsInterrupted(ErrTimeout) {
		t.Error("Expected IsInterrupted(ErrTimeout) to be false")
	}
	if IsInterrupted(errors.New("some error")) {
		t.Error("Expected IsInterrupted(some error) to be false")
	}
}

// ============================================================================
// Error Wrapping Tests
// ============================================================================

func TestErrorWrapping(t *testing.T) {
	cause := errors.New("root cause")
	err1 := NewClaudeSDKError("sdk_error", "SDK failed", cause)
	err2 := NewCLIConnectionError("connection failed", err1)

	// Test errors.Is
	if !errors.Is(err2, err1) {
		t.Error("Expected err2 to wrap err1")
	}
}

func TestErrorAs(t *testing.T) {
	cause := errors.New("root cause")
	err := NewClaudeSDKError("sdk_error", "SDK failed", cause)

	var sdkErr *ClaudeSDKError
	if !errors.As(err, &sdkErr) {
		t.Error("Expected to extract ClaudeSDKError")
	}
	if sdkErr.Type != "sdk_error" {
		t.Errorf("Expected type 'sdk_error', got %q", sdkErr.Type)
	}
}

// ============================================================================
// SDKError Alias Tests
// ============================================================================

func TestSDKErrorAlias(t *testing.T) {
	// SDKError should be an alias for ClaudeSDKError
	err := &SDKError{
		Type:    "test",
		Message: "test message",
	}
	if err.Type != "test" {
		t.Errorf("Expected type 'test', got %q", err.Type)
	}
}
