# Claude Agent SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/unitsvc/claude-agent-sdk-golang.svg)](https://pkg.go.dev/github.com/unitsvc/claude-agent-sdk-golang)
[![Go Report Card](https://goreportcard.com/badge/github.com/unitsvc/claude-agent-sdk-golang)](https://goreportcard.com/report/github.com/unitsvc/claude-agent-sdk-golang)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[中文文档](README-zh.md)

A Go SDK for building AI agents with Claude. This SDK provides a Go implementation of the [Claude Agent SDK](https://github.com/anthropics/claude-agent-sdk-python), enabling you to build AI agents that can use tools, handle permissions, and interact with MCP servers.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Core Features](#core-features)
- [Advanced Topics](#advanced-topics)
- [API Reference](#api-reference)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Testing](#testing)
- [Performance](#performance)
- [Security](#security)
- [Best Practices](#best-practices)
- [FAQ](#faq)
- [Troubleshooting](#troubleshooting)
- [Migration Guide](#migration-guide)
- [Changelog](#changelog)

## Features

| Feature | Description |
|---------|-------------|
| 🔄 **Full API Compatibility** | Compatible with Python SDK v0.1.50 |
| 📡 **Streaming Messages** | Real-time message streaming via Go channels |
| 🔌 **MCP Server Support** | Stdio, SSE, HTTP, and in-process SDK MCP servers |
| 🪝 **Hook System** | 12 hook events for tool lifecycle management |
| 🔐 **Permission Control** | Fine-grained tool permission callbacks |
| 💾 **Sessions API** | List, query, rename, and tag conversation sessions |
| 🎯 **Type Safety** | Compile-time type checking with Go generics |
| ⚡ **Concurrency** | Native goroutine + channel patterns |
| 📊 **Cost Tracking** | Built-in usage and cost tracking |
| 🛠️ **Custom Tools** | Define custom tools with JSON Schema validation |

## Prerequisites

- **Go 1.21+** (for generics support)
- **Claude Code CLI** installed and authenticated:

```bash
# Install Claude Code CLI
npm install -g @anthropic-ai/claude-code

# Authenticate with Anthropic
claude login
```

### Verify Installation

```bash
# Check Go version
go version  # Should be 1.21 or higher

# Check Claude CLI
claude --version
```

## Installation

```bash
go get github.com/unitsvc/claude-agent-sdk-golang
```

### Go Modules

```go
import claude "github.com/unitsvc/claude-agent-sdk-golang"
```

### Version Pinning

```go
// go.mod
require github.com/unitsvc/claude-agent-sdk-golang v0.1.50
```

## Quick Start

### Simple Query

The simplest way to interact with Claude:

```go
package main

import (
    "context"
    "fmt"
    "log"

    claude "github.com/unitsvc/claude-agent-sdk-golang"
    "github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
    ctx := context.Background()

    // One-shot query - creates client, sends query, and closes automatically
    msgChan, err := claude.Query(ctx, "What is 2+2?", nil)
    if err != nil {
        log.Fatal(err)
    }

    // Process streaming messages
    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.ResultMessage:
            if m.Result != nil {
                fmt.Printf("Result: %s\n", *m.Result)
            }
            fmt.Printf("Duration: %dms, Turns: %d\n", m.DurationMS, m.NumTurns)
            if m.TotalCostUSD != nil {
                fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
            }
        case *types.AssistantMessage:
            for _, block := range m.Content {
                if tb, ok := block.(types.TextBlock); ok {
                    fmt.Printf("Assistant: %s\n", tb.Text)
                }
            }
        }
    }
}
```

### Client with Options

For more control and multiple queries:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    claude "github.com/unitsvc/claude-agent-sdk-golang"
    "github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
    // Create context that cancels on Ctrl+C
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Create client with custom options
    client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
        Model:        types.String(types.ModelSonnet),
        MaxTurns:     types.Int(5),
        MaxBudgetUSD: types.Float64(1.0),
    })
    defer client.Close()

    // Connect to Claude
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    // Send a query
    msgChan, err := client.Query(ctx, "Write a haiku about programming.")
    if err != nil {
        log.Fatal(err)
    }

    // Process messages
    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.AssistantMessage:
            for _, block := range m.Content {
                if tb, ok := block.(types.TextBlock); ok {
                    fmt.Print(tb.Text)
                }
            }
        case *types.ResultMessage:
            fmt.Printf("\n\n---\nSession: %s\n", m.SessionID)
            if m.TotalCostUSD != nil {
                fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
            }
        }
    }
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Your Application                          │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Query()   │  │   Client    │  │     Sessions API        │  │
│  │   (Simple)  │  │ (Advanced)  │  │ ListSessions, etc.      │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘  │
├─────────┴────────────────┴─────────────────────┴───────────────┤
│                        SDK Core                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Message   │  │    Hook     │  │    Permission           │  │
│  │   Parser    │  │   System    │  │    Manager              │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                      Transport Layer                             │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              SubprocessCLITransport                      │    │
│  │         (Communication with Claude CLI)                  │    │
│  └─────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────┤
│                       Claude Code CLI                            │
│                    (Official Anthropic CLI)                      │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Description |
|-----------|-------------|
| **Query()** | Simple one-shot query function |
| **Client** | Full-featured client for interactive sessions |
| **Message Parser** | Parses JSONL messages from Claude CLI |
| **Hook System** | Event-driven callbacks for tool lifecycle |
| **Permission Manager** | Controls tool execution permissions |
| **Transport Layer** | Handles subprocess communication with Claude CLI |
| **Sessions API** | Manages conversation history |

### Project Structure

```
claude-agent-sdk-golang/
├── client.go              # Client implementation
├── query.go               # Query function
├── sdk.go                 # Public API exports
├── types/
│   └── types.go           # Type definitions
├── errors/
│   └── errors.go          # Error types
├── internal/
│   ├── messageparser/     # JSONL message parsing
│   ├── query/             # Query implementation
│   ├── sessions/          # Sessions API
│   └── transport/         # CLI transport layer
├── sdkmcp/
│   └── server.go          # SDK MCP server
└── examples/
    └── ...                # Usage examples
```

## Core Features

### Streaming Messages

The SDK uses Go channels for real-time message streaming:

```go
for msg := range msgChan {
    switch m := msg.(type) {
    case *types.AssistantMessage:
        // Streaming text - may arrive in multiple chunks
        for _, block := range m.Content {
            switch b := block.(type) {
            case types.TextBlock:
                fmt.Print(b.Text)  // Text content
            case types.ThinkingBlock:
                fmt.Printf("[Thinking: %s]\n", b.Thinking)  // Extended thinking
            case types.ToolUseBlock:
                fmt.Printf("[Calling tool: %s]\n", b.Name)  // Tool invocation
            }
        }
    case *types.ResultMessage:
        // Final result - contains summary info
        fmt.Printf("Session: %s\n", m.SessionID)
        fmt.Printf("Duration: %dms\n", m.DurationMS)
        fmt.Printf("Turns: %d\n", m.NumTurns)
        if m.TotalCostUSD != nil {
            fmt.Printf("Cost: $%.6f\n", *m.TotalCostUSD)
        }
        if m.StopReason != nil {
            fmt.Printf("Stop reason: %s\n", *m.StopReason)
        }
    }
}
```

### Permission Handling

Control tool execution with fine-grained permissions:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
        // Log tool usage
        log.Printf("Tool: %s, Input: %v", toolName, input)

        switch toolName {
        case "Bash":
            // Auto-approve safe commands
            if cmd, ok := input["command"].(string); ok {
                if strings.HasPrefix(cmd, "git ") || strings.HasPrefix(cmd, "go ") {
                    return types.PermissionResultAllow{Behavior: "allow"}, nil
                }
            }
            // Ask for confirmation for other commands
            return types.PermissionResultDeny{
                Behavior: "deny",
                Message:  "Command requires manual approval",
            }, nil

        case "Write":
            // Redirect writes to a sandbox directory
            if path, ok := input["file_path"].(string); ok {
                safePath := filepath.Join("/sandbox", filepath.Base(path))
                return types.PermissionResultAllow{
                    Behavior: "allow",
                    UpdatedInput: map[string]interface{}{
                        "file_path": safePath,
                        "content":   input["content"],
                    },
                }, nil
            }

        case "Read":
            // Allow all reads
            return types.PermissionResultAllow{Behavior: "allow"}, nil

        default:
            // Deny unknown tools
            return types.PermissionResultDeny{
                Behavior: "deny",
                Message:  fmt.Sprintf("Tool %s is not allowed", toolName),
            }, nil
        }
    },
})
```

### MCP Servers

Configure external MCP servers:

```go
// Stdio MCP server (local process)
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "filesystem": types.McpStdioServerConfig{
            Command: "mcp-filesystem-server",
            Args:    []string{"/allowed/path"},
            Env:     map[string]string{"DEBUG": "1"},
        },
        "database": types.McpStdioServerConfig{
            Command: "mcp-postgres-server",
            Args:    []string{"postgresql://localhost/mydb"},
        },
    },
})

// SSE MCP server (remote)
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "remote": types.McpSSEServerConfig{
            URL: "https://api.example.com/mcp/sse",
            Headers: map[string]string{
                "Authorization": "Bearer token",
            },
        },
    },
})

