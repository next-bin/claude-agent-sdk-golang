// Package transport provides transport layer implementations for the Claude Agent SDK.
//
// This package is internal and not intended for direct use by SDK consumers.
// It handles communication with the Claude CLI subprocess.
package transportimpl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/next-bin/claude-agent-sdk-golang/errors"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// SDKVersion is the current version of the Go SDK.
const SDKVersion = "0.1.0"

// DefaultMaxBufferSize is the default maximum buffer size for JSON messages (1MB).
const DefaultMaxBufferSize = 1024 * 1024

// MinimumClaudeCodeVersion is the minimum required version of Claude Code CLI.
const MinimumClaudeCodeVersion = "2.0.0"

// Transport is the interface for communication with the Claude CLI.
// This is a low-level transport interface that handles raw I/O with the Claude
// process or service.
type Transport interface {
	// Connect connects the transport and prepares for communication.
	// For subprocess transports, this starts the process.
	Connect(ctx context.Context) error

	// Close closes the transport connection and cleans up resources.
	Close(ctx context.Context) error

	// Write writes raw data to the transport.
	Write(ctx context.Context, data string) error

	// ReadMessages returns a channel that yields parsed JSON messages from the transport.
	// The channel is closed when the transport is closed or an error occurs.
	ReadMessages(ctx context.Context) <-chan map[string]interface{}

	// EndInput ends the input stream (closes stdin for process transports).
	EndInput(ctx context.Context) error

	// IsReady checks if transport is ready for communication.
	IsReady() bool
}

// SubprocessCLITransport implements Transport using a Claude CLI subprocess.
type SubprocessCLITransport struct {
	// Configuration
	prompt  interface{} // string or channel for streaming
	options *types.ClaudeAgentOptions

	// Process management
	cliPath string
	cwd     string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader

	// Streams
	stdoutReader *bufio.Reader
	stderrReader *bufio.Reader

	// State
	ready     bool
	exitError error

	// Buffer management
	maxBufferSize int

	// Concurrency
	writeLock sync.Mutex
	closeOnce sync.Once

	// Context for goroutine cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Channels
	messageChan chan map[string]interface{}
	errorChan   chan error
	stderrChan  chan string

	// Version check flag
	skipVersionCheck bool
}

// SubprocessCLITransportOption is a functional option for SubprocessCLITransport.
type SubprocessCLITransportOption func(*SubprocessCLITransport)

// WithSkipVersionCheck skips the CLI version check.
func WithSkipVersionCheck(skip bool) SubprocessCLITransportOption {
	return func(t *SubprocessCLITransport) {
		t.skipVersionCheck = skip
	}
}

