package messageparser

import (
	"testing"

	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Rate Limit Event and Forward Compatibility Tests
// ============================================================================

func TestRateLimitEventReturnsNil(t *testing.T) {
	// rate_limit_event should be silently skipped, not crash
	data := map[string]interface{}{
		"type": "rate_limit_event",
		"rate_limit_info": map[string]interface{}{
			"status":         "allowed_warning",
			"resetsAt":       float64(1700000000),
			"rateLimitType":  "five_hour",
			"utilization":    0.85,
			"isUsingOverage": false,
		},
		"uuid":       "550e8400-e29b-41d4-a716-446655440000",
		"session_id": "test-session-id",
	}

	result, err := ParseMessage(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != nil {
		t.Error("Expected nil for rate_limit_event (forward compatibility)")
	}
}

func TestRateLimitEventRejectedReturnsNil(t *testing.T) {
	// Hard rate limit (status=rejected) should also be skipped
	data := map[string]interface{}{
		"type": "rate_limit_event",
		"rate_limit_info": map[string]interface{}{
			"status":                "rejected",
			"resetsAt":              float64(1700003600),
			"rateLimitType":         "seven_day",
			"isUsingOverage":        false,
			"overageStatus":         "rejected",
			"overageDisabledReason": "out_of_credits",
		},
		"uuid":       "660e8400-e29b-41d4-a716-446655440001",
		"session_id": "test-session-id",
	}

	result, err := ParseMessage(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != nil {
		t.Error("Expected nil for rejected rate_limit_event (forward compatibility)")
	}
}

func TestUnknownMessageTypesAreHandled(t *testing.T) {
	// All unknown message types should return nil for forward compatibility
	unknownTypes := []string{
		"some_future_event_type",
		"new_feature_message",
		"debug_event",
		"performance_metric",
		"unknown_type",
	}

	for _, msgType := range unknownTypes {
		t.Run(msgType, func(t *testing.T) {
			data := map[string]interface{}{
				"type":       msgType,
				"uuid":       "770e8400-e29b-41d4-a716-446655440002",
				"session_id": "test-session-id",
				"data":       "some data",
			}

			result, err := ParseMessage(data)
			if err != nil {
				t.Errorf("Unexpected error for type %s: %v", msgType, err)
			}

			if result != nil {
				t.Errorf("Expected nil for unknown type %s (forward compatibility)", msgType)
			}
		})
	}
}

func TestKnownMessageTypesStillParsed(t *testing.T) {
	// Known message types should still be parsed normally
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "hello",
				},
			},
			"model": "claude-sonnet-4-6-20250929",
		},
	}

	result, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result for known message type")
	}

	// Verify it's an assistant message
	_, ok := result.(*types.AssistantMessage)
	if !ok {
		t.Error("Expected AssistantMessage type")
	}
}

func TestParseMessageWithMissingFields(t *testing.T) {
	// Test that messages with missing optional fields don't crash
	data := map[string]interface{}{
		"type": "result",
		// Missing many optional fields
	}

	result, err := ParseMessage(data)
	// Should not error, should handle gracefully
	if err != nil {
		t.Logf("Result message with missing fields returned error (acceptable): %v", err)
	}
	_ = result
}

func TestParseMessageWithExtraFields(t *testing.T) {
	// Test that messages with extra unknown fields are handled
	data := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "hello",
				},
			},
			"model": "claude-sonnet-4-6-20250929",
		},
		// Extra fields that might be added in future versions
		"future_field_1": "value1",
		"future_field_2": map[string]interface{}{"nested": "data"},
		"future_field_3": []interface{}{1, 2, 3},
	}

	result, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result for message with extra fields")
	}
}

func TestRateLimitEventVariations(t *testing.T) {
	// Test various rate limit event configurations
	variations := []map[string]interface{}{
		{
			"type": "rate_limit_event",
			"rate_limit_info": map[string]interface{}{
				"status": "allowed",
			},
		},
		{
			"type": "rate_limit_event",
			"rate_limit_info": map[string]interface{}{
				"status":    "warning",
				"resetTime": "2024-01-01T00:00:00Z",
			},
		},
		{
			"type": "rate_limit_event",
			"rate_limit_info": map[string]interface{}{
				"status":         "blocked",
				"retryAfter":     3600,
				"rateLimitType":  "daily",
				"utilization":    1.0,
				"isUsingOverage": true,
			},
		},
	}

	for i, data := range variations {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			result, err := ParseMessage(data)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != nil {
				t.Error("Expected nil for rate_limit_event")
			}
		})
	}
}
