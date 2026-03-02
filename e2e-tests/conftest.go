// Package e2e_tests contains end-to-end tests for the Claude Agent SDK.
//
// These tests require a valid API key configuration.
// Run with: go test ./e2e-tests/... -v
//
// To skip tests without API key: tests will be skipped automatically.
// Supported configurations (in priority order):
//  1. ANTHROPIC_API_KEY (direct API key)
//  2. CLAUDE_CODE_USE_FOUNDRY=1 + ANTHROPIC_FOUNDRY_API_KEY (Foundry)
//  3. ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (auto-convert to Foundry)
//  4. ~/.claude/settings.json with ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (auto-convert to Foundry)
package e2e_tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/unitsvc/claude-agent-sdk-golang/config"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// SkipIfNoAPIKey skips the test if no API key is configured.
func SkipIfNoAPIKey(t *testing.T) {
	if !HasAPIKey() {
		t.Skip("Skipping: No API key configured")
	}
}

// TestConfig holds E2E test configuration.
type TestConfig struct {
	// APIKey is the Anthropic API key.
	APIKey string
	// BaseURL is the API base URL (optional, for Foundry mode).
	BaseURL string
	// Model is the model to use for tests.
	Model string
	// MaxTurns is the maximum number of turns for tests.
	MaxTurns int
	// Timeout is the timeout for test operations.
	Timeout time.Duration
}

// DefaultTestConfig returns the default test configuration.
// It detects API key from multiple sources and configures the environment
// for CLI subprocess to work correctly with Foundry mode if needed.
func DefaultTestConfig() *TestConfig {
	// Detect configuration from multiple sources
	cfg := config.Detect()

	// For Foundry mode (ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL), we need to
	// inject environment variables so the CLI subprocess can recognize them.
	// This is done explicitly here, not via init(), for clarity.
	if cfg.Found && cfg.BaseURL != "" {
		// Check if we need to inject Foundry environment variables
		// (when using ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL pattern)
		if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("CLAUDE_CODE_USE_FOUNDRY") != "1" {
			os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
			os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", cfg.APIKey)
			os.Setenv("ANTHROPIC_FOUNDRY_BASE_URL", cfg.BaseURL)
		}
	}

	// Get model from env or settings
	model := os.Getenv("CLAUDE_TEST_MODEL")
	if model == "" {
		settings, err := config.LoadSettings()
		if err == nil && settings.Env != nil {
			if m := settings.Env["ANTHROPIC_MODEL"]; m != "" {
				model = m
			}
		}
	}
	if model == "" {
		model = types.ModelSonnet
	}

	return &TestConfig{
		APIKey:   cfg.APIKey,
		BaseURL:  cfg.BaseURL,
		Model:    model,
		MaxTurns: 3,
		Timeout:  60 * time.Second,
	}
}

// HasAPIKey returns true if an API key is available.
func HasAPIKey() bool {
	return config.HasAPIKey()
}

// Helper functions for pointer types

// permissionModePtr returns a pointer to a PermissionMode.
func permissionModePtr(mode types.PermissionMode) *types.PermissionMode {
	return &mode
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// float64Ptr returns a pointer to a float64.
func float64Ptr(f float64) *float64 {
	return &f
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}

// truncateString truncates a string to maxLen characters for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatContent formats content blocks for logging
func formatContent(blocks []types.ContentBlock) string {
	var result string
	for _, block := range blocks {
		// Try TextBlock
		if tb, ok := block.(types.TextBlock); ok {
			result += tb.Text
		}
	}
	return truncateString(result, 100)
}

// formatResult formats result string pointer for logging
func formatResult(result *string) string {
	if result == nil {
		return "<nil>"
	}
	return truncateString(*result, 100)
}

// consumeMessagesUntilResult consumes messages from the channel until a ResultMessage
// is received or the context is done. This prevents tests from hanging indefinitely
// when the message channel doesn't close properly.
// After receiving ResultMessage, it continues draining the channel in the background
// to prevent blocking the SDK goroutines.
// Returns the number of messages consumed and whether a ResultMessage was found.
func consumeMessagesUntilResult(ctx context.Context, msgChan <-chan types.Message) (count int, foundResult bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed
				return
			}
			count++
			if _, isResult := msg.(*types.ResultMessage); isResult {
				foundResult = true
				// Continue draining the channel in the background to prevent
				// blocking SDK goroutines that are still sending
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-msgChan:
							if !ok {
								return
							}
						}
					}
				}()
				return
			}
		}
	}
}

