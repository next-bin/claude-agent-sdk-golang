package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// ============================================================================
// Test Fixtures
// ============================================================================

// setupTestConfig creates a temporary CLAUDE_CONFIG_DIR for testing.
func setupTestConfig(t *testing.T) string {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(configDir, "projects")

	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("Failed to create projects dir: %v", err)
	}

	// Set environment variable
	originalDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", configDir)

	t.Cleanup(func() {
		if originalDir != "" {
			os.Setenv("CLAUDE_CONFIG_DIR", originalDir)
		} else {
			os.Unsetenv("CLAUDE_CONFIG_DIR")
		}
	})

	return configDir
}

// makeProjectDir creates a sanitized project directory.
func makeProjectDir(configDir, projectPath string) string {
	sanitized := sanitizePath(projectPath)
	projectDir := filepath.Join(configDir, "projects", sanitized)
	os.MkdirAll(projectDir, 0755)
	return projectDir
}

// makeSessionFile creates a test session file.
func makeSessionFile(projectDir, sessionID string, firstPrompt string) string {
	if sessionID == "" {
		sessionID = "550e8400-e29b-41d4-a716-446655440000"
	}

	filePath := filepath.Join(projectDir, sessionID+".jsonl")

	lines := []string{
		fmt.Sprintf(`{"type":"user","message":{"role":"user","content":"%s"},"uuid":"user-uuid-1","sessionId":"%s","timestamp":"2024-01-15T10:30:00Z"}`, firstPrompt, sessionID),
		fmt.Sprintf(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hi!"}]},"uuid":"asst-uuid-1","sessionId":"%s","timestamp":"2024-01-15T10:30:01Z"}`, sessionID),
	}

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return ""
	}

	return filePath
}

// makeTranscriptSession creates a session file with uuid/parentUuid chain.
// Uses proper UUID format for valid UUID validation.
func makeTranscriptSession(projectDir, sessionID string, numTurns int) (string, []string) {
	if sessionID == "" {
		sessionID = "550e8400-e29b-41d4-a716-446655440000"
	}

	filePath := filepath.Join(projectDir, sessionID+".jsonl")
	var uuids []string
	var lines []string
	var parentUUID string

	for i := 0; i < numTurns; i++ {
		// User message - use valid UUID format
		userUUID := fmt.Sprintf("10000000-0000-0000-0000-0000000000%02d", i+1)
		uuids = append(uuids, userUUID)
		userEntry := map[string]interface{}{
			"type":       "user",
			"uuid":       userUUID,
			"parentUuid": parentUUID,
			"sessionId":  sessionID,
			"timestamp":  "2024-01-15T10:30:00Z",
			"message": map[string]interface{}{
				"role":    "user",
				"content": fmt.Sprintf("Turn %d question", i+1),
			},
		}
		lines = append(lines, string(mustMarshal(userEntry)))
		parentUUID = userUUID

		// Assistant message - use valid UUID format
		asstUUID := fmt.Sprintf("20000000-0000-0000-0000-0000000000%02d", i+1)
		uuids = append(uuids, asstUUID)
		asstEntry := map[string]interface{}{
			"type":       "assistant",
			"uuid":       asstUUID,
			"parentUuid": parentUUID,
			"sessionId":  sessionID,
			"timestamp":  "2024-01-15T10:30:01Z",
			"message": map[string]interface{}{
				"role": "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Turn %d answer", i+1),
					},
				},
			},
		}
		lines = append(lines, string(mustMarshal(asstEntry)))
		parentUUID = asstUUID
	}

	content := strings.Join(lines, "\n") + "\n"
	os.WriteFile(filePath, []byte(content), 0644)

	return sessionID, uuids
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// ============================================================================
// ListSessions Tests
// ============================================================================

