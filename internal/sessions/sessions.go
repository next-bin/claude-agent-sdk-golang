// Package sessions provides functions for listing and reading Claude Code sessions.
//
// This package implements the Sessions API (v0.1.46) which allows:
//   - Listing sessions with metadata extracted from stat + head/tail reads
//   - Reading session messages by parsing JSONL transcripts and building conversation chains
//
// The implementation scans ~/.claude/projects/ directory structure and handles
// git worktree detection, path sanitization, and hash mismatch tolerance.
package sessions

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/next-bin/claude-agent-sdk-golang/types"
	"golang.org/x/text/unicode/norm"
)

// Constants matching standard SDK
const (
	// LiteReadBufSize is the size of the head/tail buffer for lite metadata reads.
	LiteReadBufSize = 65536

	// MaxSanitizedLength is the maximum length for a sanitized path component.
	// Most filesystems limit individual components to 255 bytes.
	MaxSanitizedLength = 200
)

// nfcNormalize applies Unicode NFC normalization to a string.
func nfcNormalize(s string) string {
	// Fast path: if string is already valid UTF-8 and ASCII, no normalization needed
	// Most paths are ASCII, so this avoids unnecessary work
	isASCII := true
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			isASCII = false
			break
		}
	}
	if isASCII {
		return s
	}

	// Use golang.org/x/text/unicode/norm for NFC normalization
	// This is an imported package, not a local function
	return norm.NFC.String(s)
}

// Regex patterns
var (
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

	// Pattern matching auto-generated or system messages to skip
	skipFirstPromptPattern = regexp.MustCompile(
		`^(?:<local-command-stdout>|<session-start-hook>|<tick>|<goal>|` +
			`\[Request interrupted by user[^\]]*\]|` +
			`\s*<ide_opened_file>[\s\S]*</ide_opened_file>\s*$|` +
			`\s*<ide_selection>[\s\S]*</ide_selection>\s*$)`)

	commandNameRegex = regexp.MustCompile(`<command-name>(.*?)</command-name>`)

	sanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

// ListSessions lists sessions with metadata extracted from stat + head/tail reads.
//
// When directory is provided, returns sessions for that project directory and its
// git worktrees. When empty, returns sessions across all projects.
//
// The limit parameter limits the number of sessions returned (0 means no limit).
// The includeWorktrees parameter controls whether to include git worktree sessions.
func ListSessions(directory string, limit int, includeWorktrees bool) ([]types.SDKSessionInfo, error) {
	if directory != "" {
		return listSessionsForProject(directory, limit, includeWorktrees)
	}
	return listAllSessions(limit)
}

// GetSessionMessages reads a session's conversation messages from its JSONL transcript file.
//
// Parses the full JSONL, builds the conversation chain via parentUuid links,
// and returns user/assistant messages in chronological order.
//
// The limit parameter limits the number of messages returned (0 means no limit).
// The offset parameter skips the first N messages.
func GetSessionMessages(sessionID, directory string, limit, offset int) ([]types.SessionMessage, error) {
	// Validate session ID
	if !isValidUUID(sessionID) {
		return nil, nil
	}

	content, err := readSessionFile(sessionID, directory)
	if err != nil || content == "" {
		return nil, nil
	}

	entries := parseTranscriptEntries(content)
	chain := buildConversationChain(entries)
	visible := filterVisibleMessages(chain)
	messages := convertToSessionMessages(visible)

	// Apply offset and limit
	if offset > 0 && offset < len(messages) {
		messages = messages[offset:]
	} else if offset >= len(messages) {
		return nil, nil
	}

	if limit > 0 && limit < len(messages) {
		messages = messages[:limit]
	}

	return messages, nil
}

// GetSessionInfo reads metadata for a single session by ID.
//
// Wraps readSessionLite for one file — no O(n) directory scan.
// Directory resolution matches GetSessionMessages: directory is the project path;
// when omitted, all project directories are searched for the session file.
//
// Returns SDKSessionInfo for the session, or nil if the session file
// is not found, is a sidechain session, or has no extractable summary.
//
// Example:
//
//	// Look up a session in a specific project
//	info := GetSessionInfo("550e8400-e29b-41d4-a716-446655440000", "/path/to/project")
//	if info != nil {
//	    fmt.Println(info.Summary)
//	}
//
//	// Search all projects for a session
//	info := GetSessionInfo("550e8400-e29b-41d4-a716-446655440000", "")
func GetSessionInfo(sessionID, directory string) *types.SDKSessionInfo {
	if !isValidUUID(sessionID) {
		return nil
	}
	fileName := sessionID + ".jsonl"

	if directory != "" {
		canonical := normalizePath(directory)
		projectDir := findProjectDir(canonical)
		if projectDir != "" {
			lite := readSessionLite(filepath.Join(projectDir, fileName))
			if lite != nil {
				return parseSessionInfoFromLite(sessionID, lite, canonical)
			}
		}

		// Worktree fallback — matches GetSessionMessages semantics.
		// Sessions may live under a different worktree root.
		worktreePaths := getWorktreePaths(canonical)
		for _, wt := range worktreePaths {
			if wt == canonical {
				continue
			}
			wtProjectDir := findProjectDir(wt)
			if wtProjectDir != "" {
				lite := readSessionLite(filepath.Join(wtProjectDir, fileName))
				if lite != nil {
					return parseSessionInfoFromLite(sessionID, lite, wt)
				}
			}
		}

		return nil
	}

	// No directory — search all project directories for the session file.
	projectsDir := getProjectsDir()
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		lite := readSessionLite(filepath.Join(projectsDir, entry.Name(), fileName))
		if lite != nil {
			return parseSessionInfoFromLite(sessionID, lite, "")
		}
	}
	return nil
}

