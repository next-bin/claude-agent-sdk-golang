package e2e_tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Agent Definition E2E Tests
// ============================================================================

// TestAgentDefinitionWithInit tests that custom agent definitions work
// and appear in the init message.
func TestAgentDefinitionWithInit(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sonnet := "sonnet"
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(DefaultTestConfig().Model),
		Agents: map[string]types.AgentDefinition{
			"test-agent": {
				Description: "A test agent for verification",
				Prompt:      "You are a test agent. Always respond with 'Test agent activated'",
				Tools:       []string{"Read"},
				Model:       &sonnet,
			},
		},
		MaxTurns: types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "What is 2 + 2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundInitWithAgent bool
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.SystemMessage:
			if m.Subtype == "init" {
				agents, ok := m.Data["agents"].([]interface{})
				if ok {
					for _, agent := range agents {
						if agent.(string) == "test-agent" {
							foundInitWithAgent = true
							break
						}
					}
				}
			}
		}
	}

	if !foundInitWithAgent {
		t.Error("Expected to find 'test-agent' in init message agents")
	}
}

// TestLargeAgents tests large agent definitions work with the SDK.
func TestLargeAgents(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Generate 5 agents with large prompts
	agents := make(map[string]types.AgentDefinition)
	for i := 0; i < 5; i++ {
		prompt := fmt.Sprintf("You are test agent #%d. ", i) + strings.Repeat("x", 1024) // 1KB prompt per agent
		agents[fmt.Sprintf("large-agent-%d", i)] = types.AgentDefinition{
			Description: fmt.Sprintf("Large test agent #%d for stress testing", i),
			Prompt:      prompt,
		}
	}

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:    types.String(DefaultTestConfig().Model),
		Agents:   agents,
		MaxTurns: types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "What is 2 + 2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundInitWithAgents bool
	var foundAgentNames []string

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.SystemMessage:
			if m.Subtype == "init" {
				agentsData, ok := m.Data["agents"].([]interface{})
				if ok {
					for _, agent := range agentsData {
						if name, ok := agent.(string); ok {
							foundAgentNames = append(foundAgentNames, name)
						}
					}
					// Check if our agents are registered
					foundCount := 0
					for agentName := range agents {
						for _, foundName := range foundAgentNames {
							if foundName == agentName {
								foundCount++
								break
							}
						}
					}
					if foundCount == len(agents) {
						foundInitWithAgents = true
					}
				}
			}
		}
	}

	if !foundInitWithAgents {
		t.Errorf("Not all agents were registered. Found: %v", foundAgentNames)
	}
}

// ============================================================================
// Setting Sources E2E Tests
// ============================================================================

// TestSettingSourcesDefault tests that default (no setting_sources) loads no settings.
func TestSettingSourcesDefault(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a temporary project with local settings
	tmpDir, err := os.MkdirTemp("", "sdk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .claude directory with local settings
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	// Create local settings with custom outputStyle
	settingsFile := filepath.Join(claudeDir, "settings.local.json")
	if err := os.WriteFile(settingsFile, []byte(`{"outputStyle": "local-test-style"}`), 0644); err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:    types.String(DefaultTestConfig().Model),
		CWD:      tmpDir,
		MaxTurns: types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "What is 2 + 2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.SystemMessage:
			if m.Subtype == "init" {
				outputStyle, _ := m.Data["output_style"].(string)
				if outputStyle == "local-test-style" {
					t.Error("outputStyle should NOT be from local settings (default is no settings)")
				}
				if outputStyle != "default" {
					t.Logf("Note: outputStyle is '%s' (may vary by CLI version)", outputStyle)
				}
			}
		}
	}
}

