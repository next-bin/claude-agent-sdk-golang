# Claude Agent SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/next-bin/claude-agent-sdk-golang.svg)](https://pkg.go.dev/github.com/next-bin/claude-agent-sdk-golang)
[![Go Report Card](https://goreportcard.com/badge/github.com/next-bin/claude-agent-sdk-golang)](https://goreportcard.com/report/github.com/next-bin/claude-agent-sdk-golang)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[中文文档](README-zh.md)

A Go SDK for building AI agents with [Claude Code](https://code.claude.com/). Provides a high-level API for querying Claude, managing interactive sessions, defining custom tools, intercepting agent behavior with hooks, and managing conversation sessions.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Basic Usage](#basic-usage)
- [Interactive Sessions](#interactive-sessions)
- [Custom Tools](#custom-tools)
- [Hooks](#hooks)
- [Sessions API](#sessions-api)
- [Dynamic Control](#dynamic-control)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Contributing](#contributing)

## Installation

```bash
go get github.com/next-bin/claude-agent-sdk-golang
```

**Requirements:**

| Requirement | Details |
|-------------|---------|
| **Go** | 1.21 or later |
| **Claude Code CLI** | Installed and authenticated ([install guide](https://code.claude.com/docs/en/quickstart)) |

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    claude "github.com/next-bin/claude-agent-sdk-golang"
)

func main() {
    ctx := context.Background()

    msgChan, err := claude.Query(ctx, "What is 2 + 2?", nil)
    if err != nil {
        log.Fatal(err)
    }

    for msg := range msgChan {
        fmt.Printf("%v\n", msg)
    }
}
```

## Basic Usage

### Simple Query

```go
ctx := context.Background()

// With default options
msgChan, err := claude.Query(ctx, "Hello Claude", nil)
```

### With Configuration

```go
import "github.com/next-bin/claude-agent-sdk-golang/types"

opts := &types.ClaudeAgentOptions{
    SystemPrompt: types.String("You are a helpful assistant"),
    MaxTurns:     types.Int(1),
}

msgChan, err := claude.Query(ctx, "Tell me a joke", opts)
```

### Working Directory

```go
opts := &types.ClaudeAgentOptions{
    CWD: "/path/to/project",
}
```

### Tool Permissions

By default, Claude has access to the full [Claude Code toolset](https://code.claude.com/docs/en/settings#tools-available-to-claude). `AllowedTools` is an allowlist — listed tools are auto-approved, unlisted tools fall through to `PermissionMode` and `CanUseTool`. Use `DisallowedTools` to remove tools entirely.

```go
opts := &types.ClaudeAgentOptions{
    AllowedTools:   []string{"Read", "Write", "Bash"},
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
}
```

## Interactive Sessions

For bidirectional conversations with follow-up messages, use `client.Client`:

```go
import "github.com/next-bin/claude-agent-sdk-golang/client"

c := client.NewWithOptions(&types.ClaudeAgentOptions{
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
})
defer c.Close()

// Connect with initial prompt
err := c.Connect(ctx, "Hello Claude")

// Read response
for msg := range c.ReceiveResponse(ctx) {
    fmt.Printf("%T: %v\n", msg, msg)
}

// Send follow-up
err = c.Query(ctx, "Can you explain more?")
```

### Connect Without Initial Prompt

```go
// Connect for interactive use
err := c.Connect(ctx)

// Send messages as needed
err = c.Query(ctx, "First question")
err = c.Query(ctx, "Follow-up question")
```

## Custom Tools

Define custom tools as in-process MCP servers — no subprocess management needed.

```go
import "github.com/next-bin/claude-agent-sdk-golang/sdkmcp"

// Define a tool
greetTool := sdkmcp.Tool(
    "greet",
    "Greet a user",
    sdkmcp.SimpleSchema(map[string]string{"name": "string"}),
    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
        name := args["name"].(string)
        return sdkmcp.TextResult(fmt.Sprintf("Hello, %s!", name)), nil
    },
)

// Create an SDK MCP server
server := sdkmcp.CreateSdkMcpServer("my-tools", []*sdkmcp.SdkMcpTool{greetTool})

// Use with Claude
opts := &types.ClaudeAgentOptions{
    MCPServers:   map[string]types.McpServerConfig{"tools": server},
    AllowedTools: []string{"mcp__tools__greet"},
}
```

### Mixed Server

Combine SDK MCP servers with external MCP servers:

```go
opts := &types.ClaudeAgentOptions{
    MCPServers: map[string]types.McpServerConfig{
        "internal": sdkServer, // In-process
        "external": types.McpStdioServerConfig{
            Type:    "stdio",
            Command: "my-external-server",
        },
    },
}
```

## Hooks

Hooks are functions invoked by the Claude Code application at specific points in the agent loop.

```go
type bashHook struct{}

func (h *bashHook) Execute(input types.HookInput, toolUseID *string, ctx types.HookContext) (types.HookJSONOutput, error) {
    hookInput, ok := input.(types.PreToolUseHookInput)
    if !ok {
        return types.SyncHookJSONOutput{Continue_: types.Bool(true)}, nil
    }

    command, _ := hookInput.ToolInput["command"].(string)
    if strings.Contains(command, "rm -rf") {
        reason := "Dangerous command blocked by hook"
        return types.SyncHookJSONOutput{
            Continue_: types.Bool(true),
            HookSpecificOutput: types.PreToolUseHookSpecificOutput{
                HookEventName:            "PreToolUse",
                PermissionDecision:       types.String("deny"),
                PermissionDecisionReason: &reason,
            },
        }, nil
    }

    return types.SyncHookJSONOutput{Continue_: types.Bool(true)}, nil
}

opts := &types.ClaudeAgentOptions{
    AllowedTools: []string{"Bash"},
    Hooks: map[types.HookEvent][]types.HookMatcher{
        types.HookEventPreToolUse: {
            {Matcher: "Bash", Hooks: []types.HookCallback{&bashHook{}}},
        },
    },
}
```

### Available Hook Events

| Event | Description |
|-------|-------------|
| `PreToolUse` | Before a tool is executed |
| `PostToolUse` | After a tool executes |
| `PostToolUseFailure` | When a tool fails |
| `UserPromptSubmit` | When user submits a prompt |
| `Stop` | When agent stops |
| `SubagentStart` | When a sub-agent starts |
| `SubagentStop` | When a sub-agent stops |
| `PreCompact` | Before context compaction |
| `Notification` | For notifications |
| `PermissionRequest` | When permission is requested |

## Sessions API

Manage conversation sessions programmatically.

```go
import claude "github.com/next-bin/claude-agent-sdk-golang"

// List sessions
sessions, err := claude.ListSessions("/path/to/project", 10, true)

// Get session messages
messages, err := claude.GetSessionMessages(sessionID, "/path/to/project", 0, 0)

// Get single session metadata
info := claude.GetSessionInfo(sessionID, "/path/to/project")

// Session mutations
err = claude.RenameSession(sessionID, "New Title", "/path/to/project")
err = claude.TagSession(sessionID, "experiment", "/path/to/project")
err = claude.DeleteSession(sessionID, "/path/to/project")
result, err := claude.ForkSession(sessionID, "/path/to/project", nil, nil)
```

## Dynamic Control

Control an active session at runtime.

```go
// Connect
err := c.Connect(ctx)

// Switch permission mode
err = c.SetPermissionMode(ctx, "acceptEdits")

// Switch model
err = c.SetModel(ctx, "claude-sonnet-4-6")

// Get context usage
usage, err := c.GetContextUsage(ctx)
fmt.Printf("Using %.1f%% of context\n", usage.Percentage)

// Get MCP server status
status, err := c.GetMCPStatus(ctx)

// Interrupt running conversation
err = c.Interrupt(ctx)
```

## Error Handling

```go
import claude "github.com/next-bin/claude-agent-sdk-golang"

msgChan, err := claude.Query(ctx, "Hello", nil)
if err != nil {
    switch {
    case claude.ErrNotInstalled:
        fmt.Println("Please install Claude Code")
    case claude.ErrConnectionFailed:
        fmt.Println("Failed to connect")
    case claude.ErrTimeout:
        fmt.Println("Query timed out")
    default:
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Examples

| Example | Description |
|---------|-------------|
| [quick_start](examples/quick_start/) | Basic query |
| [streaming_mode](examples/streaming_mode/) | Interactive client |
| [mcp_sdk_server](examples/mcp_sdk_server/) | Custom tools |
| [hooks](examples/hooks/) | Hook system |
| [tool_permission](examples/tool_permission/) | Permission callbacks |
| [agents](examples/agents/) | Custom agents |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Submit a pull request

### Development

```bash
git clone https://github.com/next-bin/claude-agent-sdk-golang.git
cd claude-agent-sdk-golang
go mod download
go test ./...
go vet ./...
```

## Related Projects

- [Claude Code Documentation](https://code.claude.com/docs/en) — Claude Code docs
- [MCP Specification](https://modelcontextprotocol.io/) — Model Context Protocol
- [Anthropic API](https://docs.anthropic.com/) — Anthropic API documentation
