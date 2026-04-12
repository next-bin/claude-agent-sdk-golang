package e2e_tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/next-bin/claude-agent-sdk-golang"
)

func TestDeleteSession(t *testing.T) {
	SkipIfNoAPIKey(t)

	// Create a temporary project directory
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("Failed to create projects dir: %v", err)
	}

	// Create a fake session file
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	sessionFile := filepath.Join(projectsDir, sessionID+".jsonl")
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

	// Override CLAUDE_CONFIG_DIR for test
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmpDir, ".claude"))
	defer os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)

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

func TestForkSession(t *testing.T) {
	SkipIfNoAPIKey(t)

	// Create a temporary project directory
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("Failed to create projects dir: %v", err)
	}

	// Create a fake session file
	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	sessionFile := filepath.Join(projectsDir, sessionID+".jsonl")
	content := `{"type":"user","uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"user","content":"Hello"}}
{"type":"assistant","uuid":"b1b2c3d4-e5f6-7890-abcd-ef1234567890","sessionId":"550e8400-e29b-41d4-a716-446655440000","message":{"role":"assistant","content":[{"type":"text","text":"Hi!"}]}}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}

	// Override CLAUDE_CONFIG_DIR for test
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmpDir, ".claude"))
	defer os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)

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

	forkedFile := filepath.Join(projectsDir, result.SessionID+".jsonl")
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

	// Create a temporary project directory
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("Failed to create projects dir: %v", err)
	}

	// Create a fake session file
	sessionID := "660e8400-e29b-41d4-a716-446655440001"
	sessionFile := filepath.Join(projectsDir, sessionID+".jsonl")
	content := `{"type":"user","uuid":"c1d2e3f4-a5b6-7890-abcd-ef1234567891","sessionId":"660e8400-e29b-41d4-a716-446655440001","message":{"role":"user","content":"Test"}}
{"type":"custom-title","customTitle":"Original Session","sessionId":"660e8400-e29b-41d4-a716-446655440001"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}

	// Override CLAUDE_CONFIG_DIR for test
	oldConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
	os.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(tmpDir, ".claude"))
	defer os.Setenv("CLAUDE_CONFIG_DIR", oldConfigDir)

	// Fork with custom title
	title := "My Forked Session"
	result, err := claude.ForkSession(sessionID, tmpDir, nil, &title)
	if err != nil {
		t.Fatalf("ForkSession with title failed: %v", err)
	}

	// Read the forked file and verify title
	forkedFile := filepath.Join(projectsDir, result.SessionID+".jsonl")
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