func TestListSessions_EmptyDirectory(t *testing.T) {
	setupTestConfig(t)

	sessions, err := ListSessions("/nonexistent/path", 10, false)
	if err != nil {
		t.Errorf("ListSessions should not error for empty directory: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}

func TestListSessions_SingleSession(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	makeSessionFile(projectDir, "550e8400-e29b-41d4-a716-446655440000", "Hello Claude")

	sessions, err := ListSessions(projectPath, 10, false)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	if len(sessions) > 0 {
		if sessions[0].SessionID != "550e8400-e29b-41d4-a716-446655440000" {
			t.Errorf("Expected session ID 550e8400-e29b-41d4-a716-446655440000, got %s", sessions[0].SessionID)
		}
		if sessions[0].Summary != "Hello Claude" {
			t.Errorf("Expected summary 'Hello Claude', got %s", sessions[0].Summary)
		}
	}
}

func TestListSessions_WithLimit(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		sessionID := fmt.Sprintf("550e8400-e29b-41d4-a716-44665544%04d", i)
		makeSessionFile(projectDir, sessionID, fmt.Sprintf("Prompt %d", i))
	}

	sessions, err := ListSessions(projectPath, 2, false)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions with limit, got %d", len(sessions))
	}
}

func TestListSessions_SkipSidechain(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)

	// Create a sidechain session
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	filePath := filepath.Join(projectDir, sessionID+".jsonl")
	content := `{"type":"user","isSidechain":true,"message":{"content":"sidechain"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"response"}]}}
`
	os.WriteFile(filePath, []byte(content), 0644)

	sessions, err := ListSessions(projectPath, 10, false)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions (sidechain skipped), got %d", len(sessions))
	}
}

// ============================================================================
// GetSessionInfo Tests
// ============================================================================

func TestGetSessionInfo_ExistingSession(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	makeSessionFile(projectDir, sessionID, "Test prompt")

	info := GetSessionInfo(sessionID, projectPath)
	if info == nil {
		t.Fatal("Expected session info, got nil")
	}

	if info.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, info.SessionID)
	}

	if info.Summary != "Test prompt" {
		t.Errorf("Expected summary 'Test prompt', got %s", info.Summary)
	}
}

// ============================================================================
// GetSessionMessages Tests
// ============================================================================

func TestGetSessionMessages_ExistingSession(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, _ := makeTranscriptSession(projectDir, "", 2)

	messages, err := GetSessionMessages(sessionID, projectPath, 0, 0)
	if err != nil {
		t.Fatalf("GetSessionMessages failed: %v", err)
	}

	if len(messages) != 4 {
		t.Errorf("Expected 4 messages (2 user + 2 assistant), got %d", len(messages))
	}
}

func TestGetSessionMessages_WithLimit(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, _ := makeTranscriptSession(projectDir, "", 2)

	messages, err := GetSessionMessages(sessionID, projectPath, 2, 0)
	if err != nil {
		t.Fatalf("GetSessionMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages with limit, got %d", len(messages))
	}
}

func TestGetSessionMessages_WithOffset(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, _ := makeTranscriptSession(projectDir, "", 2)

	messages, err := GetSessionMessages(sessionID, projectPath, 0, 2)
	if err != nil {
		t.Fatalf("GetSessionMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("Expected 2 messages after offset, got %d", len(messages))
	}
}

// ============================================================================
// RenameSession Tests
// ============================================================================

func TestRenameSession_InvalidUUID(t *testing.T) {
	setupTestConfig(t)

	err := RenameSession("not-a-uuid", "New Title", "/test/project")
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid session_id") {
		t.Errorf("Expected 'invalid session_id' error, got: %v", err)
	}
}

func TestRenameSession_EmptyTitle(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	makeSessionFile(projectDir, sessionID, "Test")

	err := RenameSession(sessionID, "", projectPath)
	if err == nil {
		t.Error("Expected error for empty title")
	}
	if err != nil && !strings.Contains(err.Error(), "title must be non-empty") {
		t.Errorf("Expected 'title must be non-empty' error, got: %v", err)
	}
}