// parseSessionInfoFromLite parses SDKSessionInfo fields from a lite session read (head/tail/stat).
//
// Returns nil for sidechain sessions or metadata-only sessions with no extractable summary.
func parseSessionInfoFromLite(sessionID string, lite *liteSessionFile, projectPath string) *types.SDKSessionInfo {
	head, tail, mtime, size := lite.head, lite.tail, lite.mtime, lite.size

	// Check first line for sidechain sessions
	firstNewline := strings.Index(head, "\n")
	firstLine := head
	if firstNewline >= 0 {
		firstLine = head[:firstNewline]
	}
	if strings.Contains(firstLine, `"isSidechain":true`) ||
		strings.Contains(firstLine, `"isSidechain": true`) {
		return nil
	}

	// User-set title (customTitle) wins over AI-generated title (aiTitle).
	// Head fallback covers short sessions where the title entry may not be in tail.
	customTitle := coalesce(
		extractLastJSONStringField(tail, "customTitle"),
		extractLastJSONStringField(head, "customTitle"),
		extractLastJSONStringField(tail, "aiTitle"),
		extractLastJSONStringField(head, "aiTitle"),
	)

	firstPrompt := extractFirstPromptFromHead(head)

	// Summary chain: customTitle || tail.lastPrompt || tail.summary || firstPrompt
	// lastPrompt tail entry shows what the user was most recently doing.
	summary := customTitle
	if summary == "" {
		summary = extractLastJSONStringField(tail, "lastPrompt")
	}
	if summary == "" {
		summary = extractLastJSONStringField(tail, "summary")
	}
	if summary == "" {
		summary = firstPrompt
	}

	// Skip metadata-only sessions (no title, no summary, no prompt)
	if summary == "" {
		return nil
	}

	gitBranch := extractLastJSONStringField(tail, "gitBranch")
	if gitBranch == "" {
		gitBranch = extractJSONStringField(head, "gitBranch")
	}

	sessionCWD := extractJSONStringField(head, "cwd")
	if sessionCWD == "" {
		sessionCWD = projectPath
	}

	// Tag extraction: scope to {"type":"tag"} lines only.
	tag := extractTagFromTail(tail)

	// created_at from first entry's ISO timestamp (epoch ms).
	createdAt := extractTimestampFromFirstLine(firstLine)

	info := &types.SDKSessionInfo{
		SessionID:    sessionID,
		Summary:      summary,
		LastModified: mtime,
		FileSize:     &size,
	}

	if customTitle != "" {
		info.CustomTitle = &customTitle
	}
	if firstPrompt != "" {
		info.FirstPrompt = &firstPrompt
	}
	if gitBranch != "" {
		info.GitBranch = &gitBranch
	}
	if sessionCWD != "" {
		info.CWD = &sessionCWD
	}
	if tag != "" {
		info.Tag = &tag
	}
	if createdAt != 0 {
		info.CreatedAt = &createdAt
	}

	return info
}

// ============================================================================
// Path sanitization
// ============================================================================

// simpleHash computes a 32-bit integer hash in base36 (matching JS implementation).
func simpleHash(s string) string {
	h := uint32(0)
	for _, ch := range s {
		h = (h << 5) - h + uint32(ch)
		// Emulate JS `hash |= 0` (coerce to 32-bit signed int)
		h = h & 0xFFFFFFFF
		if int32(h) < 0 {
			h = uint32(-int32(h))
		}
	}

	// Convert to base36
	if h == 0 {
		return "0"
	}

	digits := "0123456789abcdefghijklmnopqrstuvwxyz"
	out := make([]byte, 0, 8) // Pre-allocate: max 8 digits for uint32 in base36
	n := h
	for n > 0 {
		out = append(out, digits[n%36])
		n /= 36
	}
	// Reverse the slice
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// sanitizePath makes a string safe for use as a directory name.
func sanitizePath(name string) string {
	sanitized := sanitizeRegex.ReplaceAllString(name, "-")
	if len(sanitized) <= MaxSanitizedLength {
		return sanitized
	}
	h := simpleHash(name)
	return fmt.Sprintf("%s-%s", sanitized[:MaxSanitizedLength], h)
}

// ============================================================================
// Config directories
// ============================================================================

// getClaudeConfigHomeDir returns the Claude config directory (respects CLAUDE_CONFIG_DIR).
func getClaudeConfigHomeDir() string {
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir != "" {
		return normalizePath(configDir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// getProjectsDir returns the projects directory under the Claude config.
func getProjectsDir() string {
	return filepath.Join(getClaudeConfigHomeDir(), "projects")
}

// getProjectDir returns the project directory for a given path.
func getProjectDir(projectPath string) string {
	return filepath.Join(getProjectsDir(), sanitizePath(projectPath))
}

// normalizePath resolves a directory path to its canonical form.
func normalizePath(d string) string {
	resolved, err := filepath.Abs(d)
	if err != nil {
		resolved = d
	}

	// Normalize unicode to NFC form (standard text normalization)
	return nfcNormalize(resolved)
}

// findProjectDir finds the project directory for a given path.
//
// Tolerates hash mismatches for long paths (>200 chars).
func findProjectDir(projectPath string) string {
	exact := getProjectDir(projectPath)
	if _, err := os.Stat(exact); err == nil {
		return exact
	}

	// Exact match failed - try prefix matching for long paths
	sanitized := sanitizePath(projectPath)
	if len(sanitized) <= MaxSanitizedLength {
		return ""
	}

	prefix := sanitized[:MaxSanitizedLength]
	projectsDir := getProjectsDir()

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix+"-") {
			return filepath.Join(projectsDir, entry.Name())
		}
	}
	return ""
}

// ============================================================================
// UUID validation
// ============================================================================

func isValidUUID(s string) bool {
	return uuidRegex.MatchString(strings.ToLower(s))
}

// ============================================================================
// JSON string field extraction
// ============================================================================

// unescapeJSONString unescapes a JSON string value extracted as raw text.
func unescapeJSONString(raw string) string {
	if !strings.ContainsRune(raw, '\\') {
		return raw
	}

	var result string
	err := json.Unmarshal([]byte(`"`+raw+`"`), &result)
	if err != nil {
		return raw
	}
	return result
}

// extractJSONStringField extracts a simple JSON string field value without full parsing.
func extractJSONStringField(text, key string) string {
	patterns := []string{
		fmt.Sprintf(`"%s":"`, key),
		fmt.Sprintf(`"%s": "`, key),
	}

	for _, pattern := range patterns {
		idx := strings.Index(text, pattern)
		if idx < 0 {
			continue
		}

		valueStart := idx + len(pattern)
		i := valueStart
		for i < len(text) {
			if text[i] == '\\' {
				i += 2
				continue
			}
			if text[i] == '"' {
				return unescapeJSONString(text[valueStart:i])
			}
			i++
		}
	}
	return ""
}

// extractLastJSONStringField extracts the LAST occurrence of a JSON string field.
func extractLastJSONStringField(text, key string) string {
	patterns := []string{
		fmt.Sprintf(`"%s":"`, key),
		fmt.Sprintf(`"%s": "`, key),
	}

	var lastValue string
	for _, pattern := range patterns {
		searchFrom := 0
		for {
			idx := strings.Index(text[searchFrom:], pattern)
			if idx < 0 {
				break
			}
			idx += searchFrom

			valueStart := idx + len(pattern)
			i := valueStart
			for i < len(text) {
				if text[i] == '\\' {
					i += 2
					continue
				}
				if text[i] == '"' {
					lastValue = unescapeJSONString(text[valueStart:i])
					break
				}
				i++
			}
			searchFrom = i + 1
		}
	}
	return lastValue
}

// coalesce returns the first non-empty string from the provided arguments.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// extractTagFromTail extracts the tag value scoped to {"type":"tag"} lines.
// This avoids matching "tag" in tool_use inputs (git tag, Docker tags, etc.).
func extractTagFromTail(tail string) string {
	// Find the last line that starts with {"type":"tag"
	lines := strings.Split(tail, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, `{"type":"tag"`) {
			return extractLastJSONStringField(line, "tag")
		}
	}
	return ""
}

// extractTimestampFromFirstLine extracts timestamp from the first JSONL line only.
// This avoids false matches from later entries in the head buffer.
func extractTimestampFromFirstLine(firstLine string) int64 {
	if firstLine == "" {
		return 0
	}
	ts := extractJSONStringField(firstLine, "timestamp")
	if ts == "" {
		return 0
	}
	// Parse ISO timestamp and convert to epoch ms
	return parseTimestamp(ts)
}

// parseTimestamp parses an ISO timestamp string and returns epoch milliseconds.
func parseTimestamp(ts string) int64 {
	// Remove timezone suffix for parsing
	// Handle both "2024-01-15T10:30:00Z" and "2024-01-15T10:30:00.123Z" formats
	cleanTs := ts
	if strings.HasSuffix(ts, "Z") {
		cleanTs = ts[:len(ts)-1]
	} else if idx := strings.LastIndex(ts, "+"); idx > 0 {
		cleanTs = ts[:idx]
	} else if idx := strings.LastIndex(ts, "-"); idx > 0 && idx > 10 {
		// Only strip timezone if after the date portion
		cleanTs = ts[:idx]
	}

	// Try parsing with nanoseconds
	t, err := time.Parse("2006-01-02T15:04:05.000000000", cleanTs)
	if err != nil {
		// Try with less precision
		t, err = time.Parse("2006-01-02T15:04:05.000", cleanTs)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05", cleanTs)
			if err != nil {
				return 0
			}
		}
	}
	return t.UnixMilli()
}

