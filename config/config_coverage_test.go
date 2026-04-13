package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestDetect_DirectAPIKeyWithoutBaseURL tests Priority 1 with API key but no base URL
func TestDetect_DirectAPIKeyWithoutBaseURL(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Set direct API key without base URL
	os.Setenv("ANTHROPIC_API_KEY", "test-api-key-no-base")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	cfg := Detect()

	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "test-api-key-no-base" {
		t.Errorf("Expected APIKey 'test-api-key-no-base', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "" {
		t.Errorf("Expected BaseURL to be empty, got '%s'", cfg.BaseURL)
	}
}

// TestDetect_FoundryModeWithoutFoundryBaseURL tests Priority 2 with foundry API key but no base URL
func TestDetect_FoundryModeWithoutFoundryBaseURL(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Set Foundry config without base URL
	os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
	os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", "foundry-api-key-no-base")
	defer func() {
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	}()

	cfg := Detect()

	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "foundry-api-key-no-base" {
		t.Errorf("Expected APIKey 'foundry-api-key-no-base', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "" {
		t.Errorf("Expected BaseURL to be empty, got '%s'", cfg.BaseURL)
	}
}

// TestDetect_FoundryModeWithoutAPIKey tests Priority 2 with CLAUDE_CODE_USE_FOUNDRY=1 but no API key
func TestDetect_FoundryModeWithoutAPIKey(t *testing.T) {
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

	// Set Foundry flag without API key
	os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
	defer os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")

	cfg := Detect()

	// Should not find config since no ANTHROPIC_FOUNDRY_API_KEY
	if cfg.Found {
		t.Error("Expected Found to be false when CLAUDE_CODE_USE_FOUNDRY=1 but no ANTHROPIC_FOUNDRY_API_KEY")
	}
}

// TestDetect_FoundryModeNotEnabled tests when CLAUDE_CODE_USE_FOUNDRY is set but not "1"
func TestDetect_FoundryModeNotEnabled(t *testing.T) {
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

	// Set Foundry flag to something other than "1"
	os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "true")
	os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", "foundry-key")
	defer func() {
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	}()

	cfg := Detect()

	// Should not use foundry mode since CLAUDE_CODE_USE_FOUNDRY != "1"
	if cfg.Found {
		t.Error("Expected Found to be false when CLAUDE_CODE_USE_FOUNDRY is not '1'")
	}
}

// TestDetect_AuthTokenWithoutBaseURL tests Priority 3 with auth token but no base URL
func TestDetect_AuthTokenWithoutBaseURL(t *testing.T) {
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

	// Set auth token without base URL (should fall through to settings.json)
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "auth-token-no-base")
	defer os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	cfg := Detect()

	// Should not find config since ANTHROPIC_BASE_URL is required for Priority 3
	if cfg.Found {
		t.Error("Expected Found to be false when ANTHROPIC_AUTH_TOKEN set but no ANTHROPIC_BASE_URL")
	}
}