func TestRenameSession_Success(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	filePath := makeSessionFile(projectDir, sessionID, "Test")

	err := RenameSession(sessionID, "My New Title", projectPath)
	if err != nil {
		t.Fatalf("RenameSession failed: %v", err)
	}

	// Verify the title entry was appended
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	lastLine := lines[len(lines)-1]

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(lastLine), &entry); err != nil {
		t.Fatalf("Failed to parse last line: %v", err)
	}

	if entry["type"] != "custom-title" {
		t.Errorf("Expected type 'custom-title', got %v", entry["type"])
	}
	if entry["customTitle"] != "My New Title" {
		t.Errorf("Expected customTitle 'My New Title', got %v", entry["customTitle"])
	}
}

func TestRenameSession_TitleTrimmed(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	filePath := makeSessionFile(projectDir, sessionID, "Test")

	err := RenameSession(sessionID, "  Trimmed Title  ", projectPath)
	if err != nil {
		t.Fatalf("RenameSession failed: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry map[string]interface{}
	json.Unmarshal([]byte(lines[len(lines)-1]), &entry)

	if entry["customTitle"] != "Trimmed Title" {
		t.Errorf("Expected trimmed title, got %v", entry["customTitle"])
	}
}

func TestRenameSession_MultipleRenames(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	makeSessionFile(projectDir, sessionID, "Test")

	RenameSession(sessionID, "First Title", projectPath)
	RenameSession(sessionID, "Second Title", projectPath)
	RenameSession(sessionID, "Final Title", projectPath)

	// Verify last title wins
	sessions, _ := ListSessions(projectPath, 10, false)
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].CustomTitle == nil || *sessions[0].CustomTitle != "Final Title" {
		t.Errorf("Expected last title 'Final Title', got %v", sessions[0].CustomTitle)
	}
}

// ============================================================================
// TagSession Tests
// ============================================================================

func TestTagSession_InvalidUUID(t *testing.T) {
	setupTestConfig(t)

	err := TagSession("not-a-uuid", "tag", "/test/project")
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
}

func TestTagSession_EmptyTag(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	makeSessionFile(projectDir, sessionID, "Test")

	err := TagSession(sessionID, "", projectPath)
	// Empty tag should succeed (clears the tag)
	if err != nil {
		t.Errorf("Empty tag should succeed (clears tag), got error: %v", err)
	}
}

func TestTagSession_Success(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	filePath := makeSessionFile(projectDir, sessionID, "Test")

	err := TagSession(sessionID, "experiment", projectPath)
	if err != nil {
		t.Fatalf("TagSession failed: %v", err)
	}

	// Verify the tag entry was appended
	content, _ := os.ReadFile(filePath)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry map[string]interface{}
	json.Unmarshal([]byte(lines[len(lines)-1]), &entry)

	if entry["type"] != "tag" {
		t.Errorf("Expected type 'tag', got %v", entry["type"])
	}
	if entry["tag"] != "experiment" {
		t.Errorf("Expected tag 'experiment', got %v", entry["tag"])
	}
}

func TestTagSession_TagTrimmed(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	filePath := makeSessionFile(projectDir, sessionID, "Test")

	TagSession(sessionID, "  my-tag  ", projectPath)

	content, _ := os.ReadFile(filePath)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	var entry map[string]interface{}
	json.Unmarshal([]byte(lines[len(lines)-1]), &entry)

	if entry["tag"] != "my-tag" {
		t.Errorf("Expected trimmed tag, got %v", entry["tag"])
	}
}

func TestTagSession_ClearTag(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	makeSessionFile(projectDir, sessionID, "Test")

	// Set a tag
	TagSession(sessionID, "original-tag", projectPath)

	// Clear it
	TagSession(sessionID, "", projectPath)

	sessions, _ := ListSessions(projectPath, 10, false)
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	// Tag should be empty string or nil
	if sessions[0].Tag != nil && *sessions[0].Tag != "" {
		t.Errorf("Expected cleared tag, got %v", sessions[0].Tag)
	}
}

func TestTagSession_MultipleTags(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	makeSessionFile(projectDir, sessionID, "Test")

	TagSession(sessionID, "first", projectPath)
	TagSession(sessionID, "second", projectPath)
	TagSession(sessionID, "third", projectPath)

	sessions, _ := ListSessions(projectPath, 10, false)
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].Tag == nil || *sessions[0].Tag != "third" {
		t.Errorf("Expected last tag 'third', got %v", sessions[0].Tag)
	}
}

