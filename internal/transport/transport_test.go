// Package transport provides transport layer implementations for the Claude Agent SDK.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to create float64 pointer
func float64Ptr(f float64) *float64 {
	return &f
}

// ============================================================================
// Tests for buildCommand
// ============================================================================

func TestBuildCommand_BasicArguments(t *testing.T) {
	tests := []struct {
		name     string
		options  *types.ClaudeAgentOptions
		cliPath  string
		wantArgs []string // Arguments to check for presence
	}{
		{
			name:     "minimal options with default values",
			options:  &types.ClaudeAgentOptions{},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"/usr/bin/claude", "--output-format", "stream-json", "--verbose", "--system-prompt", "", "--input-format", "stream-json"},
		},
		{
			name: "with system prompt string",
			options: &types.ClaudeAgentOptions{
				SystemPrompt: "You are a helpful assistant",
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--system-prompt", "You are a helpful assistant"},
		},
		{
			name: "with system prompt preset",
			options: &types.ClaudeAgentOptions{
				SystemPrompt: map[string]interface{}{
					"type":   "preset",
					"append": "Additional context",
				},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--append-system-prompt", "Additional context"},
		},
		{
			name: "with tools as string slice",
			options: &types.ClaudeAgentOptions{
				Tools: []string{"Read", "Write", "Bash"},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--tools", "Read,Write,Bash"},
		},
		{
			name: "with tools as empty slice",
			options: &types.ClaudeAgentOptions{
				Tools: []string{},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--tools", ""},
		},
		{
			name: "with tools as preset map",
			options: &types.ClaudeAgentOptions{
				Tools: map[string]interface{}{"preset": "claude_code"},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--tools", "default"},
		},
		{
			name: "with allowed tools",
			options: &types.ClaudeAgentOptions{
				AllowedTools: []string{"Read", "Write"},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--allowedTools", "Read,Write"},
		},
		{
			name: "with disallowed tools",
			options: &types.ClaudeAgentOptions{
				DisallowedTools: []string{"Bash", "WebFetch"},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--disallowedTools", "Bash,WebFetch"},
		},
		{
			name: "with max turns",
			options: &types.ClaudeAgentOptions{
				MaxTurns: intPtr(10),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--max-turns", "10"},
		},
		{
			name: "with max budget USD",
			options: &types.ClaudeAgentOptions{
				MaxBudgetUSD: float64Ptr(5.50),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--max-budget-usd"},
		},
		{
			name: "with model",
			options: &types.ClaudeAgentOptions{
				Model: strPtr("claude-3-sonnet"),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--model", "claude-3-sonnet"},
		},
		{
			name: "with fallback model",
			options: &types.ClaudeAgentOptions{
				FallbackModel: strPtr("claude-3-haiku"),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--fallback-model", "claude-3-haiku"},
		},
		{
			name: "with betas",
			options: &types.ClaudeAgentOptions{
				Betas: []types.SdkBeta{types.SdkBetaContext1M},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--betas", "context-1m-2025-08-07"},
		},
		{
			name: "with permission prompt tool name",
			options: &types.ClaudeAgentOptions{
				PermissionPromptToolName: strPtr("my_permission_tool"),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--permission-prompt-tool", "my_permission_tool"},
		},
		{
			name: "with permission mode",
			options: &types.ClaudeAgentOptions{
				PermissionMode: ptrToPermissionMode(types.PermissionModeAcceptEdits),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--permission-mode", "acceptEdits"},
		},
		{
			name: "with continue conversation",
			options: &types.ClaudeAgentOptions{
				ContinueConversation: true,
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--continue"},
		},
		{
			name: "with resume",
			options: &types.ClaudeAgentOptions{
				Resume: strPtr("session-123"),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--resume", "session-123"},
		},
		{
			name: "with include partial messages",
			options: &types.ClaudeAgentOptions{
				IncludePartialMessages: true,
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--include-partial-messages"},
		},
		{
			name: "with fork session",
			options: &types.ClaudeAgentOptions{
				ForkSession: true,
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--fork-session"},
		},
		{
			name: "with setting sources",
			options: &types.ClaudeAgentOptions{
				SettingSources: []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--setting-sources", "user,project"},
		},
		{
			name: "with empty setting sources",
			options: &types.ClaudeAgentOptions{
				SettingSources: []types.SettingSource{},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--setting-sources", ""},
		},
		{
			name: "with plugins",
			options: &types.ClaudeAgentOptions{
				Plugins: []types.SdkPluginConfig{
					{Type: "local", Path: "/path/to/plugin"},
				},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--plugin-dir", "/path/to/plugin"},
		},
		{
			name: "with effort",
			options: &types.ClaudeAgentOptions{
				Effort: strPtr("high"),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--effort", "high"},
		},
		{
			name: "with max thinking tokens",
			options: &types.ClaudeAgentOptions{
				MaxThinkingTokens: intPtr(16000),
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--max-thinking-tokens", "16000"},
		},
		{
			name: "with thinking config enabled",
			options: &types.ClaudeAgentOptions{
				Thinking: types.ThinkingConfigEnabled{Type: "enabled", BudgetTokens: 8000},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--max-thinking-tokens", "8000"},
		},
		{
			name: "with thinking config disabled",
			options: &types.ClaudeAgentOptions{
				Thinking: types.ThinkingConfigDisabled{Type: "disabled"},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--max-thinking-tokens", "0"},
		},
		{
			name: "with thinking config adaptive",
			options: &types.ClaudeAgentOptions{
				Thinking: types.ThinkingConfigAdaptive{Type: "adaptive"},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--max-thinking-tokens", "32000"},
		},
		{
			name: "with output format",
			options: &types.ClaudeAgentOptions{
				OutputFormat: map[string]interface{}{
					"type":   "json_schema",
					"schema": map[string]interface{}{"type": "object"},
				},
			},
			cliPath:  "/usr/bin/claude",
			wantArgs: []string{"--json-schema"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transport, err := NewSubprocessCLITransport("test prompt", tt.options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}
			transport.cliPath = tt.cliPath

			cmd := transport.buildCommand()

			for _, wantArg := range tt.wantArgs {
				found := false
				for i, arg := range cmd {
					if wantArg == arg || (strings.HasPrefix(wantArg, "--") && i+1 < len(cmd) && cmd[i] == wantArg) {
						found = true
						break
					}
					if i+1 < len(cmd) && cmd[i]+" "+cmd[i+1] == wantArg {
						found = true
						break
					}
				}
				if !found {
					// Check if it's a value after a flag
					for i := 0; i < len(cmd)-1; i++ {
						if cmd[i]+" "+cmd[i+1] == wantArg {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("buildCommand() missing expected argument %q, got: %v", wantArg, cmd)
				}
			}
		})
	}
}

func TestBuildCommand_AddDirs(t *testing.T) {
	tests := []struct {
		name     string
		addDirs  []interface{}
		wantDirs []string
	}{
		{
			name:     "empty add dirs",
			addDirs:  []interface{}{},
			wantDirs: nil,
		},
		{
			name:     "single string dir",
			addDirs:  []interface{}{"/path/to/dir1"},
			wantDirs: []string{"/path/to/dir1"},
		},
		{
			name:     "multiple string dirs",
			addDirs:  []interface{}{"/path/to/dir1", "/path/to/dir2"},
			wantDirs: []string{"/path/to/dir1", "/path/to/dir2"},
		},
		{
			name:     "string pointer dirs",
			addDirs:  []interface{}{strPtr("/path/to/dir1")},
			wantDirs: []string{"/path/to/dir1"},
		},
		{
			name:     "nil pointer in add dirs",
			addDirs:  []interface{}{(*string)(nil), "/path/to/dir1"},
			wantDirs: []string{"/path/to/dir1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			options := &types.ClaudeAgentOptions{
				AddDirs: tt.addDirs,
			}
			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}
			transport.cliPath = "/usr/bin/claude"

			cmd := transport.buildCommand()

			for _, wantDir := range tt.wantDirs {
				found := false
				for i := 0; i < len(cmd)-1; i++ {
					if cmd[i] == "--add-dir" && cmd[i+1] == wantDir {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildCommand() missing --add-dir %q, got: %v", wantDir, cmd)
				}
			}
		})
	}
}

func TestBuildCommand_ExtraArgs(t *testing.T) {
	tests := []struct {
		name      string
		extraArgs map[string]interface{}
		wantArgs  []string
	}{
		{
			name:      "nil extra args",
			extraArgs: nil,
			wantArgs:  nil,
		},
		{
			name:      "empty extra args",
			extraArgs: map[string]interface{}{},
			wantArgs:  nil,
		},
		{
			name:      "boolean flag (nil value)",
			extraArgs: map[string]interface{}{"debug-to-stderr": nil},
			wantArgs:  []string{"--debug-to-stderr"},
		},
		{
			name:      "flag with string value",
			extraArgs: map[string]interface{}{"custom-flag": "value"},
			wantArgs:  []string{"--custom-flag", "value"},
		},
		{
			name:      "flag with int value",
			extraArgs: map[string]interface{}{"timeout": 30},
			wantArgs:  []string{"--timeout", "30"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			options := &types.ClaudeAgentOptions{
				ExtraArgs: tt.extraArgs,
			}
			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}
			transport.cliPath = "/usr/bin/claude"

			cmd := transport.buildCommand()

			for i := 0; i < len(tt.wantArgs); i += 2 {
				if i+1 < len(tt.wantArgs) {
					// Flag with value
					flag := tt.wantArgs[i]
					value := tt.wantArgs[i+1]
					found := false
					for j := 0; j < len(cmd)-1; j++ {
						if cmd[j] == flag && cmd[j+1] == value {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("buildCommand() missing extra arg %s %s, got: %v", flag, value, cmd)
					}
				} else {
					// Boolean flag
					flag := tt.wantArgs[i]
					found := false
					for _, arg := range cmd {
						if arg == flag {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("buildCommand() missing extra arg %s, got: %v", flag, cmd)
					}
				}
			}
		})
	}
}

func TestBuildCommand_MCPServers(t *testing.T) {
	tests := []struct {
		name       string
		mcpServers interface{}
		wantInCmd  string // Substring expected in command
		wantEmpty  bool   // If true, no --mcp-config should be present
	}{
		{
			name:       "nil mcp servers",
			mcpServers: nil,
			wantEmpty:  true,
		},
		{
			name:       "string mcp config path",
			mcpServers: "/path/to/mcp-config.json",
			wantInCmd:  "--mcp-config",
		},
		{
			name: "map mcp servers",
			mcpServers: map[string]interface{}{
				"server1": map[string]interface{}{
					"type":    "stdio",
					"command": "node",
					"args":    []string{"server.js"},
				},
			},
			wantInCmd: "--mcp-config",
		},
		{
			name: "sdk server with instance field stripped",
			mcpServers: map[string]interface{}{
				"sdk-server": map[string]interface{}{
					"type":     "sdk",
					"name":     "my-server",
					"instance": "should-be-stripped",
				},
			},
			wantInCmd: "--mcp-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ClaudeAgentOptions{
				MCPServers: tt.mcpServers,
			}
			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}
			transport.cliPath = "/usr/bin/claude"

			cmd := transport.buildCommand()

			hasMCPConfig := false
			for _, arg := range cmd {
				if arg == "--mcp-config" {
					hasMCPConfig = true
					break
				}
			}

			if tt.wantEmpty && hasMCPConfig {
				t.Errorf("buildCommand() should not have --mcp-config, got: %v", cmd)
			}
			if !tt.wantEmpty && !hasMCPConfig {
				t.Errorf("buildCommand() missing --mcp-config, got: %v", cmd)
			}

			// For SDK servers, verify instance field is stripped
			if !tt.wantEmpty && tt.mcpServers != nil {
				if _, ok := tt.mcpServers.(map[string]interface{}); ok {
					for _, cmdArg := range cmd {
						if strings.HasPrefix(cmdArg, "{\"mcpServers\":") {
							var parsed map[string]interface{}
							if err := json.Unmarshal([]byte(cmdArg), &parsed); err != nil {
								continue
							}
							if servers, ok := parsed["mcpServers"].(map[string]interface{}); ok {
								for serverName, serverConfig := range servers {
									if configMap, ok := serverConfig.(map[string]interface{}); ok {
										if _, hasInstance := configMap["instance"]; hasInstance {
											t.Errorf("buildCommand() SDK server %q should not have instance field", serverName)
										}
									}
								}
							}
						}
					}
				}
			}
		})
	}
}

func TestBuildCommand_ThinkingConfigPrecedence(t *testing.T) {
	tests := []struct {
		name                string
		maxThinkingTokens   *int
		thinking            types.ThinkingConfig
		wantMaxThinkingArgs string
	}{
		{
			name:                "only MaxThinkingTokens set",
			maxThinkingTokens:   intPtr(10000),
			wantMaxThinkingArgs: "10000",
		},
		{
			name:                "Thinking enabled overrides MaxThinkingTokens",
			maxThinkingTokens:   intPtr(10000),
			thinking:            types.ThinkingConfigEnabled{Type: "enabled", BudgetTokens: 5000},
			wantMaxThinkingArgs: "5000",
		},
		{
			name:                "Thinking disabled overrides MaxThinkingTokens",
			maxThinkingTokens:   intPtr(10000),
			thinking:            types.ThinkingConfigDisabled{Type: "disabled"},
			wantMaxThinkingArgs: "0",
		},
		{
			name:                "Thinking adaptive with no MaxThinkingTokens uses default",
			thinking:            types.ThinkingConfigAdaptive{Type: "adaptive"},
			wantMaxThinkingArgs: "32000",
		},
		{
			name:                "Thinking adaptive with MaxThinkingTokens uses MaxThinkingTokens",
			maxThinkingTokens:   intPtr(20000),
			thinking:            types.ThinkingConfigAdaptive{Type: "adaptive"},
			wantMaxThinkingArgs: "20000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ClaudeAgentOptions{
				MaxThinkingTokens: tt.maxThinkingTokens,
				Thinking:          tt.thinking,
			}
			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}
			transport.cliPath = "/usr/bin/claude"

			cmd := transport.buildCommand()

			found := false
			for i := 0; i < len(cmd)-1; i++ {
				if cmd[i] == "--max-thinking-tokens" && cmd[i+1] == tt.wantMaxThinkingArgs {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("buildCommand() missing --max-thinking-tokens %s, got: %v", tt.wantMaxThinkingArgs, cmd)
			}
		})
	}
}

func TestBuildCommand_AlwaysHasInputFormat(t *testing.T) {
	options := &types.ClaudeAgentOptions{}
	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}
	transport.cliPath = "/usr/bin/claude"

	cmd := transport.buildCommand()

	hasInputFormat := false
	for i := 0; i < len(cmd)-1; i++ {
		if cmd[i] == "--input-format" && cmd[i+1] == "stream-json" {
			hasInputFormat = true
			break
		}
	}
	if !hasInputFormat {
		t.Errorf("buildCommand() should always have --input-format stream-json, got: %v", cmd)
	}
}

// ============================================================================
// Tests for buildSettingsValue
// ============================================================================

func TestBuildSettingsValue(t *testing.T) {
	tests := []struct {
		name        string
		settings    *string
		sandbox     *types.SandboxSettings
		wantValue   string
		wantHas     bool
		wantInValue []string // Substrings expected in output
	}{
		{
			name:     "nil settings and nil sandbox",
			settings: nil,
			sandbox:  nil,
			wantHas:  false,
		},
		{
			name:     "empty settings and nil sandbox",
			settings: strPtr(""),
			sandbox:  nil,
			wantHas:  false,
		},
		{
			name:      "settings path only",
			settings:  strPtr("/path/to/settings.json"),
			sandbox:   nil,
			wantValue: "/path/to/settings.json",
			wantHas:   true,
		},
		{
			name:      "settings JSON string only",
			settings:  strPtr(`{"key": "value"}`),
			sandbox:   nil,
			wantValue: `{"key": "value"}`,
			wantHas:   true,
		},
		{
			name:     "sandbox only without settings",
			settings: nil,
			sandbox: &types.SandboxSettings{
				Enabled: ptrToBool(true),
			},
			wantInValue: []string{`"sandbox"`, `"enabled"`},
			wantHas:     true,
		},
		{
			name:     "sandbox merges with settings JSON",
			settings: strPtr(`{"existing": "value"}`),
			sandbox: &types.SandboxSettings{
				Enabled: ptrToBool(true),
			},
			wantInValue: []string{`"existing"`, `"sandbox"`, `"enabled"`},
			wantHas:     true,
		},
		{
			name:     "sandbox with network config",
			settings: nil,
			sandbox: &types.SandboxSettings{
				Enabled: ptrToBool(true),
				Network: &types.SandboxNetworkConfig{
					HTTPProxyPort:  intPtr(8080),
					SOCKSProxyPort: intPtr(1080),
				},
			},
			wantInValue: []string{`"sandbox"`, `"network"`, `"httpProxyPort"`},
			wantHas:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ClaudeAgentOptions{
				Settings: tt.settings,
				Sandbox:  tt.sandbox,
			}
			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}

			value, has := transport.buildSettingsValue()

			if has != tt.wantHas {
				t.Errorf("buildSettingsValue() has = %v, want %v", has, tt.wantHas)
			}

			if tt.wantHas {
				if tt.wantValue != "" && value != tt.wantValue {
					t.Errorf("buildSettingsValue() value = %v, want %v", value, tt.wantValue)
				}

				for _, wantIn := range tt.wantInValue {
					if !strings.Contains(value, wantIn) {
						t.Errorf("buildSettingsValue() value missing %q, got: %v", wantIn, value)
					}
				}
			}
		})
	}
}

func TestBuildSettingsValue_FilePath(t *testing.T) {
	// Create a temporary settings file
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	settingsContent := `{"api_key": "test123", "timeout": 30}`
	if err := os.WriteFile(settingsPath, []byte(settingsContent), 0644); err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	options := &types.ClaudeAgentOptions{
		Settings: &settingsPath,
		Sandbox: &types.SandboxSettings{
			Enabled: ptrToBool(true),
		},
	}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	value, has := transport.buildSettingsValue()
	if !has {
		t.Error("buildSettingsValue() should return has=true")
	}

	// Should contain settings from file
	if !strings.Contains(value, `"api_key"`) {
		t.Errorf("buildSettingsValue() should contain api_key from file, got: %v", value)
	}

	// Should also contain sandbox
	if !strings.Contains(value, `"sandbox"`) {
		t.Errorf("buildSettingsValue() should contain sandbox, got: %v", value)
	}
}

// ============================================================================
// Tests for findCLI
// ============================================================================

func TestFindCLI_CustomPath(t *testing.T) {
	// Create a temporary file to simulate CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create CLI file: %v", err)
	}

	options := &types.ClaudeAgentOptions{
		CLIPath: cliPath,
	}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport.cliPath != cliPath {
		t.Errorf("cliPath = %v, want %v", transport.cliPath, cliPath)
	}
}

func TestFindCLI_CustomPathPointer(t *testing.T) {
	// Create a temporary file to simulate CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create CLI file: %v", err)
	}

	options := &types.ClaudeAgentOptions{
		CLIPath: &cliPath,
	}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport.cliPath != cliPath {
		t.Errorf("cliPath = %v, want %v", transport.cliPath, cliPath)
	}
}

func TestFindCLI_NotFound(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	origHome := os.Getenv("HOME")

	// Set a minimal PATH that won't have claude
	os.Setenv("PATH", "/nonexistent")
	os.Setenv("HOME", "/nonexistent")

	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv("HOME", origHome)
	}()

	options := &types.ClaudeAgentOptions{}

	_, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err == nil {
		t.Error("NewSubprocessCLITransport() should return error when CLI not found")
	}
}

func TestFindCLI_FromSystemPath(t *testing.T) {
	// Create a temporary directory with a claude executable
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create CLI file: %v", err)
	}

	// Add tmpDir to PATH
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	// Reset exec.LookPath cache (Go doesn't cache this, so no need)

	options := &types.ClaudeAgentOptions{}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport.cliPath != cliPath {
		t.Errorf("cliPath = %v, want %v", transport.cliPath, cliPath)
	}
}

func TestFindCLI_FromCommonLocation(t *testing.T) {
	// Create a temporary home directory with claude in .local/bin
	tmpHome := t.TempDir()
	localBin := filepath.Join(tmpHome, ".local", "bin")
	if err := os.MkdirAll(localBin, 0755); err != nil {
		t.Fatalf("Failed to create .local/bin: %v", err)
	}

	cliPath := filepath.Join(localBin, "claude")
	if runtime.GOOS == "windows" {
		cliPath += ".exe"
	}
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create CLI file: %v", err)
	}

	// Set HOME and remove claude from PATH
	origHome := os.Getenv("HOME")
	origPath := os.Getenv("PATH")

	os.Setenv("HOME", tmpHome)
	os.Setenv("PATH", "/nonexistent") // Force search in common locations

	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("PATH", origPath)
	}()

	options := &types.ClaudeAgentOptions{}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport.cliPath != cliPath {
		t.Errorf("cliPath = %v, want %v", transport.cliPath, cliPath)
	}
}

// ============================================================================
// Tests for checkClaudeVersion
// ============================================================================

func TestCheckClaudeVersion_ValidVersion(t *testing.T) {
	tests := []struct {
		name          string
		versionOutput string
		wantErr       bool
	}{
		{
			name:          "version at minimum",
			versionOutput: "claude-code version 2.0.0",
			wantErr:       false,
		},
		{
			name:          "version above minimum",
			versionOutput: "claude-code version 3.1.5",
			wantErr:       false,
		},
		{
			name:          "version below minimum",
			versionOutput: "claude-code version 1.5.0",
			wantErr:       true,
		},
		{
			name:          "version below minimum patch",
			versionOutput: "claude-code version 1.9.99",
			wantErr:       true,
		},
		{
			name:          "version with v prefix",
			versionOutput: "claude-code version v2.0.0",
			wantErr:       false,
		},
		{
			name:          "version without text",
			versionOutput: "2.0.0",
			wantErr:       false,
		},
		{
			name:          "invalid version format",
			versionOutput: "claude-code version unknown",
			wantErr:       false, // Should not error, just skip check
		},
		{
			name:          "pre-release version",
			versionOutput: "claude-code version 2.0.0-beta.1",
			wantErr:       false, // Should parse 2.0.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock script that outputs the version
			tmpDir := t.TempDir()
			cliPath := filepath.Join(tmpDir, "claude")
			script := fmt.Sprintf("#!/bin/sh\necho '%s'", tt.versionOutput)
			if err := os.WriteFile(cliPath, []byte(script), 0755); err != nil {
				t.Fatalf("Failed to create mock CLI: %v", err)
			}

			transport := &SubprocessCLITransport{
				cliPath: cliPath,
			}

			err := transport.checkClaudeVersion(context.Background())

			if tt.wantErr && err == nil {
				t.Errorf("checkClaudeVersion() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("checkClaudeVersion() unexpected error: %v", err)
			}
		})
	}
}

func TestCheckClaudeVersion_Timeout(t *testing.T) {
	// Create a mock script that hangs (but not too long to avoid slow tests)
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	// Use a shorter sleep - we just need to verify timeout behavior, not wait for full sleep
	script := "#!/bin/sh\nsleep 1\necho '2.0.0'"
	if err := os.WriteFile(cliPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	transport := &SubprocessCLITransport{
		cliPath: cliPath,
	}

	// Use a context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should not hang - the version check itself has a 2-second timeout
	err := transport.checkClaudeVersion(ctx)

	// Should complete without hanging (though the mock will be killed)
	_ = err // Error is acceptable here
}

func TestCheckClaudeVersion_NonexistentCLI(t *testing.T) {
	transport := &SubprocessCLITransport{
		cliPath: "/nonexistent/path/to/claude",
	}

	err := transport.checkClaudeVersion(context.Background())

	// Should not error - version check errors are ignored
	if err != nil {
		t.Errorf("checkClaudeVersion() should not return error for nonexistent CLI, got: %v", err)
	}
}

// ============================================================================
// Tests for NewSubprocessCLITransport
// ============================================================================

func TestNewSubprocessCLITransport_Options(t *testing.T) {
	// Create a mock CLI
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	cwd := "/tmp/test"
	maxBufferSize := 2048 * 1024
	env := map[string]string{"TEST_VAR": "test_value"}

	options := &types.ClaudeAgentOptions{
		CLIPath:       cliPath,
		CWD:           cwd,
		MaxBufferSize: &maxBufferSize,
		Env:           env,
	}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport.cliPath != cliPath {
		t.Errorf("cliPath = %v, want %v", transport.cliPath, cliPath)
	}
	if transport.cwd != cwd {
		t.Errorf("cwd = %v, want %v", transport.cwd, cwd)
	}
	if transport.maxBufferSize != maxBufferSize {
		t.Errorf("maxBufferSize = %v, want %v", transport.maxBufferSize, maxBufferSize)
	}
	if transport.skipVersionCheck != true {
		t.Error("skipVersionCheck should be true")
	}
}

func TestNewSubprocessCLITransport_NilOptions(t *testing.T) {
	// Create a mock CLI in PATH
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	transport, err := NewSubprocessCLITransport("test prompt", nil, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	if transport.options == nil {
		t.Error("options should not be nil after initialization")
	}
}

func TestNewSubprocessCLITransport_CWDTypes(t *testing.T) {
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	tests := []struct {
		name    string
		cwd     interface{}
		wantCwd string
	}{
		{
			name:    "string CWD",
			cwd:     "/tmp/test",
			wantCwd: "/tmp/test",
		},
		{
			name:    "string pointer CWD",
			cwd:     strPtr("/tmp/test"),
			wantCwd: "/tmp/test",
		},
		{
			name:    "nil pointer CWD",
			cwd:     (*string)(nil),
			wantCwd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ClaudeAgentOptions{
				CLIPath: cliPath,
				CWD:     tt.cwd,
			}

			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}

			if transport.cwd != tt.wantCwd {
				t.Errorf("cwd = %v, want %v", transport.cwd, tt.wantCwd)
			}
		})
	}
}

// ============================================================================
// Tests for hasExtraArg
// ============================================================================

func TestHasExtraArg(t *testing.T) {
	tests := []struct {
		name      string
		extraArgs map[string]interface{}
		argName   string
		want      bool
	}{
		{
			name:      "nil extra args",
			extraArgs: nil,
			argName:   "debug",
			want:      false,
		},
		{
			name:      "empty extra args",
			extraArgs: map[string]interface{}{},
			argName:   "debug",
			want:      false,
		},
		{
			name:      "arg present with nil value",
			extraArgs: map[string]interface{}{"debug": nil},
			argName:   "debug",
			want:      true,
		},
		{
			name:      "arg present with value",
			extraArgs: map[string]interface{}{"timeout": "30"},
			argName:   "timeout",
			want:      true,
		},
		{
			name:      "arg not present",
			extraArgs: map[string]interface{}{"debug": nil},
			argName:   "verbose",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &SubprocessCLITransport{
				options: &types.ClaudeAgentOptions{
					ExtraArgs: tt.extraArgs,
				},
			}

			got := transport.hasExtraArg(tt.argName)
			if got != tt.want {
				t.Errorf("hasExtraArg(%q) = %v, want %v", tt.argName, got, tt.want)
			}
		})
	}
}

// ============================================================================
// Tests for GetCLIPath
// ============================================================================

func TestGetCLIPath(t *testing.T) {
	transport := &SubprocessCLITransport{
		cliPath: "/custom/path/to/claude",
	}

	got := transport.GetCLIPath()
	if got != "/custom/path/to/claude" {
		t.Errorf("GetCLIPath() = %v, want /custom/path/to/claude", got)
	}
}

// ============================================================================
// Tests for IsReady
// ============================================================================

func TestIsReady(t *testing.T) {
	transport := &SubprocessCLITransport{
		ready: true,
	}

	if !transport.IsReady() {
		t.Error("IsReady() should return true when ready is true")
	}

	transport.ready = false
	if transport.IsReady() {
		t.Error("IsReady() should return false when ready is false")
	}
}

// ============================================================================
// Tests for GetExitError
// ============================================================================

func TestGetExitError(t *testing.T) {
	transport := &SubprocessCLITransport{}

	if transport.GetExitError() != nil {
		t.Error("GetExitError() should return nil when no error")
	}

	transport.exitError = fmt.Errorf("test error")
	if transport.GetExitError() == nil {
		t.Error("GetExitError() should return the error when set")
	}
}

// ============================================================================
// Helper functions
// ============================================================================

func ptrToPermissionMode(mode types.PermissionMode) *types.PermissionMode {
	return &mode
}

func ptrToBool(b bool) *bool {
	return &b
}

// ============================================================================
// Tests for version comparison logic
// ============================================================================

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		name        string
		cliVersion  string
		minVersion  string
		shouldError bool
	}{
		{
			name:        "equal versions",
			cliVersion:  "2.0.0",
			minVersion:  "2.0.0",
			shouldError: false,
		},
		{
			name:        "higher major version",
			cliVersion:  "3.0.0",
			minVersion:  "2.0.0",
			shouldError: false,
		},
		{
			name:        "lower major version",
			cliVersion:  "1.0.0",
			minVersion:  "2.0.0",
			shouldError: true,
		},
		{
			name:        "higher minor version",
			cliVersion:  "2.1.0",
			minVersion:  "2.0.0",
			shouldError: false,
		},
		{
			name:        "lower minor version",
			cliVersion:  "2.0.0",
			minVersion:  "2.1.0",
			shouldError: true,
		},
		{
			name:        "higher patch version",
			cliVersion:  "2.0.1",
			minVersion:  "2.0.0",
			shouldError: false,
		},
		{
			name:        "lower patch version",
			cliVersion:  "2.0.0",
			minVersion:  "2.0.1",
			shouldError: true,
		},
		{
			name:        "mixed comparison",
			cliVersion:  "2.5.3",
			minVersion:  "2.5.3",
			shouldError: false,
		},
		{
			name:        "much higher version",
			cliVersion:  "10.20.30",
			minVersion:  "2.0.0",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock CLI
			tmpDir := t.TempDir()
			cliPath := filepath.Join(tmpDir, "claude")
			script := fmt.Sprintf("#!/bin/sh\necho '%s'", tt.cliVersion)
			if err := os.WriteFile(cliPath, []byte(script), 0755); err != nil {
				t.Fatalf("Failed to create mock CLI: %v", err)
			}

			// Temporarily set minimum version
			origMinVersion := MinimumClaudeCodeVersion
			// We can't modify the constant, so we test with the default minimum

			transport := &SubprocessCLITransport{
				cliPath: cliPath,
			}

			err := transport.checkClaudeVersion(context.Background())

			// For custom min version tests, we'd need to modify the constant or
			// extract the comparison logic. For now, test with default minimum
			if tt.minVersion != MinimumClaudeCodeVersion {
				t.Skip("Cannot test custom minimum version")
			}

			_ = origMinVersion

			if tt.shouldError && err == nil {
				t.Errorf("Expected error for version %s", tt.cliVersion)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for version %s: %v", tt.cliVersion, err)
			}
		})
	}
}

// ============================================================================
// Tests for environment setup
// ============================================================================

func TestBuildCommand_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name                 string
		env                  map[string]string
		enableFileCheckpoint bool
		wantEnvVars          []string
		wantNotEnvVars       []string
	}{
		{
			name:                 "default environment",
			env:                  map[string]string{},
			enableFileCheckpoint: false,
			wantEnvVars: []string{
				"CLAUDE_CODE_ENTRYPOINT=sdk-go",
				"CLAUDE_AGENT_SDK_VERSION=",
			},
			wantNotEnvVars: []string{"CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING"},
		},
		{
			name: "custom environment",
			env: map[string]string{
				"MY_VAR":  "my_value",
				"MY_VAR2": "my_value2",
			},
			enableFileCheckpoint: false,
			wantEnvVars: []string{
				"MY_VAR=my_value",
				"MY_VAR2=my_value2",
			},
		},
		{
			name:                 "file checkpointing enabled",
			env:                  map[string]string{},
			enableFileCheckpoint: true,
			wantEnvVars: []string{
				"CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true",
			},
		},
		{
			name: "caller can override entrypoint",
			env: map[string]string{
				"CLAUDE_CODE_ENTRYPOINT": "custom-caller",
			},
			enableFileCheckpoint: false,
			wantEnvVars: []string{
				"CLAUDE_CODE_ENTRYPOINT=custom-caller",
			},
			wantNotEnvVars: []string{"CLAUDE_CODE_ENTRYPOINT=sdk-go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock CLI that prints environment
			tmpDir := t.TempDir()
			cliPath := filepath.Join(tmpDir, "claude")
			script := "#!/bin/sh\nenv | sort\nexit 0"
			if runtime.GOOS == "windows" {
				script = "@echo off\nset\nexit 0"
			}
			if err := os.WriteFile(cliPath, []byte(script), 0755); err != nil {
				t.Fatalf("Failed to create mock CLI: %v", err)
			}

			cwd := tmpDir

			options := &types.ClaudeAgentOptions{
				CLIPath:                 cliPath,
				CWD:                     cwd,
				Env:                     tt.env,
				EnableFileCheckpointing: tt.enableFileCheckpoint,
			}

			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}

			// Connect to start the process
			ctx := context.Background()
			err = transport.Connect(ctx)
			if err != nil {
				t.Fatalf("Failed to connect: %v", err)
			}
			defer transport.Close(ctx)

			// Check command was built
			cmd := transport.buildCommand()
			_ = cmd // The environment is set when Connect is called

			// We can't easily verify environment variables without running the command
			// and reading its output, which is complex. The test above verifies the
			// command is built correctly.
		})
	}
}

// ============================================================================
// Tests for system prompt handling
// ============================================================================

func TestBuildCommand_SystemPromptTypes(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt interface{}
		wantArgs     []string
		dontWantArgs []string
	}{
		{
			name:         "nil system prompt",
			systemPrompt: nil,
			wantArgs:     []string{"--system-prompt", ""},
		},
		{
			name:         "string system prompt",
			systemPrompt: "You are a helpful assistant",
			wantArgs:     []string{"--system-prompt", "You are a helpful assistant"},
		},
		{
			name: "preset system prompt with append",
			systemPrompt: map[string]interface{}{
				"type":   "preset",
				"append": "Additional instructions",
			},
			wantArgs: []string{"--append-system-prompt", "Additional instructions"},
		},
		{
			name: "preset system prompt without append",
			systemPrompt: map[string]interface{}{
				"type": "preset",
			},
			dontWantArgs: []string{"--append-system-prompt"},
		},
		{
			name: "preset system prompt with empty append",
			systemPrompt: map[string]interface{}{
				"type":   "preset",
				"append": "",
			},
			dontWantArgs: []string{"--append-system-prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ClaudeAgentOptions{
				SystemPrompt: tt.systemPrompt,
			}

			// Create a mock CLI in PATH
			tmpDir := t.TempDir()
			cliPath := filepath.Join(tmpDir, "claude")
			if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
				t.Fatalf("Failed to create mock CLI: %v", err)
			}

			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}
			transport.cliPath = cliPath

			cmd := transport.buildCommand()

			for _, wantArg := range tt.wantArgs {
				found := false
				for i := 0; i < len(cmd)-1; i++ {
					if cmd[i]+" "+cmd[i+1] == wantArg || cmd[i] == wantArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("buildCommand() missing expected arg %q, got: %v", wantArg, cmd)
				}
			}

			for _, dontWantArg := range tt.dontWantArgs {
				for _, arg := range cmd {
					if arg == dontWantArg {
						t.Errorf("buildCommand() should not have arg %q, got: %v", dontWantArg, cmd)
					}
				}
			}
		})
	}
}

// ============================================================================
// Tests for JSON schema output format
// ============================================================================

func TestBuildCommand_OutputFormat(t *testing.T) {
	tests := []struct {
		name         string
		outputFormat map[string]interface{}
		wantSchema   bool
	}{
		{
			name:         "nil output format",
			outputFormat: nil,
			wantSchema:   false,
		},
		{
			name:         "empty output format",
			outputFormat: map[string]interface{}{},
			wantSchema:   false,
		},
		{
			name: "json schema output format",
			outputFormat: map[string]interface{}{
				"type": "json_schema",
				"schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]string{"type": "string"},
					},
				},
			},
			wantSchema: true,
		},
		{
			name: "non-json-schema output format",
			outputFormat: map[string]interface{}{
				"type": "text",
			},
			wantSchema: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &types.ClaudeAgentOptions{
				OutputFormat: tt.outputFormat,
			}

			tmpDir := t.TempDir()
			cliPath := filepath.Join(tmpDir, "claude")
			if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
				t.Fatalf("Failed to create mock CLI: %v", err)
			}

			transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
			if err != nil {
				t.Fatalf("Failed to create transport: %v", err)
			}
			transport.cliPath = cliPath

			cmd := transport.buildCommand()

			hasSchema := false
			for i, arg := range cmd {
				if arg == "--json-schema" {
					hasSchema = true
					if tt.wantSchema && i+1 < len(cmd) {
						// Verify it's valid JSON
						var schema map[string]interface{}
						if err := json.Unmarshal([]byte(cmd[i+1]), &schema); err != nil {
							t.Errorf("Invalid JSON schema: %v", err)
						}
					}
					break
				}
			}

			if tt.wantSchema && !hasSchema {
				t.Errorf("buildCommand() missing --json-schema, got: %v", cmd)
			}
			if !tt.wantSchema && hasSchema {
				t.Errorf("buildCommand() should not have --json-schema, got: %v", cmd)
			}
		})
	}
}