// ============================================================================
// First prompt extraction
// ============================================================================

// processPromptText trims whitespace and replaces newlines with spaces in a single pass.
// This is more efficient than calling strings.TrimSpace and strings.ReplaceAll separately.
func processPromptText(s string) string {
	// Fast path: empty string
	if len(s) == 0 {
		return ""
	}

	// Use strings.Builder for efficient string building
	var b strings.Builder
	b.Grow(len(s))

	// Track if we're at the start (for trimming leading whitespace)
	start := true
	// Track if previous char was a space (for collapsing multiple spaces)
	prevSpace := false

	for _, r := range s {
		switch r {
		case '\n', '\r':
			// Replace newlines with space
			if !start && !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		case ' ', '\t':
			// Collapse multiple spaces/tabs
			if !start && !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		default:
			// Regular character
			b.WriteRune(r)
			start = false
			prevSpace = false
		}
	}

	// Trim trailing space
	result := b.String()
	if len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}

	return result
}

// extractFirstPromptFromHead extracts the first meaningful user prompt from a JSONL head chunk.
func extractFirstPromptFromHead(head string) string {
	scanner := bufio.NewScanner(strings.NewReader(head))
	var commandFallback string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Check for user message
		if !strings.Contains(line, `"type":"user"`) && !strings.Contains(line, `"type": "user"`) {
			continue
		}
		if strings.Contains(line, `"tool_result"`) {
			continue
		}
		if strings.Contains(line, `"isMeta":true`) || strings.Contains(line, `"isMeta": true`) {
			continue
		}
		if strings.Contains(line, `"isCompactSummary":true`) || strings.Contains(line, `"isCompactSummary": true`) {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entryType, ok := entry["type"].(string); !ok || entryType != "user" {
			continue
		}

		message, ok := entry["message"].(map[string]interface{})
		if !ok {
			continue
		}

		// Pre-allocate texts slice with reasonable capacity
		var texts []string
		switch content := message["content"].(type) {
		case string:
			texts = []string{content}
		case []interface{}:
			texts = make([]string, 0, len(content))
			for _, block := range content {
				if b, ok := block.(map[string]interface{}); ok {
					if t, ok := b["type"].(string); ok && t == "text" {
						if text, ok := b["text"].(string); ok {
							texts = append(texts, text)
						}
					}
				}
			}
		}

		for _, raw := range texts {
			// Optimize string processing: trim and replace newlines in one pass
			result := processPromptText(raw)
			if result == "" {
				continue
			}

			// Skip slash-command messages but remember first as fallback
			if cmdMatch := commandNameRegex.FindStringSubmatch(result); cmdMatch != nil {
				if commandFallback == "" {
					commandFallback = cmdMatch[1]
				}
				continue
			}

			if skipFirstPromptPattern.MatchString(result) {
				continue
			}

			if utf8.RuneCountInString(result) > 200 {
				result = string([]rune(result)[:200]) + "…"
			}
			return result
		}
	}

	if commandFallback != "" {
		return commandFallback
	}
	return ""
}

// ============================================================================
// File I/O
// ============================================================================

// liteSessionFile holds the result of reading a session file's head, tail, mtime and size.
type liteSessionFile struct {
	mtime int64
	size  int64
	head  string
	tail  string
}

// readSessionLite opens a session file, stats it, and reads head + tail.
func readSessionLite(filePath string) *liteSessionFile {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil
	}

	size := stat.Size()
	mtime := stat.ModTime().UnixMilli()

	// Read head
	headBuf := make([]byte, LiteReadBufSize)
	n, err := f.Read(headBuf)
	if err != nil || n == 0 {
		return nil
	}
	head := string(headBuf[:n])

	// Read tail
	var tail string
	tailOffset := size - LiteReadBufSize
	if tailOffset <= 0 {
		tail = head
	} else {
		_, err = f.Seek(tailOffset, 0)
		if err != nil {
			return nil
		}
		tailBuf := make([]byte, LiteReadBufSize)
		n, err = f.Read(tailBuf)
		if err != nil {
			return nil
		}
		tail = string(tailBuf[:n])
	}

	return &liteSessionFile{
		mtime: mtime,
		size:  size,
		head:  head,
		tail:  tail,
	}
}

// ============================================================================
// Git worktree detection
// ============================================================================

// getWorktreePaths returns absolute worktree paths for the git repo containing cwd.
func getWorktreePaths(cwd string) []string {
	ctx, cancel := createCommandContext()
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var paths []string
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			path := normalizePath(strings.TrimPrefix(line, "worktree "))
			paths = append(paths, path)
		}
	}
	return paths
}

// createCommandContext creates a context with timeout for command execution.
func createCommandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// ============================================================================
// Core implementation
// ============================================================================

