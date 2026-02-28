// Package e2e_tests contains end-to-end tests for the Claude Agent SDK.
//
// These tests require a valid ANTHROPIC_API_KEY environment variable.
// Run with: go test ./e2e-tests/... -v
//
// To skip tests without API key: tests will be skipped automatically.
package e2e_tests

import (
	"os"
	"testing"
	"time"

	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// SkipIfNoAPIKey skips the test if ANTHROPIC_API_KEY is not set.
func SkipIfNoAPIKey(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}
}

// TestConfig holds E2E test configuration.
type TestConfig struct {
	// APIKey is the Anthropic API key.
	APIKey string
	// Model is the model to use for tests.
	Model string
	// MaxTurns is the maximum number of turns for tests.
	MaxTurns int
	// Timeout is the timeout for test operations.
	Timeout time.Duration
}

// DefaultTestConfig returns the default test configuration.
func DefaultTestConfig() *TestConfig {
	model := os.Getenv("CLAUDE_TEST_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	return &TestConfig{
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		Model:    model,
		MaxTurns: 3,
		Timeout:  60 * time.Second,
	}
}

// HasAPIKey returns true if API key is available.
func HasAPIKey() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != ""
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
