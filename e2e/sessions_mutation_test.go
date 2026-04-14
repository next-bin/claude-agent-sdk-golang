package e2e_tests

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang"
)

// sanitizePathForTest replicates the sessions package's sanitizePath logic.
// This ensures tests create files in the correct project directory.
var sanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

func sanitizePathForTest(name string) string {
	return sanitizeRegex.ReplaceAllString(name, "-")
}

// setupTestProject creates the project directory structure and returns the project dir.
func setupTestProject(t *testing.T) (tmpDir, configDir, projectDir string) {
	t.Helper()
	tmpDir = t.TempDir()
	configDir = filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(configDir, "projects")
	projectName := sanitizePathForTest(tmpDir)
	projectDir = filepath.Join(projectsDir, projectName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create projects dir: %v", err)
	}

	// Set CLAUDE_CONFIG_DIR
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	t.Cleanup(func() { os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir) })
	os.Setenv("CLAUDE_CONFIG_DIR", configDir)

	return tmpDir, configDir, projectDir
}

func TestDeleteSession(t *testing.T) {
	SkipIfNoAPIKey(t)

	tmpDir, _, projectDir := setupTestProject(t)

	// Create a fake session file
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	sessionFile := filepath.Join(projectDir, sessionID+".jsonl")
	content := `{"type":"user","uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"b1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"assistant","content":[{"type":"text","text":"Hi!"}]}}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}

	// Verify session exists
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Fatal("Session file should exist before delete")
	}

	// Delete the session
	err := claude.DeleteSession(sessionID, tmpDir)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify session is deleted
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Fatal("Session file should be deleted after DeleteSession")
	}
}

func TestDeleteSessionNotFound(t *testing.T) {
	SkipIfNoAPIKey(t)

	// Try to delete a non-existent session
	err := claude.DeleteSession("00000000-0000-0000-0000-000000000000", "/nonexistent/path")
	if err == nil {
		t.Fatal("DeleteSession should fail for non-existent session")
	}
}

// TestDeleteSessionCascadeSubagentDir tests that DeleteSession removes
// the sibling {session_id}/ subdirectory that holds subagent transcripts.
// This matches Python SDK v0.1.59 behavior.
func TestDeleteSessionCascadeSubagentDir(t *testing.T) {
	SkipIfNoAPIKey(t)

	tmpDir, _, projectDir := setupTestProject(t)

	// Create a fake session file
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	sessionFile := filepath.Join(projectDir, sessionID+".jsonl")
	content := `{"type":"user","uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"b1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"assistant","content":[{"type":"text","text":"Hi!"}]}}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}

	// Create subagent transcript directory (sibling {session_id}/ dir)
	subagentDir := filepath.Join(projectDir, sessionID)
	if err := os.MkdirAll(subagentDir, 0755); err != nil {
		t.Fatalf("Failed to create subagent dir: %v", err)
	}

	// Create a fake subagent transcript file
	subagentFile := filepath.Join(subagentDir, "660e8400-e29b-41d4-a716-446655440001.jsonl")
	if err := os.WriteFile(subagentFile, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("Failed to create subagent transcript file: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Fatal("Session file should exist before delete")
	}
	if _, err := os.Stat(subagentDir); os.IsNotExist(err) {
		t.Fatal("Subagent dir should exist before delete")
	}

	// Delete the session
	err := claude.DeleteSession(sessionID, tmpDir)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify session file is deleted
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Fatal("Session file should be deleted after DeleteSession")
	}

	// Verify subagent directory is also deleted (cascade)
	if _, err := os.Stat(subagentDir); !os.IsNotExist(err) {
		t.Fatal("Subagent dir should be deleted (cascade) after DeleteSession")
	}
}

func TestForkSession(t *testing.T) {
	SkipIfNoAPIKey(t)

	tmpDir, _, projectDir := setupTestProject(t)

	// Create a fake session file
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	sessionFile := filepath.Join(projectDir, sessionID+".jsonl")
	content := `{"type":"user","uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"b1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"assistant","content":[{"type":"text","text":"Hi!"}]}}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}

	// Fork the session
	result, err := claude.ForkSession(sessionID, tmpDir, nil, nil)
	if err != nil {
		t.Fatalf("ForkSession failed: %v", err)
	}

	// Verify forked session exists
	if result.SessionID == "" {
		t.Fatal("Forked session ID should not be empty")
	}
	if result.SessionID == sessionID {
		t.Fatal("Forked session ID should be different from original")
	}

	forkedFile := filepath.Join(projectDir, result.SessionID+".jsonl")
	if _, err := os.Stat(forkedFile); os.IsNotExist(err) {
		t.Fatalf("Forked session file should exist: %s", forkedFile)
	}

	// Verify original session still exists
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Fatal("Original session file should still exist after fork")
	}
}

func TestForkSessionWithCustomTitle(t *testing.T) {
	SkipIfNoAPIKey(t)

	tmpDir, _, projectDir := setupTestProject(t)

	// Create a fake session file
	sessionID := "660e8400-e29b-41d4-a716-446655440001"
	sessionFile := filepath.Join(projectDir, sessionID+".jsonl")
	content := `{"type":"user","uuid":"c1d2e3f4-a5b6-7890-abcd-ef1234567891","sessionId":"660e8400-e29b-41d4-a716-446655440001","message":{"role":"user","content":"Test"}}
{"type":"custom-title","customTitle":"Original Session","sessionId":"660e8400-e29b-41d4-a716-446655440001"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}

	// Fork with custom title
	title := "My Forked Session"
	result, err := claude.ForkSession(sessionID, tmpDir, nil, &title)
	if err != nil {
		t.Fatalf("ForkSession with title failed: %v", err)
	}

	// Read the forked file and verify title
	forkedFile := filepath.Join(projectDir, result.SessionID+".jsonl")
	forkedContent, err := os.ReadFile(forkedFile)
	if err != nil {
		t.Fatalf("Failed to read forked session: %v", err)
	}

	// Check that the custom title is in the forked content
	if !containsString(string(forkedContent), "My Forked Session") {
		t.Errorf("Forked session should contain custom title. Content: %s", forkedContent)
	}
}

func TestForkSessionInvalidUUID(t *testing.T) {
	SkipIfNoAPIKey(t)

	_, err := claude.ForkSession("invalid-uuid", "", nil, nil)
	if err == nil {
		t.Fatal("ForkSession should fail for invalid UUID")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s != "" && substr != "" && (s == substr || len(s) > 0 && containsStringAt(s, substr))
}

func containsStringAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