// readSessionsFromDir reads session files from a single project directory.
func readSessionsFromDir(projectDir, projectPath string) []types.SDKSessionInfo {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil
	}

	var results []types.SDKSessionInfo

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(name, ".jsonl")
		if !isValidUUID(sessionID) {
			continue
		}

		filePath := filepath.Join(projectDir, name)
		lite := readSessionLite(filePath)
		if lite == nil {
			continue
		}

		// Check first line for sidechain sessions
		firstLine := lite.head
		if idx := strings.Index(lite.head, "\n"); idx >= 0 {
			firstLine = lite.head[:idx]
		}
		if strings.Contains(firstLine, `"isSidechain":true`) ||
			strings.Contains(firstLine, `"isSidechain": true`) {
			continue
		}

		// Custom title chain: tail.customTitle || head.customTitle || tail.aiTitle || head.aiTitle
		// User-set title (customTitle) wins over AI-generated title (aiTitle).
		// Head fallback covers short sessions where the title entry may not be in tail.
		customTitle := coalesce(
			extractLastJSONStringField(lite.tail, "customTitle"),
			extractLastJSONStringField(lite.head, "customTitle"),
			extractLastJSONStringField(lite.tail, "aiTitle"),
			extractLastJSONStringField(lite.head, "aiTitle"),
		)

		firstPrompt := extractFirstPromptFromHead(lite.head)

		// Summary chain: customTitle || tail.lastPrompt || tail.summary || firstPrompt
		// lastPrompt tail entry shows what the user was most recently doing.
		summary := customTitle
		if summary == "" {
			summary = extractLastJSONStringField(lite.tail, "lastPrompt")
		}
		if summary == "" {
			summary = extractLastJSONStringField(lite.tail, "summary")
		}
		if summary == "" {
			summary = firstPrompt
		}

		// Skip metadata-only sessions
		if summary == "" {
			continue
		}

		gitBranch := extractLastJSONStringField(lite.tail, "gitBranch")
		if gitBranch == "" {
			gitBranch = extractJSONStringField(lite.head, "gitBranch")
		}

		sessionCWD := extractJSONStringField(lite.head, "cwd")
		if sessionCWD == "" {
			sessionCWD = projectPath
		}

		// Tag extraction: scope to {"type":"tag"} lines only.
		// A bare tail scan for "tag" would match tool_use inputs
		// (git tag, Docker tags, cloud resource tags).
		tag := extractTagFromTail(lite.tail)

		// created_at from first entry's ISO timestamp (epoch ms).
		// Scope to first JSONL line to avoid false matches.
		createdAt := extractTimestampFromFirstLine(firstLine)

		info := types.SDKSessionInfo{
			SessionID:    sessionID,
			Summary:      summary,
			LastModified: lite.mtime,
			FileSize:     &lite.size,
		}

		if customTitle != "" {
			info.CustomTitle = &customTitle
		}
		if firstPrompt != "" {
			info.FirstPrompt = &firstPrompt
		}
		if gitBranch != "" {
			info.GitBranch = &gitBranch
		}
		if sessionCWD != "" {
			info.CWD = &sessionCWD
		}
		if tag != "" {
			info.Tag = &tag
		}
		if createdAt != 0 {
			info.CreatedAt = &createdAt
		}

		results = append(results, info)
	}

	return results
}

// deduplicateBySessionID deduplicates sessions by session_id, keeping the newest.
func deduplicateBySessionID(sessions []types.SDKSessionInfo) []types.SDKSessionInfo {
	byID := make(map[string]types.SDKSessionInfo)
	for _, s := range sessions {
		existing, exists := byID[s.SessionID]
		if !exists || s.LastModified > existing.LastModified {
			byID[s.SessionID] = s
		}
	}

	result := make([]types.SDKSessionInfo, 0, len(byID))
	for _, s := range byID {
		result = append(result, s)
	}
	return result
}

// applySortAndLimit sorts sessions by last_modified descending and applies limit.
func applySortAndLimit(sessions []types.SDKSessionInfo, limit int) []types.SDKSessionInfo {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastModified > sessions[j].LastModified
	})

	if limit > 0 && limit < len(sessions) {
		return sessions[:limit]
	}
	return sessions
}

// listSessionsForProject lists sessions for a specific project directory.
func listSessionsForProject(directory string, limit int, includeWorktrees bool) ([]types.SDKSessionInfo, error) {
	canonicalDir := normalizePath(directory)

	var worktreePaths []string
	if includeWorktrees {
		worktreePaths = getWorktreePaths(canonicalDir)
	}

	// No worktrees - just scan the single project dir
	if len(worktreePaths) <= 1 {
		projectDir := findProjectDir(canonicalDir)
		if projectDir == "" {
			return nil, nil
		}
		sessions := readSessionsFromDir(projectDir, canonicalDir)
		return applySortAndLimit(sessions, limit), nil
	}

	// Worktree-aware scanning
	projectsDir := getProjectsDir()
	caseInsensitive := runtime.GOOS == "windows"

	// Sort worktree paths by sanitized prefix length (longest first)
	type indexedWorktree struct {
		path   string
		prefix string
	}
	var indexed []indexedWorktree
	for _, wt := range worktreePaths {
		sanitized := sanitizePath(wt)
		prefix := sanitized
		if caseInsensitive {
			prefix = strings.ToLower(sanitized)
		}
		indexed = append(indexed, indexedWorktree{path: wt, prefix: prefix})
	}
	sort.Slice(indexed, func(i, j int) bool {
		return len(indexed[i].prefix) > len(indexed[j].prefix)
	})

	allDirents, err := os.ReadDir(projectsDir)
	if err != nil {
		// Fall back to single project dir
		projectDir := findProjectDir(canonicalDir)
		if projectDir == "" {
			return nil, nil
		}
		sessions := readSessionsFromDir(projectDir, canonicalDir)
		return applySortAndLimit(sessions, limit), nil
	}

	var allSessions []types.SDKSessionInfo
	seenDirs := make(map[string]bool)

	// Always include the user's actual directory
	canonicalProjectDir := findProjectDir(canonicalDir)
	if canonicalProjectDir != "" {
		dirBase := filepath.Base(canonicalProjectDir)
		if caseInsensitive {
			dirBase = strings.ToLower(dirBase)
		}
		seenDirs[dirBase] = true
		sessions := readSessionsFromDir(canonicalProjectDir, canonicalDir)
		allSessions = append(allSessions, sessions...)
	}

	for _, entry := range allDirents {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		if caseInsensitive {
			dirName = strings.ToLower(dirName)
		}
		if seenDirs[dirName] {
			continue
		}

		for _, idx := range indexed {
			prefix := idx.prefix
			if caseInsensitive {
				prefix = strings.ToLower(prefix)
			}

			// Match exact or prefix with hash suffix
			isMatch := dirName == prefix ||
				(len(prefix) >= MaxSanitizedLength && strings.HasPrefix(dirName, prefix+"-"))

			if isMatch {
				seenDirs[dirName] = true
				sessions := readSessionsFromDir(filepath.Join(projectsDir, entry.Name()), idx.path)
				allSessions = append(allSessions, sessions...)
				break
			}
		}
	}

	deduped := deduplicateBySessionID(allSessions)
	return applySortAndLimit(deduped, limit), nil
}