// NewSubprocessCLITransport creates a new subprocess transport.
func NewSubprocessCLITransport(prompt interface{}, options *types.ClaudeAgentOptions, opts ...SubprocessCLITransportOption) (*SubprocessCLITransport, error) {
	if options == nil {
		options = &types.ClaudeAgentOptions{}
	}

	t := &SubprocessCLITransport{
		prompt:        prompt,
		options:       options,
		maxBufferSize: DefaultMaxBufferSize,
		messageChan:   make(chan map[string]interface{}, 100),
		errorChan:     make(chan error, 1),
		stderrChan:    make(chan string, 100),
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	// Set CLI path
	if options.CLIPath != nil {
		switch v := options.CLIPath.(type) {
		case string:
			t.cliPath = v
		case *string:
			if v != nil {
				t.cliPath = *v
			}
		}
	}
	if t.cliPath == "" {
		cliPath, err := t.findCLI()
		if err != nil {
			return nil, err
		}
		t.cliPath = cliPath
	}

	// Set working directory
	if options.CWD != nil {
		switch v := options.CWD.(type) {
		case string:
			t.cwd = v
		case *string:
			if v != nil {
				t.cwd = *v
			}
		}
	}

	// Set max buffer size
	if options.MaxBufferSize != nil {
		t.maxBufferSize = *options.MaxBufferSize
	}

	return t, nil
}

// findCLI finds the Claude Code CLI binary.
func (t *SubprocessCLITransport) findCLI() (string, error) {
	// First, check for bundled CLI
	if bundledCLI := t.findBundledCLI(); bundledCLI != "" {
		return bundledCLI, nil
	}

	// Fall back to system-wide search
	cliPath, err := exec.LookPath("claude")
	if err == nil {
		return cliPath, nil
	}

	// Check common locations
	homeDir, err := os.UserHomeDir()
	if err == nil {
		locations := []string{
			filepath.Join(homeDir, ".npm-global/bin/claude"),
			"/usr/local/bin/claude",
			filepath.Join(homeDir, ".local/bin/claude"),
			filepath.Join(homeDir, "node_modules/.bin/claude"),
			filepath.Join(homeDir, ".yarn/bin/claude"),
			filepath.Join(homeDir, ".claude/local/claude"),
		}

		for _, path := range locations {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", errors.NewCLINotFoundError(
		"Claude Code not found. Install with:\n"+
			"  npm install -g @anthropic-ai/claude-code\n"+
			"\nIf already installed locally, try:\n"+
			"  export PATH=\"$HOME/node_modules/.bin:$PATH\"\n"+
			"\nOr provide the path via ClaudeAgentOptions:\n"+
			"  ClaudeAgentOptions{CLIPath: stringPtr(\"/path/to/claude\")}",
		"",
	)
}

// findBundledCLI finds bundled CLI binary if it exists.
func (t *SubprocessCLITransport) findBundledCLI() string {
	// Determine the CLI binary name based on platform
	cliName := "claude"
	if runtime.GOOS == "windows" {
		cliName = "claude.exe"
	}

	// Get the path to the bundled CLI
	// The bundled directory is relative to this module
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}

	// Check for bundled CLI in various locations relative to the executable
	bundledPaths := []string{
		filepath.Join(filepath.Dir(execPath), "_bundled", cliName),
		filepath.Join(filepath.Dir(execPath), "..", "_bundled", cliName),
	}

	for _, bundledPath := range bundledPaths {
		if _, err := os.Stat(bundledPath); err == nil {
			return bundledPath
		}
	}

	return ""
}

// buildSettingsValue builds settings value, merging sandbox settings if provided.
func (t *SubprocessCLITransport) buildSettingsValue() (string, bool) {
	hasSettings := t.options.Settings != nil && *t.options.Settings != ""
	hasSandbox := t.options.Sandbox != nil

	if !hasSettings && !hasSandbox {
		return "", false
	}

	// If only settings path and no sandbox, pass through as-is
	if hasSettings && !hasSandbox {
		return *t.options.Settings, true
	}

	// If we have sandbox settings, we need to merge into a JSON object
	settingsObj := make(map[string]interface{})

	if hasSettings {
		settingsStr := strings.TrimSpace(*t.options.Settings)
		// Check if settings is a JSON string or a file path
		if strings.HasPrefix(settingsStr, "{") && strings.HasSuffix(settingsStr, "}") {
			// Parse JSON string
			if err := json.Unmarshal([]byte(settingsStr), &settingsObj); err != nil {
				// If parsing fails, treat as file path - try to read the file
				settingsBytes, readErr := os.ReadFile(settingsStr)
				if readErr == nil {
					json.Unmarshal(settingsBytes, &settingsObj)
				}
			}
		} else {
			// It's a file path - read and parse
			settingsBytes, err := os.ReadFile(settingsStr)
			if err == nil {
				json.Unmarshal(settingsBytes, &settingsObj)
			}
		}
	}

	// Merge sandbox settings
	if hasSandbox {
		settingsObj["sandbox"] = t.options.Sandbox
	}

	settingsJSON, err := json.Marshal(settingsObj)
	if err != nil {
		return "", false
	}
	return string(settingsJSON), true
}

// buildCommand builds CLI command with arguments.
func (t *SubprocessCLITransport) buildCommand() []string {
	cmd := []string{t.cliPath, "--output-format", "stream-json", "--verbose"}

	// System prompt
	if t.options.SystemPrompt == nil {
		cmd = append(cmd, "--system-prompt", "")
	} else {
		switch v := t.options.SystemPrompt.(type) {
		case string:
			cmd = append(cmd, "--system-prompt", v)
		case types.SystemPromptPreset:
			if v.Append != nil && *v.Append != "" {
				cmd = append(cmd, "--append-system-prompt", *v.Append)
			}
		case types.SystemPromptFile:
			cmd = append(cmd, "--system-prompt-file", v.Path)
		case map[string]interface{}:
			if preset, ok := v["type"].(string); ok && preset == "preset" {
				if appendStr, ok := v["append"].(string); ok && appendStr != "" {
					cmd = append(cmd, "--append-system-prompt", appendStr)
				}
			} else if file, ok := v["type"].(string); ok && file == "file" {
				if path, ok := v["path"].(string); ok && path != "" {
					cmd = append(cmd, "--system-prompt-file", path)
				}
			}
		}
	}

	// Tools option (base set of tools)
	if t.options.Tools != nil {
		switch v := t.options.Tools.(type) {
		case []string:
			if len(v) == 0 {
				cmd = append(cmd, "--tools", "")
			} else {
				cmd = append(cmd, "--tools", strings.Join(v, ","))
			}
		case map[string]interface{}:
			// Preset object - 'claude_code' preset maps to 'default'
			cmd = append(cmd, "--tools", "default")
		}
	}

	if len(t.options.AllowedTools) > 0 {
		cmd = append(cmd, "--allowedTools", strings.Join(t.options.AllowedTools, ","))
	}

	if t.options.MaxTurns != nil {
		cmd = append(cmd, "--max-turns", fmt.Sprintf("%d", *t.options.MaxTurns))
	}

	if t.options.MaxBudgetUSD != nil {
		cmd = append(cmd, "--max-budget-usd", fmt.Sprintf("%f", *t.options.MaxBudgetUSD))
	}

	if len(t.options.DisallowedTools) > 0 {
		cmd = append(cmd, "--disallowedTools", strings.Join(t.options.DisallowedTools, ","))
	}

	if t.options.TaskBudget != nil {
		cmd = append(cmd, "--task-budget", fmt.Sprintf("%d", t.options.TaskBudget.Total))
	}

	if t.options.Model != nil {
		cmd = append(cmd, "--model", *t.options.Model)
	}

	if t.options.FallbackModel != nil {
		cmd = append(cmd, "--fallback-model", *t.options.FallbackModel)
	}

	if t.options.SessionID != nil && *t.options.SessionID != "" {
		cmd = append(cmd, "--session-id", *t.options.SessionID)
	}

	if len(t.options.Betas) > 0 {
		betas := make([]string, len(t.options.Betas))
		for i, b := range t.options.Betas {
			betas[i] = string(b)
		}
		cmd = append(cmd, "--betas", strings.Join(betas, ","))
	}

	if t.options.PermissionPromptToolName != nil {
		cmd = append(cmd, "--permission-prompt-tool", *t.options.PermissionPromptToolName)
	}

	if t.options.PermissionMode != nil {
		cmd = append(cmd, "--permission-mode", string(*t.options.PermissionMode))
	}

	if t.options.ContinueConversation {
		cmd = append(cmd, "--continue")
	}

	if t.options.Resume != nil {
		cmd = append(cmd, "--resume", *t.options.Resume)
	}

	// Handle settings and sandbox: merge sandbox into settings if both are provided
	if settingsValue, ok := t.buildSettingsValue(); ok {
		cmd = append(cmd, "--settings", settingsValue)
	}

	if len(t.options.AddDirs) > 0 {
		for _, dir := range t.options.AddDirs {
			var dirStr string
			switch v := dir.(type) {
			case string:
				dirStr = v
			case *string:
				if v != nil {
					dirStr = *v
				}
			}
			if dirStr != "" {
				cmd = append(cmd, "--add-dir", dirStr)
			}
		}
	}

	if t.options.MCPServers != nil {
		switch v := t.options.MCPServers.(type) {
		case map[string]types.McpServerConfig:
			// Handle typed map[string]McpServerConfig (recommended Go API)
			serversForCLI := make(map[string]interface{})
			for name, config := range v {
				// Convert McpServerConfig to map for JSON serialization
				configMap := config.ToMap()
				if serverType, hasType := configMap["type"]; hasType && serverType == "sdk" {
					// For SDK servers, pass everything except the instance field
					sdkConfig := make(map[string]interface{})
					for k, val := range configMap {
						if k != "instance" {
							sdkConfig[k] = val
						}
					}
					serversForCLI[name] = sdkConfig
				} else {
					serversForCLI[name] = configMap
				}
			}
			if len(serversForCLI) > 0 {
				mcpConfig := map[string]interface{}{"mcpServers": serversForCLI}
				mcpJSON, err := json.Marshal(mcpConfig)
				if err == nil {
					cmd = append(cmd, "--mcp-config", string(mcpJSON))
				}
			}
		case map[string]interface{}:
			// Process all servers, stripping instance field from SDK servers
			serversForCLI := make(map[string]interface{})
			for name, config := range v {
				if configMap, ok := config.(map[string]interface{}); ok {
					if serverType, hasType := configMap["type"]; hasType && serverType == "sdk" {
						// For SDK servers, pass everything except the instance field
						sdkConfig := make(map[string]interface{})
						for k, val := range configMap {
							if k != "instance" {
								sdkConfig[k] = val
							}
						}
						serversForCLI[name] = sdkConfig
					} else {
						// For external servers, pass as-is
						serversForCLI[name] = config
					}
				} else {
					serversForCLI[name] = config
				}
			}

			// Pass all servers to CLI
			if len(serversForCLI) > 0 {
				mcpConfig := map[string]interface{}{"mcpServers": serversForCLI}
				mcpJSON, err := json.Marshal(mcpConfig)
				if err == nil {
					cmd = append(cmd, "--mcp-config", string(mcpJSON))
				}
			}
		case string:
			cmd = append(cmd, "--mcp-config", v)
		}
	}

	if t.options.IncludePartialMessages {
		cmd = append(cmd, "--include-partial-messages")
	}

	if t.options.ForkSession {
		cmd = append(cmd, "--fork-session")
	}

	// Setting sources
	sourcesValue := ""
	if len(t.options.SettingSources) > 0 {
		sources := make([]string, len(t.options.SettingSources))
		for i, s := range t.options.SettingSources {
			sources[i] = string(s)
		}
		sourcesValue = strings.Join(sources, ",")
	}
	cmd = append(cmd, "--setting-sources", sourcesValue)

	// Add plugin directories
	for _, plugin := range t.options.Plugins {
		if plugin.Type == "local" {
			cmd = append(cmd, "--plugin-dir", plugin.Path)
		}
	}

	// Add extra args for future CLI flags
	for flag, value := range t.options.ExtraArgs {
		if value == nil {
			// Boolean flag without value
			cmd = append(cmd, fmt.Sprintf("--%s", flag))
		} else {
			// Flag with value
			cmd = append(cmd, fmt.Sprintf("--%s", flag), fmt.Sprintf("%v", value))
		}
	}

	// Resolve thinking config -> --thinking and --max-thinking-tokens
	// `thinking` takes precedence over the deprecated `max_thinking_tokens`
	resolvedMaxThinkingTokens := t.options.MaxThinkingTokens
	if t.options.Thinking != nil {
		switch v := t.options.Thinking.(type) {
		case types.ThinkingConfigAdaptive:
			cmd = append(cmd, "--thinking", "adaptive")
			if resolvedMaxThinkingTokens == nil {
				val := 32000
				resolvedMaxThinkingTokens = &val
			}
		case types.ThinkingConfigEnabled:
			// Note: "enabled" type does NOT use --thinking flag
			// Only passes --max-thinking-tokens
			resolvedMaxThinkingTokens = &v.BudgetTokens
		case types.ThinkingConfigDisabled:
			cmd = append(cmd, "--thinking", "disabled")
			val := 0
			resolvedMaxThinkingTokens = &val
		default:
			// Handle generic ThinkingConfig interface by checking GetType()
			switch t.options.Thinking.GetType() {
			case "adaptive":
				cmd = append(cmd, "--thinking", "adaptive")
				if resolvedMaxThinkingTokens == nil {
					val := 32000
					resolvedMaxThinkingTokens = &val
				}
			case "disabled":
				cmd = append(cmd, "--thinking", "disabled")
				val := 0
				resolvedMaxThinkingTokens = &val
			}
		}
	}
	if resolvedMaxThinkingTokens != nil {
		cmd = append(cmd, "--max-thinking-tokens", fmt.Sprintf("%d", *resolvedMaxThinkingTokens))
	}

	if t.options.Effort != nil {
		cmd = append(cmd, "--effort", *t.options.Effort)
	}

	// Extract schema from output_format structure if provided
	// Expected: {"type": "json_schema", "schema": {...}}
	if len(t.options.OutputFormat) > 0 {
		if of := t.options.OutputFormat; of["type"] == "json_schema" {
			if schema, hasSchema := of["schema"]; hasSchema {
				schemaJSON, err := json.Marshal(schema)
				if err == nil {
					cmd = append(cmd, "--json-schema", string(schemaJSON))
				}
			}
		}
	}

	// Always use streaming mode with stdin (matching TypeScript SDK)
	// This allows agents and other large configs to be sent via initialize request
	cmd = append(cmd, "--input-format", "stream-json")

	return cmd
}

// Connect starts the subprocess.
func (t *SubprocessCLITransport) Connect(ctx context.Context) error {
	if t.cmd != nil {
		return nil
	}

	// Create cancellable context for goroutine lifecycle management
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// Check CLI version unless skipped
	if !t.skipVersionCheck && os.Getenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK") == "" {
		if err := t.checkClaudeVersion(ctx); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	cmd := t.buildCommand()

	// Create the command
	t.cmd = exec.CommandContext(ctx, cmd[0], cmd[1:]...)

	// Set working directory
	if t.cwd != "" {
		t.cmd.Dir = t.cwd
	}

	// Merge environment variables. CLAUDE_CODE_ENTRYPOINT defaults to
	// sdk-go regardless of inherited process env; options.env can override it.
	// CLAUDE_AGENT_SDK_VERSION is always set by the SDK.
	// Start with system environment but filter out CLAUDECODE to allow nested SDK sessions
	envs := os.Environ()
	// Pre-allocate with extra capacity for custom env vars
	processEnv := make([]string, 0, len(envs)+5+len(t.options.Env))
	for _, env := range envs {
		// Skip CLAUDECODE to allow running SDK inside Claude Code session
		if strings.HasPrefix(env, "CLAUDECODE=") {
			continue
		}
		processEnv = append(processEnv, env)
	}
	// Set default entrypoint before user options so it can be overridden
	processEnv = append(processEnv, fmt.Sprintf("CLAUDE_CODE_ENTRYPOINT=%s", "sdk-go"))
	for k, v := range t.options.Env {
		processEnv = append(processEnv, fmt.Sprintf("%s=%s", k, v))
	}
	processEnv = append(processEnv, fmt.Sprintf("CLAUDE_AGENT_SDK_VERSION=%s", SDKVersion))

	// Enable file checkpointing if requested
	if t.options.EnableFileCheckpointing {
		processEnv = append(processEnv, "CLAUDE_CODE_ENABLE_SDK_FILE_CHECKPOINTING=true")
	}

	// Enable fine-grained tool streaming when partial messages are requested.
	// --include-partial-messages emits stream_event messages, but tool input
	// parameters are still buffered by the API unless eager_input_streaming is
	// also enabled at the per-tool level via this env var.
	// User-supplied value in options.Env takes precedence (setdefault pattern).
	if t.options.IncludePartialMessages {
		hasUserFGTS := false
		for _, env := range processEnv {
			if strings.HasPrefix(env, "CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=") {
				hasUserFGTS = true
				break
			}
		}
		if !hasUserFGTS {
			processEnv = append(processEnv, "CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=1")
		}
	}

	// Set PWD for cwd if specified
	if t.cwd != "" {
		processEnv = append(processEnv, fmt.Sprintf("PWD=%s", t.cwd))
	}

	t.cmd.Env = processEnv

	// Set up pipes
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		t.exitError = errors.NewCLIConnectionError("failed to create stdin pipe", err)
		return t.exitError
	}
	t.stdin = stdin

	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		t.exitError = errors.NewCLIConnectionError("failed to create stdout pipe", err)
		return t.exitError
	}
	t.stdout = stdout
	t.stdoutReader = bufio.NewReader(stdout)

	// Pipe stderr if we have a callback OR debug mode is enabled
	shouldPipeStderr := t.options.Stderr != nil || t.hasExtraArg("debug-to-stderr")
	if shouldPipeStderr {
		stderr, err := t.cmd.StderrPipe()
		if err != nil {
			t.exitError = errors.NewCLIConnectionError("failed to create stderr pipe", err)
			return t.exitError
		}
		t.stderr = stderr
		t.stderrReader = bufio.NewReader(stderr)

		// Start goroutine to read stderr
		go t.handleStderr()
	}

	// Start the process
	if err := t.cmd.Start(); err != nil {
		// Check if the error comes from the working directory or the CLI
		if t.cwd != "" {
			if _, statErr := os.Stat(t.cwd); os.IsNotExist(statErr) {
				t.exitError = errors.NewCLIConnectionError(fmt.Sprintf("Working directory does not exist: %s", t.cwd), err)
				return t.exitError
			}
		}
		t.exitError = errors.NewCLINotFoundError(fmt.Sprintf("Claude Code not found at: %s", t.cliPath), t.cliPath)
		return t.exitError
	}

	t.ready = true

	// Start goroutine to read messages
	go t.readMessagesLoop()

	return nil
}

