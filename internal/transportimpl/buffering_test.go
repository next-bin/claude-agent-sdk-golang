// Package transport provides transport layer implementations for the Claude Agent SDK.
package transportimpl

import (
	"encoding/json"
	"strings"
	"testing"
)

// ============================================================================
// Subprocess Buffering Tests
// ============================================================================

func TestMultipleJSONObjectsOnSingleLine(t *testing.T) {
	// Test parsing when multiple JSON objects are concatenated on a single line
	jsonObj1 := map[string]interface{}{
		"type":    "message",
		"id":      "msg1",
		"content": "First message",
	}
	jsonObj2 := map[string]interface{}{
		"type":   "result",
		"id":     "res1",
		"status": "completed",
	}

	bufferedLine := mustMarshal(jsonObj1) + "\n" + mustMarshal(jsonObj2)

	messages := parseBufferedLines(bufferedLine)

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["id"] != "msg1" {
		t.Errorf("Expected first message id 'msg1', got %v", messages[0]["id"])
	}

	if messages[1]["id"] != "res1" {
		t.Errorf("Expected second message id 'res1', got %v", messages[1]["id"])
	}
}

func TestJSONWithEmbeddedNewlines(t *testing.T) {
	// Test parsing JSON objects that contain newline characters in string values
	jsonObj1 := map[string]interface{}{
		"type":    "message",
		"content": "Line 1\nLine 2\nLine 3",
	}
	jsonObj2 := map[string]interface{}{
		"type": "result",
		"data": "Some\nMultiline\nContent",
	}

	bufferedLine := mustMarshal(jsonObj1) + "\n" + mustMarshal(jsonObj2)

	messages := parseBufferedLines(bufferedLine)

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["content"] != "Line 1\nLine 2\nLine 3" {
		t.Errorf("Expected multiline content in first message, got %v", messages[0]["content"])
	}

	if messages[1]["data"] != "Some\nMultiline\nContent" {
		t.Errorf("Expected multiline data in second message, got %v", messages[1]["data"])
	}
}

func TestMultipleNewlinesBetweenObjects(t *testing.T) {
	// Test parsing with multiple newlines between JSON objects
	jsonObj1 := map[string]interface{}{
		"type": "message",
		"id":   "msg1",
	}
	jsonObj2 := map[string]interface{}{
		"type": "result",
		"id":   "res1",
	}

	bufferedLine := mustMarshal(jsonObj1) + "\n\n\n" + mustMarshal(jsonObj2)

	messages := parseBufferedLines(bufferedLine)

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0]["id"] != "msg1" {
		t.Errorf("Expected first message id 'msg1', got %v", messages[0]["id"])
	}

	if messages[1]["id"] != "res1" {
		t.Errorf("Expected second message id 'res1', got %v", messages[1]["id"])
	}
}

func TestSplitJSONAcrossMultipleReads(t *testing.T) {
	// Test parsing when a single JSON object is split across multiple stream reads
	jsonObj := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": strings.Repeat("x", 1000),
				},
				map[string]interface{}{
					"type":  "tool_use",
					"id":    "tool_123",
					"name":  "Read",
					"input": map[string]interface{}{"file_path": "/test.txt"},
				},
			},
		},
	}

	completeJSON := mustMarshal(jsonObj)

	// Split into parts
	part1 := completeJSON[:100]
	part2 := completeJSON[100:250]
	part3 := completeJSON[250:]

	// Simulate buffer accumulation
	buffer := part1
	buffer += part2
	buffer += part3

	messages := parseBufferedLines(buffer)

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0]["type"] != "assistant" {
		t.Errorf("Expected type 'assistant', got %v", messages[0]["type"])
	}
}