// listAllSessions lists sessions across all project directories.
func listAllSessions(limit int) ([]types.SDKSessionInfo, error) {
	projectsDir := getProjectsDir()

	projectDirs, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, nil
	}

	var allSessions []types.SDKSessionInfo
	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}
		sessions := readSessionsFromDir(filepath.Join(projectsDir, projectDir.Name()), "")
		allSessions = append(allSessions, sessions...)
	}

	deduped := deduplicateBySessionID(allSessions)
	return applySortAndLimit(deduped, limit), nil
}

// ============================================================================
// GetSessionMessages implementation
// ============================================================================

// transcriptEntryTypes are the types that carry uuid + parentUuid chain links.
var transcriptEntryTypes = map[string]bool{
	"user":       true,
	"assistant":  true,
	"progress":   true,
	"system":     true,
	"attachment": true,
}

// transcriptEntry represents a parsed JSONL transcript entry.
type transcriptEntry map[string]interface{}

// readSessionFile finds and reads the session JSONL file.
func readSessionFile(sessionID, directory string) (string, error) {
	fileName := sessionID + ".jsonl"

	if directory != "" {
		canonicalDir := normalizePath(directory)

		// Try the exact/prefix-matched project directory first
		projectDir := findProjectDir(canonicalDir)
		if projectDir != "" {
			content, err := os.ReadFile(filepath.Join(projectDir, fileName))
			if err == nil {
				return string(content), nil
			}
		}

		// Try worktree paths
		worktreePaths := getWorktreePaths(canonicalDir)
		for _, wt := range worktreePaths {
			if wt == canonicalDir {
				continue
			}
			wtProjectDir := findProjectDir(wt)
			if wtProjectDir != "" {
				content, err := os.ReadFile(filepath.Join(wtProjectDir, fileName))
				if err == nil {
					return string(content), nil
				}
			}
		}

		return "", os.ErrNotExist
	}

	// No directory provided - search all project directories
	projectsDir := getProjectsDir()
	dirents, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", err
	}

	for _, entry := range dirents {
		content, err := os.ReadFile(filepath.Join(projectsDir, entry.Name(), fileName))
		if err == nil {
			return string(content), nil
		}
	}

	return "", os.ErrNotExist
}

// parseTranscriptEntries parses JSONL content into transcript entries.
func parseTranscriptEntries(content string) []transcriptEntry {
	entries, _ := parseTranscriptEntriesWithReplacements(content, "")
	return entries
}

// parseTranscriptEntriesWithReplacements parses JSONL content into transcript entries
// and collects content-replacement records for a specific session.
func parseTranscriptEntriesWithReplacements(content string, sessionID string) ([]transcriptEntry, []interface{}) {
	// Pre-allocate based on estimated line count
	lineCount := strings.Count(content, "\n") + 1
	entries := make([]transcriptEntry, 0, lineCount)
	contentReplacements := make([]interface{}, 0)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		entryType, ok := entry["type"].(string)
		if !ok {
			continue
		}

		// Handle transcript entries
		if transcriptEntryTypes[entryType] {
			if _, ok := entry["uuid"].(string); ok {
				entries = append(entries, entry)
			}
			continue
		}

		// Handle content-replacement entries
		if entryType == "content-replacement" {
			entrySessionID, _ := entry["sessionId"].(string)
			if sessionID == "" || entrySessionID == sessionID {
				replacements, ok := entry["replacements"].([]interface{})
				if ok {
					contentReplacements = append(contentReplacements, replacements...)
				}
			}
		}
	}

	return entries, contentReplacements
}

// buildConversationChain builds the conversation chain by finding the leaf and walking parentUuid.
func buildConversationChain(entries []transcriptEntry) []transcriptEntry {
	if len(entries) == 0 {
		return nil
	}

	// Index by uuid for O(1) parent lookup
	byUUID := make(map[string]transcriptEntry)
	entryIndex := make(map[string]int)
	for i, entry := range entries {
		uuid, _ := entry["uuid"].(string)
		byUUID[uuid] = entry
		entryIndex[uuid] = i
	}

	// Find terminal messages (no children point to them via parentUuid)
	parentUUIDs := make(map[string]bool)
	for _, entry := range entries {
		if parent, ok := entry["parentUuid"].(string); ok && parent != "" {
			parentUUIDs[parent] = true
		}
	}

	// Pre-allocate terminals slice
	terminals := make([]transcriptEntry, 0, len(entries))
	for _, entry := range entries {
		uuid, ok := entry["uuid"].(string)
		if !ok {
			continue
		}
		if !parentUUIDs[uuid] {
			terminals = append(terminals, entry)
		}
	}

	// From each terminal, walk back to find the nearest user/assistant leaf
	leaves := make([]transcriptEntry, 0, len(terminals))
	for _, terminal := range terminals {
		cur := terminal
		seen := make(map[string]bool)

		for {
			uuid, ok := cur["uuid"].(string)
			if !ok {
				break
			}
			if seen[uuid] {
				break
			}
			seen[uuid] = true

			if entryType, ok := cur["type"].(string); ok && (entryType == "user" || entryType == "assistant") {
				leaves = append(leaves, cur)
				break
			}

			parent, ok := cur["parentUuid"].(string)
			if !ok || parent == "" {
				break
			}
			cur = byUUID[parent]
			if cur == nil {
				break
			}
		}
	}

	if len(leaves) == 0 {
		return nil
	}

	// Pick the leaf from the main chain (not sidechain/team/meta)
	mainLeaves := make([]transcriptEntry, 0, len(leaves))
	for _, leaf := range leaves {
		if leaf["isSidechain"] == true {
			continue
		}
		if leaf["teamName"] != nil {
			continue
		}
		if leaf["isMeta"] == true {
			continue
		}
		mainLeaves = append(mainLeaves, leaf)
	}

	candidates := mainLeaves
	if len(candidates) == 0 {
		candidates = leaves
	}

	// Pick the one with highest position in entries array
	best := candidates[0]
	bestIdx := entryIndex[best["uuid"].(string)]
	for _, cur := range candidates[1:] {
		curIdx := entryIndex[cur["uuid"].(string)]
		if curIdx > bestIdx {
			best = cur
			bestIdx = curIdx
		}
	}

	// Walk from leaf to root via parentUuid
	chain := make([]transcriptEntry, 0, len(entries))
	chainSeen := make(map[string]bool)
	cur := best

	for cur != nil {
		uuid := cur["uuid"].(string)
		if chainSeen[uuid] {
			break
		}
		chainSeen[uuid] = true
		chain = append(chain, cur)

		parent, ok := cur["parentUuid"].(string)
		if !ok || parent == "" {
			break
		}
		cur = byUUID[parent]
	}

	// Reverse to get chronological order
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain
}

