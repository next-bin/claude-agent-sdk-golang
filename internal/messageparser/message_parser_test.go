package messageparser

import (
	"testing"

	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// ParseMessage Tests
// ============================================================================

func TestParseMessageNilData(t *testing.T) {
	_, err := ParseMessage(nil)
	if err == nil {
		t.Error("Expected error for nil data")
	}
}

func TestParseMessageMissingType(t *testing.T) {
	_, err := ParseMessage(map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing type field")
	}
}

func TestParseMessageUnknownType(t *testing.T) {
	msg, err := ParseMessage(map[string]interface{}{
		"type": "unknown_type",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if msg != nil {
		t.Error("Expected nil for unknown message type (forward compatibility)")
	}
}

// ============================================================================
// User Message Tests
// ============================================================================

func TestParseUserMessageString(t *testing.T) {
	data := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"content": "Hello, Claude!",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	userMsg, ok := msg.(*types.UserMessage)
	if !ok {
		t.Fatal("Expected UserMessage")
	}

	content, ok := userMsg.Content.(string)
	if !ok {
		t.Fatal("Expected string content")
	}
	if content != "Hello, Claude!" {
		t.Errorf("Expected 'Hello, Claude!', got %q", content)
	}
}

func TestParseUserMessageWithUUID(t *testing.T) {
	data := map[string]interface{}{
		"type": "user",
		"uuid": "test-uuid-123",
		"message": map[string]interface{}{
			"content": "Hello",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	userMsg := msg.(*types.UserMessage)
	if userMsg.UUID == nil || *userMsg.UUID != "test-uuid-123" {
		t.Errorf("Expected UUID 'test-uuid-123', got %v", userMsg.UUID)
	}
}

func TestParseUserMessageWithContentBlocks(t *testing.T) {
	data := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello",
				},
				map[string]interface{}{
					"type": "tool_use",
					"id":   "tool-123",
					"name": "test_tool",
				},
			},
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	userMsg := msg.(*types.UserMessage)
	blocks, ok := userMsg.Content.([]types.ContentBlock)
	if !ok {
		t.Fatal("Expected content blocks")
	}
	if len(blocks) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(blocks))
	}
}

func TestParseUserMessageMissingMessage(t *testing.T) {
	data := map[string]interface{}{
		"type": "user",
	}

	_, err := ParseMessage(data)
	if err == nil {
		t.Error("Expected error for missing message field")
	}
}

func TestParseUserMessageWithToolResult(t *testing.T) {
	data := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"content": "result",
		},
		"tool_use_result": map[string]interface{}{
			"output": "tool output",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	userMsg := msg.(*types.UserMessage)
	if userMsg.ToolUseResult == nil {
		t.Error("Expected tool_use_result to be set")
	}
}

func TestParseUserMessageWithParentToolUseID(t *testing.T) {
	data := map[string]interface{}{
		"type":               "user",
		"parent_tool_use_id": "parent-123",
		"message": map[string]interface{}{
			"content": "Hello",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	userMsg := msg.(*types.UserMessage)
	if userMsg.ParentToolUseID == nil || *userMsg.ParentToolUseID != "parent-123" {
		t.Errorf("Expected parent_tool_use_id 'parent-123', got %v", userMsg.ParentToolUseID)
	}
}

// ============================================================================
// Assistant Message Tests
// ============================================================================

func TestParseAssistantMessage(t *testing.T) {
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "Hello!",
				},
			},
			"model": "claude-sonnet-4-20250514",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	assistantMsg, ok := msg.(*types.AssistantMessage)
	if !ok {
		t.Fatal("Expected AssistantMessage")
	}
	if assistantMsg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected model 'claude-sonnet-4-20250514', got %q", assistantMsg.Model)
	}
}

func TestParseAssistantMessageWithThinking(t *testing.T) {
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "thinking",
					"thinking":  "Let me think...",
					"signature": "sig-123",
				},
			},
			"model": "claude-sonnet-4-20250514",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	assistantMsg := msg.(*types.AssistantMessage)
	blocks := assistantMsg.Content
	if len(blocks) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(blocks))
	}

	thinkingBlock, ok := blocks[0].(types.ThinkingBlock)
	if !ok {
		t.Fatal("Expected ThinkingBlock")
	}
	if thinkingBlock.Thinking != "Let me think..." {
		t.Errorf("Expected thinking 'Let me think...', got %q", thinkingBlock.Thinking)
	}
}

func TestParseAssistantMessageWithToolUse(t *testing.T) {
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":  "tool_use",
					"id":    "tool-123",
					"name":  "test_tool",
					"input": map[string]interface{}{"arg": "value"},
				},
			},
			"model": "claude-sonnet-4-20250514",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	assistantMsg := msg.(*types.AssistantMessage)
	blocks := assistantMsg.Content
	toolUseBlock, ok := blocks[0].(types.ToolUseBlock)
	if !ok {
		t.Fatal("Expected ToolUseBlock")
	}
	if toolUseBlock.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got %q", toolUseBlock.Name)
	}
}