// TestDetect_SettingsWithAuthTokenAndBaseURL tests Priority 4 with valid settings.json
func TestDetect_SettingsWithAuthTokenAndBaseURL(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Create a temp home dir with settings.json
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settings := Settings{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "settings-auth-token",
			"ANTHROPIC_BASE_URL":   "https://settings.example.com",
		},
	}
	settingsContent, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal settings: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, settingsContent, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := Detect()

	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "settings-auth-token" {
		t.Errorf("Expected APIKey 'settings-auth-token', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://settings.example.com" {
		t.Errorf("Expected BaseURL 'https://settings.example.com', got '%s'", cfg.BaseURL)
	}
}

// TestDetect_SettingsWithOnlyAuthToken tests Priority 4 fallback with only auth token
func TestDetect_SettingsWithOnlyAuthToken(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Create a temp home dir with settings.json containing only auth token
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settings := Settings{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "settings-only-token",
		},
	}
	settingsContent, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal settings: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, settingsContent, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := Detect()

	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "settings-only-token" {
		t.Errorf("Expected APIKey 'settings-only-token', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "" {
		t.Errorf("Expected BaseURL to be empty, got '%s'", cfg.BaseURL)
	}
}

// TestDetect_SettingsWithEmptyEnv tests Priority 4 with empty env map
func TestDetect_SettingsWithEmptyEnv(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Create a temp home dir with settings.json with empty env
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settings := Settings{
		Env: map[string]string{},
	}
	settingsContent, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal settings: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, settingsContent, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := Detect()

	if cfg.Found {
		t.Error("Expected Found to be false when settings has empty env")
	}
}

// TestDetect_PriorityOrder tests that priority 3 overrides settings
func TestDetect_Priority3OverridesSettings(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Create a temp home dir with settings.json
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settings := Settings{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "settings-token",
			"ANTHROPIC_BASE_URL":   "https://settings.example.com",
		},
	}
	settingsContent, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal settings: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, settingsContent, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Set priority 3 env vars
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "env-auth-token")
	os.Setenv("ANTHROPIC_BASE_URL", "https://env.example.com")
	defer func() {
		os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
		os.Unsetenv("ANTHROPIC_BASE_URL")
	}()

	cfg := Detect()

	// Priority 3 should win over settings.json
	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
	if cfg.APIKey != "env-auth-token" {
		t.Errorf("Expected Priority 3 to override settings, got APIKey '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://env.example.com" {
		t.Errorf("Expected BaseURL from env, got '%s'", cfg.BaseURL)
	}
}

// TestLoadSettings_InvalidJSON tests LoadSettings with an invalid JSON file
func TestLoadSettings_InvalidJSON(t *testing.T) {
	// Create a temp home dir with invalid settings.json
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	invalidJSON := `{invalid json content`
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	settings, err := LoadSettings()

	if err == nil {
		t.Error("Expected error when settings file contains invalid JSON")
	}
	if settings != nil {
		t.Error("Expected nil settings when JSON is invalid")
	}
}

// TestLoadSettings_EmptyFile tests LoadSettings with an empty file
func TestLoadSettings_EmptyFile(t *testing.T) {
	// Create a temp home dir with empty settings.json
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	settings, err := LoadSettings()

	if err == nil {
		t.Error("Expected error when settings file is empty")
	}
	if settings != nil {
		t.Error("Expected nil settings when file is empty")
	}
}

// TestLoadSettings_FileWithNoEnv tests LoadSettings with file that has no env field
func TestLoadSettings_FileWithNoEnv(t *testing.T) {
	// Create a temp home dir with settings.json without env field
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settingsContent := `{"otherField": "value"}`
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
	if settings.Env != nil {
		t.Errorf("Expected Env to be nil, got %v", settings.Env)
	}
}

// TestDetect_SettingsWithNilEnv tests Priority 4 with nil env map
func TestDetect_SettingsWithNilEnv(t *testing.T) {
	// Clear all relevant env vars
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Create a temp home dir with settings.json without env field
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settingsContent := `{"otherField": "value"}`
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := Detect()

	if cfg.Found {
		t.Error("Expected Found to be false when settings has nil env")
	}
}

// TestDetect_Priority2OverridesPriority3 tests that Priority 2 overrides Priority 3
func TestDetect_Priority2OverridesPriority3(t *testing.T) {
	// Clear all relevant env vars first
	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_BASE_URL")
	os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
	os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
	os.Unsetenv("ANTHROPIC_AUTH_TOKEN")

	// Set both priority 2 and priority 3
	os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
	os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", "foundry-key")
	os.Setenv("ANTHROPIC_FOUNDRY_BASE_URL", "https://foundry.example.com")
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "auth-token")
	os.Setenv("ANTHROPIC_BASE_URL", "https://custom.example.com")
	defer func() {
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_BASE_URL")
		os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
		os.Unsetenv("ANTHROPIC_BASE_URL")
	}()

	cfg := Detect()

	// Priority 2 should win
	if cfg.APIKey != "foundry-key" {
		t.Errorf("Expected Priority 2 to override, got APIKey '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://foundry.example.com" {
		t.Errorf("Expected BaseURL from Priority 2, got '%s'", cfg.BaseURL)
	}
}

// TestDetect_AllPrioritiesSet tests that Priority 1 wins when all priorities are set
func TestDetect_AllPrioritiesSet(t *testing.T) {
	// Create a temp home dir with settings.json
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	settings := Settings{
		Env: map[string]string{
			"ANTHROPIC_AUTH_TOKEN": "settings-token",
			"ANTHROPIC_BASE_URL":   "https://settings.example.com",
		},
	}
	settingsContent, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal settings: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, settingsContent, 0644); err != nil {
		t.Fatalf("Failed to write settings.json: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Set all priorities
	os.Setenv("ANTHROPIC_API_KEY", "direct-key")
	os.Setenv("CLAUDE_CODE_USE_FOUNDRY", "1")
	os.Setenv("ANTHROPIC_FOUNDRY_API_KEY", "foundry-key")
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "auth-token")
	os.Setenv("ANTHROPIC_BASE_URL", "https://custom.example.com")
	defer func() {
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("CLAUDE_CODE_USE_FOUNDRY")
		os.Unsetenv("ANTHROPIC_FOUNDRY_API_KEY")
		os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
		os.Unsetenv("ANTHROPIC_BASE_URL")
	}()

	cfg := Detect()

	// Priority 1 should win
	if cfg.APIKey != "direct-key" {
		t.Errorf("Expected Priority 1 to win, got APIKey '%s'", cfg.APIKey)
	}
}

// TestConfig_StructFields tests that Config struct fields are properly set
func TestConfig_StructFields(t *testing.T) {
	cfg := &Config{
		APIKey:  "test-key",
		BaseURL: "https://test.example.com",
		Found:   true,
	}

	if cfg.APIKey != "test-key" {
		t.Errorf("Expected APIKey 'test-key', got '%s'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://test.example.com" {
		t.Errorf("Expected BaseURL 'https://test.example.com', got '%s'", cfg.BaseURL)
	}
	if !cfg.Found {
		t.Error("Expected Found to be true")
	}
}

// TestSettings_StructFields tests that Settings struct fields are properly set
func TestSettings_StructFields(t *testing.T) {
	settings := &Settings{
		Env: map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		},
	}

	if len(settings.Env) != 2 {
		t.Errorf("Expected 2 env entries, got %d", len(settings.Env))
	}
	if settings.Env["KEY1"] != "value1" {
		t.Errorf("Expected KEY1='value1', got '%s'", settings.Env["KEY1"])
	}
	if settings.Env["KEY2"] != "value2" {
		t.Errorf("Expected KEY2='value2', got '%s'", settings.Env["KEY2"])
	}
}