// filterVisibleMessages filters to visible user/assistant messages.
func filterVisibleMessages(entries []transcriptEntry) []transcriptEntry {
	visible := make([]transcriptEntry, 0, len(entries))
	for _, entry := range entries {
		entryType, ok := entry["type"].(string)
		if !ok {
			continue
		}
		if entryType != "user" && entryType != "assistant" {
			continue
		}
		if entry["isMeta"] == true {
			continue
		}
		if entry["isSidechain"] == true {
			continue
		}
		if entry["teamName"] != nil {
			continue
		}
		visible = append(visible, entry)
	}
	return visible
}

// convertToSessionMessages converts transcript entries to SessionMessage objects.
func convertToSessionMessages(entries []transcriptEntry) []types.SessionMessage {
	messages := make([]types.SessionMessage, 0, len(entries))
	for _, entry := range entries {
		msgType, _ := entry["type"].(string)
		uuid, _ := entry["uuid"].(string)
		sessionID, _ := entry["sessionId"].(string)
		message := entry["message"]

		messages = append(messages, types.SessionMessage{
			Type:      msgType,
			UUID:      uuid,
			SessionID: sessionID,
			Message:   message,
		})
	}
	return messages
}

// ============================================================================
// Session Mutations (v0.1.49)
// ============================================================================

// RenameSession renames a session by appending a custom-title entry.
//
// ListSessions reads the LAST custom-title from the file tail, so
// repeated calls are safe — the most recent wins.
//
// Parameters:
//   - sessionID: UUID of the session to rename.
//   - title: New session title. Leading/trailing whitespace is stripped.
//     Must be non-empty after stripping.
//   - directory: Project directory path (same semantics as ListSessions).
//     When empty, all project directories are searched for the session file.
//
// Returns an error if:
//   - sessionID is not a valid UUID
//   - title is empty/whitespace-only
//   - the session file cannot be found
func RenameSession(sessionID, title, directory string) error {
	if !isValidUUID(sessionID) {
		return fmt.Errorf("invalid session_id: %s", sessionID)
	}

	// Matches CLI guard — empty/whitespace titles are rejected rather than
	// overloaded as "clear title".
	stripped := strings.TrimSpace(title)
	if stripped == "" {
		return fmt.Errorf("title must be non-empty")
	}

	data := fmt.Sprintf(`{"type":"custom-title","customTitle":%q,"sessionId":%q}%s`,
		stripped, sessionID, "\n")

	return appendToSession(sessionID, data, directory)
}

// TagSession tags a session. Pass empty string to clear the tag.
//
// Appends a {"type":"tag","tag":<tag>,"sessionId":<id>} JSONL entry.
// ListSessions reads the LAST tag from the file tail — most recent wins.
// Passing empty string appends an empty-string tag entry which is treated
// as cleared.
//
// Tags are Unicode-sanitized before storing (removes zero-width chars,
// directional marks, private-use characters, etc.) for CLI filter
// compatibility.
//
// Parameters:
//   - sessionID: UUID of the session to tag.
//   - tag: Tag string, or empty string to clear. Leading/trailing whitespace
//     is stripped. Must be non-empty after sanitization and stripping
//     (unless clearing).
//   - directory: Project directory path (same semantics as ListSessions).
//     When empty, all project directories are searched for the session file.
//
// Returns an error if:
//   - sessionID is not a valid UUID
//   - tag is empty/whitespace-only after sanitization (and not clearing)
//   - the session file cannot be found
func TagSession(sessionID, tag, directory string) error {
	if !isValidUUID(sessionID) {
		return fmt.Errorf("invalid session_id: %s", sessionID)
	}

	var tagValue string
	if tag != "" {
		sanitized := sanitizeUnicode(tag)
		stripped := strings.TrimSpace(sanitized)
		if stripped == "" {
			return fmt.Errorf("tag must be non-empty (use empty string to clear)")
		}
		tagValue = stripped
	}
	// Empty tagValue means clear the tag

	data := fmt.Sprintf(`{"type":"tag","tag":%q,"sessionId":%q}%s`,
		tagValue, sessionID, "\n")

	return appendToSession(sessionID, data, directory)
}

// appendToSession appends data to an existing session file.
//
// Searches candidate paths and tries the append directly — no existence check.
// Uses O_WRONLY | O_APPEND (without O_CREAT) so the open fails with ENOENT
// for missing files, avoiding TOCTOU.
func appendToSession(sessionID, data, directory string) error {
	fileName := sessionID + ".jsonl"

	if directory != "" {
		canonical := normalizePath(directory)

		// Try the exact/prefix-matched project directory first.
		projectDir := findProjectDir(canonical)
		if projectDir != "" && tryAppend(filepath.Join(projectDir, fileName), data) {
			return nil
		}

		// Worktree fallback — matches ListSessions/GetSessionMessages.
		// Sessions may live under a different worktree root.
		worktreePaths := getWorktreePaths(canonical)
		for _, wt := range worktreePaths {
			if wt == canonical {
				continue // already tried above
			}
			wtProjectDir := findProjectDir(wt)
			if wtProjectDir != "" && tryAppend(filepath.Join(wtProjectDir, fileName), data) {
				return nil
			}
		}

		return fmt.Errorf("session %s not found in project directory for %s", sessionID, directory)
	}

	// No directory — search all project directories by trying each directly.
	projectsDir := getProjectsDir()
	dirents, err := os.ReadDir(projectsDir)
	if err != nil {
		return fmt.Errorf("session %s not found (no projects directory)", sessionID)
	}

	for _, entry := range dirents {
		if tryAppend(filepath.Join(projectsDir, entry.Name(), fileName), data) {
			return nil
		}
	}

	return fmt.Errorf("session %s not found in any project directory", sessionID)
}

// tryAppend tries appending to a path.
//
// Opens with O_WRONLY | O_APPEND (no O_CREAT), so the open fails with ENOENT
// if the file does not exist — no separate existence check.
//
// Returns true on successful write, false if the file does not exist or is 0-byte.
// A 0-byte .jsonl is a "session not here, keep searching" signal.
func tryAppend(path, data string) bool {
	// Open with append mode, no create
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return false
	}
	defer f.Close()

	// Check file size - skip empty files
	stat, err := f.Stat()
	if err != nil || stat.Size() == 0 {
		return false
	}

	_, err = f.WriteString(data)
	return err == nil
}

// ============================================================================
// DeleteSession (v0.1.50)
// ============================================================================