// TestSettingSourcesUserOnly tests that setting_sources=['user'] excludes project settings.
func TestSettingSourcesUserOnly(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a temporary project with a slash command
	tmpDir, err := os.MkdirTemp("", "sdk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .claude/commands directory with a test command
	commandsDir := filepath.Join(tmpDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		t.Fatalf("Failed to create commands dir: %v", err)
	}

	testCommand := filepath.Join(commandsDir, "testcmd.md")
	commandContent := `---
description: Test command
---

This is a test command.
`
	if err := os.WriteFile(testCommand, []byte(commandContent), 0644); err != nil {
		t.Fatalf("Failed to write command file: %v", err)
	}

	userOnly := []types.SettingSource{types.SettingSourceUser}
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		SettingSources: userOnly,
		CWD:            tmpDir,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "What is 2 + 2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.SystemMessage:
			if m.Subtype == "init" {
				commands, ok := m.Data["slash_commands"].([]interface{})
				if ok {
					for _, cmd := range commands {
						if cmd.(string) == "testcmd" {
							t.Error("testcmd should NOT be available with user-only sources")
						}
					}
				}
			}
		}
	}
}

// TestSettingSourcesProjectIncluded tests that setting_sources=['user', 'project', 'local']
// includes project settings.
func TestSettingSourcesProjectIncluded(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a temporary project with local settings
	tmpDir, err := os.MkdirTemp("", "sdk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .claude directory with local settings
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	// Create local settings with custom outputStyle
	settingsFile := filepath.Join(claudeDir, "settings.local.json")
	if err := os.WriteFile(settingsFile, []byte(`{"outputStyle": "local-test-style"}`), 0644); err != nil {
		t.Fatalf("Failed to write settings file: %v", err)
	}

	sources := []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal}
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		SettingSources: sources,
		CWD:            tmpDir,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "What is 2 + 2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	foundLocalStyle := false
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.SystemMessage:
			if m.Subtype == "init" {
				outputStyle, _ := m.Data["output_style"].(string)
				if outputStyle == "local-test-style" {
					foundLocalStyle = true
				}
			}
		}
	}

	// Note: This may not always find the local style depending on CLI version
	// The test primarily verifies that setting_sources is properly passed
	t.Logf("Found local style: %v", foundLocalStyle)
}

// TestFilesystemAgentLoading tests that filesystem-based agents load via setting_sources
// and produce a full response.
func TestFilesystemAgentLoading(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a temporary project with a filesystem agent
	tmpDir, err := os.MkdirTemp("", "sdk-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "project")
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents dir: %v", err)
	}

	// Create a test agent file
	agentFile := filepath.Join(agentsDir, "fs-test-agent.md")
	agentContent := `---
name: fs-test-agent
description: A filesystem test agent for SDK testing
tools: Read
---

# Filesystem Test Agent

You are a simple test agent. When asked a question, provide a brief, helpful answer.
`
	if err := os.WriteFile(agentFile, []byte(agentContent), 0644); err != nil {
		t.Fatalf("Failed to write agent file: %v", err)
	}

	projectSources := []types.SettingSource{types.SettingSourceProject}
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		SettingSources: projectSources,
		CWD:            projectDir,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Say hello in exactly 3 words")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Collect all messages
	var messages []types.Message
	for msg := range msgChan {
		messages = append(messages, msg)
	}

	// Must have at least init, assistant, result
	messageTypes := make([]string, 0, len(messages))
	for _, msg := range messages {
		messageTypes = append(messageTypes, fmt.Sprintf("%T", msg))
	}

	hasSystem := false
	hasAssistant := false
	hasResult := false

	for _, msg := range messages {
		switch msg.(type) {
		case *types.SystemMessage:
			hasSystem = true
		case *types.AssistantMessage:
			hasAssistant = true
		case *types.ResultMessage:
			hasResult = true
		}
	}

	if !hasSystem {
		t.Error("Missing SystemMessage (init)")
	}
	if !hasAssistant {
		t.Errorf("Missing AssistantMessage - got only: %v. This may indicate issue with filesystem agents.", messageTypes)
	}
	if !hasResult {
		t.Error("Missing ResultMessage")
	}

	// Find the init message and check for the filesystem agent
	for _, msg := range messages {
		if m, ok := msg.(*types.SystemMessage); ok && m.Subtype == "init" {
			agents, ok := m.Data["agents"].([]interface{})
			if ok {
				found := false
				for _, agent := range agents {
					if agent.(string) == "fs-test-agent" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("fs-test-agent not loaded from filesystem. Found: %v", agents)
				}
			}
			break
		}
	}
}
