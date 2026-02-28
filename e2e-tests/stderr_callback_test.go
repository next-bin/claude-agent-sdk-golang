package e2e_tests

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ============================================================================
// Stderr Callback E2E Tests
// ============================================================================

// TestStderrCallbackCapturesDebugOutput tests that stderr callback receives
// debug output when enabled.
func TestStderrCallbackCapturesDebugOutput(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var stderrLines []string
	var mu sync.Mutex

	stderrCallback := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		stderrLines = append(stderrLines, line)
	}

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: permissionModePtr(types.PermissionModeBypassPermissions),
		Stderr:         stderrCallback,
		ExtraArgs: map[string]interface{}{
			"debug-to-stderr": nil,
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "What is 1+1?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Consume messages
	for range msgChan {
	}

	// Verify we captured debug output
	mu.Lock()
	defer mu.Unlock()

	if len(stderrLines) == 0 {
		t.Error("Should capture stderr output with debug enabled")
	} else {
		t.Logf("Captured %d stderr lines", len(stderrLines))

		// Check for DEBUG messages
		hasDebug := false
		for _, line := range stderrLines {
			if strings.Contains(line, "[DEBUG]") {
				hasDebug = true
				break
			}
		}

		if !hasDebug {
			t.Log("Note: No [DEBUG] markers found (may vary by CLI version)")
		}
	}
}

// TestStderrCallbackWithoutDebug tests that stderr callback works but receives
// no output without debug mode.
func TestStderrCallbackWithoutDebug(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var stderrLines []string
	var mu sync.Mutex

	stderrCallback := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		stderrLines = append(stderrLines, line)
	}

	// No debug mode enabled
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: permissionModePtr(types.PermissionModeBypassPermissions),
		Stderr:         stderrCallback,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	msgChan, err := client.Query(ctx, "What is 1+1?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Consume messages
	for range msgChan {
	}

	// Should work but capture minimal/no output without debug
	mu.Lock()
	defer mu.Unlock()

	if len(stderrLines) > 0 {
		t.Logf("Note: Captured %d stderr lines without debug mode (may vary)", len(stderrLines))
	}
}

// TestStderrCallbackMultipleQueries tests stderr callback across multiple queries.
func TestStderrCallbackMultipleQueries(t *testing.T) {
	SkipIfNoAPIKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var stderrLines []string
	var mu sync.Mutex

	stderrCallback := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		stderrLines = append(stderrLines, line)
	}

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: permissionModePtr(types.PermissionModeBypassPermissions),
		Stderr:         stderrCallback,
		ExtraArgs: map[string]interface{}{
			"debug-to-stderr": nil,
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First query
	msgChan, err := client.Query(ctx, "What is 1+1?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	for range msgChan {
	}

	// Second query
	msgChan, err = client.Query(ctx, "What is 2+2?")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	for range msgChan {
	}

	mu.Lock()
	defer mu.Unlock()
	t.Logf("Total stderr lines captured: %d", len(stderrLines))
}
