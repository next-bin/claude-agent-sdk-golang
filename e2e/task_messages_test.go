package e2e_tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Task Message E2E Tests - TaskStartedMessage, TaskProgressMessage, TaskNotificationMessage
// ============================================================================

// TestTaskMessages tests that task messages are properly parsed and delivered.
// Task messages (task_started, task_progress, task_notification) are emitted
// when Claude uses subagents or performs background tasks.
func TestTaskMessages(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 120*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(5),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	// Track all message types received
	var (
		taskStartedMsgs      []*types.TaskStartedMessage
		taskProgressMsgs     []*types.TaskProgressMessage
		taskNotificationMsgs []*types.TaskNotificationMessage
		resultMsg            *types.ResultMessage
	)

	// Make a query that might trigger subagent usage
	if err := client.Query(ctx, "List the files in the current directory using Bash tool"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	// Consume all messages
	for {
		select {
		case <-ctx.Done():
			t.Logf("Context done")
			goto done
		case msg, ok := <-msgChan:
			if !ok {
				t.Logf("Channel closed")
				goto done
			}

			switch m := msg.(type) {
			case *types.TaskStartedMessage:
				taskStartedMsgs = append(taskStartedMsgs, m)
				t.Logf("TaskStarted: ID=%s, Description=%s", m.TaskID, m.Description)
			case *types.TaskProgressMessage:
				taskProgressMsgs = append(taskProgressMsgs, m)
				t.Logf("TaskProgress: ID=%s, Usage={Tokens:%d, Tools:%d, Duration:%dms}",
					m.TaskID, m.Usage.TotalTokens, m.Usage.ToolUses, m.Usage.DurationMs)
			case *types.TaskNotificationMessage:
				taskNotificationMsgs = append(taskNotificationMsgs, m)
				t.Logf("TaskNotification: ID=%s, Status=%s, Summary=%s",
					m.TaskID, m.Status, m.Summary)
			case *types.ResultMessage:
				resultMsg = m
				t.Logf("Result: IsError=%v", m.IsError)
				// Continue draining in background
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-msgChan:
							if !ok {
								return
							}
						}
					}
				}()
				goto done
			default:
				// Other message types
			}
		}
	}
done:

	// Log summary
	t.Logf("\n=== Message Summary ===")
	t.Logf("TaskStartedMessages: %d", len(taskStartedMsgs))
	t.Logf("TaskProgressMessages: %d", len(taskProgressMsgs))
	t.Logf("TaskNotificationMessages: %d", len(taskNotificationMsgs))

	if resultMsg == nil {
		t.Error("Expected to receive a ResultMessage")
	}

	// Validate task messages if any were received
	for _, m := range taskStartedMsgs {
		if m.TaskID == "" {
			t.Error("TaskStartedMessage should have TaskID")
		}
		if m.Description == "" {
			t.Error("TaskStartedMessage should have Description")
		}
		if m.UUID == "" {
			t.Error("TaskStartedMessage should have UUID")
		}
		if m.SessionID == "" {
			t.Error("TaskStartedMessage should have SessionID")
		}
		// Verify it's also a SystemMessage
		if m.Subtype != "task_started" {
			t.Errorf("TaskStartedMessage.Subtype should be 'task_started', got %s", m.Subtype)
		}
	}

	for _, m := range taskProgressMsgs {
		if m.TaskID == "" {
			t.Error("TaskProgressMessage should have TaskID")
		}
		// Usage should be populated
		if m.Usage.TotalTokens < 0 {
			t.Error("TaskProgressMessage.Usage.TotalTokens should be >= 0")
		}
	}

	for _, m := range taskNotificationMsgs {
		if m.TaskID == "" {
			t.Error("TaskNotificationMessage should have TaskID")
		}
		// Status should be one of the valid values
		validStatus := m.Status == types.TaskNotificationStatusCompleted ||
			m.Status == types.TaskNotificationStatusFailed ||
			m.Status == types.TaskNotificationStatusStopped
		if !validStatus {
			t.Errorf("TaskNotificationMessage.Status should be valid, got %s", m.Status)
		}
	}
}

