package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect_Priority1_DirectAPIKey(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Set direct API key
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	os.Setenv("ANTHROPIC_BASE_URL", "https://test.example.com")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ANTHROPIC_BASE_URL")
	}()

	cfg := Detect()

	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey 'test-api-key', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://test.example.com" {
		t.Errorf("Expected BaseURL 'https://test.example.com', got '%s'", cfg.BaseURL)
	}
}

func TestDetect_Priority2_FoundryMode(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Set Foundry config
	os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
	os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", "foundry-api-key")
	os.Setenv("ANTHROPIC_FOUNDRY_BASE_URL", "https://foundry.example.com")
	defer func() {
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	}()

	cfg := Detect()

	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "foundry-api-key" {
		t.Errorf("Expected APIKey 'foundry-api-key', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://foundry.example.com" {
		t.Errorf("Expected BaseURL 'https://foundry.example.com', got '%s'", cfg.BaseURL)
	}
}

func TestDetect_Priority3_AuthToken(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Set auth token config
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "auth-token")
	os.Setenv("ANTHROPIC_BASE_URL", "https://custom.example.com")
	defer func() {
		os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
		os.Unsetenv("ANTHROPIC_BASE_URL")
	}()

	cfg := Detect()

	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "auth-token" {
		t.Errorf("Expected APIKey 'auth-token', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://custom.example.com" {
		t.Errorf("Expected BaseURL 'https://custom.example.com', got '%s'", cfg.BaseURL)
	}
}

func TestDetect_NoConfig(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Create a temp home dir without settings.json
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := Detect()

	if cfg.Found {
		t.Error("Expected Found to be false when no config is available")
	}
}

func TestDetect_Priority1_OverridesPriority2(t *testing.T) {
	// Clear all relevant env vars first
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Set both priority 1 and priority 2
	os.Setenv("ANTHROPIC_API_KEY", "direct-key")
	os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
	os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", "foundry-key")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	}()

	cfg := Detect()

	// Priority 1 should win
	if cfg.APIKey != "direct-key" {
		t.Errorf("Expected Priority 1 to override, got APIKey '%s'", cfg.APIKey)
	}
}

func TestLoadSettings_FileNotExists(t *testing.T) {
	// Create a temp home dir without settings.json
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	settings, err := LoadSettings()

	if err == nil {
		t.Error("Expected error when settings file doesn't exist")
	}
	if settings != nil {
		t.Error("Expected nil settings when file doesn't exist")
	}
}

func TestLoadSettings_ValidFile(t *testing.T) {
	// Create a temp home dir with settings.json
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settingsContent := `{"env": {"ANTHROPIC_AUTH_TOKEN": "test-token", "ANTHROPIC_BASE_URL": "https://test.example.com"}}`
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	settings, err := LoadSettings()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if settings == nil {
		t.Fatal("Expected non-nil settings")
	}
	if settings.Env["ANTHROPIC_AUTH_TOKEN"] != "test-token" {
		t.Errorf("Expected ANTHROPIC_AUTH_TOKEN 'test-token', got '%s'", settings.Env["ANTHROPIC_AUTH_TOKEN"])
	}
}

func TestHasAPIKey(t *testing.T) {
	// Clear all env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Create a temp home dir without settings.json
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	if HasAPIKey() {
		t.Error("Expected HasAPIKey to return false when no config available")
	}

	// Set API key
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	if !HasAPIKey() {
		t.Error("Expected HasAPIKey to return true when API key is set")
	}
}
