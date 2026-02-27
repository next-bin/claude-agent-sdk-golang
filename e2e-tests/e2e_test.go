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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:    types.String(DefaultTestConfig().Model),
		MaxTurns: types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Say 'Hello, World!' and nothing else")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	for msg := range msgChan {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			// Check content
			if len(m.Content) == 0 {
				t.Error("Expected assistant message to have content")
			}
		case *types.ResultMessage:
			foundResult = true
			if m.IsError {
				t.Errorf("Result was an error: %v", m)
			}
		}
	}

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

func TestQueryWithSystemPrompt(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(DefaultTestConfig().Model),
		SystemPrompt: "You are a helpful assistant that only responds with the word 'PONG'.",
		MaxTurns:     types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Say something")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	for msg := range msgChan {
		switch msg.(type) {
		case *types.ResultMessage:
			foundResult = true
		}
	}

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}

func TestQueryWithBudget(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Set a very low budget to test budget handling
	budget := 0.001
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:        types.String(DefaultTestConfig().Model),
		MaxBudgetUSD: &budget,
		MaxTurns:     types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Write a very long story about a cat")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Just verify we get messages - budget may or may not trigger
	var receivedMessages bool
	for range msgChan {
		receivedMessages = true
	}

	if !receivedMessages {
		t.Error("Expected to receive some messages")
	}
}

func TestStreamingMode(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:    types.String(DefaultTestConfig().Model),
		MaxTurns: types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Count from 1 to 5")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	messageCount := 0
	for msg := range msgChan {
		messageCount++
		// Just consume messages
		_ = msg
	}

	if messageCount == 0 {
		t.Error("Expected to receive at least one message")
	}
}

// ============================================================================
// Permission Mode Tests
// ============================================================================

func TestBypassPermissionsMode(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Say 'test passed'")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var foundResult bool
	for msg := range msgChan {
		switch msg.(type) {
		case *types.ResultMessage:
			foundResult = true
		}
	}

	if !foundResult {
		t.Error("Expected to receive a result message")
	}
}