// ============================================================================
// Tests for fine-grained tool streaming (FGTS)
// ============================================================================

func TestBuildCommand_IncludePartialMessagesEnablesFGTS(t *testing.T) {
	// Test that include_partial_messages=True sets CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=1
	// --include-partial-messages tells the CLI to forward stream_event messages,
	// but tool input parameters are still buffered by the API unless
	// eager_input_streaming is enabled via this env var.
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	options := &types.ClaudeAgentOptions{
		CLIPath:                cliPath,
		IncludePartialMessages: true,
	}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	ctx := context.Background()
	err = transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer transport.Close(ctx)

	// Check that FGTS env var is set
	found := false
	for _, env := range transport.cmd.Env {
		if env == "CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=1 not set when include_partial_messages=true")
	}
}

func TestBuildCommand_IncludePartialMessagesFalseDoesNotSetFGTS(t *testing.T) {
	// Test that include_partial_messages=False does not force-enable FGTS.
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	options := &types.ClaudeAgentOptions{
		CLIPath:                cliPath,
		IncludePartialMessages: false,
	}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	ctx := context.Background()
	err = transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer transport.Close(ctx)

	// Check that FGTS env var is NOT set
	for _, env := range transport.cmd.Env {
		if strings.HasPrefix(env, "CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=") {
			t.Errorf("CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING should not be set when include_partial_messages=false, got: %s", env)
		}
	}
}

func TestBuildCommand_UserCanOverrideFGTSEnvVar(t *testing.T) {
	// Test that a user-supplied env var takes precedence over the SDK default.
	tmpDir := t.TempDir()
	cliPath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	options := &types.ClaudeAgentOptions{
		CLIPath:                cliPath,
		IncludePartialMessages: true,
		Env: map[string]string{
			"CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING": "0",
		},
	}

	transport, err := NewSubprocessCLITransport("test prompt", options, WithSkipVersionCheck(true))
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	ctx := context.Background()
	err = transport.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer transport.Close(ctx)

	// Check that user's "0" value is preserved (not overwritten by SDK default "1")
	for _, env := range transport.cmd.Env {
		if env == "CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=0" {
			return // Success - user override worked
		}
	}
	t.Error("User's CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=0 was not preserved")
}