// HTTP MCP server
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "api": types.McpHttpServerConfig{
            URL: "https://api.example.com/mcp",
        },
    },
})
```

### SDK MCP Server (In-Process)

Create custom tools without external processes:

```go
import "github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"

// Define a calculator tool
calculator := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{
    sdkmcp.Tool("add", "Add two numbers",
        sdkmcp.Schema(map[string]interface{}{
            "a": sdkmcp.NumberProperty("First number"),
            "b": sdkmcp.NumberProperty("Second number"),
        }, []string{"a", "b"}),
        func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
            a, _ := args["a"].(float64)
            b, _ := args["b"].(float64)
            return sdkmcp.TextResult(fmt.Sprintf("%.2f", a+b)), nil
        }),

    sdkmcp.Tool("multiply", "Multiply two numbers",
        sdkmcp.Schema(map[string]interface{}{
            "a": sdkmcp.NumberProperty("First number"),
            "b": sdkmcp.NumberProperty("Second number"),
        }, []string{"a", "b"}),
        func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
            a, _ := args["a"].(float64)
            b, _ := args["b"].(float64)
            return sdkmcp.TextResult(fmt.Sprintf("%.2f", a*b)), nil
        }),
})

// Use the tool with client
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "calc": types.McpSdkServerConfig{
            Type:     "sdk",
            Name:     "calculator",
            Instance: calculator,
        },
    },
    AllowedTools: []string{"mcp__calc__add", "mcp__calc__multiply"},
})
```

### Hooks

Register callbacks for tool lifecycle events:

```go
// Define a hook callback
type LoggingHook struct{}