// hasExtraArg checks if an extra arg is set.
func (t *SubprocessCLITransport) hasExtraArg(name string) bool {
	if t.options.ExtraArgs == nil {
		return false
	}
	_, exists := t.options.ExtraArgs[name]
	return exists
}

// handleStderr handles stderr stream - read and invoke callbacks.
func (t *SubprocessCLITransport) handleStderr() {
	if t.stderrReader == nil {
		return
	}

	scanner := bufio.NewScanner(t.stderrReader)
	for scanner.Scan() {
		// Check for context cancellation
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Call the stderr callback if provided
		if t.options.Stderr != nil {
			t.options.Stderr(line)
		}

		// Send to stderr channel for debugging
		select {
		case t.stderrChan <- line:
		default:
			// Channel full, drop message
		}
	}
}

// readMessagesLoop reads and parses messages from stdout.
func (t *SubprocessCLITransport) readMessagesLoop() {
	defer close(t.messageChan)

	if t.stdoutReader == nil {
		return
	}

	var jsonBuffer strings.Builder

	scanner := bufio.NewScanner(t.stdoutReader)
	// Set a larger buffer size for the scanner
	scanner.Buffer(make([]byte, t.maxBufferSize), t.maxBufferSize)

	for scanner.Scan() {
		// Check for context cancellation
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Accumulate partial JSON until we can parse it
		// Note: bufio.Scanner can truncate long lines, so we need to buffer
		// and speculatively parse until we get a complete JSON object
		jsonLines := strings.Split(line, "\n")

		for _, jsonLine := range jsonLines {
			jsonLine = strings.TrimSpace(jsonLine)
			if jsonLine == "" {
				continue
			}

			// Keep accumulating partial JSON until we can parse it
			jsonBuffer.WriteString(jsonLine)

			if jsonBuffer.Len() > t.maxBufferSize {
				bufferLength := jsonBuffer.Len()
				jsonBuffer.Reset()
				select {
				case t.errorChan <- errors.NewCLIJSONDecodeError(
					fmt.Sprintf("JSON message exceeded maximum buffer size of %d bytes", t.maxBufferSize),
					fmt.Errorf("buffer size %d exceeds limit %d", bufferLength, t.maxBufferSize),
				):
				default:
				}
				return
			}

			// Try to parse the JSON
			bufferStr := jsonBuffer.String()
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(bufferStr), &data); err == nil {
				jsonBuffer.Reset()
				select {
				case t.messageChan <- data:
				case <-t.ctx.Done():
					// Context cancelled, exit
					return
				case <-t.errorChan:
					// Error channel closed, exit
					return
				}
			}
			// If JSON decode fails, continue accumulating
		}
	}

	// Note: Don't call cmd.Wait() here - let Close() handle process cleanup
	// This prevents race conditions when Close() is called while this loop is exiting
}

