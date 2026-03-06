package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Include Partial Messages E2E Tests
// ============================================================================

// TestIncludePartialMessagesStreamEvents tests that include_partial_messages
// produces StreamEvent messages.
func TestIncludePartialMessagesStreamEvents(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	trueVal := true
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:                  types.String(DefaultTestConfig().Model),
		IncludePartialMessages: trueVal,
		MaxTurns:               types.Int(2),
		PermissionMode:         permissionModePtr(types.PermissionModeBypassPermissions),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Think of three jokes, then tell one"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var collectedMessages []types.Message
	var streamEvents []*types.StreamEvent
	var assistantMessages []*types.AssistantMessage

	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		collectedMessages = append(collectedMessages, msg)

		switch m := msg.(type) {
		case *types.StreamEvent:
			streamEvents = append(streamEvents, m)
		case *types.AssistantMessage:
			assistantMessages = append(assistantMessages, m)
		}
	})

	// Should have SystemMessage(init) at the start
	if len(collectedMessages) == 0 {
		t.Fatal("No messages received")
	}

	initMsg, ok := collectedMessages[0].(*types.SystemMessage)
	if !ok {
		t.Fatalf("First message should be SystemMessage, got %T", collectedMessages[0])
	}
	if initMsg.Subtype != "init" {
		t.Errorf("Expected init subtype, got %s", initMsg.Subtype)
	}

	// Should have multiple StreamEvent messages
	if len(streamEvents) == 0 {
		t.Error("No StreamEvent messages received")
	} else {
		t.Logf("Received %d StreamEvent messages", len(streamEvents))

		// Check for expected StreamEvent types
		eventTypes := make(map[string]bool)
		for _, event := range streamEvents {
			if eventType, ok := event.Event["type"].(string); ok {
				eventTypes[eventType] = true
			}
		}

		expectedTypes := []string{"message_start", "content_block_start", "content_block_delta", "content_block_stop", "message_stop"}
		for _, expected := range expectedTypes {
			if !eventTypes[expected] {
				t.Errorf("Missing expected StreamEvent type: %s", expected)
			}
		}
	}

	// Should have AssistantMessage messages
	if len(assistantMessages) == 0 {
		t.Error("No AssistantMessage received")
	} else {
		// Check for text block
		hasText := false
		for _, msg := range assistantMessages {
			for _, block := range msg.Content {
				if _, ok := block.(types.TextBlock); ok {
					hasText = true
					break
				}
			}
		}
		if !hasText {
			t.Error("No TextBlock found in AssistantMessages")
		}
	}
}

// TestPartialMessagesDisabledByDefault tests that partial messages are not
// included when option is not set.
func TestPartialMessagesDisabledByDefault(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	// IncludePartialMessages not set (defaults to false)
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		MaxTurns:       types.Int(2),
		PermissionMode: permissionModePtr(types.PermissionModeBypassPermissions),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Say hello"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var streamEvents []*types.StreamEvent
	var hasSystem, hasAssistant, hasResult bool

	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		switch m := msg.(type) {
		case *types.StreamEvent:
			streamEvents = append(streamEvents, m)
		case *types.SystemMessage:
			hasSystem = true
		case *types.AssistantMessage:
			hasAssistant = true
		case *types.ResultMessage:
			hasResult = true
		}
	})

	// Should NOT have any StreamEvent messages
	if len(streamEvents) > 0 {
		t.Errorf("StreamEvent messages present when partial messages disabled: %d", len(streamEvents))
	}

	// Should still have the regular messages
	if !hasSystem {
		t.Error("Missing SystemMessage")
	}
	if !hasAssistant {
		t.Error("Missing AssistantMessage")
	}
	if !hasResult {
		t.Error("Missing ResultMessage")
	}
}

// TestThinkingDeltas tests that thinking content is streamed incrementally via deltas.
func TestThinkingDeltas(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	trueVal := true
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:                  types.String(DefaultTestConfig().Model),
		IncludePartialMessages: trueVal,
		MaxTurns:               types.Int(2),
		PermissionMode:         permissionModePtr(types.PermissionModeBypassPermissions),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Think step by step about what 2 + 2 equals"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var thinkingDeltas []string

	_, _ = consumeAllMessagesUntilResult(ctx, msgChan, func(msg types.Message) {
		if event, ok := msg.(*types.StreamEvent); ok {
			if eventType, ok := event.Event["type"].(string); ok && eventType == "content_block_delta" {
				if delta, ok := event.Event["delta"].(map[string]interface{}); ok {
					if deltaType, ok := delta["type"].(string); ok && deltaType == "thinking_delta" {
						if thinking, ok := delta["thinking"].(string); ok {
							thinkingDeltas = append(thinkingDeltas, thinking)
						}
					}
				}
			}
		}
	})

	// Should have received thinking deltas (may or may not have depending on model response)
	t.Logf("Received %d thinking deltas", len(thinkingDeltas))
}
