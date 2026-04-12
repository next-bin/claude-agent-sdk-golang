package e2e_tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Agent Definition E2E Tests
// ============================================================================

// TestAgentDefinitionWithInit tests that custom agent definitions work
// and appear in the init message.
func TestAgentDefinitionWithInit(t *testing.T) {
	SkipIfNoAPIKey(t)
	t.Log("Starting TestAgentDefinitionWithInit...")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	cfg := DefaultTestConfig()
	t.Logf("Using model: %s", cfg.Model)

	sonnet := "sonnet"
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model: types.String(cfg.Model),
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

	t.Log("Connecting to Claude...")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(ctx)

	t.Log("Sending query: What is 2 + 2?")
	if err := client.Query(ctx, "What is 2 + 2?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundInitWithAgent bool
	var msgCount int
	for msg := range msgChan {
		msgCount++
		switch m := msg.(type) {
		case *types.SystemMessage:
			t.Logf("Received SystemMessage: subtype=%s", m.Subtype)
			if m.Subtype == "init" {
				agents, ok := m.Data["agents"].([]interface{})
				if ok {
					t.Logf("Found agents in init: %v", agents)
					for _, agent := range agents {
						if agent.(string) == "test-agent" {
							foundInitWithAgent = true
							t.Log("SUCCESS: Found 'test-agent' in init message")
							break
						}
					}
				}
			}
		case *types.AssistantMessage:
			t.Logf("Received AssistantMessage: %s", formatContent(m.Content))
		case *types.ResultMessage:
			t.Logf("Received ResultMessage: %s", formatResult(m.Result))
		}
	}

	t.Logf("Total messages received: %d", msgCount)

	if !foundInitWithAgent {
		t.Error("Expected to find 'test-agent' in init message agents")
	} else {
		t.Log("TEST PASSED: Agent found in init message")
	}
}

// TestLargeAgents tests large agent definitions work with the SDK.
// This tests ~260KB of agent definitions (20 agents x 13KB each).
func TestLargeAgents(t *testing.T) {
	SkipIfNoAPIKey(t)
	t.Log("Starting TestLargeAgents...")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel()

	// Generate 20 agents with 13KB prompts each = ~260KB total (matching upstream SDK)
	t.Log("Generating 20 large agents (~260KB total)...")
	agents := make(map[string]types.AgentDefinition)
	for i := 0; i < 20; i++ {
		prompt := fmt.Sprintf("You are test agent #%d. ", i) + strings.Repeat("x", 13*1024) // 13KB prompt per agent
		agents[fmt.Sprintf("large-agent-%d", i)] = types.AgentDefinition{
			Description: fmt.Sprintf("Large test agent #%d for stress testing", i),
			Prompt:      prompt,
		}
	}

	cfg := DefaultTestConfig()
	t.Logf("Using model: %s", cfg.Model)

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:    types.String(cfg.Model),
		Agents:   agents,
		MaxTurns: types.Int(1),
	})
	defer client.Close()

	t.Log("Connecting to Claude...")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	t.Log("Sending query: What is 2 + 2?")
	if err := client.Query(ctx, "What is 2 + 2?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundInitWithAgents bool
	var foundAgentNames []string
	var msgCount int

	for msg := range msgChan {
		msgCount++
		switch m := msg.(type) {
		case *types.SystemMessage:
			t.Logf("Received SystemMessage: subtype=%s", m.Subtype)
			if m.Subtype == "init" {
				agentsData, ok := m.Data["agents"].([]interface{})
				if ok {
					for _, agent := range agentsData {
						if name, ok := agent.(string); ok {
							foundAgentNames = append(foundAgentNames, name)
						}
					}
					t.Logf("Found %d agents in init message", len(foundAgentNames))
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
					t.Logf("Matched %d/%d large agents", foundCount, len(agents))
					if foundCount == len(agents) {
						foundInitWithAgents = true
					}
				}
			}
		case *types.AssistantMessage:
			t.Logf("Received AssistantMessage: %s", formatContent(m.Content))
		case *types.ResultMessage:
			t.Logf("Received ResultMessage: %s", formatResult(m.Result))
			// ResultMessage indicates the conversation is complete
			t.Logf("Total messages received: %d", msgCount)
			if !foundInitWithAgents {
				t.Errorf("Not all agents were registered. Found: %v", foundAgentNames)
			} else {
				t.Log("TEST PASSED: All large agents found in init message")
			}
			return
		}
	}

	t.Logf("Total messages received: %d", msgCount)

	if !foundInitWithAgents {
		t.Errorf("Not all agents were registered. Found: %v", foundAgentNames)
	} else {
		t.Log("TEST PASSED: All large agents found in init message")
	}
}