// consumeAllMessagesUntilResult consumes all messages including the result message.
// It processes each message through the provided handler function.
// Returns when a ResultMessage is received or context is done.
// After receiving ResultMessage, it continues draining the channel in the background.
func consumeAllMessagesUntilResult(ctx context.Context, msgChan <-chan types.Message, handler func(types.Message)) (count int, foundResult bool) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgChan:
			if !ok {
				// Channel closed
				return
			}
			count++
			if handler != nil {
				handler(msg)
			}
			if _, isResult := msg.(*types.ResultMessage); isResult {
				foundResult = true
				// Continue draining the channel in the background
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-msgChan:
							if !ok {
								return
							}
						}
					}
				}()
				return
			}
		}
	}
}

// ============================================================================
// Verbose Test Helpers - Display execution status during tests
// ============================================================================

// TestLogger provides verbose logging for E2E tests
type TestLogger struct {
	t       *testing.T
	prefix  string
	enabled bool
}

// NewTestLogger creates a new test logger
func NewTestLogger(t *testing.T, prefix string) *TestLogger {
	return &TestLogger{
		t:       t,
		prefix:  prefix,
		enabled: true,
	}
}

// Log logs a message with the test logger prefix
func (l *TestLogger) Log(format string, args ...interface{}) {
	if l.enabled {
		msg := fmt.Sprintf(format, args...)
		l.t.Logf("[%s] %s", l.prefix, msg)
	}
}

// Step logs a test step with a clear marker
func (l *TestLogger) Step(step string) {
	if l.enabled {
		l.t.Logf("\n========== [%s] %s ==========", l.prefix, step)
	}
}

// Message logs a received message with details
func (l *TestLogger) Message(msgType string, details string) {
	if l.enabled {
		l.t.Logf("  📩 [%s] %s", msgType, details)
	}
}

// Status logs a status update
func (l *TestLogger) Status(status string) {
	if l.enabled {
		l.t.Logf("  ⚡ %s", status)
	}
}

// Error logs an error
func (l *TestLogger) Error(err error) {
	if l.enabled {
		l.t.Logf("  ❌ Error: %v", err)
	}
}

// Result logs the final result
func (l *TestLogger) Result(success bool, details string) {
	if l.enabled {
		if success {
			l.t.Logf("  ✅ Result: %s", details)
		} else {
			l.t.Logf("  ❌ Result: %s", details)
		}
	}
}

// ConsumeMessagesVerbose consumes messages with verbose output showing execution status
func ConsumeMessagesVerbose(ctx context.Context, t *testing.T, msgChan <-chan types.Message, testName string) (count int, foundResult bool, resultMsg *types.ResultMessage) {
	logger := NewTestLogger(t, testName)
	logger.Step("Waiting for messages")

	for {
		select {
		case <-ctx.Done():
			logger.Status("Context done (timeout)")
			return
		case msg, ok := <-msgChan:
			if !ok {
				logger.Status("Channel closed")
				return
			}
			count++

			switch m := msg.(type) {
			case *types.AssistantMessage:
				details := formatContent(m.Content)
				logger.Message("AssistantMessage", details)
			case *types.ResultMessage:
				foundResult = true
				resultMsg = m
				costStr := "<nil>"
				if m.TotalCostUSD != nil {
					costStr = fmt.Sprintf("%.6f", *m.TotalCostUSD)
				}
				logger.Result(!m.IsError, fmt.Sprintf("IsError=%v, TotalCostUSD=%s", m.IsError, costStr))
				// Continue draining in background
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-msgChan:
							if !ok {
								return
							}
						}
					}
				}()
				return
			default:
				logger.Message("Message", fmt.Sprintf("%T", msg))
			}
		}
	}
}

// CreateVerboseHook creates a hook callback that logs execution status
func CreateVerboseHook(t *testing.T, hookName string) types.HookCallback {
	return &verboseHookCallback{
		t:        t,
		hookName: hookName,
	}
}

type verboseHookCallback struct {
	t        *testing.T
	hookName string
}

func (h *verboseHookCallback) Execute(input types.HookInput, toolUseID *string, context types.HookContext) (types.HookJSONOutput, error) {
	h.t.Logf("  🪝 [%s] Hook executed: %T", h.hookName, input)
	return types.SyncHookJSONOutput{
		Continue_: types.Bool(true),
	}, nil
}

// PrintTestHeader prints a formatted test header
func PrintTestHeader(t *testing.T, testName string) {
	t.Logf("\n============================================================")
	t.Logf("  TEST: %s", testName)
	t.Logf("============================================================")
}

// PrintTestSummary prints a formatted test summary
func PrintTestSummary(t *testing.T, testName string, success bool, messageCount int, duration time.Duration) {
	status := "✅ PASSED"
	if !success {
		status = "❌ FAILED"
	}
	t.Logf("\n------------------------------------------------------------")
	t.Logf("  %s: %s", status, testName)
	t.Logf("  Messages: %d | Duration: %v", messageCount, duration)
	t.Logf("------------------------------------------------------------")
}