// DeleteSession deletes a session file and its subagent transcript directory.
//
// This is a hard delete — the {session_id}.jsonl file is removed permanently,
// along with the sibling {session_id}/ subdirectory that holds subagent
// transcripts (if it exists).
//
// Parameters:
//   - sessionID: UUID of the session to delete.
//   - directory: Project directory path (same semantics as ListSessions).
//     When empty, all project directories are searched for the session file.
//
// Returns an error if:
//   - sessionID is not a valid UUID
//   - the session file cannot be found
func DeleteSession(sessionID, directory string) error {
	if !isValidUUID(sessionID) {
		return fmt.Errorf("invalid session_id: %s", sessionID)
	}

	fileName := sessionID + ".jsonl"

	if directory != "" {
		canonical := normalizePath(directory)

		// Try the exact/prefix-matched project directory first.
		projectDir := findProjectDir(canonical)
		if projectDir != "" {
			filePath := filepath.Join(projectDir, fileName)
			if tryDeleteWithCascade(filePath, sessionID) {
				return nil
			}
		}

		// Worktree fallback
		worktreePaths := getWorktreePaths(canonical)
		for _, wt := range worktreePaths {
			if wt == canonical {
				continue
			}
			wtProjectDir := findProjectDir(wt)
			if wtProjectDir != "" {
				filePath := filepath.Join(wtProjectDir, fileName)
				if tryDeleteWithCascade(filePath, sessionID) {
					return nil
				}
			}
		}

		return fmt.Errorf("session %s not found in project directory for %s", sessionID, directory)
	}

	// No directory — search all project directories
	projectsDir := getProjectsDir()
	dirents, err := os.ReadDir(projectsDir)
	if err != nil {
		return fmt.Errorf("session %s not found (no projects directory)", sessionID)
	}

	for _, entry := range dirents {
		filePath := filepath.Join(projectsDir, entry.Name(), fileName)
		if tryDeleteWithCascade(filePath, sessionID) {
			return nil
		}
	}

	return fmt.Errorf("session %s not found in any project directory", sessionID)
}

// tryDeleteWithCascade tries to delete a session file and its subagent transcript directory.
// Returns true if deletion succeeded, false if file doesn't exist.
func tryDeleteWithCascade(path, sessionID string) bool {
	err := os.Remove(path)
	if err != nil {
		return false
	}

	// Subagent transcripts live in a sibling {session_id}/ dir; often absent.
	// Security: validate the subagent directory path to prevent path traversal attacks.
	subagentDir := filepath.Join(filepath.Dir(path), sessionID)

	// Security checks before deletion:
	// 1. Verify sessionID is a valid UUID (prevents path injection)
	if !isValidUUID(sessionID) {
		return true // Session file deleted, skip subagent dir
	}

	// 2. Verify subagentDir is within ~/.claude/projects (no path traversal outside config dir)
	projectsDir := getProjectsDir()
	absSubagentDir, err := filepath.Abs(subagentDir)
	if err != nil {
		return true // Cannot resolve path, skip deletion
	}
	absProjectsDir, err := filepath.Abs(projectsDir)
	if err != nil {
		return true // Cannot resolve projects dir, skip deletion
	}
	// Ensure subagentDir is under projectsDir (no path traversal)
	if !strings.HasPrefix(absSubagentDir, absProjectsDir+string(filepath.Separator)) {
		return true // Path outside ~/.claude/projects, skip deletion
	}

	// 3. Verify subagentDir is within the same parent directory (no sibling traversal)
	parentDir := filepath.Dir(path)
	relPath, err := filepath.Rel(parentDir, subagentDir)
	if err != nil || relPath != sessionID {
		return true // Invalid path, skip deletion
	}

	// 4. Check if directory exists and is actually a directory
	info, err := os.Stat(subagentDir)
	if err != nil || !info.IsDir() {
		return true // Not a directory or doesn't exist
	}

	// Safe to remove subagent directory
	_ = os.RemoveAll(subagentDir)
	return true
}

// ============================================================================
// ForkSession (v0.1.50)
// ============================================================================

// ForkSessionResult represents the result of a fork_session operation.
type ForkSessionResult struct {
	SessionID string `json:"session_id"`
}

