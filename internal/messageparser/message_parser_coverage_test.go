// Package messageparser provides coverage tests for task message parsing.
package messageparser

import (
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func TestParseTaskStartedMessageValid(t *testing.T) {
	data := map[string]interface{}{
		"type":        "system",
		"subtype":     "task_started",
		"task_id":     "task-123",
		"description": "Running analysis",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
		"tool_use_id": "tool-use-1",
		"task_type":   "analysis",
	}

	msg, err := parseTaskStartedMessage(data)
	if err != nil {
		t.Fatalf("parseTaskStartedMessage failed: %v", err)
	}

	if msg.TaskID != "task-123" {
		t.Errorf("TaskID = %s, want task-123", msg.TaskID)
	}
	if msg.Description != "Running analysis" {
		t.Errorf("Description = %s, want Running analysis", msg.Description)
	}
	if msg.UUID != "uuid-abc" {
		t.Errorf("UUID = %s, want uuid-abc", msg.UUID)
	}
	if msg.SessionID != "session-xyz" {
		t.Errorf("SessionID = %s, want session-xyz", msg.SessionID)
	}
	if msg.ToolUseID == nil || *msg.ToolUseID != "tool-use-1" {
		t.Errorf("ToolUseID mismatch")
	}
	if msg.TaskType == nil || *msg.TaskType != "analysis" {
		t.Errorf("TaskType mismatch")
	}
	if msg.Subtype != "task_started" {
		t.Errorf("Subtype = %s, want task_started", msg.Subtype)
	}
}

func TestParseTaskStartedMessageMinimal(t *testing.T) {
	data := map[string]interface{}{
		"task_id":     "task-123",
		"description": "Test",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
	}

	msg, err := parseTaskStartedMessage(data)
	if err != nil {
		t.Fatalf("parseTaskStartedMessage failed: %v", err)
	}

	// Optional fields should be nil
	if msg.ToolUseID != nil {
		t.Errorf("ToolUseID should be nil")
	}
	if msg.TaskType != nil {
		t.Errorf("TaskType should be nil")
	}
}

func TestParseTaskStartedMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{"missing task_id", map[string]interface{}{"description": "Test", "uuid": "u", "session_id": "s"}},
		{"missing description", map[string]interface{}{"task_id": "t", "uuid": "u", "session_id": "s"}},
		{"missing uuid", map[string]interface{}{"task_id": "t", "description": "Test", "session_id": "s"}},
		{"missing session_id", map[string]interface{}{"task_id": "t", "description": "Test", "uuid": "u"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTaskStartedMessage(tt.data)
			if err == nil {
				t.Error("Expected error")
			}
		})
	}
}

func TestParseTaskProgressMessageValid(t *testing.T) {
	data := map[string]interface{}{
		"task_id":     "task-123",
		"description": "Processing",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
		"usage": map[string]interface{}{
			"total_tokens": float64(100), // JSON numbers are float64
			"tool_uses":    float64(5),
			"duration_ms":  float64(1000),
		},
	}

	msg, err := parseTaskProgressMessage(data)
	if err != nil {
		t.Fatalf("parseTaskProgressMessage failed: %v", err)
	}

	if msg.TaskID != "task-123" {
		t.Errorf("TaskID = %s, want task-123", msg.TaskID)
	}
	if msg.Description != "Processing" {
		t.Errorf("Description mismatch")
	}
	if msg.Usage.TotalTokens != 100 {
		t.Errorf("TotalTokens = %d, want 100", msg.Usage.TotalTokens)
	}
	if msg.Usage.ToolUses != 5 {
		t.Errorf("ToolUses = %d, want 5", msg.Usage.ToolUses)
	}
	if msg.Usage.DurationMs != 1000 {
		t.Errorf("DurationMs = %d, want 1000", msg.Usage.DurationMs)
	}
}

func TestParseTaskProgressMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{"missing task_id", map[string]interface{}{"description": "Test", "uuid": "u", "usage": map[string]interface{}{"total_tokens": 1}}},
		{"missing description", map[string]interface{}{"task_id": "t", "uuid": "u", "usage": map[string]interface{}{"total_tokens": 1}}},
		{"missing uuid", map[string]interface{}{"task_id": "t", "description": "Test", "usage": map[string]interface{}{"total_tokens": 1}}},
		{"missing usage", map[string]interface{}{"task_id": "t", "description": "Test", "uuid": "u"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTaskProgressMessage(tt.data)
			if err == nil {
				t.Error("Expected error")
			}
		})
	}
}

