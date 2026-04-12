package messageparser

import (
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Rate Limit Event Tests
// ============================================================================

func TestParseRateLimitEvent(t *testing.T) {
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
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result for rate_limit_event")
	}

	// Verify it's a RateLimitEvent
	event, ok := result.(*types.RateLimitEvent)
	if !ok {
		t.Fatalf("Expected RateLimitEvent type, got %T", result)
	}

	// Verify fields
	if event.UUID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("Expected UUID '550e8400-e29b-41d4-a716-446655440000', got %s", event.UUID)
	}
	if event.SessionID != "test-session-id" {
		t.Errorf("Expected SessionID 'test-session-id', got %s", event.SessionID)
	}
	if event.RateLimitInfo.Status != types.RateLimitStatusAllowedWarning {
		t.Errorf("Expected status 'allowed_warning', got %s", event.RateLimitInfo.Status)
	}
	if event.RateLimitInfo.ResetsAt == nil || *event.RateLimitInfo.ResetsAt != 1700000000 {
		t.Errorf("Expected ResetsAt 1700000000, got %v", event.RateLimitInfo.ResetsAt)
	}
	if event.RateLimitInfo.RateLimitType == nil || *event.RateLimitInfo.RateLimitType != types.RateLimitTypeFiveHour {
		t.Errorf("Expected RateLimitType 'five_hour', got %v", event.RateLimitInfo.RateLimitType)
	}
	if event.RateLimitInfo.Utilization == nil || *event.RateLimitInfo.Utilization != 0.85 {
		t.Errorf("Expected Utilization 0.85, got %v", event.RateLimitInfo.Utilization)
	}
}

func TestParseRateLimitEventRejected(t *testing.T) {
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
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result for rate_limit_event")
	}

	event, ok := result.(*types.RateLimitEvent)
	if !ok {
		t.Fatalf("Expected RateLimitEvent type, got %T", result)
	}

	if event.RateLimitInfo.Status != types.RateLimitStatusRejected {
		t.Errorf("Expected status 'rejected', got %s", event.RateLimitInfo.Status)
	}
	if event.RateLimitInfo.OverageStatus == nil || *event.RateLimitInfo.OverageStatus != types.RateLimitStatusRejected {
		t.Errorf("Expected OverageStatus 'rejected', got %v", event.RateLimitInfo.OverageStatus)
	}
	if event.RateLimitInfo.OverageDisabledReason == nil || *event.RateLimitInfo.OverageDisabledReason != "out_of_credits" {
		t.Errorf("Expected OverageDisabledReason 'out_of_credits', got %v", event.RateLimitInfo.OverageDisabledReason)
	}
}

func TestParseRateLimitEventMinimal(t *testing.T) {
	// Minimal rate limit event with only required fields
	data := map[string]interface{}{
		"type": "rate_limit_event",
		"rate_limit_info": map[string]interface{}{
			"status": "allowed",
		},
		"uuid":       "770e8400-e29b-41d4-a716-446655440002",
		"session_id": "test-session-id",
	}

	result, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result for rate_limit_event")
	}

	event, ok := result.(*types.RateLimitEvent)
	if !ok {
		t.Fatalf("Expected RateLimitEvent type, got %T", result)
	}

	if event.RateLimitInfo.Status != types.RateLimitStatusAllowed {
		t.Errorf("Expected status 'allowed', got %s", event.RateLimitInfo.Status)
	}
	// Optional fields should be nil
	if event.RateLimitInfo.ResetsAt != nil {
		t.Errorf("Expected ResetsAt to be nil, got %v", event.RateLimitInfo.ResetsAt)
	}
	if event.RateLimitInfo.RateLimitType != nil {
		t.Errorf("Expected RateLimitType to be nil, got %v", event.RateLimitInfo.RateLimitType)
	}
}

func TestParseRateLimitEventMissingRequiredFields(t *testing.T) {
	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "missing_uuid",
			data: map[string]interface{}{
				"type": "rate_limit_event",
				"rate_limit_info": map[string]interface{}{
					"status": "allowed",
				},
				"session_id": "test-session-id",
			},
		},
		{
			name: "missing_session_id",
			data: map[string]interface{}{
				"type": "rate_limit_event",
				"rate_limit_info": map[string]interface{}{
					"status": "allowed",
				},
				"uuid": "770e8400-e29b-41d4-a716-446655440002",
			},
		},
		{
			name: "missing_rate_limit_info",
			data: map[string]interface{}{
				"type":       "rate_limit_event",
				"uuid":       "770e8400-e29b-41d4-a716-446655440002",
				"session_id": "test-session-id",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMessage(tc.data)
			if err == nil {
				t.Error("Expected error for missing required field")
			}
		})
	}
}

func TestRateLimitEventGetSessionID(t *testing.T) {
	event := &types.RateLimitEvent{
		SessionID: "test-session-123",
	}

	if event.GetSessionID() != "test-session-123" {
		t.Errorf("Expected GetSessionID to return 'test-session-123', got %s", event.GetSessionID())
	}
}

// ============================================================================
// Forward Compatibility Tests
// ============================================================================

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
