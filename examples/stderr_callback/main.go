// Example stderr_callback demonstrates handling stderr output in the Claude Agent SDK for Go.
//
// The SDK can output diagnostic information to stderr. This example shows
// how to capture and handle stderr output for logging, debugging, or display.
//
// This example shows:
// 1. Basic stderr callback configuration
// 2. Logging stderr to a file
// 3. Filtering and processing stderr output
// 4. Real-time stderr display
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	claude "github.com/unitsvc/claude-agent-sdk-golang"
	"github.com/unitsvc/claude-agent-sdk-golang/examples/internal"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// StderrLogger is a thread-safe stderr logger.
type StderrLogger struct {
	mu      sync.Mutex
	lines   []string
	enabled bool
}

// NewStderrLogger creates a new stderr logger.
func NewStderrLogger() *StderrLogger {
	return &StderrLogger{
		lines:   make([]string, 0),
		enabled: true,
	}
}

// Callback returns a function that can be used as the Stderr callback.
func (l *StderrLogger) Callback() func(string) {
	return func(line string) {
		if !l.enabled {
			return
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		l.lines = append(l.lines, line)
	}
}

// GetLines returns all captured stderr lines.
func (l *StderrLogger) GetLines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]string, len(l.lines))
	copy(result, l.lines)
	return result
}

// Disable stops capturing stderr.
func (l *StderrLogger) Disable() {
	l.enabled = false
}

// Enable starts capturing stderr.
func (l *StderrLogger) Enable() {
	l.enabled = true
}

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	fmt.Println("=== Claude Agent SDK Go - Stderr Callback Example ===")
	fmt.Println()

	// Example 1: Basic stderr callback
	basicStderrCallback(ctx)

	// Example 2: Logging stderr to a file
	fileLoggingExample(ctx)

	// Example 3: Filtering stderr output
	filteredStderrExample(ctx)

	// Example 4: Real-time stderr display
	realtimeStderrExample(ctx)
}

// basicStderrCallback demonstrates the simplest stderr callback.
func basicStderrCallback(ctx context.Context) {
	fmt.Println("--- Example 1: Basic Stderr Callback ---")
	fmt.Println("Capture stderr output using a simple callback.")
	fmt.Println()

	// Create a logger
	logger := NewStderrLogger()

	// Configure the client with stderr callback
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:  types.String(types.ModelSonnet),
		Stderr: logger.Callback(), // Capture stderr output
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Stderr callback configured.")
	fmt.Println("All stderr output from Claude CLI will be captured in the logger.")
	fmt.Println()

	// After queries, you can retrieve captured stderr
	// lines := logger.GetLines()
	// for _, line := range lines {
	//     fmt.Printf("STDERR: %s\n", line)
	// }
}

// fileLoggingExample demonstrates logging stderr to a file.
func fileLoggingExample(ctx context.Context) {
	fmt.Println("--- Example 2: Logging Stderr to File ---")
	fmt.Println("Write stderr output to a log file for debugging.")
	fmt.Println()

	// Create a log file
	logFile, err := os.CreateTemp("", "claude-stderr-*.log")
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return
	}
	defer logFile.Close()

	// Create a file-logging callback
	fileCallback := func(line string) {
		logFile.WriteString(line + "\n")
	}

	// Configure the client
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:  types.String(types.ModelSonnet),
		Stderr: fileCallback,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Printf("Stderr will be logged to: %s\n", logFile.Name())
	fmt.Println("This is useful for:")
	fmt.Println("  - Debugging SDK behavior")
	fmt.Println("  - Tracking CLI communication")
	fmt.Println("  - Auditing operations")
	fmt.Println()
}