// ForkSession creates a fork (copy) of a session with remapped UUIDs.
//
// The fork preserves the conversation structure but:
//   - Assigns new UUIDs to all messages
//   - Remaps parentUuid references
//   - Sets a new sessionId
//   - Adds forkedFrom field pointing to original
//   - Clears stale fields (teamName, agentName, slug)
//
// Parameters:
//   - sessionID: UUID of the session to fork.
//   - directory: Project directory path. When empty, searches all projects.
//   - upToMessageID: Optional UUID to slice the fork at that message (inclusive).
//   - title: Optional custom title for the fork. Default is original title + " (fork)".
//
// Returns ForkSessionResult with the new session ID, or an error if:
//   - sessionID is not a valid UUID
//   - upToMessageID is provided but not a valid UUID
//   - the session file cannot be found
//   - upToMessageID is not found in the session
func ForkSession(sessionID, directory string, upToMessageID *string, title *string) (*ForkSessionResult, error) {
	if !isValidUUID(sessionID) {
		return nil, fmt.Errorf("invalid session_id: %s", sessionID)
	}

	if upToMessageID != nil && !isValidUUID(*upToMessageID) {
		return nil, fmt.Errorf("invalid up_to_message_id: %s", *upToMessageID)
	}

	// Read original session content
	content, err := readSessionFile(sessionID, directory)
	if err != nil {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// Parse entries and collect content-replacements
	entries, contentReplacements := parseTranscriptEntriesWithReplacements(content, sessionID)

	// Find the slice point if upToMessageID is specified
	var sliceIndex int = -1
	if upToMessageID != nil {
		for i, entry := range entries {
			if uuid, ok := entry["uuid"].(string); ok && uuid == *upToMessageID {
				sliceIndex = i
				break
			}
		}
		if sliceIndex == -1 {
			return nil, fmt.Errorf("message %s not found in session", *upToMessageID)
		}
	}

	// Generate new session ID
	newSessionID := generateUUID()

	// Build UUID remapping table
	uuidMap := make(map[string]string)
	for _, entry := range entries {
		if originalUUID, ok := entry["uuid"].(string); ok {
			uuidMap[originalUUID] = generateUUID()
		}
	}

	// Fork entries
	var forkedEntries []map[string]interface{}
	for i, entry := range entries {
		if sliceIndex >= 0 && i > sliceIndex {
			break // Stop after slice point
		}

		forked := make(map[string]interface{})
		for k, v := range entry {
			// Skip stale fields
			if k == "teamName" || k == "agentName" || k == "slug" {
				continue
			}

			// Remap UUIDs
			if k == "uuid" {
				if originalUUID, ok := v.(string); ok {
					forked["uuid"] = uuidMap[originalUUID]
				}
				continue
			}
			if k == "parentUuid" {
				if originalUUID, ok := v.(string); ok {
					if originalUUID == "" {
						// Preserve empty parentUuid for root message
						forked["parentUuid"] = ""
					} else if newUUID, exists := uuidMap[originalUUID]; exists {
						forked["parentUuid"] = newUUID
					}
				}
				continue
			}
			if k == "sessionId" {
				forked["sessionId"] = newSessionID
				continue
			}

			// Add forkedFrom for user/assistant messages
			if entry["type"] == "user" || entry["type"] == "assistant" {
				if k == "message" {
					forked["forkedFrom"] = map[string]interface{}{
						"sessionId": sessionID,
					}
				}
			}

			forked[k] = v
		}
		forkedEntries = append(forkedEntries, forked)
	}

	// Determine fork title
	forkTitle := ""
	if title != nil && *title != "" {
		forkTitle = *title
	} else {
		// Default: find original title/summary and append " (fork)"
		originalTitle := extractLastJSONStringField(content, "customTitle")
		if originalTitle == "" {
			originalTitle = extractLastJSONStringField(content, "aiTitle")
		}
		if originalTitle == "" {
			originalTitle = extractFirstPromptFromHead(content[:min(len(content), LiteReadBufSize)])
		}
		if originalTitle != "" {
			forkTitle = originalTitle + " (fork)"
		}
	}

	// Add custom-title entry for the fork
	if forkTitle != "" {
		titleEntry := map[string]interface{}{
			"type":        "custom-title",
			"customTitle": forkTitle,
			"sessionId":   newSessionID,
		}
		forkedEntries = append(forkedEntries, titleEntry)
	}

	// Append content-replacement entry (if any) with the fork's sessionId
	if len(contentReplacements) > 0 {
		replacementEntry := map[string]interface{}{
			"type":         "content-replacement",
			"sessionId":    newSessionID,
			"replacements": contentReplacements,
		}
		forkedEntries = append(forkedEntries, replacementEntry)
	}

	// Write forked session file
	var lines []string
	for _, entry := range forkedEntries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		lines = append(lines, string(data))
	}

	forkContent := strings.Join(lines, "\n") + "\n"

	// Determine target directory
	targetDir := ""
	if directory != "" {
		canonical := normalizePath(directory)
		targetDir = findProjectDir(canonical)
		if targetDir == "" {
			// Try worktrees
			worktreePaths := getWorktreePaths(canonical)
			for _, wt := range worktreePaths {
				if wt != canonical {
					wtProjectDir := findProjectDir(wt)
					if wtProjectDir != "" {
						targetDir = wtProjectDir
						break
					}
				}
			}
		}
	}

	if targetDir == "" {
		// Find where the original session lives
		originalPath, err := findSessionFilePath(sessionID, directory)
		if err != nil {
			return nil, err
		}
		targetDir = filepath.Dir(originalPath)
	}

	// Write the fork file
	forkPath := filepath.Join(targetDir, newSessionID+".jsonl")
	if err := os.WriteFile(forkPath, []byte(forkContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write fork session: %w", err)
	}

	return &ForkSessionResult{SessionID: newSessionID}, nil
}

// findSessionFilePath finds the file path for a session.
func findSessionFilePath(sessionID, directory string) (string, error) {
	fileName := sessionID + ".jsonl"

	if directory != "" {
		canonical := normalizePath(directory)
		projectDir := findProjectDir(canonical)
		if projectDir != "" {
			path := filepath.Join(projectDir, fileName)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}

		worktreePaths := getWorktreePaths(canonical)
		for _, wt := range worktreePaths {
			if wt != canonical {
				wtProjectDir := findProjectDir(wt)
				if wtProjectDir != "" {
					path := filepath.Join(wtProjectDir, fileName)
					if _, err := os.Stat(path); err == nil {
						return path, nil
					}
				}
			}
		}

		return "", fmt.Errorf("session %s not found", sessionID)
	}

	// Search all project directories
	projectsDir := getProjectsDir()
	dirents, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	for _, entry := range dirents {
		path := filepath.Join(projectsDir, entry.Name(), fileName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("session %s not found", sessionID)
}

// generateUUID generates a new UUID string using crypto/rand.
func generateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based if crypto/rand fails (extremely unlikely)
		return time.Now().Format("20060102-150405-000000000000000000000000")[:36]
	}
	// Set version 4 and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// Unicode Sanitization
// ============================================================================

// sanitizeUnicode sanitizes a string by removing dangerous Unicode characters.
//
// Ported from TypeScript partiallySanitizeUnicode. Iteratively applies NFKC
// normalization and strips format/private-use/unassigned characters until
// no more changes occur (max 10 iterations).
func sanitizeUnicode(value string) string {
	current := value
	for i := 0; i < 10; i++ {
		previous := current

		// Apply NFKC normalization to handle composed character sequences
		current = strings.ToValidUTF8(current, "")

		// Strip dangerous Unicode characters
		current = stripDangerousUnicode(current)

		// Strip format, private use, and unassigned category characters
		current = stripUnicodeCategories(current)

		if current == previous {
			break
		}
	}
	return current
}

// stripDangerousUnicode removes known dangerous Unicode character ranges.
func stripDangerousUnicode(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		// Skip dangerous ranges
		switch {
		case r >= '\u200b' && r <= '\u200f': // Zero-width spaces, LTR/RTL marks
			continue
		case r >= '\u202a' && r <= '\u202e': // Directional formatting characters
			continue
		case r >= '\u2066' && r <= '\u2069': // Directional isolates
			continue
		case r == '\ufeff': // Byte order mark
			continue
		case r >= '\ue000' && r <= '\uf8ff': // BMP private use
			continue
		}
		b.WriteRune(r)
	}

	return b.String()
}

// stripUnicodeCategories removes characters in Cf (format), Co (private use), Cn (unassigned).
func stripUnicodeCategories(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		cat := unicodeCategory(r)
		if cat == "Cf" || cat == "Co" || cat == "Cn" {
			continue
		}
		b.WriteRune(r)
	}

	return b.String()
}

// unicodeCategory returns the Unicode category for a rune.
// This is a simplified implementation that checks common format characters.
func unicodeCategory(r rune) string {
	// Common format characters (Cf category)
	switch r {
	case '\u00AD', '\u034F', '\u1806', '\u180B', '\u180C', '\u180D', '\u180E',
		'\u200B', '\u200C', '\u200D', '\u200E', '\u200F',
		'\u202A', '\u202B', '\u202C', '\u202D', '\u202E',
		'\u2060', '\u2061', '\u2062', '\u2063', '\u2064', '\u2066', '\u2067', '\u2068', '\u2069',
		'\uFE00', '\uFE01', '\uFE02', '\uFE03', '\uFE04', '\uFE05', '\uFE06', '\uFE07',
		'\uFE08', '\uFE09', '\uFE0A', '\uFE0B', '\uFE0C', '\uFE0D', '\uFE0E', '\uFE0F',
		'\uFEFF', '\uFFF9', '\uFFFA', '\uFFFB':
		return "Cf"
	}

	// Private use areas (Co category) - Supplementary Private Use Area-A and B
	// Use numeric comparison to avoid invalid Unicode escape issues
	if r >= 0xE000 && r <= 0xF8FF {
		return "Co"
	}
	if r >= 0xF0000 && r <= 0x10FFFF {
		// This covers supplementary private use areas
		return "Co"
	}

	// Default: assume it's a regular character
	return "Lo"
}