func (h *LoggingHook) Call(input interface{}) (interface{}, error) {
    switch i := input.(type) {
    case types.PreToolUseHookInput:
        log.Printf("[PRE] Tool: %s", i.ToolName)
        log.Printf("[PRE] Input: %v", i.ToolInput)
    case types.PostToolUseHookInput:
        log.Printf("[POST] Tool: %s", i.ToolName)
        log.Printf("[POST] Result: %v", i.ToolResult)
    }
    return nil, nil // Return nil to continue, error to block
}

// Register hooks
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Hooks: map[types.HookEvent][]types.HookMatcher{
        types.HookEventPreToolUse: {
            {
                Matcher: "Bash",  // Regex pattern
                Hooks: []types.HookCallback{&LoggingHook{}},
            },
        },
        types.HookEventPostToolUse: {
            {
                Matcher: ".*",  // Match all tools
                Hooks: []types.HookCallback{&LoggingHook{}},
            },
        },
    },
})
```

#### Hook Events

| Event | When Fired | Input Type |
|-------|------------|------------|
| `HookEventPreToolUse` | Before tool execution | `PreToolUseHookInput` |
| `HookEventPostToolUse` | After successful tool execution | `PostToolUseHookInput` |
| `HookEventPostToolUseFailure` | After failed tool execution | `PostToolUseFailureHookInput` |
| `HookEventUserPromptSubmit` | When user submits a prompt | `UserPromptSubmitHookInput` |
| `HookEventStop` | When conversation stops | `StopHookInput` |
| `HookEventSubagentStart` | When subagent starts | `SubagentStartHookInput` |
| `HookEventSubagentStop` | When subagent stops | `SubagentStopHookInput` |
| `HookEventPreCompact` | Before context compaction | `PreCompactHookInput` |
| `HookEventNotification` | For notifications | `NotificationHookInput` |
| `HookEventPermissionRequest` | For permission requests | `PermissionRequestHookInput` |
| `HookEventSessionStart` | When session starts | `SessionStartHookInput` |
| `HookEventSessionEnd` | When session ends | `SessionEndHookInput` |

### Sessions API

Manage conversation history:

```go
// List sessions for a project
sessions, err := claude.ListSessions("/path/to/project", 10, true)
if err != nil {
    log.Fatal(err)
}