// ============================================================================
// DeleteSession Tests
// ============================================================================

func TestDeleteSession_InvalidUUID(t *testing.T) {
	setupTestConfig(t)

	err := DeleteSession("not-a-uuid", "/test/project")
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
}

func TestDeleteSession_NotFound(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	makeProjectDir(configDir, projectPath)

	err := DeleteSession("550e8400-e29b-41d4-a716-446655440000", projectPath)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestDeleteSession_Success(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	filePath := makeSessionFile(projectDir, sessionID, "Test")

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Session file should exist before delete")
	}

	err := DeleteSession(sessionID, projectPath)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Session file should be deleted")
	}
}

func TestDeleteSession_NoLongerInList(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	makeSessionFile(projectDir, sessionID, "Test")

	// Verify session is in list
	sessions, _ := ListSessions(projectPath, 10, false)
	if len(sessions) != 1 {
		t.Fatal("Expected 1 session before delete")
	}

	DeleteSession(sessionID, projectPath)

	// Verify session is no longer in list
	sessions, _ = ListSessions(projectPath, 10, false)
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions after delete, got %d", len(sessions))
	}
}

// ============================================================================
// ForkSession Tests
// ============================================================================

func TestForkSession_InvalidUUID(t *testing.T) {
	setupTestConfig(t)

	result, err := ForkSession("not-a-uuid", "/test/project", nil, nil)
	if err == nil {
		t.Error("Expected error for invalid UUID")
	}
	if result != nil {
		t.Error("Expected nil result for invalid UUID")
	}
}

func TestForkSession_NotFound(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	makeProjectDir(configDir, projectPath)

	result, err := ForkSession("550e8400-e29b-41d4-a716-446655440000", projectPath, nil, nil)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	if result != nil {
		t.Error("Expected nil result for non-existent session")
	}
}

func TestForkSession_InvalidUpToMessageID(t *testing.T) {
	setupTestConfig(t)

	invalidID := "not-a-uuid"
	result, err := ForkSession("550e8400-e29b-41d4-a716-446655440000", "/test/project", &invalidID, nil)
	if err == nil {
		t.Error("Expected error for invalid up_to_message_id")
	}
	if result != nil {
		t.Error("Expected nil result")
	}
}

func TestForkSession_Success(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, _ := makeTranscriptSession(projectDir, "", 2)

	result, err := ForkSession(sessionID, projectPath, nil, nil)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected fork result, got nil")
	}

	if result.SessionID == sessionID {
		t.Error("Fork session ID should be different from original")
	}

	// Verify fork file exists
	forkPath := filepath.Join(projectDir, result.SessionID+".jsonl")
	if _, err := os.Stat(forkPath); os.IsNotExist(err) {
		t.Error("Fork session file should exist")
	}
}

func TestForkSession_SameMessageCount(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, _ := makeTranscriptSession(projectDir, "", 3)

	// Read original file for debugging
	originalPath := filepath.Join(projectDir, sessionID+".jsonl")
	originalContent, _ := os.ReadFile(originalPath)
	t.Logf("Original file content:\n%s", string(originalContent))

	result, err := ForkSession(sessionID, projectPath, nil, nil)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	// Read fork file for debugging
	forkPath := filepath.Join(projectDir, result.SessionID+".jsonl")
	forkContent, _ := os.ReadFile(forkPath)
	t.Logf("Fork file content:\n%s", string(forkContent))

	// Original messages
	originalMsgs, _ := GetSessionMessages(sessionID, projectPath, 0, 0)
	t.Logf("Original messages count: %d", len(originalMsgs))

	// Fork messages
	forkMsgs, _ := GetSessionMessages(result.SessionID, projectPath, 0, 0)
	t.Logf("Fork messages count: %d", len(forkMsgs))

	if len(originalMsgs) != len(forkMsgs) {
		t.Errorf("Fork should have same message count: original=%d, fork=%d", len(originalMsgs), len(forkMsgs))
	}
}