// TestAgentDefinitionWithQueryFunction tests that custom agent definitions
// work with the query package function.
func TestAgentDefinitionWithQueryFunction(t *testing.T) {
	SkipIfNoAPIKey(t)
	t.Log("Starting TestAgentDefinitionWithQueryFunction...")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	cfg := DefaultTestConfig()
	t.Logf("Using model: %s", cfg.Model)

	// Use query package
	options := &types.ClaudeAgentOptions{
		Model: types.String(cfg.Model),
		Agents: map[string]types.AgentDefinition{
			"test-agent-query": {
				Description: "A test agent for query function verification",
				Prompt:      "You are a test agent.",
			},
		},
		MaxTurns: types.Int(1),
	}

	client := claude.NewClientWithOptions(options)
	defer client.Close()

	t.Log("Connecting to Claude...")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	t.Log("Sending query: What is 2 + 2?")
	if err := client.Query(ctx, "What is 2 + 2?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	foundAgent := false
	var msgCount int
	for msg := range msgChan {
		msgCount++
		switch m := msg.(type) {
		case *types.SystemMessage:
			t.Logf("Received SystemMessage: subtype=%s", m.Subtype)
			if m.Subtype == "init" {
				agents, ok := m.Data["agents"].([]interface{})
				if ok {
					t.Logf("Found agents in init: %v", agents)
					for _, agent := range agents {
						if agent.(string) == "test-agent-query" {
							foundAgent = true
							t.Log("SUCCESS: Found 'test-agent-query' in init message")
							break
						}
					}
				}
			}
		case *types.AssistantMessage:
			t.Logf("Received AssistantMessage: %s", formatContent(m.Content))
		case *types.ResultMessage:
			t.Logf("Received ResultMessage: %s", formatResult(m.Result))
			// ResultMessage indicates the conversation is complete
			t.Logf("Total messages received: %d", msgCount)
			if !foundAgent {
				t.Error("Should have received init message with test-agent-query")
			} else {
				t.Log("TEST PASSED: Agent found in init message")
			}
			return
		}
	}

	t.Logf("Total messages received: %d", msgCount)

	if !foundAgent {
		t.Error("Should have received init message with test-agent-query")
	} else {
		t.Log("TEST PASSED: Agent found in init message")
	}
}

// ============================================================================
// Setting Sources E2E Tests
// ============================================================================

// TestSettingSourcesDefault tests that default (no setting_sources) loads no settings.
func TestSettingSourcesDefault(t *testing.T) {
	SkipIfNoAPIKey(t)
	t.Log("Starting TestSettingSourcesDefault...")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	// Create a temporary project with local settings
	t.Log("Creating temporary project with local settings...")
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
	t.Logf("Created temp dir: %s", tmpDir)

	cfg := DefaultTestConfig()
	t.Logf("Using model: %s", cfg.Model)

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:    types.String(cfg.Model),
		CWD:      tmpDir,
		MaxTurns: types.Int(1),
	})
	defer client.Close()

	t.Log("Connecting to Claude...")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	t.Log("Sending query: What is 2 + 2?")
	if err := client.Query(ctx, "What is 2 + 2?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var msgCount int
	for msg := range msgChan {
		msgCount++
		switch m := msg.(type) {
		case *types.SystemMessage:
			t.Logf("Received SystemMessage: subtype=%s", m.Subtype)
			if m.Subtype == "init" {
				outputStyle, _ := m.Data["output_style"].(string)
				t.Logf("outputStyle: %s", outputStyle)
				if outputStyle == "local-test-style" {
					t.Error("outputStyle should NOT be from local settings (default is no settings)")
				}
			}
		case *types.AssistantMessage:
			t.Logf("Received AssistantMessage: %s", formatContent(m.Content))
		case *types.ResultMessage:
			t.Logf("Received ResultMessage: %s", formatResult(m.Result))
			t.Logf("Total messages received: %d", msgCount)
			t.Log("TEST PASSED: Setting sources default test completed")
			return
		}
	}

	t.Logf("Total messages received: %d", msgCount)
	t.Log("TEST PASSED: Setting sources default test completed")
}

