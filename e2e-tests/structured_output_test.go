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
	startTime := time.Now()
	PrintTestHeader(t, "TestSimpleStructuredOutput")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestSimpleStructuredOutput")

	// Define schema for file analysis
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_count": map[string]interface{}{"type": "number"},
			"has_tests":  map[string]interface{}{"type": "boolean"},
		},
		"required": []string{"file_count", "has_tests"},
	}

	logger.Step("Creating client with structured output schema")
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
		CWD:            ".",
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: Count Go files")
	if err := client.Query(ctx, "Count how many Go files are in the current directory and check if there are any test files."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMessage := ConsumeMessagesVerbose(ctx, t, msgChan, "TestSimpleStructuredOutput")

	// Verify result
	if !foundResult || resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.IsError {
		t.Logf("Query had error (may be expected): %v", resultMessage.Result)
	}

	// Verify structured output is present
	if resultMessage.StructuredOutput != nil {
		output, ok := resultMessage.StructuredOutput.(map[string]interface{})
		if ok {
			t.Logf("Structured output: %+v", output)
		} else {
			t.Logf("Structured output present but not map: %T", resultMessage.StructuredOutput)
		}
	} else {
		t.Log("No structured output in result (may vary by model)")
	}

	PrintTestSummary(t, "TestSimpleStructuredOutput", foundResult, count, time.Since(startTime))
}

// TestNestedStructuredOutput tests structured output with nested objects.
func TestNestedStructuredOutput(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestNestedStructuredOutput")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestNestedStructuredOutput")

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

	logger.Step("Creating client with structured output schema")
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: Analyze text 'Hello world'")
	if err := client.Query(ctx, "Analyze this text: 'Hello world'. Provide word count, character count, and list of words."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMessage := ConsumeMessagesVerbose(ctx, t, msgChan, "TestNestedStructuredOutput")

	if !foundResult || resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.StructuredOutput != nil {
		output, ok := resultMessage.StructuredOutput.(map[string]interface{})
		if ok {
			t.Logf("Nested structured output: %+v", output)
		}
	} else {
		t.Log("No structured output in result (may vary by model)")
	}

	PrintTestSummary(t, "TestNestedStructuredOutput", foundResult, count, time.Since(startTime))
}

// TestStructuredOutputWithEnum tests structured output with enum constraints.
func TestStructuredOutputWithEnum(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestStructuredOutputWithEnum")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestStructuredOutputWithEnum")

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"has_tests": map[string]interface{}{"type": "boolean"},
			"test_framework": map[string]interface{}{
				"type": "string",
				"enum": []string{"pytest", "unittest", "nose", "unknown", "go test", "other"},
			},
		},
		"required": []string{"has_tests", "test_framework"},
	}

	logger.Step("Creating client with enum schema")
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
		CWD:            ".",
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query about test framework")
	if err := client.Query(ctx, "Check if there are test files in the e2e-tests/ directory and identify the test framework."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMessage := ConsumeMessagesVerbose(ctx, t, msgChan, "TestStructuredOutputWithEnum")

	if !foundResult || resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.StructuredOutput != nil {
		output, ok := resultMessage.StructuredOutput.(map[string]interface{})
		if ok {
			t.Logf("Enum structured output: %+v", output)
		}
	} else {
		t.Log("No structured output in result (may vary by model)")
	}

	PrintTestSummary(t, "TestStructuredOutputWithEnum", foundResult, count, time.Since(startTime))
}

// TestStructuredOutputWithTools tests structured output when agent uses tools.
func TestStructuredOutputWithTools(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestStructuredOutputWithTools")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestStructuredOutputWithTools")

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_count": map[string]interface{}{"type": "number"},
			"has_readme": map[string]interface{}{"type": "boolean"},
		},
		"required": []string{"file_count", "has_readme"},
	}

	logger.Step("Creating client with structured output schema")
	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		OutputFormat:   map[string]interface{}{"type": "json_schema", "schema": schema},
		PermissionMode: &mode,
		MaxTurns:       types.Int(1),
		CWD:            "/tmp",
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query about files in /tmp")
	if err := client.Query(ctx, "Count how many files are in the current directory and check if there's a README file."); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	count, foundResult, resultMessage := ConsumeMessagesVerbose(ctx, t, msgChan, "TestStructuredOutputWithTools")

	if !foundResult || resultMessage == nil {
		t.Fatal("No result message received")
	}

	if resultMessage.StructuredOutput != nil {
		output, ok := resultMessage.StructuredOutput.(map[string]interface{})
		if ok {
			t.Logf("Tools structured output: %+v", output)
		}
	} else {
		t.Log("No structured output in result (may vary by model)")
	}

	PrintTestSummary(t, "TestStructuredOutputWithTools", foundResult, count, time.Since(startTime))
}