func TestForkSession_UpToMessageID(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, uuids := makeTranscriptSession(projectDir, "", 3)

	// Fork up to the first assistant response (uuid index 1)
	cutoffUUID := uuids[1]
	result, err := ForkSession(sessionID, projectPath, &cutoffUUID, nil)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	forkMsgs, _ := GetSessionMessages(result.SessionID, projectPath, 0, 0)

	// Should have 2 messages (first user + first assistant)
	if len(forkMsgs) != 2 {
		t.Errorf("Expected 2 messages after slice, got %d", len(forkMsgs))
	}
}

func TestForkSession_CustomTitle(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, _ := makeTranscriptSession(projectDir, "", 2)

	title := "My Custom Fork"
	result, err := ForkSession(sessionID, projectPath, nil, &title)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	sessions, _ := ListSessions(projectPath, 10, false)
	var forkInfo *types.SDKSessionInfo
	for _, s := range sessions {
		if s.SessionID == result.SessionID {
			forkInfo = &s
			break
		}
	}

	if forkInfo == nil {
		t.Fatal("Fork session should be in list")
	}

	if forkInfo.CustomTitle == nil || *forkInfo.CustomTitle != "My Custom Fork" {
		t.Errorf("Expected custom title 'My Custom Fork', got %v", forkInfo.CustomTitle)
	}
}

func TestForkSession_DefaultTitle(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, _ := makeTranscriptSession(projectDir, "", 2)

	result, err := ForkSession(sessionID, projectPath, nil, nil)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	sessions, _ := ListSessions(projectPath, 10, false)
	var forkInfo *types.SDKSessionInfo
	for _, s := range sessions {
		if s.SessionID == result.SessionID {
			forkInfo = &s
			break
		}
	}

	if forkInfo == nil {
		t.Fatal("Fork session should be in list")
	}

	if forkInfo.CustomTitle == nil {
		t.Error("Expected default title with '(fork)' suffix")
	}

	if !strings.HasSuffix(*forkInfo.CustomTitle, "(fork)") {
		t.Errorf("Expected title ending with '(fork)', got %s", *forkInfo.CustomTitle)
	}
}

func TestForkSession_RemapUUIDs(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID, originalUUIDs := makeTranscriptSession(projectDir, "", 2)

	result, err := ForkSession(sessionID, projectPath, nil, nil)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	forkPath := filepath.Join(projectDir, result.SessionID+".jsonl")
	content, _ := os.ReadFile(forkPath)

	// Verify original UUIDs don't appear in fork
	for _, originalUUID := range originalUUIDs {
		if strings.Contains(string(content), originalUUID) {
			t.Errorf("Original UUID %s should not appear in fork", originalUUID)
		}
	}
}

func TestForkSession_ClearsStaleFields(t *testing.T) {
	configDir := setupTestConfig(t)
	projectPath := "/test/project"
	projectDir := makeProjectDir(configDir, projectPath)
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	filePath := filepath.Join(projectDir, sessionID+".jsonl")

	// Create a session with stale fields
	entry := map[string]interface{}{
		"type":       "user",
		"uuid":       "user-uuid-1",
		"parentUuid": nil,
		"sessionId":  sessionID,
		"timestamp":  "2024-01-15T10:30:00Z",
		"teamName":   "test-team",
		"agentName":  "test-agent",
		"slug":       "test-slug",
		"message": map[string]interface{}{
			"role":    "user",
			"content": "Hello",
		},
	}
	os.WriteFile(filePath, mustMarshal(entry), 0644)

	result, err := ForkSession(sessionID, projectPath, nil, nil)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	forkPath := filepath.Join(projectDir, result.SessionID+".jsonl")
	content, _ := os.ReadFile(forkPath)

	// Verify stale fields are removed
	if strings.Contains(string(content), "teamName") {
		t.Error("Fork should not contain teamName")
	}
	if strings.Contains(string(content), "agentName") {
		t.Error("Fork should not contain agentName")
	}
	if strings.Contains(string(content), "slug") {
		t.Error("Fork should not contain slug")
	}
}
