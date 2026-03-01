// Package e2e_tests contains end-to-end tests for the Claude Agent SDK.
//
// These tests require a valid API key configuration.
// Run with: go test ./e2e-tests/... -v
//
// To skip tests without API key: tests will be skipped automatically.
// Supported configurations (in priority order):
//   1. ANTHROPIC_API_KEY (direct API key)
//   2. CLAUDE_CODE_USE_FOUNDRY=1 + ANTHROPIC_FOUNDRY_API_KEY (Foundry)
//   3. ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (auto-convert to Foundry)
//   4. ~/.claude/settings.json with ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (auto-convert to Foundry)
package e2e_tests

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ClaudeSettings represents the structure of ~/.claude/settings.json
type ClaudeSettings struct {
	Env map[string]string `json:"env"`
}

// loadClaudeSettings loads settings from ~/.claude/settings.json
func loadClaudeSettings() *ClaudeSettings {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}

	return &settings
}

// FoundryConfig holds Foundry-style configuration
type FoundryConfig struct {
	APIKey  string
	BaseURL string
	Found   bool
}

// foundryOnce ensures Foundry config is only detected once
var foundryOnce sync.Once

// foundryConfig caches the detected Foundry configuration
var foundryConfig *FoundryConfig

// detectFoundryConfig detects and returns Foundry configuration from multiple sources.
// Priority:
// 1. Environment variable ANTHROPIC_API_KEY (not Foundry, just direct key)
// 2. CLAUDE_CODE_USE_FOUNDRY=1 + ANTHROPIC_FOUNDRY_API_KEY
// 3. ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (auto-convert to Foundry)
// 4. ~/.claude/settings.json ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (auto-convert to Foundry)
func detectFoundryConfig() *FoundryConfig {
	foundryOnce.Do(func() {
		cfg := &FoundryConfig{}

		// Priority 1: Direct API key (not Foundry)
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			cfg.APIKey = apiKey
			cfg.BaseURL = os.Getenv("ANTHROPIC_BASE_URL")
			cfg.Found = true
			foundryConfig = cfg
			return
		}

		// Priority 2: Explicit Foundry configuration
		if os.Getenv("CLAUDE_CODE_USE_FOUNDRY") == "1" {
			if apiKey := os.Getenv("ANTHROPIC_FOUNDRY_API_KEY"); apiKey != "" {
				cfg.APIKey = apiKey
				cfg.BaseURL = os.Getenv("ANTHROPIC_FOUNDRY_BASE_URL")
				cfg.Found = true
				foundryConfig = cfg
				return
			}
		}

		// Priority 3: ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (auto-convert to Foundry)
		if authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN"); authToken != "" {
			baseURL := os.Getenv("ANTHROPIC_BASE_URL")
			if baseURL != "" {
				cfg.APIKey = authToken
				cfg.BaseURL = baseURL
				cfg.Found = true
				// Inject into environment as Foundry config
				os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
				os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", authToken)
				os.Setenv("ANTHROPIC_FOUNDRY_BASE_URL", baseURL)
				foundryConfig = cfg
				return
			}
		}

		// Priority 4: ~/.claude/settings.json (auto-convert to Foundry)
		settings := loadClaudeSettings()
		if settings != nil && settings.Env != nil {
			authToken := settings.Env["ANTHROPIC_AUTH_TOKEN"]
			baseURL := settings.Env["ANTHROPIC_BASE_URL"]
			if authToken != "" && baseURL != "" {
				cfg.APIKey = authToken
				cfg.BaseURL = baseURL
				cfg.Found = true
				// Inject into environment as Foundry config
				os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
				os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", authToken)
				os.Setenv("ANTHROPIC_FOUNDRY_BASE_URL", baseURL)
				foundryConfig = cfg
				return
			}
			// Fallback: only ANTHROPIC_AUTH_TOKEN without base URL
			if authToken != "" {
				cfg.APIKey = authToken
				cfg.Found = true
				foundryConfig = cfg
				return
			}
		}

		foundryConfig = cfg
	})

	return foundryConfig
}

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
func DefaultTestConfig() *TestConfig {
	// Detect Foundry configuration (auto-injects env vars)
	foundryCfg := detectFoundryConfig()

	// Get model from env or settings
	model := os.Getenv("CLAUDE_TEST_MODEL")
	if model == "" {
		settings := loadClaudeSettings()
		if settings != nil && settings.Env != nil {
			if m := settings.Env["ANTHROPIC_MODEL"]; m != "" {
				model = m
			}
		}
	}
	if model == "" {
		model = types.ModelSonnet
	}

	return &TestConfig{
		APIKey:   foundryCfg.APIKey,
		BaseURL:  foundryCfg.BaseURL,
		Model:    model,
		MaxTurns: 3,
		Timeout:  60 * time.Second,
	}
}

// HasAPIKey returns true if an API key is available.
func HasAPIKey() bool {
	cfg := detectFoundryConfig()
	return cfg.Found && cfg.APIKey != ""
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

// init automatically configures Foundry environment variables at package load time.
// This ensures the CLI subprocess receives the correct environment configuration.
func init() {
	// Auto-detect and inject Foundry configuration
	detectFoundryConfig()
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