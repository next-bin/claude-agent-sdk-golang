// Package config provides configuration detection for the Claude Agent SDK.
//
// It supports multiple configuration sources with the following priority:
//  1. ANTHROPIC_API_KEY environment variable (direct API key)
//  2. CLAUDE_CODE_USE_FOUNDRY=1 + ANTHROPIC_FOUNDRY_API_KEY (Foundry mode)
//  3. ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (custom endpoint)
//  4. ~/.claude/settings.json with ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL
//
// Example usage:
//
//	cfg := config.Detect()
//	if cfg.Found {
//	    client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
//	        // Use cfg.APIKey and cfg.BaseURL as needed
//	    })
//	}
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds detected SDK configuration.
type Config struct {
	// APIKey is the Anthropic API key or auth token.
	APIKey string

	// BaseURL is the API base URL (optional, for Foundry or custom endpoints).
	BaseURL string

	// Found indicates whether a valid configuration was detected.
	Found bool
}

// Settings represents the structure of ~/.claude/settings.json.
type Settings struct {
	// Env contains environment variables from the settings file.
	Env map[string]string `json:"env"`
}

// Detect automatically detects SDK configuration from multiple sources.
//
// Priority order:
//  1. ANTHROPIC_API_KEY environment variable (direct API key)
//  2. CLAUDE_CODE_USE_FOUNDRY=1 + ANTHROPIC_FOUNDRY_API_KEY (Foundry mode)
//  3. ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL (custom endpoint)
//  4. ~/.claude/settings.json with ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL
//
// This function only reads configuration and does not modify environment variables.
func Detect() *Config {
	cfg := &Config{}

	// Priority 1: Direct API key
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
		cfg.BaseURL = os.Getenv("ANTHROPIC_BASE_URL")
		cfg.Found = true
		return cfg
	}

	// Priority 2: Explicit Foundry configuration
	if os.Getenv("CLAUDE_CODE_USE_FOUNDRY") == "1" {
		if apiKey := os.Getenv("ANTHROPIC_FOUNDRY_API_KEY"); apiKey != "" {
			cfg.APIKey = apiKey
			cfg.BaseURL = os.Getenv("ANTHROPIC_FOUNDRY_BASE_URL")
			cfg.Found = true
			return cfg
		}
	}

	// Priority 3: ANTHROPIC_AUTH_TOKEN + ANTHROPIC_BASE_URL
	if authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN"); authToken != "" {
		baseURL := os.Getenv("ANTHROPIC_BASE_URL")
		if baseURL != "" {
			cfg.APIKey = authToken
			cfg.BaseURL = baseURL
			cfg.Found = true
			return cfg
		}
	}

	// Priority 4: ~/.claude/settings.json
	settings, err := LoadSettings()
	if err == nil && settings.Env != nil {
		authToken := settings.Env["ANTHROPIC_AUTH_TOKEN"]
		baseURL := settings.Env["ANTHROPIC_BASE_URL"]
		if authToken != "" && baseURL != "" {
			cfg.APIKey = authToken
			cfg.BaseURL = baseURL
			cfg.Found = true
			return cfg
		}
		// Fallback: only ANTHROPIC_AUTH_TOKEN without base URL
		if authToken != "" {
			cfg.APIKey = authToken
			cfg.Found = true
			return cfg
		}
	}

	return cfg
}

// LoadSettings loads settings from ~/.claude/settings.json.
// Returns nil if the file doesn't exist or cannot be parsed.
func LoadSettings() (*Settings, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// HasAPIKey returns true if an API key is available from any source.
// This is a convenience function equivalent to Detect().Found.
func HasAPIKey() bool {
	return Detect().Found
}
