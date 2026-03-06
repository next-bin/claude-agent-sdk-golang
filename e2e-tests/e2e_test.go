package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Basic E2E Tests
// ============================================================================

func TestBasicQuery(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestBasicQuery")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestBasicQuery")
	logger.Step("Creating client")

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: \"Say 'Hello, World!' and nothing else\"")
	if err := client.Query(ctx, "Say 'Hello, World!' and nothing else"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "TestBasicQuery")

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
	if resultMsg != nil && resultMsg.IsError {
		t.Errorf("Result was an error: %v", resultMsg)
	}

	PrintTestSummary(t, "TestBasicQuery", foundResult && (resultMsg == nil || !resultMsg.IsError), count, time.Since(startTime))
}

func TestQueryWithSystemPrompt(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestQueryWithSystemPrompt")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestQueryWithSystemPrompt")
	logger.Step("Creating client with custom system prompt")

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		SystemPrompt:   "You are a helpful assistant that only responds with the word 'PONG'.",
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: \"Say something\"")
	if err := client.Query(ctx, "Say something"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestQueryWithSystemPrompt")

	if !foundResult {
		t.Error("Expected to receive a result message")
	}

	PrintTestSummary(t, "TestQueryWithSystemPrompt", foundResult, count, time.Since(startTime))
}

func TestQueryWithBudget(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestQueryWithBudget")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestQueryWithBudget")
	logger.Step("Creating client with low budget (0.001 USD)")

	// Set a very low budget to test budget handling
	budget := 0.001
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		MaxBudgetUSD:   &budget,
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: \"Write a very long story about a cat\"")
	if err := client.Query(ctx, "Write a very long story about a cat"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, _, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestQueryWithBudget")

	// Just verify we get messages - budget may or may not trigger
	if count == 0 {
		t.Error("Expected to receive some messages")
	}

	PrintTestSummary(t, "TestQueryWithBudget", count > 0, count, time.Since(startTime))
}

func TestStreamingMode(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestStreamingMode")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestStreamingMode")
	logger.Step("Creating client")

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: \"Count from 1 to 5\"")
	if err := client.Query(ctx, "Count from 1 to 5"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, _, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestStreamingMode")

	if count == 0 {
		t.Error("Expected to receive at least one message")
	}

	PrintTestSummary(t, "TestStreamingMode", count > 0, count, time.Since(startTime))
}

// ============================================================================
// Permission Mode Tests
// ============================================================================

func TestBypassPermissionsMode(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestBypassPermissionsMode")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestBypassPermissionsMode")
	logger.Step("Creating client with BypassPermissions mode")

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: \"Say 'test passed'\"")
	if err := client.Query(ctx, "Say 'test passed'"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMsg := ConsumeMessagesVerbose(ctx, t, msgChan, "TestBypassPermissionsMode")

	if !foundResult {
		t.Error("Expected to receive a result message")
	}

	PrintTestSummary(t, "TestBypassPermissionsMode", foundResult && (resultMsg == nil || !resultMsg.IsError), count, time.Since(startTime))
}