// filteredStderrExample demonstrates filtering stderr output.
func filteredStderrExample(ctx context.Context) {
	fmt.Println("--- Example 3: Filtered Stderr Output ---")
	fmt.Println("Process and filter stderr output based on content.")
	fmt.Println()

	// Create a filtered callback
	var warningLines []string
	var errorLines []string
	var debugLines []string

	filteredCallback := func(line string) {
		lowerLine := strings.ToLower(line)

		// Categorize by content
		switch {
		case strings.Contains(lowerLine, "error"):
			errorLines = append(errorLines, line)
			fmt.Printf("[ERROR] %s\n", line)
		case strings.Contains(lowerLine, "warning"):
			warningLines = append(warningLines, line)
			fmt.Printf("[WARN] %s\n", line)
		case strings.Contains(lowerLine, "debug"):
			debugLines = append(debugLines, line)
			// Don't print debug lines by default
		default:
			// Regular output - could log or ignore
		}
	}

	// Configure the client
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:  types.String(types.ModelSonnet),
		Stderr: filteredCallback,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Filtered stderr callback configured.")
	fmt.Println("  - Errors are highlighted in red")
	fmt.Println("  - Warnings are highlighted in yellow")
	fmt.Println("  - Debug lines are filtered out")
	fmt.Println()

	// After queries, you can access categorized lines
	// fmt.Printf("Total errors: %d\n", len(errorLines))
	// fmt.Printf("Total warnings: %d\n", len(warningLines))
}

// realtimeStderrExample demonstrates real-time stderr display.
func realtimeStderrExample(ctx context.Context) {
	fmt.Println("--- Example 4: Real-Time Stderr Display ---")
	fmt.Println("Display stderr output in real-time with timestamps.")
	fmt.Println()

	// Create a real-time callback with timestamps
	realtimeCallback := func(line string) {
		if line == "" {
			return
		}
		timestamp := "12:00:00" // In real code: time.Now().Format("15:04:05")

		// Prefix with timestamp
		fmt.Printf("[%s] %s\n", timestamp, line)
	}

	// Configure the client
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:  types.String(types.ModelSonnet),
		Stderr: realtimeCallback,
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Real-time stderr display configured.")
	fmt.Println("Each line will be prefixed with a timestamp.")
	fmt.Println()
}

// Example: Combined stderr handler with multiple features
// This demonstrates a more sophisticated stderr handling approach.
func combinedStderrHandler(ctx context.Context) {
	fmt.Println("--- Bonus: Combined Stderr Handler ---")

	// Create a combined handler
	type LogLevel string
	const (
		LogLevelDebug   LogLevel = "DEBUG"
		LogLevelInfo    LogLevel = "INFO"
		LogLevelWarning LogLevel = "WARN"
		LogLevelError   LogLevel = "ERROR"
	)

	// Thread-safe combined handler
	handler := struct {
		logFile *os.File
		enabled bool
	}{
		enabled: true,
	}

	// Create log file
	logFile, err := os.CreateTemp("", "claude-combined-*.log")
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return
	}
	handler.logFile = logFile

	combinedCallback := func(line string) {
		if !handler.enabled || line == "" {
			return
		}

		// Determine log level
		var level LogLevel = LogLevelInfo
		lowerLine := strings.ToLower(line)
		switch {
		case strings.Contains(lowerLine, "error"):
			level = LogLevelError
		case strings.Contains(lowerLine, "warning"):
			level = LogLevelWarning
		case strings.Contains(lowerLine, "debug"):
			level = LogLevelDebug
		}

		// Format the log entry
		formatted := fmt.Sprintf("[%s] %s\n", level, line)

		// Write to file
		handler.logFile.WriteString(formatted)

		// Print to console based on level
		switch level {
		case LogLevelError:
			fmt.Printf("\033[31m%s\033[0m", formatted) // Red
		case LogLevelWarning:
			fmt.Printf("\033[33m%s\033[0m", formatted) // Yellow
		case LogLevelDebug:
			// Skip debug output on console
		default:
			fmt.Print(formatted)
		}
	}

	// Configure the client
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		Model:  types.String(types.ModelSonnet),
		Stderr: combinedCallback,
	})
	defer client.Close()
	defer handler.logFile.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Combined handler configured:")
	fmt.Println("  - Logs everything to file")
	fmt.Println("  - Errors displayed in red")
	fmt.Println("  - Warnings displayed in yellow")
	fmt.Println("  - Debug output suppressed on console")
	fmt.Printf("  - Log file: %s\n", handler.logFile.Name())
	fmt.Println()
}