func TestParseTaskNotificationMessageValid(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   types.TaskNotificationStatus
	}{
		{"completed", "completed", types.TaskNotificationStatusCompleted},
		{"failed", "failed", types.TaskNotificationStatusFailed},
		{"stopped", "stopped", types.TaskNotificationStatusStopped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]interface{}{
				"task_id":     "task-123",
				"status":      tt.status,
				"output_file": "/path/to/output.txt",
				"summary":     "Test summary",
				"uuid":        "uuid-abc",
				"session_id":  "session-xyz",
			}

			msg, err := parseTaskNotificationMessage(data)
			if err != nil {
				t.Fatalf("parseTaskNotificationMessage failed: %v", err)
			}

			if msg.Status != tt.want {
				t.Errorf("Status = %s, want %s", msg.Status, tt.want)
			}
			if msg.OutputFile != "/path/to/output.txt" {
				t.Errorf("OutputFile mismatch")
			}
		})
	}
}

func TestParseTaskNotificationMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{"missing task_id", map[string]interface{}{"status": "completed", "output_file": "out", "summary": "Test", "uuid": "u"}},
		{"missing status", map[string]interface{}{"task_id": "t", "output_file": "out", "summary": "Test", "uuid": "u"}},
		{"missing output_file", map[string]interface{}{"task_id": "t", "status": "completed", "summary": "Test", "uuid": "u"}},
		{"missing summary", map[string]interface{}{"task_id": "t", "status": "completed", "output_file": "out", "uuid": "u"}},
		{"missing uuid", map[string]interface{}{"task_id": "t", "status": "completed", "output_file": "out", "summary": "Test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTaskNotificationMessage(tt.data)
			if err == nil {
				t.Error("Expected error")
			}
		})
	}
}

func TestParseTaskUsageValid(t *testing.T) {
	data := map[string]interface{}{
		"total_tokens": float64(1000), // JSON numbers are float64
		"tool_uses":    float64(10),
		"duration_ms":  float64(5000),
	}

	usage, err := parseTaskUsage(data)
	if err != nil {
		t.Fatalf("parseTaskUsage failed: %v", err)
	}

	if usage.TotalTokens != 1000 {
		t.Errorf("TotalTokens = %d, want 1000", usage.TotalTokens)
	}
	if usage.ToolUses != 10 {
		t.Errorf("ToolUses = %d, want 10", usage.ToolUses)
	}
	if usage.DurationMs != 5000 {
		t.Errorf("DurationMs = %d, want 5000", usage.DurationMs)
	}
}

func TestParseTaskUsageErrors(t *testing.T) {
	// parseTaskUsage doesn't return errors for missing fields, returns zero values
	// Test with empty data
	data := map[string]interface{}{}

	usage, err := parseTaskUsage(data)
	if err != nil {
		t.Fatalf("parseTaskUsage should not error: %v", err)
	}

	// All fields should be zero
	if usage.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", usage.TotalTokens)
	}
	if usage.ToolUses != 0 {
		t.Errorf("ToolUses = %d, want 0", usage.ToolUses)
	}
	if usage.DurationMs != 0 {
		t.Errorf("DurationMs = %d, want 0", usage.DurationMs)
	}
}

func TestParseTaskProgressMessageMissingSessionID(t *testing.T) {
	data := map[string]interface{}{
		"task_id":     "task-123",
		"description": "Test",
		"uuid":        "uuid-abc",
		"usage":       map[string]interface{}{"total_tokens": float64(1)},
	}

	_, err := parseTaskProgressMessage(data)
	if err == nil {
		t.Error("Expected error for missing session_id")
	}
}

func TestParseTaskProgressMessageWithOptionalFields(t *testing.T) {
	data := map[string]interface{}{
		"task_id":        "task-123",
		"description":    "Processing",
		"uuid":           "uuid-abc",
		"session_id":     "session-xyz",
		"tool_use_id":    "tool-123",
		"last_tool_name": "Read",
		"usage": map[string]interface{}{
			"total_tokens": float64(100),
			"tool_uses":    float64(5),
			"duration_ms":  float64(1000),
		},
	}

	msg, err := parseTaskProgressMessage(data)
	if err != nil {
		t.Fatalf("parseTaskProgressMessage failed: %v", err)
	}

	if msg.ToolUseID == nil || *msg.ToolUseID != "tool-123" {
		t.Errorf("ToolUseID mismatch")
	}
	if msg.LastToolName == nil || *msg.LastToolName != "Read" {
		t.Errorf("LastToolName mismatch")
	}
}

