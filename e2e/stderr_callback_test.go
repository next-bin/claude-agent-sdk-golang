package e2e_tests

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Stderr Callback E2E Tests
// ============================================================================

// TestStderrCallbackCapturesDebugOutput tests that stderr callback receives
// debug output when enabled.
func TestStderrCallbackCapturesDebugOutput(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestStderrCallbackCapturesDebugOutput")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestStderrCallbackCapturesDebugOutput")

	var stderrLines []string
	var mu sync.Mutex

	stderrCallback := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		stderrLines = append(stderrLines, line)
	}

	logger.Step("Creating client with stderr callback and debug mode")
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: permissionModePtr(types.PermissionModeBypassPermissions),
		Stderr:         stderrCallback,
		ExtraArgs: map[string]interface{}{
			"debug-to-stderr": nil,
		},
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: What is 1+1?")
	if err := client.Query(ctx, "What is 1+1?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Consume messages with verbose output
	count, foundResult, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestStderrCallbackCapturesDebugOutput")

	// Verify we captured debug output
	mu.Lock()
	stderrCount := len(stderrLines)
	mu.Unlock()

	if stderrCount == 0 {
		t.Error("Should capture stderr output with debug enabled")
	} else {
		t.Logf("Captured %d stderr lines", stderrCount)

		// Check for DEBUG messages
		mu.Lock()
		hasDebug := false
		for _, line := range stderrLines {
			if strings.Contains(line, "[DEBUG]") {
				hasDebug = true
				break
			}
		}
		mu.Unlock()

		if !hasDebug {
			t.Log("Note: No [DEBUG] markers found (may vary by CLI version)")
		}
	}

	PrintTestSummary(t, "TestStderrCallbackCapturesDebugOutput", foundResult, count, time.Since(startTime))
}

// TestStderrCallbackWithoutDebug tests that stderr callback works but receives
// no output without debug mode.
func TestStderrCallbackWithoutDebug(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestStderrCallbackWithoutDebug")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestStderrCallbackWithoutDebug")

	var stderrLines []string
	var mu sync.Mutex

	stderrCallback := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		stderrLines = append(stderrLines, line)
	}

	logger.Step("Creating client with stderr callback (no debug mode)")
	// No debug mode enabled
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: permissionModePtr(types.PermissionModeBypassPermissions),
		Stderr:         stderrCallback,
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	logger.Step("Sending query: What is 1+1?")
	if err := client.Query(ctx, "What is 1+1?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Consume messages with verbose output
	count, foundResult, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "TestStderrCallbackWithoutDebug")

	// Should work but capture minimal/no output without debug
	mu.Lock()
	stderrCount := len(stderrLines)
	mu.Unlock()

	if stderrCount > 0 {
		t.Logf("Note: Captured %d stderr lines without debug mode (may vary)", stderrCount)
	}

	PrintTestSummary(t, "TestStderrCallbackWithoutDebug", foundResult, count, time.Since(startTime))
}

// TestStderrCallbackMultipleQueries tests stderr callback across multiple queries.
func TestStderrCallbackMultipleQueries(t *testing.T) {
	SkipIfNoAPIKey(t)
	startTime := time.Now()
	PrintTestHeader(t, "TestStderrCallbackMultipleQueries")

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel()

	logger := NewTestLogger(t, "TestStderrCallbackMultipleQueries")

	var stderrLines []string
	var mu sync.Mutex

	stderrCallback := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		stderrLines = append(stderrLines, line)
	}

	logger.Step("Creating client with stderr callback")
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: permissionModePtr(types.PermissionModeBypassPermissions),
		Stderr:         stderrCallback,
		ExtraArgs: map[string]interface{}{
			"debug-to-stderr": nil,
		},
	})
	defer client.Close()

	logger.Step("Connecting")
	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	logger.Status("Connected successfully")

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// First query
	logger.Step("Sending first query: What is 1+1?")
	if err := client.Query(ctx, "What is 1+1?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	count1, foundResult1, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "Query1")
	logger.Log("First query completed: %d messages, foundResult=%v", count1, foundResult1)

	// Second query - need to ensure first query is fully complete
	logger.Step("Sending second query: What is 2+2?")
	if err := client.Query(ctx, "What is 2+2?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	count2, foundResult2, _ := ConsumeMessagesVerbose(ctx, t, msgChan, "Query2")
	logger.Log("Second query completed: %d messages, foundResult=%v", count2, foundResult2)

	mu.Lock()
	totalLines := len(stderrLines)
	mu.Unlock()

	PrintTestSummary(t, "TestStderrCallbackMultipleQueries", foundResult1 && foundResult2, count1+count2, time.Since(startTime))
	t.Logf("Total stderr lines captured: %d", totalLines)
}
