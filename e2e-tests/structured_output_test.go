package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Structured Output E2E Tests
// ============================================================================

// TestSimpleStructuredOutput tests structured output with file counting.
func TestSimpleStructuredOutput(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Define schema for file analysis
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_count": map[string]interface{}{"type": "number"},
			"has_tests":  map[string]interface{}{"type": "boolean"},
		},
		"required": []string{"file_count", "has_tests"},
	}

	mode := types.PermissionModeAcceptEdits
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
		CWD:            ".",
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Count how many Go files are in the current directory and check if there are any test files. Use tools to explore the filesystem.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var resultMessage *types.ResultMessage

	for msg := range msgChan {
		if m, ok := msg.(*types.ResultMessage); ok {
			resultMessage = m
		}
	}

	// Verify result
	if resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.IsError {
		t.Fatalf("Query failed: %v", resultMessage.Result)
	}

	// Verify structured output is present
	if resultMessage.StructuredOutput == nil {
		t.Fatal("No structured output in result")
	}

	output, ok := resultMessage.StructuredOutput.(map[string]interface{})
	if !ok {
		t.Fatalf("Structured output is not a map: %T", resultMessage.StructuredOutput)
	}

	// Check required fields
	if _, ok := output["file_count"]; !ok {
		t.Error("Missing required field: file_count")
	}
	if _, ok := output["has_tests"]; !ok {
		t.Error("Missing required field: has_tests")
	}

	t.Logf("Structured output: %+v", output)
}

// TestNestedStructuredOutput tests structured output with nested objects.
func TestNestedStructuredOutput(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Define a schema with nested structure
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"analysis": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"word_count":      map[string]interface{}{"type": "number"},
					"character_count": map[string]interface{}{"type": "number"},
				},
				"required": []string{"word_count", "character_count"},
			},
			"words": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
		"required": []string{"analysis", "words"},
	}

	mode := types.PermissionModeAcceptEdits
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Analyze this text: 'Hello world'. Provide word count, character count, and list of words.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var resultMessage *types.ResultMessage

	for msg := range msgChan {
		if m, ok := msg.(*types.ResultMessage); ok {
			resultMessage = m
		}
	}

	if resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.IsError {
		t.Fatalf("Query failed: %v", resultMessage.Result)
	}

	if resultMessage.StructuredOutput == nil {
		t.Fatal("No structured output in result")
	}

	output, ok := resultMessage.StructuredOutput.(map[string]interface{})
	if !ok {
		t.Fatalf("Structured output is not a map: %T", resultMessage.StructuredOutput)
	}

	// Check nested structure
	if _, ok := output["analysis"]; !ok {
		t.Error("Missing 'analysis' field")
	}
	if _, ok := output["words"]; !ok {
		t.Error("Missing 'words' field")
	}

	t.Logf("Nested structured output: %+v", output)
}

// TestStructuredOutputWithEnum tests structured output with enum constraints.
func TestStructuredOutputWithEnum(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"has_tests": map[string]interface{}{"type": "boolean"},
			"test_framework": map[string]interface{}{
				"type": "string",
				"enum": []string{"pytest", "unittest", "nose", "unknown"},
			},
		},
		"required": []string{"has_tests", "test_framework"},
	}

	mode := types.PermissionModeAcceptEdits
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
		CWD:            ".",
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Search for test files in the e2e-tests/ directory. Determine which test framework is being used (go test).")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var resultMessage *types.ResultMessage

	for msg := range msgChan {
		if m, ok := msg.(*types.ResultMessage); ok {
			resultMessage = m
		}
	}

	if resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.StructuredOutput == nil {
		t.Fatal("No structured output in result")
	}

	output, ok := resultMessage.StructuredOutput.(map[string]interface{})
	if !ok {
		t.Fatalf("Structured output is not a map: %T", resultMessage.StructuredOutput)
	}

	// Check enum values are valid
	if framework, ok := output["test_framework"].(string); ok {
		validFrameworks := map[string]bool{"pytest": true, "unittest": true, "nose": true, "unknown": true}
		if !validFrameworks[framework] {
			t.Errorf("Invalid test_framework value: %s", framework)
		}
	}

	if hasTests, ok := output["has_tests"].(bool); ok {
		t.Logf("has_tests: %v", hasTests)
	}

	t.Logf("Enum structured output: %+v", output)
}

// TestStructuredOutputWithTools tests structured output when agent uses tools.
func TestStructuredOutputWithTools(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_count": map[string]interface{}{"type": "number"},
			"has_readme": map[string]interface{}{"type": "boolean"},
		},
		"required": []string{"file_count", "has_readme"},
	}

	mode := types.PermissionModeAcceptEdits
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
		CWD:            "/tmp",
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "Count how many files are in the current directory and check if there's a README file. Use tools as needed.")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var resultMessage *types.ResultMessage

	for msg := range msgChan {
		if m, ok := msg.(*types.ResultMessage); ok {
			resultMessage = m
		}
	}

	if resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.StructuredOutput == nil {
		t.Fatal("No structured output in result")
	}

	output, ok := resultMessage.StructuredOutput.(map[string]interface{})
	if !ok {
		t.Fatalf("Structured output is not a map: %T", resultMessage.StructuredOutput)
	}

	// Check structure
	if _, ok := output["file_count"]; !ok {
		t.Error("Missing 'file_count' field")
	}
	if _, ok := output["has_readme"]; !ok {
		t.Error("Missing 'has_readme' field")
	}

	t.Logf("Tools structured output: %+v", output)
}