for _, sess := range sessions {
    fmt.Printf("Session: %s\n", sess.SessionID)
    fmt.Printf("  Summary: %s\n", sess.Summary)
    fmt.Printf("  Modified: %s\n", time.UnixMilli(sess.LastModified).Format(time.RFC3339))

    if sess.CustomTitle != nil {
        fmt.Printf("  Title: %s\n", *sess.CustomTitle)
    }
    if sess.Tag != nil {
        fmt.Printf("  Tag: %s\n", *sess.Tag)
    }
    if sess.CreatedAt != nil {
        fmt.Printf("  Created: %s\n", time.UnixMilli(*sess.CreatedAt).Format(time.RFC3339))
    }
}

// Get single session info (no directory scan)
info := claude.GetSessionInfo("550e8400-e29b-41d4-a716-446655440000", "/path/to/project")
if info != nil {
    fmt.Printf("Session: %s\n", info.Summary)
}

// Get messages from a session
messages, err := claude.GetSessionMessages(
    "550e8400-e29b-41d4-a716-446655440000",
    "/path/to/project",
    10,   // limit (0 = no limit)
    0,    // offset
)
for _, msg := range messages {
    fmt.Printf("[%s] %v\n", msg.Type, msg.Message)
}

// Rename a session
err := claude.RenameSession(
    "550e8400-e29b-41d4-a716-446655440000",
    "My Important Session",
    "/path/to/project",
)

// Tag a session
err := claude.TagSession(
    "550e8400-e29b-41d4-a716-446655440000",
    "important",
    "/path/to/project",
)
```

### Permission Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `PermissionModeDefault` | Prompts for permissions | Interactive applications |
| `PermissionModeAcceptEdits` | Auto-accept file edits | Code editing tools |
| `PermissionModePlan` | Planning mode | Complex multi-step tasks |
| `PermissionModeBypassPermissions` | Bypass all permissions | Trusted environments only |

```go
// Accept edits automatically
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
})
```

### Custom System Prompt

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    SystemPrompt: `You are a helpful coding assistant specialized in Go.

Follow these guidelines:
1. Always use idiomatic Go code
2. Prefer standard library when possible
3. Include error handling in examples
4. Add comments for complex logic`,
})
```

### Agents

Define specialized agents for different tasks:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Agents: []types.AgentDefinition{
        {
            Name:        "go-expert",
            Description: "Go programming expert for writing and reviewing code",
            Prompt:      "You are a Go programming expert with deep knowledge of the standard library, best practices, and common patterns.",
            Tools:       []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep"},
            Model:       types.String(types.ModelSonnet),
        },
        {
            Name:        "security-reviewer",
            Description: "Security-focused code reviewer",
            Prompt:      "You are a security expert focused on identifying vulnerabilities and suggesting fixes.",
            Tools:       []string{"Read", "Grep"},
            Model:       types.String(types.ModelOpus),
        },
    },
})
```

### Fine-Grained Tool Streaming

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: types.Bool(true),
})
```

When enabled, tool input deltas stream in real-time, allowing progressive UI updates.

## Advanced Topics

### Concurrent Queries

Handle multiple queries concurrently:

```go
func processQuery(ctx context.Context, prompt string) error {
    msgChan, err := claude.Query(ctx, prompt, nil)
    if err != nil {
        return err
    }

    for msg := range msgChan {
        if m, ok := msg.(*types.ResultMessage); ok {
            fmt.Printf("Result: %s\n", *m.Result)
        }
    }
    return nil
}

// Run multiple queries concurrently
var wg sync.WaitGroup
prompts := []string{"What is 1+1?", "What is 2+2?", "What is 3+3?"}

for _, p := range prompts {
    wg.Add(1)
    go func(prompt string) {
        defer wg.Done()
        if err := processQuery(ctx, prompt); err != nil {
            log.Printf("Error: %v", err)
        }
    }(p)
}
wg.Wait()
```

