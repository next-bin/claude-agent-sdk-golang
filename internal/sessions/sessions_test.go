package sessions

import (
	"strings"
	"testing"
)

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true},
		{"not-a-uuid", false},
		{"", false},
		{"550e8400-e29b-41d4-a716", false},
	}

	for _, tt := range tests {
		result := isValidUUID(tt.input)
		if result != tt.expected {
			t.Errorf("isValidUUID(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestSimpleHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string // We just verify it produces consistent output
	}{
		{"test", ""},
		{"hello world", ""},
		{"/home/user/project", ""},
	}

	for _, tt := range tests {
		result := simpleHash(tt.input)
		if result == "" {
			t.Errorf("simpleHash(%q) returned empty string", tt.input)
		}
		// Verify consistency - same input should produce same output
		result2 := simpleHash(tt.input)
		if result != result2 {
			t.Errorf("simpleHash(%q) inconsistent: %q != %q", tt.input, result, result2)
		}
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "with-spaces"},
		{"with/slashes", "with-slashes"},
		{"with:colons", "with-colons"},
	}

	for _, tt := range tests {
		result := sanitizePath(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractJSONStringField(t *testing.T) {
	tests := []struct {
		json     string
		key      string
		expected string
	}{
		{`{"name":"value"}`, "name", "value"},
		{`{"name": "value"}`, "name", "value"},
		{`{"other":"data","name":"value"}`, "name", "value"},
		{`{"name":"escaped\"value"}`, "name", "escaped\"value"},
		{`{"missing":"data"}`, "name", ""},
	}

	for _, tt := range tests {
		result := extractJSONStringField(tt.json, tt.key)
		if result != tt.expected {
			t.Errorf("extractJSONStringField(%q, %q) = %q, want %q", tt.json, tt.key, result, tt.expected)
		}
	}
}

func TestExtractLastJSONStringField(t *testing.T) {
	tests := []struct {
		json     string
		key      string
		expected string
	}{
		{`{"name":"first","name":"last"}`, "name", "last"},
		{`{"name": "first", "name": "last"}`, "name", "last"},
		{`{"missing":"data"}`, "name", ""},
	}

	for _, tt := range tests {
		result := extractLastJSONStringField(tt.json, tt.key)
		if result != tt.expected {
			t.Errorf("extractLastJSONStringField(%q, %q) = %q, want %q", tt.json, tt.key, result, tt.expected)
		}
	}
}

func TestUnescapeJSONString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"escaped\\nnewline", "escaped\nnewline"},
		{"escaped\\ttab", "escaped\ttab"},
		{"escaped\\\"quote", "escaped\"quote"},
	}

	for _, tt := range tests {
		result := unescapeJSONString(tt.input)
		if result != tt.expected {
			t.Errorf("unescapeJSONString(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractFirstPromptFromHead(t *testing.T) {
	tests := []struct {
		name     string
		head     string
		expected string
	}{
		{
			name:     "empty head",
			head:     "",
			expected: "",
		},
		{
			name: "user message with text",
			head: `{"type":"user","message":{"content":"Hello world"}}
`,
			expected: "Hello world",
		},
		{
			name: "skip tool_result",
			head: `{"type":"user","tool_result":"something","message":{"content":"ignored"}}
{"type":"user","message":{"content":"real prompt"}}
`,
			expected: "real prompt",
		},
		{
			name: "skip isMeta",
			head: `{"type":"user","isMeta":true,"message":{"content":"ignored"}}
{"type":"user","message":{"content":"real prompt"}}
`,
			expected: "real prompt",
		},
		{
			name: "truncate long prompt",
			head: `{"type":"user","message":{"content":"` + strings.Repeat("a", 300) + `"}}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirstPromptFromHead(tt.head)
			// For the truncate test, just check the suffix
			if tt.name == "truncate long prompt" {
				if !strings.HasSuffix(result, "…") {
					t.Errorf("expected truncated prompt with … suffix, got %q", result)
				}
				// 200 runes + "…" (1 rune) = 201 runes
				if len([]rune(result)) != 201 {
					t.Errorf("expected 201 runes, got %d", len([]rune(result)))
				}
				return
			}
			if result != tt.expected {
				t.Errorf("extractFirstPromptFromHead() = %q, want %q", result, tt.expected)
			}
		})
	}
}
