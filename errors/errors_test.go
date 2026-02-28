package errors

import (
	"errors"
	"strings"
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

// ============================================================================
// Error Type Hierarchy Tests
// ============================================================================

func TestErrorTypeHierarchy(t *testing.T) {
	// Test that specific error types can be checked
	tests := []struct {
		name          string
		err           error
		isTimeout     bool
		isInterrupted bool
	}{
		{"ErrTimeout", ErrTimeout, true, false},
		{"ErrInterrupted", ErrInterrupted, false, true},
		{"ErrNoAPIKey", ErrNoAPIKey, false, false},
		{"ErrNotInstalled", ErrNotInstalled, false, false},
		{"ErrConnectionFailed", ErrConnectionFailed, false, false},
		{"ClaudeSDKError", NewClaudeSDKError("test", "msg", nil), false, false},
		{"CLIConnectionError", NewCLIConnectionError("conn", nil), false, false},
		{"CLINotFoundError", NewCLINotFoundError("not found", ""), false, false},
		{"ProcessError", NewProcessError("proc", nil, ""), false, false},
		{"CLIJSONDecodeError", NewCLIJSONDecodeError("line", nil), false, false},
		{"MessageParseError", NewMessageParseError("parse", nil), false, false},
		{"CLIError", NewCLIError(1, "cli", ""), false, false},
		{"ToolExecutionError", NewToolExecutionError("tool", "exec", nil), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTimeout(tt.err); got != tt.isTimeout {
				t.Errorf("IsTimeout(%s) = %v, want %v", tt.name, got, tt.isTimeout)
			}
			if got := IsInterrupted(tt.err); got != tt.isInterrupted {
				t.Errorf("IsInterrupted(%s) = %v, want %v", tt.name, got, tt.isInterrupted)
			}
		})
	}
}

// ============================================================================
// Error Message Formatting Tests
// ============================================================================

func TestErrorMessagesContainRelevantInfo(t *testing.T) {
	t.Run("ProcessError contains exit code and stderr", func(t *testing.T) {
		exitCode := 127
		err := NewProcessError("process failed", &exitCode, "command not found")
		errStr := err.Error()

		if !containsString(errStr, "process failed") {
			t.Errorf("Expected error to contain 'process failed', got %q", errStr)
		}
		if !containsString(errStr, "exit code: 127") {
			t.Errorf("Expected error to contain 'exit code: 127', got %q", errStr)
		}
		if !containsString(errStr, "command not found") {
			t.Errorf("Expected error to contain 'command not found', got %q", errStr)
		}
	})

	t.Run("CLIJSONDecodeError truncates long lines", func(t *testing.T) {
		longLine := strings.Repeat("x", 200)
		err := NewCLIJSONDecodeError(longLine, nil)
		errStr := err.Error()

		// Should truncate and add "..."
		if len(errStr) > 150 {
			t.Errorf("Expected truncated error message, got length %d: %q", len(errStr), errStr)
		}
		if !containsString(errStr, "...") {
			t.Errorf("Expected error to contain '...' for truncation, got %q", errStr)
		}
	})

	t.Run("CLIError includes stderr when present", func(t *testing.T) {
		err := NewCLIError(1, "error occurred", "stderr output")
		errStr := err.Error()

		if !containsString(errStr, "exit code 1") {
			t.Errorf("Expected error to contain 'exit code 1', got %q", errStr)
		}
		if !containsString(errStr, "error occurred") {
			t.Errorf("Expected error to contain 'error occurred', got %q", errStr)
		}
		if !containsString(errStr, "stderr output") {
			t.Errorf("Expected error to contain 'stderr output', got %q", errStr)
		}
	})
}

// ============================================================================
// Error Chaining Tests
// ============================================================================

func TestErrorChaining(t *testing.T) {
	t.Run("ClaudeSDKError chains cause correctly", func(t *testing.T) {
		cause := errors.New("root cause")
		err := NewClaudeSDKError("type", "message", cause)

		unwrapped := err.Unwrap()
		if unwrapped == nil {
			t.Fatal("Expected non-nil unwrapped error")
		}
		if unwrapped.Error() != "root cause" {
			t.Errorf("Expected unwrapped error 'root cause', got %q", unwrapped.Error())
		}

		// Test errors.Is works through the chain
		if !errors.Is(err, cause) {
			t.Error("Expected errors.Is to find the cause")
		}
	})

	t.Run("CLIConnectionError chains cause correctly", func(t *testing.T) {
		cause := errors.New("network error")
		err := NewCLIConnectionError("connection failed", cause)

		unwrapped := err.Unwrap()
		if unwrapped == nil {
			t.Fatal("Expected non-nil unwrapped error")
		}
		if unwrapped.Error() != "network error" {
			t.Errorf("Expected unwrapped error 'network error', got %q", unwrapped.Error())
		}
	})

	t.Run("ToolExecutionError chains cause correctly", func(t *testing.T) {
		cause := errors.New("execution failed")
		err := NewToolExecutionError("my_tool", "tool error", cause)

		unwrapped := err.Unwrap()
		if unwrapped == nil {
			t.Fatal("Expected non-nil unwrapped error")
		}
		if unwrapped.Error() != "execution failed" {
			t.Errorf("Expected unwrapped error 'execution failed', got %q", unwrapped.Error())
		}
	})

	t.Run("CLIJSONDecodeError chains cause correctly", func(t *testing.T) {
		cause := errors.New("json parse error")
		err := NewCLIJSONDecodeError(`{"invalid":}`, cause)

		unwrapped := err.Unwrap()
		if unwrapped == nil {
			t.Fatal("Expected non-nil unwrapped error")
		}
		if unwrapped.Error() != "json parse error" {
			t.Errorf("Expected unwrapped error 'json parse error', got %q", unwrapped.Error())
		}
	})
}

// ============================================================================
// Error Interface Compliance Tests
// ============================================================================

func TestErrorInterfaceCompliance(t *testing.T) {
	// Ensure all error types implement the error interface
	var _ error = (*ClaudeSDKError)(nil)
	var _ error = (*CLIConnectionError)(nil)
	var _ error = (*CLINotFoundError)(nil)
	var _ error = (*ProcessError)(nil)
	var _ error = (*CLIJSONDecodeError)(nil)
	var _ error = (*MessageParseError)(nil)
	var _ error = (*CLIError)(nil)
	var _ error = (*ToolExecutionError)(nil)
	var _ error = (*SDKError)(nil) // Alias
}

// ============================================================================
// Nil and Edge Case Tests
// ============================================================================

func TestErrorNilHandling(t *testing.T) {
	t.Run("ClaudeSDKError with nil cause", func(t *testing.T) {
		err := NewClaudeSDKError("type", "message", nil)
		if err.Unwrap() != nil {
			t.Error("Expected nil unwrap for nil cause")
		}
	})

	t.Run("CLIConnectionError with nil cause", func(t *testing.T) {
		err := NewCLIConnectionError("message", nil)
		if err.Unwrap() != nil {
			t.Error("Expected nil unwrap for nil cause")
		}
	})

	t.Run("ProcessError with nil exit code", func(t *testing.T) {
		err := NewProcessError("message", nil, "stderr")
		if err.ExitCode != nil {
			t.Error("Expected nil exit code")
		}
	})

	t.Run("MessageParseError with nil data", func(t *testing.T) {
		err := NewMessageParseError("message", nil)
		if err.Data != nil {
			t.Error("Expected nil data")
		}
	})
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}