### Context Cancellation

Properly handle context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

msgChan, err := client.Query(ctx, "Hello")
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case msg, ok := <-msgChan:
        if !ok {
            return // Channel closed
        }
        // Process message
    case <-ctx.Done():
        log.Println("Context cancelled")
        client.Interrupt(context.Background())
        return
    }
}
```

### Custom Transport

Implement custom transport for testing or special needs:

```go
type MockTransport struct {
    messages []string
}

func (t *MockTransport) Start(ctx context.Context) error { return nil }
func (t *MockTransport) Close() error                    { return nil }
func (t *MockTransport) SendPrompt(ctx context.Context, prompt string, opts *types.ClaudeAgentOptions) error {
    return nil
}
func (t *MockTransport) Messages() <-chan string { /* ... */ }
func (t *MockTransport) Stderr() <-chan string   { /* ... */ }

// Use custom transport
client := client.NewWithOptions(&types.ClaudeAgentOptions{
    Transport: &MockTransport{},
})
```

### Error Recovery

Recover from errors and continue:

```go
for {
    msgChan, err := client.Query(ctx, prompt)
    if err != nil {
        if errors.Is(err, claude.ErrConnectionFailed) {
            // Try to reconnect
            time.Sleep(time.Second)
            if err := client.Connect(ctx); err != nil {
                log.Printf("Reconnect failed: %v", err)
                continue
            }
        }
        continue
    }

    // Process messages...
    break
}
```

## API Reference

### Package Functions

```go
// Simple query (creates client internally)
msgChan, err := claude.Query(ctx, "prompt", opts)

// Query with existing client
msgChan, err := claude.QueryWithClient(ctx, client, "prompt")

// Create clients
client := claude.NewClient()
client := claude.NewClientWithOptions(opts)

// Sessions API
sessions, err := claude.ListSessions(directory, limit, includeWorktrees)
info := claude.GetSessionInfo(sessionID, directory)
messages, err := claude.GetSessionMessages(sessionID, directory, limit, offset)
err := claude.RenameSession(sessionID, title, directory)
err := claude.TagSession(sessionID, tag, directory)
```

### Client Methods

```go
// Connection
client.Connect(ctx) error
client.Close() error

// Query
client.Query(ctx, prompt) (<-chan Message, error)
client.ReceiveMessages(ctx) (<-chan Message, error)

// Control
client.Interrupt(ctx) error
client.StopTask(ctx) error
client.SetPermissionMode(ctx, mode) error
client.SetModel(ctx, model) error

// MCP
client.ReconnectMCPServer(ctx, name) error
client.ToggleMCPServer(ctx, name, enabled) error
client.GetMCPStatus(ctx) (*McpStatusResponse, error)

// Info
client.GetServerInfo() *ServerInfo
```

### Options Reference

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Model` | `*string` | `"sonnet"` | AI model: `"opus"`, `"sonnet"`, `"haiku"` |
| `SystemPrompt` | `string` | `""` | Custom system prompt |
| `CWD` | `*string` | current dir | Working directory |
| `MaxTurns` | `*int` | unlimited | Maximum conversation turns |
| `MaxBudgetUSD` | `*float64` | unlimited | Maximum budget in USD |
| `PermissionMode` | `*PermissionMode` | `default` | Permission handling mode |
| `CanUseTool` | `func` | `nil` | Tool permission callback |
| `Hooks` | `map` | `nil` | Event hooks |
| `MCPServers` | `map` | `nil` | MCP server configurations |
| `AllowedTools` | `[]string` | all | Tools to allow |
| `DisallowedTools` | `[]string` | none | Tools to disallow |
| `IncludePartialMessages` | `*bool` | `false` | Enable partial streaming |
| `Agents` | `[]AgentDefinition` | `nil` | Custom agent definitions |
| `CLIPath` | `*string` | auto | Path to Claude CLI |
| `Env` | `map[string]string` | `nil` | Additional environment variables |

### Helper Functions