// checkClaudeVersion checks Claude Code version and warns if below minimum.
func (t *SubprocessCLITransport) checkClaudeVersion(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.cliPath, "-v")
	output, err := cmd.Output()
	if err != nil {
		return nil // Ignore version check errors
	}

	versionOutput := strings.TrimSpace(string(output))

	// Extract version number
	re := regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+)`)
	match := re.FindStringSubmatch(versionOutput)
	if match == nil {
		return nil
	}

	version := match[1]
	versionParts := strings.Split(version, ".")
	minParts := strings.Split(MinimumClaudeCodeVersion, ".")

	if len(versionParts) >= 3 && len(minParts) >= 3 {
		for i := 0; i < 3; i++ {
			var vPart, mPart int
			fmt.Sscanf(versionParts[i], "%d", &vPart)
			fmt.Sscanf(minParts[i], "%d", &mPart)

			if vPart < mPart {
				return fmt.Errorf("warning: Claude Code version %s is unsupported in the Agent SDK. "+
					"Minimum required version is %s. Some features may not work correctly", version, MinimumClaudeCodeVersion)
			} else if vPart > mPart {
				break
			}
		}
	}

	return nil
}

// Close closes the transport and cleans up resources.
func (t *SubprocessCLITransport) Close(ctx context.Context) error {
	var err error

	t.closeOnce.Do(func() {
		// Cancel context to signal goroutines to stop
		if t.cancel != nil {
			t.cancel()
		}

		// Set ready to false inside lock to prevent race with write()
		t.writeLock.Lock()
		t.ready = false
		t.writeLock.Unlock()

		// Close stdin
		if t.stdin != nil {
			t.stdin.Close()
			t.stdin = nil
		}

		// Close stdout
		if t.stdout != nil {
			if closer, ok := t.stdout.(io.Closer); ok {
				closer.Close()
			}
			t.stdout = nil
		}

		// Close stderr
		if t.stderr != nil {
			if closer, ok := t.stderr.(io.Closer); ok {
				closer.Close()
			}
			t.stderr = nil
		}

		// Wait for graceful shutdown after stdin EOF, then terminate if needed.
		// The subprocess needs time to flush its session file after receiving
		// EOF on stdin. Without this grace period, SIGTERM can interrupt the
		// write and cause the last assistant message to be lost (see #625).
		if t.cmd != nil && t.cmd.Process != nil {
			process := t.cmd.Process

			// Wait for process with timeout to prevent indefinite blocking
			done := make(chan struct{})
			go func() {
				t.cmd.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Process exited gracefully after stdin EOF
			case <-time.After(5 * time.Second):
				// Graceful shutdown timed out - force terminate
				process.Kill()
				// Wait for kill to complete
				select {
				case <-done:
				case <-time.After(1 * time.Second):
					// Timeout waiting for kill, continue cleanup
				}
			}
		}

		t.cmd = nil
		t.stdoutReader = nil
		t.stderrReader = nil
		t.exitError = nil
	})

	return err
}

// Write writes raw data to the transport.
func (t *SubprocessCLITransport) Write(ctx context.Context, data string) error {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()

	// All checks inside lock to prevent TOCTOU races with close()/end_input()
	if !t.ready || t.stdin == nil {
		return errors.NewCLIConnectionError("ProcessTransport is not ready for writing", nil)
	}

	if t.cmd != nil && t.cmd.ProcessState != nil && t.cmd.ProcessState.Exited() {
		exitCode := t.cmd.ProcessState.ExitCode()
		return errors.NewCLIConnectionError(
			fmt.Sprintf("Cannot write to terminated process (exit code: %d)", exitCode),
			nil,
		)
	}

	if t.exitError != nil {
		return errors.NewCLIConnectionError(
			fmt.Sprintf("Cannot write to process that exited with error: %v", t.exitError),
			t.exitError,
		)
	}

	_, err := t.stdin.Write([]byte(data))
	if err != nil {
		t.ready = false
		t.exitError = errors.NewCLIConnectionError("Failed to write to process stdin", err)
		return t.exitError
	}

	return nil
}

// EndInput ends the input stream (closes stdin).
func (t *SubprocessCLITransport) EndInput(ctx context.Context) error {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()

	if t.stdin != nil {
		err := t.stdin.Close()
		t.stdin = nil
		return err
	}
	return nil
}

// ReadMessages returns a channel that yields parsed JSON messages from the transport.
func (t *SubprocessCLITransport) ReadMessages(ctx context.Context) <-chan map[string]interface{} {
	return t.messageChan
}

// IsReady checks if transport is ready for communication.
func (t *SubprocessCLITransport) IsReady() bool {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()
	return t.ready
}

// GetStderrChan returns a channel for receiving stderr output.
func (t *SubprocessCLITransport) GetStderrChan() <-chan string {
	return t.stderrChan
}

// GetExitError returns the exit error if the process exited with an error.
func (t *SubprocessCLITransport) GetExitError() error {
	return t.exitError
}

// GetCLIPath returns the path to the CLI binary.
func (t *SubprocessCLITransport) GetCLIPath() string {
	return t.cliPath
}