// TestSettingSourcesUserOnly tests that setting_sources=['user'] excludes project settings.
func TestSettingSourcesUserOnly(t *testing.T) {
	SkipIfNoAPIKey(t)
	t.Log("Starting TestSettingSourcesUserOnly...")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	// Create a temporary project with a slash command
	t.Log("Creating temporary project with slash command...")
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
	t.Logf("Created temp dir: %s", tmpDir)

	cfg := DefaultTestConfig()
	t.Logf("Using model: %s", cfg.Model)

	userOnly := []types.SettingSource{types.SettingSourceUser}
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(cfg.Model),
		SettingSources: userOnly,
		CWD:            tmpDir,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	t.Log("Connecting to Claude...")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	t.Log("Sending query: What is 2 + 2?")
	if err := client.Query(ctx, "What is 2 + 2?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var msgCount int
	for msg := range msgChan {
		msgCount++
		switch m := msg.(type) {
		case *types.SystemMessage:
			t.Logf("Received SystemMessage: subtype=%s", m.Subtype)
			if m.Subtype == "init" {
				commands, ok := m.Data["slash_commands"].([]interface{})
				if ok {
					t.Logf("Found slash commands: %v", commands)
					for _, cmd := range commands {
						if cmd.(string) == "testcmd" {
							t.Error("testcmd should NOT be available with user-only sources")
						}
					}
				}
			}
		case *types.AssistantMessage:
			t.Logf("Received AssistantMessage: %s", formatContent(m.Content))
		case *types.ResultMessage:
			t.Logf("Received ResultMessage: %s", formatResult(m.Result))
			t.Logf("Total messages received: %d", msgCount)
			t.Log("TEST PASSED: User-only setting sources test completed")
			return
		}
	}

	t.Logf("Total messages received: %d", msgCount)
	t.Log("TEST PASSED: User-only setting sources test completed")
}

// TestSettingSourcesProjectIncluded tests that setting_sources=['user', 'project', 'local']
// includes project settings.
func TestSettingSourcesProjectIncluded(t *testing.T) {
	SkipIfNoAPIKey(t)
	t.Log("Starting TestSettingSourcesProjectIncluded...")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	// Create a temporary project with local settings
	t.Log("Creating temporary project with local settings...")
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
	t.Logf("Created temp dir: %s", tmpDir)

	cfg := DefaultTestConfig()
	t.Logf("Using model: %s", cfg.Model)

	sources := []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject, types.SettingSourceLocal}
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(cfg.Model),
		SettingSources: sources,
		CWD:            tmpDir,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	t.Log("Connecting to Claude...")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	t.Log("Sending query: What is 2 + 2?")
	if err := client.Query(ctx, "What is 2 + 2?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	foundLocalStyle := false
	var msgCount int
	for {
		select {
		case <-ctx.Done():
			t.Logf("Context done, total messages received: %d", msgCount)
			t.Logf("Found local style: %v", foundLocalStyle)
			if foundLocalStyle {
				t.Log("TEST PASSED: Found local-test-style before timeout")
				return
			}
			t.Fatal("Test timed out before completion")
		case msg, ok := <-msgChan:
			if !ok {
				t.Logf("Message channel closed, total messages: %d", msgCount)
				return
			}
			msgCount++
			switch m := msg.(type) {
			case *types.SystemMessage:
				t.Logf("Received SystemMessage: subtype=%s", m.Subtype)
				if m.Subtype == "init" {
					outputStyle, _ := m.Data["output_style"].(string)
					t.Logf("outputStyle: %s", outputStyle)
					if outputStyle == "local-test-style" {
						foundLocalStyle = true
						t.Log("SUCCESS: Found local-test-style in outputStyle")
					}
				}
			case *types.AssistantMessage:
				t.Logf("Received AssistantMessage: %s", formatContent(m.Content))
			case *types.ResultMessage:
				t.Logf("Received ResultMessage: %s", formatResult(m.Result))
				t.Logf("Total messages received: %d", msgCount)
				t.Logf("Found local style: %v", foundLocalStyle)
				t.Log("TEST PASSED: Setting sources project included test completed")
				return
			default:
				t.Logf("Received unknown message type: %T", msg)
			}
		}
	}
}

// TestFilesystemAgentLoading tests that filesystem-based agents load via setting_sources
// and produce a full response.
func TestFilesystemAgentLoading(t *testing.T) {
	SkipIfNoAPIKey(t)
	t.Log("Starting TestFilesystemAgentLoading...")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	// Create a temporary project with a filesystem agent
	t.Log("Creating temporary project with filesystem agent...")
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
	t.Logf("Created project dir: %s", projectDir)

	cfg := DefaultTestConfig()
	t.Logf("Using model: %s", cfg.Model)

	projectSources := []types.SettingSource{types.SettingSourceProject}
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(cfg.Model),
		SettingSources: projectSources,
		CWD:            projectDir,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	t.Log("Connecting to Claude...")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Log("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	t.Log("Sending query: Say hello in exactly 3 words")
	if err := client.Query(ctx, "Say hello in exactly 3 words"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Collect all messages
	var messages []types.Message
	for msg := range msgChan {
		messages = append(messages, msg)
		switch m := msg.(type) {
		case *types.SystemMessage:
			t.Logf("Received SystemMessage: subtype=%s", m.Subtype)
		case *types.AssistantMessage:
			t.Logf("Received AssistantMessage: %s", formatContent(m.Content))
		case *types.ResultMessage:
			t.Logf("Received ResultMessage: %s", formatResult(m.Result))
			// ResultMessage indicates the conversation is complete
			t.Logf("Total messages received: %d", len(messages))
			// Check messages
			hasSystem := false
			hasAssistant := false
			for _, msg := range messages {
				switch msg.(type) {
				case *types.SystemMessage:
					hasSystem = true
				case *types.AssistantMessage:
					hasAssistant = true
				}
			}
			if !hasSystem {
				t.Error("Missing SystemMessage (init)")
			} else {
				t.Log("Found SystemMessage (init)")
			}
			if !hasAssistant {
				t.Errorf("Missing AssistantMessage - this may indicate issue with filesystem agents.")
			} else {
				t.Log("Found AssistantMessage")
			}
			// Find the init message and check for the filesystem agent
			for _, msg := range messages {
				if sm, ok := msg.(*types.SystemMessage); ok && sm.Subtype == "init" {
					agents, ok := sm.Data["agents"].([]interface{})
					if ok {
						t.Logf("Found agents in init: %v", agents)
						found := false
						for _, agent := range agents {
							if agent.(string) == "fs-test-agent" {
								found = true
								t.Log("SUCCESS: Found 'fs-test-agent' in init message")
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
			t.Log("TEST PASSED: Filesystem agent loading test completed")
			return
		}
	}

	t.Logf("Total messages received: %d", len(messages))

	// Must have at least init, assistant, result
	messageTypes := make([]string, 0, len(messages))
	for _, msg := range messages {
		messageTypes = append(messageTypes, fmt.Sprintf("%T", msg))
	}
	t.Logf("Message types: %v", messageTypes)

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
	} else {
		t.Log("Found SystemMessage (init)")
	}
	if !hasAssistant {
		t.Errorf("Missing AssistantMessage - got only: %v. This may indicate issue with filesystem agents.", messageTypes)
	} else {
		t.Log("Found AssistantMessage")
	}
	if !hasResult {
		t.Error("Missing ResultMessage")
	} else {
		t.Log("Found ResultMessage")
	}

	// Find the init message and check for the filesystem agent
	for _, msg := range messages {
		if m, ok := msg.(*types.SystemMessage); ok && m.Subtype == "init" {
			agents, ok := m.Data["agents"].([]interface{})
			if ok {
				t.Logf("Found agents in init: %v", agents)
				found := false
				for _, agent := range agents {
					if agent.(string) == "fs-test-agent" {
						found = true
						t.Log("SUCCESS: Found 'fs-test-agent' in init message")
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

	t.Log("TEST PASSED: Filesystem agent loading test completed")
}