```go
// Pointer helpers
types.String("value")           // *string
types.Int(10)                   // *int
types.Float64(1.5)              // *float64
types.Bool(true)                // *bool
types.PermissionModePtr(mode)   // *PermissionMode

// Schema helpers for MCP tools
sdkmcp.Schema(props, required)      // Full schema with required fields
sdkmcp.SimpleSchema(props)          // Simple schema (no required fields)
sdkmcp.StringProperty(desc)         // String property
sdkmcp.NumberProperty(desc)         // Number property
sdkmcp.BooleanProperty(desc)        // Boolean property
sdkmcp.ObjectProperty(props, req)   // Nested object property
sdkmcp.ArrayProperty(items)         // Array property
```

## Message Types

### Top-Level Messages

| Type | Description | Key Fields |
|------|-------------|------------|
| `ResultMessage` | Final result | `Result`, `SessionID`, `TotalCostUSD`, `DurationMS`, `NumTurns`, `StopReason` |
| `AssistantMessage` | Claude's response | `Content`, `Model`, `Usage` |
| `UserMessage` | User input | `Content` |
| `SystemMessage` | System events | `Subtype`, `Data` |
| `StreamEvent` | Streaming event | `Type`, `Data` |
| `RateLimitEvent` | Rate limit info | `Type`, `Data` |

### Content Blocks

| Type | Description | Key Fields |
|------|-------------|------------|
| `TextBlock` | Text content | `Text` |
| `ThinkingBlock` | Extended thinking | `Thinking` |
| `ToolUseBlock` | Tool request | `ID`, `Name`, `Input` |
| `ToolResultBlock` | Tool result | `ToolUseID`, `Content`, `IsError` |
| `GenericContentBlock` | Unknown type | `Type`, `Raw` |

## Error Handling

```go
import (
    "errors"
    "log"

    claude "github.com/unitsvc/claude-agent-sdk-golang"
    sdkerrors "github.com/unitsvc/claude-agent-sdk-golang/errors"
)

msgChan, err := client.Query(ctx, "Hello")
if err != nil {
    // Check sentinel errors
    switch {
    case errors.Is(err, claude.ErrNoAPIKey):
        log.Fatal("API key not configured. Run: claude login")

    case errors.Is(err, claude.ErrNotInstalled):
        log.Fatal("Claude CLI not installed. Run: npm install -g @anthropic-ai/claude-code")

    case errors.Is(err, claude.ErrConnectionFailed):
        log.Fatal("Connection failed. Is Claude CLI running?")

    case errors.Is(err, claude.ErrTimeout):
        log.Fatal("Operation timed out")

    case errors.Is(err, claude.ErrInterrupted):
        log.Println("Operation was interrupted")
        return
    }

    // Check error types
    var cliErr *sdkerrors.CLIError
    if errors.As(err, &cliErr) {
        log.Printf("CLI Error: %s (exit code: %d)", cliErr.Message, cliErr.ExitCode)
        log.Printf("Stderr: %s", cliErr.Stderr)
    }

    var procErr *sdkerrors.ProcessError
    if errors.As(err, &procErr) {
        log.Printf("Process Error: %v", procErr)
    }

    log.Fatal(err)
}
```

## Examples

The [examples](examples/) directory contains comprehensive examples:

| Example | Description |
|---------|-------------|
| `quick_start` | Basic usage patterns |
| `streaming_mode` | Message streaming techniques |
| `streaming_interactive` | Interactive streaming with context |
| `streaming_goroutines` | Concurrent streaming patterns |
| `hooks` | Hook system with all events |
| `tool_permission` | Permission callback examples |
| `mcp_calculator` | MCP server integration |
| `mcp_sdk_simple` | Simple in-process MCP server |
| `mcp_sdk_server` | Full-featured SDK MCP server |
| `mcp_control` | MCP server runtime control |
| `agents` | Custom agent definitions |
| `system_prompt` | System prompt configuration |
| `setting_sources` | Settings and configuration |
| `budget` | Budget management |
| `include_partial_messages` | Partial message handling |
| `stderr_callback` | Stderr output handling |
| `tools_option` | Tool configuration |
| `filesystem_agents` | File system operations |
| `task_messages` | Task event handling |
| `plugin_example` | Plugin integration |

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run short tests (skip E2E)
go test -short ./...

# Run specific package
go test ./internal/sessions/... -v

# Run with race detector
go test -race ./...
```

## Performance

### Benchmarks

```bash
# Run benchmarks
go test -bench=. ./...

