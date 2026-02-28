// Package types_test contains tests for model constants.
package types_test

import (
	"testing"

	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func TestModelConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		// Short names
		{"ModelOpus", types.ModelOpus, "opus"},
		{"ModelSonnet", types.ModelSonnet, "sonnet"},
		{"ModelHaiku", types.ModelHaiku, "haiku"},
		// Concrete names
		{"ModelClaudeOpus", types.ModelClaudeOpus, "claude-opus-4-6"},
		{"ModelClaudeSonnet", types.ModelClaudeSonnet, "claude-sonnet-4-6"},
		{"ModelClaudeHaiku", types.ModelClaudeHaiku, "claude-haiku-4-5-20251001"},
		// Agent value
		{"ModelInherit", types.ModelInherit, "inherit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestModelConstantsNotEmpty(t *testing.T) {
	models := []struct {
		name  string
		value string
	}{
		{"ModelOpus", types.ModelOpus},
		{"ModelSonnet", types.ModelSonnet},
		{"ModelHaiku", types.ModelHaiku},
		{"ModelClaudeOpus", types.ModelClaudeOpus},
		{"ModelClaudeSonnet", types.ModelClaudeSonnet},
		{"ModelClaudeHaiku", types.ModelClaudeHaiku},
		{"ModelInherit", types.ModelInherit},
	}

	for _, m := range models {
		if m.value == "" {
			t.Errorf("%s should not be empty", m.name)
		}
	}
}