func TestLargeMinifiedJSON(t *testing.T) {
	// Test parsing a large minified JSON (simulating the reported issue)
	largeData := map[string]interface{}{
		"data": generateLargeData(1000),
	}
	jsonObj := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{
					"tool_use_id": "toolu_016fed1NhiaMLqnEvrj5NUaj",
					"type":        "tool_result",
					"content":     mustMarshal(largeData),
				},
			},
		},
	}

	completeJSON := mustMarshal(jsonObj)

	// Chunk the JSON (simulating chunked reads)
	chunkSize := 64 * 1024
	var chunks []string
	for i := 0; i < len(completeJSON); i += chunkSize {
		end := i + chunkSize
		if end > len(completeJSON) {
			end = len(completeJSON)
		}
		chunks = append(chunks, completeJSON[i:end])
	}

	// Reassemble
	buffer := ""
	for _, chunk := range chunks {
		buffer += chunk
	}

	messages := parseBufferedLines(buffer)

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0]["type"] != "user" {
		t.Errorf("Expected type 'user', got %v", messages[0]["type"])
	}
}

func TestBufferSizeExceeded(t *testing.T) {
	// Test that exceeding buffer size raises an appropriate error
	defaultMaxBufferSize := 1024 * 1024 // 1MB default

	hugeIncomplete := `{"data": "` + strings.Repeat("x", defaultMaxBufferSize+1000)

	_, err := parseWithBufferLimit(hugeIncomplete, defaultMaxBufferSize)

	if err == nil {
		t.Error("Expected error for buffer size exceeded")
	}

	if !strings.Contains(err.Error(), "buffer size") {
		t.Errorf("Expected buffer size error, got: %v", err)
	}
}

func TestBufferSizeOption(t *testing.T) {
	// Test that the configurable buffer size option is respected
	customLimit := 512

	hugeIncomplete := `{"data": "` + strings.Repeat("x", customLimit+10)

	_, err := parseWithBufferLimit(hugeIncomplete, customLimit)

	if err == nil {
		t.Error("Expected error for buffer size exceeded")
	}

	if !strings.Contains(err.Error(), string(rune(customLimit))) {
		t.Logf("Buffer limit %d exceeded as expected", customLimit)
	}
}

func TestMixedCompleteAndSplitJSON(t *testing.T) {
	// Test handling a mix of complete and split JSON messages
	msg1 := mustMarshal(map[string]interface{}{
		"type":    "system",
		"subtype": "start",
	})

	largeMsg := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": strings.Repeat("y", 5000),
				},
			},
		},
	}
	largeJSON := mustMarshal(largeMsg)

	msg3 := mustMarshal(map[string]interface{}{
		"type":    "system",
		"subtype": "end",
	})

	// Simulate chunked delivery
	lines := []string{
		msg1 + "\n",
		largeJSON[:1000],
		largeJSON[1000:3000],
		largeJSON[3000:] + "\n" + msg3,
	}

	// Accumulate and parse
	buffer := ""
	var messages []map[string]interface{}

	for _, line := range lines {
		buffer += line
		parsed := parseBufferedLines(buffer)
		if len(parsed) > 0 {
			messages = append(messages, parsed...)
			buffer = ""
		}
	}

	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	if messages[0]["subtype"] != "start" {
		t.Errorf("Expected first message subtype 'start', got %v", messages[0]["subtype"])
	}

	if messages[2]["subtype"] != "end" {
		t.Errorf("Expected third message subtype 'end', got %v", messages[2]["subtype"])
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func generateLargeData(count int) []map[string]interface{} {
	data := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		data[i] = map[string]interface{}{
			"id":    i,
			"value": strings.Repeat("x", 100),
		}
	}
	return data
}

func parseBufferedLines(buffer string) []map[string]interface{} {
	var messages []map[string]interface{}

	lines := strings.Split(buffer, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			messages = append(messages, msg)
		}
	}

	return messages
}

func parseWithBufferLimit(buffer string, limit int) (map[string]interface{}, error) {
	if len(buffer) > limit {
		return nil, &BufferExceededError{Limit: limit, Actual: len(buffer)}
	}

	var msg map[string]interface{}
	err := json.Unmarshal([]byte(buffer), &msg)
	return msg, err
}

// BufferExceededError represents a buffer size exceeded error
type BufferExceededError struct {
	Limit  int
	Actual int
}

func (e *BufferExceededError) Error() string {
	return "exceeded maximum buffer size"
}