# Run with memory profiling
go test -bench=. -benchmem ./...
```

### Optimization Tips

1. **Reuse Clients**: Create one client and reuse for multiple queries
2. **Pool Goroutines**: Use worker pools for concurrent queries
3. **Buffer Channels**: Use buffered channels for high-throughput scenarios
4. **Context Timeouts**: Set appropriate timeouts to prevent hangs

```go
// Good: Reuse client
client := claude.NewClientWithOptions(opts)
defer client.Close()
client.Connect(ctx)

for _, prompt := range prompts {
    msgChan, _ := client.Query(ctx, prompt)
    // Process messages...
}

// Bad: Create new client each time
for _, prompt := range prompts {
    msgChan, _ := claude.Query(ctx, prompt, opts) // Creates new client each time
    // Process messages...
}
```

## Security

### Best Practices

1. **Never hardcode API keys** - Use environment variables or secure storage
2. **Validate inputs** - Sanitize user inputs before sending to Claude
3. **Limit permissions** - Use `AllowedTools` and `CanUseTool` to restrict tool access
4. **Sandbox writes** - Redirect file writes to safe directories
5. **Audit hooks** - Log all tool executions for security auditing

```go
// Example: Secure configuration
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    // Restrict tools
    AllowedTools: []string{"Read", "Bash"},

    // Validate and sandbox
    CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
        // Log for auditing
        log.Printf("Tool request: %s by user", toolName)

        // Validate inputs
        if toolName == "Bash" {
            if cmd, ok := input["command"].(string); ok {
                // Block dangerous commands
                if strings.Contains(cmd, "rm -rf") {
                    return types.PermissionResultDeny{
                        Behavior: "deny",
                        Message:  "Destructive commands not allowed",
                    }, nil
                }
            }
        }

        return types.PermissionResultAllow{Behavior: "allow"}, nil
    },
})
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `CLAUDE_CONFIG_DIR` | Custom config directory |
| `CLAUDE_CODE_ENTRYPOINT` | Entry point identifier |

## Best Practices

### Resource Management

```go
// Always close clients
client := claude.NewClientWithOptions(opts)
defer client.Close()

// Always cancel contexts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### Error Handling

```go
// Check all errors
msgChan, err := client.Query(ctx, prompt)
if err != nil {
    // Handle specific errors
    switch {
    case errors.Is(err, claude.ErrNoAPIKey):
        // Handle missing API key
    case errors.Is(err, claude.ErrTimeout):
        // Handle timeout
    default:
        // Handle other errors
    }
}
```

### Concurrency

```go
// Use sync.WaitGroup for coordination
var wg sync.WaitGroup
for _, prompt := range prompts {
    wg.Add(1)
    go func(p string) {
        defer wg.Done()
        processQuery(ctx, p)
    }(prompt)
}
wg.Wait()
```

## FAQ

### How do I set a custom working directory?

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CWD: types.String("/path/to/project"),
})
```

### How do I limit the number of turns?

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MaxTurns: types.Int(5),
})
```

### How do I set a budget limit?

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MaxBudgetUSD: types.Float64(1.0),  // $1.00 max
})
```

### How do I use a specific model?

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Model: types.String(types.ModelOpus),  // "opus", "sonnet", "haiku"
})
```

### How do I handle Ctrl+C gracefully?

```go
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer cancel()

client := claude.NewClientWithOptions(opts)
defer client.Close()

if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}

// Use ctx for all operations - it will cancel on Ctrl+C
msgChan, err := client.Query(ctx, "Hello")
```

### How do I access the session ID after a query?

```go
for msg := range msgChan {
    if m, ok := msg.(*types.ResultMessage); ok {
        fmt.Printf("Session ID: %s\n", m.SessionID)
    }
}
```

### How do I stream partial messages?

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: types.Bool(true),
})
```

### How do I use multiple MCP servers?

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "fs": types.McpStdioServerConfig{
            Command: "mcp-filesystem-server",
            Args:    []string{"/allowed"},
        },
        "db": types.McpStdioServerConfig{
            Command: "mcp-postgres-server",
            Args:    []string{"postgres://localhost/db"},
        },
    },
    AllowedTools: []string{"mcp__fs__read", "mcp__db__query"},
})
```

## Troubleshooting

### "Claude CLI not installed"

```bash
# Install Claude CLI
npm install -g @anthropic-ai/claude-code