func TestParseTaskNotificationMessageMissingSessionID(t *testing.T) {
	data := map[string]interface{}{
		"task_id":     "task-123",
		"status":      "completed",
		"output_file": "out",
		"summary":     "Test",
		"uuid":        "uuid-abc",
	}

	_, err := parseTaskNotificationMessage(data)
	if err == nil {
		t.Error("Expected error for missing session_id")
	}
}

func TestParseTaskNotificationMessageWithUsage(t *testing.T) {
	data := map[string]interface{}{
		"task_id":     "task-123",
		"status":      "completed",
		"output_file": "/path/to/output.txt",
		"summary":     "Test summary",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
		"tool_use_id": "tool-123",
		"usage": map[string]interface{}{
			"total_tokens": float64(500),
			"tool_uses":    float64(10),
			"duration_ms":  float64(2000),
		},
	}

	msg, err := parseTaskNotificationMessage(data)
	if err != nil {
		t.Fatalf("parseTaskNotificationMessage failed: %v", err)
	}

	if msg.ToolUseID == nil || *msg.ToolUseID != "tool-123" {
		t.Errorf("ToolUseID mismatch")
	}
	if msg.Usage == nil {
		t.Fatal("Expected usage to be set")
	}
	if msg.Usage.TotalTokens != 500 {
		t.Errorf("TotalTokens = %d, want 500", msg.Usage.TotalTokens)
	}
}

func TestParseTaskNotificationMessageInvalidUsageType(t *testing.T) {
	data := map[string]interface{}{
		"task_id":     "task-123",
		"status":      "completed",
		"output_file": "/path/to/output.txt",
		"summary":     "Test summary",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
		"usage":       "invalid", // Not a map - should be ignored
	}

	msg, err := parseTaskNotificationMessage(data)
	if err != nil {
		t.Fatalf("parseTaskNotificationMessage should not fail: %v", err)
	}

	// Invalid usage type should result in nil usage
	if msg.Usage != nil {
		t.Errorf("Usage should be nil for invalid type")
	}
}

func TestParseSystemMessageTaskStarted(t *testing.T) {
	data := map[string]interface{}{
		"type":        "system",
		"subtype":     "task_started",
		"task_id":     "task-123",
		"description": "Test",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	taskMsg, ok := msg.(*types.TaskStartedMessage)
	if !ok {
		t.Fatal("Expected TaskStartedMessage")
	}
	if taskMsg.TaskID != "task-123" {
		t.Errorf("TaskID = %s, want task-123", taskMsg.TaskID)
	}
}

func TestParseSystemMessageTaskProgress(t *testing.T) {
	data := map[string]interface{}{
		"type":        "system",
		"subtype":     "task_progress",
		"task_id":     "task-123",
		"description": "Test",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
		"usage": map[string]interface{}{
			"total_tokens": float64(100),
		},
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	progressMsg, ok := msg.(*types.TaskProgressMessage)
	if !ok {
		t.Fatal("Expected TaskProgressMessage")
	}
	if progressMsg.TaskID != "task-123" {
		t.Errorf("TaskID = %s, want task-123", progressMsg.TaskID)
	}
}

func TestParseSystemMessageTaskNotification(t *testing.T) {
	data := map[string]interface{}{
		"type":        "system",
		"subtype":     "task_notification",
		"task_id":     "task-123",
		"status":      "completed",
		"output_file": "/path/to/output.txt",
		"summary":     "Test summary",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
	}

	msg, err := ParseMessage(data)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	notificationMsg, ok := msg.(*types.TaskNotificationMessage)
	if !ok {
		t.Fatal("Expected TaskNotificationMessage")
	}
	if notificationMsg.TaskID != "task-123" {
		t.Errorf("TaskID = %s, want task-123", notificationMsg.TaskID)
	}
}

func TestParseTaskUsageNilData(t *testing.T) {
	usage, err := parseTaskUsage(nil)
	if err != nil {
		t.Fatalf("parseTaskUsage should not error: %v", err)
	}
	if usage.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0", usage.TotalTokens)
	}
}

func TestParseTaskProgressMessageInvalidUsage(t *testing.T) {
	data := map[string]interface{}{
		"task_id":     "task-123",
		"description": "Test",
		"uuid":        "uuid-abc",
		"session_id":  "session-xyz",
		"usage":       "not a map", // Invalid usage type
	}

	_, err := parseTaskProgressMessage(data)
	if err == nil {
		t.Error("Expected error for invalid usage type")
	}
}