// TestTaskMessageFields tests specific fields of task messages.
func TestTaskMessageFields(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 90*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(3),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "What is 1+1?"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var receivedMessages []types.Message
	for {
		select {
		case <-ctx.Done():
			goto done
		case msg, ok := <-msgChan:
			if !ok {
				goto done
			}
			receivedMessages = append(receivedMessages, msg)
			if _, isResult := msg.(*types.ResultMessage); isResult {
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-msgChan:
							if !ok {
								return
							}
						}
					}
				}()
				goto done
			}
		}
	}
done:

	// Verify that task messages, if present, have correct field types
	for _, msg := range receivedMessages {
		switch m := msg.(type) {
		case *types.TaskStartedMessage:
			// Check inherited SystemMessage fields
			if m.Subtype != "task_started" {
				t.Errorf("TaskStartedMessage.Subtype = %s, want task_started", m.Subtype)
			}
			if m.Data == nil {
				t.Error("TaskStartedMessage.Data should not be nil")
			}
		case *types.TaskProgressMessage:
			if m.Subtype != "task_progress" {
				t.Errorf("TaskProgressMessage.Subtype = %s, want task_progress", m.Subtype)
			}
		case *types.TaskNotificationMessage:
			if m.Subtype != "task_notification" {
				t.Errorf("TaskNotificationMessage.Subtype = %s, want task_notification", m.Subtype)
			}
		}
	}
}

// TestResultMessageStopReason tests that ResultMessage contains StopReason field.
func TestResultMessageStopReason(t *testing.T) {
	SkipIfNoAPIKey(t)

	bgCtx := context.Background()
	ctx, cancel := context.WithTimeout(bgCtx, 60*time.Second)
	defer cancel()

	mode := types.PermissionModeBypassPermissions
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:          types.String(DefaultTestConfig().Model),
		PermissionMode: &mode,
		MaxTurns:       types.Int(2),
	})
	defer client.Close()

	if err := client.Connect(bgCtx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create message channel once and reuse for all queries
	msgChan := client.ReceiveMessages(bgCtx)

	if err := client.Query(ctx, "Say hello"); err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	var resultMsg *types.ResultMessage
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Context done without receiving result")
		case msg, ok := <-msgChan:
			if !ok {
				t.Fatal("Channel closed without receiving result")
			}
			if m, isResult := msg.(*types.ResultMessage); isResult {
				resultMsg = m
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-msgChan:
							if !ok {
								return
							}
						}
					}
				}()
				goto done
			}
		}
	}
done:

	if resultMsg == nil {
		t.Fatal("Expected to receive a ResultMessage")
	}

	// Log the result message details
	t.Logf("ResultMessage.Subtype: %s", resultMsg.Subtype)
	t.Logf("ResultMessage.SessionID: %s", resultMsg.SessionID)
	t.Logf("ResultMessage.NumTurns: %d", resultMsg.NumTurns)
	t.Logf("ResultMessage.IsError: %v", resultMsg.IsError)

	// StopReason may be nil or have a value
	if resultMsg.StopReason != nil {
		t.Logf("ResultMessage.StopReason: %s", *resultMsg.StopReason)
	} else {
		t.Log("ResultMessage.StopReason: nil")
	}

	// Verify required fields are present
	if resultMsg.SessionID == "" {
		t.Error("ResultMessage should have SessionID")
	}
	if resultMsg.Subtype == "" {
		t.Error("ResultMessage should have Subtype")
	}
}

// TestTaskNotificationStatusValues tests the TaskNotificationStatus constants.
func TestTaskNotificationStatusValues(t *testing.T) {
	// Test that the constants have correct values
	if types.TaskNotificationStatusCompleted != "completed" {
		t.Errorf("TaskNotificationStatusCompleted = %s, want completed",
			types.TaskNotificationStatusCompleted)
	}
	if types.TaskNotificationStatusFailed != "failed" {
		t.Errorf("TaskNotificationStatusFailed = %s, want failed",
			types.TaskNotificationStatusFailed)
	}
	if types.TaskNotificationStatusStopped != "stopped" {
		t.Errorf("TaskNotificationStatusStopped = %s, want stopped",
			types.TaskNotificationStatusStopped)
	}
	t.Log("TaskNotificationStatus constants verified")
}