# Verify installation
claude --version
```

### "API key not configured"

```bash
# Login to Anthropic
claude login

# Or set environment variable
export ANTHROPIC_API_KEY=your-key-here
```

### "Connection failed"

1. Check if Claude CLI is in your PATH
2. Try running `claude` directly in terminal
3. Check for any CLI updates: `npm update -g @anthropic-ai/claude-code`

### "Tool not found"

MCP tools are named with the pattern `mcp__<server>__<tool>`:

```go
// Correct tool name format
AllowedTools: []string{"mcp__calc__add", "mcp__calc__multiply"}
```

### "Context deadline exceeded"

Increase timeout for long-running queries:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### "Permission denied"

Check your `CanUseTool` callback or `PermissionMode`:

```go
// Option 1: Use bypass mode (not recommended for production)
PermissionMode: types.PermissionModePtr(types.PermissionModeBypassPermissions)

// Option 2: Add tools to allowed list
AllowedTools: []string{"Bash", "Read", "Write"}

// Option 3: Implement CanUseTool callback
CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
    return types.PermissionResultAllow{Behavior: "allow"}, nil
}
```

## Migration Guide

### From Python SDK

| Python | Go |
|--------|-----|
| `from claude_agent_sdk import Query` | `import claude "github.com/unitsvc/claude-agent-sdk-golang"` |
| `async for msg in query(...)` | `for msg := range claude.Query(...)` |
| `options=ClaudeAgentOptions(...)` | `&types.ClaudeAgentOptions{...}` |
| `permission_result_allow()` | `types.PermissionResultAllow{Behavior: "allow"}` |
| `@tool` decorator | `sdkmcp.Tool(...)` |

### Key Differences

1. **Async vs Channels**: Python uses `async/await`, Go uses channels
2. **Options**: Go uses struct pointers for optional fields
3. **Errors**: Go returns errors as values, Python raises exceptions
4. **Context**: Go requires explicit context for cancellation

## Changelog

### v0.1.50 (2026-03-23)

- Added `GetSessionInfo()` for single-session lookup
- Changed `SDKSessionInfo.FileSize` to optional for remote storage
- Updated to Python SDK v0.1.50

### v0.1.49

- Added `RenameSession()` and `TagSession()` functions
- Added `Tag` and `CreatedAt` fields to `SDKSessionInfo`
- Fixed session title and summary chain extraction

### v0.1.48

- Added fine-grained tool streaming support
- Added `Usage` field to `AssistantMessage`
- Fixed graceful subprocess shutdown

### v0.1.46

- Added Sessions API: `ListSessions()`, `GetSessionMessages()`
- Added agent context fields to hook inputs

### v0.1.45

- Added `StopReason` field to `ResultMessage`
- Added task message types
- Added MCP control methods: `ReconnectMCPServer()`, `ToggleMCPServer()`, `StopTask()`

See [CHANGELOG.md](CHANGELOG.md) for full history.

## Go SDK Advantages

| Feature | Python SDK | Go SDK |
|---------|-----------|--------|
| Hook Events | 10 | 12 (+SessionStart, SessionEnd) |
| Unit Tests | 153 | 360+ |
| E2E Tests | 32 | 55+ |
| Schema Helpers | Limited | Full (Schema, SimpleSchema, Property helpers) |
| Transport | Internal | Exported interface |
| Rate Limit Events | No | Yes (RateLimitEvent) |
| Generic Content Blocks | No | Yes (GenericContentBlock) |
| Concurrent Safe | Partial | Yes (channel-based) |
| Memory Footprint | Higher | Lower |

## Version

**Current Version**: 0.1.50-a7fd631

Synced with [Python SDK v0.1.50](https://github.com/anthropics/claude-agent-sdk-python).

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/unitsvc/claude-agent-sdk-golang.git
cd claude-agent-sdk-golang

# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter
go vet ./...
```

## Related Projects

- [Claude Agent SDK (Python)](https://github.com/anthropics/claude-agent-sdk-python) - Official Python SDK
- [Claude Code](https://github.com/anthropics/claude-code) - Official CLI tool
- [MCP Specification](https://modelcontextprotocol.io/) - Model Context Protocol
- [Anthropic API](https://docs.anthropic.com/) - Anthropic API documentation