func TestParseAssistantMessageWithError(t *testing.T) {
	data := map[string]interface{}{
		"type":  "assistant",
		"error": "rate_limit",
		"message": map[string]interface{}{
			"content": []interface{}{},
			"model":   "claude-sonnet-4-20250514",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	assistantMsg := msg.(*types.AssistantMessage)
	if assistantMsg.Error == nil || *assistantMsg.Error != "rate_limit" {
		t.Errorf("Expected error 'rate_limit', got %v", assistantMsg.Error)
	}
}

func TestParseAssistantMessageMissingModel(t *testing.T) {
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{},
		},
	}

	_, err := ParseMessage(data)
	if err == nil {
		t.Error("Expected error for missing model field")
	}
}

// ============================================================================
// System Message Tests
// ============================================================================

func TestParseSystemMessage(t *testing.T) {
	data := map[string]interface{}{
		"type":    "system",
		"subtype": "init",
		"data":    map[string]interface{}{"key": "value"},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	systemMsg, ok := msg.(*types.SystemMessage)
	if !ok {
		t.Fatal("Expected SystemMessage")
	}
	if systemMsg.Subtype != "init" {
		t.Errorf("Expected subtype 'init', got %q", systemMsg.Subtype)
	}
}

func TestParseSystemMessageMissingSubtype(t *testing.T) {
	data := map[string]interface{}{
		"type": "system",
	}

	_, err := ParseMessage(data)
	if err == nil {
		t.Error("Expected error for missing subtype field")
	}
}

// ============================================================================
// Result Message Tests
// ============================================================================

func TestParseResultMessage(t *testing.T) {
	data := map[string]interface{}{
		"type":            "result",
		"subtype":         "success",
		"duration_ms":     1000.0,
		"duration_api_ms": 500.0,
		"is_error":        false,
		"num_turns":       2.0,
		"session_id":      "session-123",
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultMsg, ok := msg.(*types.ResultMessage)
	if !ok {
		t.Fatal("Expected ResultMessage")
	}
	if resultMsg.SessionID != "session-123" {
		t.Errorf("Expected session_id 'session-123', got %q", resultMsg.SessionID)
	}
	if resultMsg.DurationMs != 1000 {
		t.Errorf("Expected duration_ms 1000, got %d", resultMsg.DurationMs)
	}
}

func TestParseResultMessageWithCost(t *testing.T) {
	cost := 0.05
	data := map[string]interface{}{
		"type":            "result",
		"subtype":         "success",
		"duration_ms":     1000.0,
		"duration_api_ms": 500.0,
		"is_error":        false,
		"num_turns":       2.0,
		"session_id":      "session-123",
		"total_cost_usd":  cost,
		"result":          "Task completed",
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultMsg := msg.(*types.ResultMessage)
	if resultMsg.TotalCostUSD == nil || *resultMsg.TotalCostUSD != 0.05 {
		t.Errorf("Expected total_cost_usd 0.05, got %v", resultMsg.TotalCostUSD)
	}
	if resultMsg.Result == nil || *resultMsg.Result != "Task completed" {
		t.Errorf("Expected result 'Task completed', got %v", resultMsg.Result)
	}
}

func TestParseResultMessageWithError(t *testing.T) {
	data := map[string]interface{}{
		"type":            "result",
		"subtype":         "error",
		"duration_ms":     100.0,
		"duration_api_ms": 50.0,
		"is_error":        true,
		"num_turns":       1.0,
		"session_id":      "session-123",
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultMsg := msg.(*types.ResultMessage)
	if !resultMsg.IsError {
		t.Error("Expected is_error to be true")
	}
}

func TestParseResultMessageWithStructuredOutput(t *testing.T) {
	data := map[string]interface{}{
		"type":              "result",
		"subtype":           "success",
		"duration_ms":       1000.0,
		"duration_api_ms":   500.0,
		"is_error":          false,
		"num_turns":         2.0,
		"session_id":        "session-123",
		"structured_output": map[string]interface{}{"key": "value"},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultMsg := msg.(*types.ResultMessage)
	if resultMsg.StructuredOutput == nil {
		t.Error("Expected structured_output to be set")
	}
}

// ============================================================================
// Stream Event Tests
// ============================================================================

func TestParseStreamEvent(t *testing.T) {
	data := map[string]interface{}{
		"type":       "stream_event",
		"uuid":       "event-123",
		"session_id": "session-123",
		"event": map[string]interface{}{
			"type": "content_block_start",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	streamEvent, ok := msg.(*types.StreamEvent)
	if !ok {
		t.Fatal("Expected StreamEvent")
	}
	if streamEvent.UUID != "event-123" {
		t.Errorf("Expected uuid 'event-123', got %q", streamEvent.UUID)
	}
}

func TestParseStreamEventWithParentToolUseID(t *testing.T) {
	data := map[string]interface{}{
		"type":               "stream_event",
		"uuid":               "event-123",
		"session_id":         "session-123",
		"parent_tool_use_id": "parent-456",
		"event": map[string]interface{}{
			"type": "content_block_delta",
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	streamEvent := msg.(*types.StreamEvent)
	if streamEvent.ParentToolUseID == nil || *streamEvent.ParentToolUseID != "parent-456" {
		t.Errorf("Expected parent_tool_use_id 'parent-456', got %v", streamEvent.ParentToolUseID)
	}
}

// ============================================================================
// Content Block Tests
// ============================================================================

func TestParseTextBlock(t *testing.T) {
	data := map[string]interface{}{
		"type": "text",
		"text": "Hello, World!",
	}

	block, err := parseTextBlock(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if block.Text != "Hello, World!" {
		t.Errorf("Expected text 'Hello, World!', got %q", block.Text)
	}
}

func TestParseTextBlockMissingText(t *testing.T) {
	data := map[string]interface{}{
		"type": "text",
	}

	_, err := parseTextBlock(data)
	if err == nil {
		t.Error("Expected error for missing text field")
	}
}

func TestParseThinkingBlock(t *testing.T) {
	data := map[string]interface{}{
		"type":      "thinking",
		"thinking":  "Let me think about this...",
		"signature": "sig-123",
	}

	block, err := parseThinkingBlock(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if block.Thinking != "Let me think about this..." {
		t.Errorf("Expected thinking 'Let me think about this...', got %q", block.Thinking)
	}
	if block.Signature != "sig-123" {
		t.Errorf("Expected signature 'sig-123', got %q", block.Signature)
	}
}

func TestParseThinkingBlockMissingThinking(t *testing.T) {
	data := map[string]interface{}{
		"type": "thinking",
	}

	_, err := parseThinkingBlock(data)
	if err == nil {
		t.Error("Expected error for missing thinking field")
	}
}

func TestParseToolUseBlock(t *testing.T) {
	data := map[string]interface{}{
		"type":  "tool_use",
		"id":    "tool-123",
		"name":  "test_tool",
		"input": map[string]interface{}{"arg": "value"},
	}

	block, err := parseToolUseBlock(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if block.ID != "tool-123" {
		t.Errorf("Expected id 'tool-123', got %q", block.ID)
	}
	if block.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got %q", block.Name)
	}
}

func TestParseToolUseBlockMissingID(t *testing.T) {
	data := map[string]interface{}{
		"type": "tool_use",
		"name": "test_tool",
	}

	_, err := parseToolUseBlock(data)
	if err == nil {
		t.Error("Expected error for missing id field")
	}
}

func TestParseToolResultBlock(t *testing.T) {
	data := map[string]interface{}{
		"type":        "tool_result",
		"tool_use_id": "tool-123",
		"content":     "result content",
		"is_error":    false,
	}

	block, err := parseToolResultBlock(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if block.ToolUseID != "tool-123" {
		t.Errorf("Expected tool_use_id 'tool-123', got %q", block.ToolUseID)
	}
}

func TestParseToolResultBlockWithError(t *testing.T) {
	data := map[string]interface{}{
		"type":        "tool_result",
		"tool_use_id": "tool-123",
		"content":     "error occurred",
		"is_error":    true,
	}

	block, err := parseToolResultBlock(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if block.IsError == nil || !*block.IsError {
		t.Error("Expected is_error to be true")
	}
}

func TestParseToolResultBlockMissingToolUseID(t *testing.T) {
	data := map[string]interface{}{
		"type": "tool_result",
	}

	_, err := parseToolResultBlock(data)
	if err == nil {
		t.Error("Expected error for missing tool_use_id field")
	}
}

// ============================================================================
// Content Blocks Array Tests
// ============================================================================

func TestParseContentBlocks(t *testing.T) {
	blocks := []interface{}{
		map[string]interface{}{
			"type": "text",
			"text": "Hello",
		},
		map[string]interface{}{
			"type": "tool_use",
			"id":   "tool-1",
			"name": "test_tool",
		},
		map[string]interface{}{
			"type":        "tool_result",
			"tool_use_id": "tool-1",
			"content":     "result",
		},
	}

	result, err := parseContentBlocks(blocks)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 content blocks, got %d", len(result))
	}
}

func TestParseContentBlocksWithUnknownType(t *testing.T) {
	blocks := []interface{}{
		map[string]interface{}{
			"type": "unknown_block",
			"data": "some data",
		},
	}

	result, err := parseContentBlocks(blocks)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 content block (generic), got %d", len(result))
	}
}

func TestParseContentBlocksEmpty(t *testing.T) {
	result, err := parseContentBlocks([]interface{}{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 content blocks, got %d", len(result))
	}
}

func TestParseContentBlocksInvalidBlock(t *testing.T) {
	blocks := []interface{}{
		"not a map",
		map[string]interface{}{
			"type": "text",
			"text": "valid",
		},
	}

	result, err := parseContentBlocks(blocks)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 content block (invalid skipped), got %d", len(result))
	}
}
