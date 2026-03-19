# Claude Agent SDK for Go

[中文版本](README-zh.md)

A Go SDK for building AI agents with Claude. This SDK provides a Go implementation of the Claude Agent SDK, enabling you to build AI agents that can use tools, handle permissions, and interact with MCP servers.

## Prerequisites

- Go 1.21 or later
- Claude Code CLI installed and authenticated:
  ```bash
  npm install -g @anthropic-ai/claude-code
  claude login
  ```

## Installation

```bash
go get github.com/unitsvc/claude-agent-sdk-golang
```

## Quick Start

### Simple Query

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

    // Simple one-shot query
    msgChan, err := claude.Query(ctx, "What is 2+2?", nil)
    if err != nil {
        log.Fatal(err)
    }

    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.ResultMessage:
            if m.Result != nil {
                fmt.Printf("Result: %s\n", *m.Result)
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

    // Create client with custom options
    client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
        Model: types.String(types.ModelSonnet),
    })
    defer client.Close()

    // Connect to Claude
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    // Send a query
    msgChan, err := client.Query(ctx, "Tell me a short joke.")
    if err != nil {
        log.Fatal(err)
    }

    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.ResultMessage:
            if m.Result != nil {
                fmt.Printf("Result: %s\n", *m.Result)
            }
        }
    }
}
```

## Features

### Streaming Messages

The SDK streams messages as they are generated:

```go
for msg := range msgChan {
    switch m := msg.(type) {
    case *types.AssistantMessage:
        // Streaming text from the assistant
        for _, block := range m.Content {
            if tb, ok := block.(types.TextBlock); ok {
                fmt.Print(tb.Text)
            }
            if tb, ok := block.(types.ThinkingBlock); ok {
                fmt.Printf("[Thinking: %s]\n", tb.Thinking)
            }
        }
    case *types.ResultMessage:
        // Final result
        if m.TotalCostUSD != nil {
            fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
        }
        fmt.Printf("Duration: %dms\n", m.DurationMS)
    }
}
```

### Permission Handling

Control tool permissions with callbacks:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
        if toolName == "Bash" {
            // Auto-approve Bash commands
            return types.PermissionResultAllow{
                Behavior: "allow",
            }, nil
        }
        // Deny other tools
        return types.PermissionResultDeny{
            Behavior: "deny",
            Message:  "Permission denied",
        }, nil
    },
})
```

### Custom System Prompt

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    SystemPrompt: "You are a helpful coding assistant specialized in Go.",
})
```

### Working Directory

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CWD: types.String("/path/to/project"),
})
```

### MCP Servers

Configure MCP (Model Context Protocol) servers:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "myServer": types.McpStdioServerConfig{
            Command: "my-mcp-server",
            Args:    []string{"--port", "8080"},
        },
    },
})
```

### SDK MCP Server (In-Process)

Create in-process MCP servers with custom tools:

```go
import "github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"

// Define a tool
addTool := sdkmcp.Tool("add", "Add two numbers",
    sdkmcp.Schema(map[string]interface{}{
        "a": sdkmcp.NumberProperty("First number"),
        "b": sdkmcp.NumberProperty("Second number"),
    }, []string{"a", "b"}),
    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
        a, _ := args["a"].(float64)
        b, _ := args["b"].(float64)
        return sdkmcp.TextResult(fmt.Sprintf("Result: %.2f", a+b)), nil
    })

// Create the server
calcServer := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{addTool})

// Use with client
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "calc": types.McpSdkServerConfig{
            Type:     "sdk",
            Name:     "calculator",
            Instance: calcServer,
        },
    },
    AllowedTools: []string{"mcp__calc__add"},
})
```

### Hooks

Register hooks for tool events:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Hooks: map[types.HookEvent][]types.HookMatcher{
        types.HookEventPreToolUse: {
            {
                Matcher: "Bash",
                Hooks: []types.HookCallback{
                    &MyHookCallback{},
                },
            },
        },
    },
})
```

A **hook** is a Go function that the Claude Code _application_ (_not_ Claude) invokes at specific points of the Claude agent loop. Hooks can provide deterministic processing and automated feedback for Claude. Read more in [Claude Code Hooks Reference](https://docs.anthropic.com/en/docs/claude-code/hooks).

**Available Hook Events:**
- `HookEventPreToolUse` - Before tool execution
- `HookEventPostToolUse` - After successful tool execution
- `HookEventPostToolUseFailure` - After failed tool execution
- `HookEventUserPromptSubmit` - When user submits a prompt
- `HookEventStop` - When conversation stops
- `HookEventSubagentStart` - When subagent starts
- `HookEventSubagentStop` - When subagent stops
- `HookEventPreCompact` - Before context compaction
- `HookEventNotification` - For notifications
- `HookEventPermissionRequest` - For permission requests
- `HookEventSessionStart` - When session starts (Go SDK only)
- `HookEventSessionEnd` - When session ends (Go SDK only)

For comprehensive examples, see [examples/hooks/main.go](examples/hooks/main.go).

### Permission Modes

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
})
```

Available modes:
- `PermissionModeDefault` - Default behavior
- `PermissionModeAcceptEdits` - Auto-accept file edits
- `PermissionModePlan` - Planning mode
- `PermissionModeBypassPermissions` - Bypass all permissions

### Sessions API

List and retrieve session history:

```go
// List all sessions
sessions, err := claude.ListSessions(ctx)
for _, sess := range sessions {
    fmt.Printf("Session: %s (%s)\n", sess.SessionID, sess.CustomTitle)
}

// Get messages from a specific session
messages, err := claude.GetSessionMessages(ctx, sessionID)
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.Role, msg.Content)
}
```

### Session Mutations

Rename and tag sessions:

```go
// Rename a session
err := claude.RenameSession(ctx, sessionID, "My New Title")

// Tag a session
err := claude.TagSession(ctx, sessionID, "important", "golang")
```

### Client Control Methods

```go
// Reconnect an MCP server
err := client.ReconnectMCPServer(ctx, "myServer")

// Toggle an MCP server on/off
err := client.ToggleMCPServer(ctx, "myServer", false)

// Stop the current task
err := client.StopTask(ctx)
```

### Fine-Grained Tool Streaming

Enable detailed tool streaming with `include_partial_messages`:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: true,
})
```

When enabled, the SDK automatically sets `CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=1` to get detailed tool input deltas.

### Agents

Define custom agents with specific configurations:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Agents: []types.AgentDefinition{
        {
            Description: "Go expert",
            Prompt:      "You are a Go programming expert.",
            Tools:       []string{"Bash", "Read", "Write"},
            Model:       types.String(types.ModelSonnet),
            Skills:      []string{"golang"},
        },
    },
})
```

### Streaming Modes

The SDK supports multiple streaming patterns. See [examples/streaming_mode/main.go](examples/streaming_mode/main.go) for full demonstrations:

```go
// Interactive streaming with message type filtering
for msg := range msgChan {
    switch m := msg.(type) {
    case *types.AssistantMessage:
        // Handle streaming text
    case *types.ResultMessage:
        // Handle final result
    }
}
```

### Include Partial Messages

Receive partial message content as it streams:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: true,
})
```

## API Reference

### `Query(ctx, prompt, options)`

Simple one-shot query function.

```go
msgChan, err := claude.Query(ctx, "What is 2+2?", nil)
```

### `QueryWithClient(ctx, client, prompt)`

Query using an existing client.

```go
msgChan, err := claude.QueryWithClient(ctx, client, "Hello!")
```

### `Client`

Full-featured client for interactive conversations.

#### Methods:
- `Connect(ctx)` - Establish connection
- `Query(ctx, prompt)` - Send a query
- `ReceiveMessages(ctx)` - Receive messages until ResultMessage
- `Interrupt(ctx)` - Interrupt current operation
- `SetPermissionMode(ctx, mode)` - Change permission mode
- `SetModel(ctx, model)` - Change AI model
- `ReconnectMCPServer(ctx, name)` - Reconnect an MCP server
- `ToggleMCPServer(ctx, name, enabled)` - Toggle MCP server
- `StopTask(ctx)` - Stop the running task
- `GetServerInfo()` - Get server initialization info
- `GetMCPStatus()` - Get MCP server status
- `Close()` - Close the connection

### Sessions API Functions

- `ListSessions(ctx)` - List all sessions
- `GetSessionMessages(ctx, sessionID)` - Get messages from a session
- `RenameSession(ctx, sessionID, title)` - Rename a session
- `TagSession(ctx, sessionID, tags...)` - Tag a session

### Options

| Option | Type | Description |
|--------|------|-------------|
| `Model` | `*string` | AI model to use (`"opus"`, `"sonnet"`, `"haiku"`, `"inherit"`) |
| `SystemPrompt` | `string` | Custom system prompt |
| `CWD` | `*string` | Working directory |
| `MaxTurns` | `*int` | Maximum conversation turns |
| `MaxBudgetUSD` | `*float64` | Maximum budget in USD |
| `PermissionMode` | `*PermissionMode` | Permission handling mode |
| `CanUseTool` | `func` | Permission callback |
| `Hooks` | `map` | Event hooks |
| `MCPServers` | `map` | MCP server configurations |
| `AllowedTools` | `[]string` | Tools to allow |
| `DisallowedTools` | `[]string` | Tools to disallow |
| `IncludePartialMessages` | `*bool` | Enable partial message streaming |
| `Agents` | `[]AgentDefinition` | Agent definitions |
| `SystemPromptPresets` | `[]SystemPromptPreset` | System prompt presets |
| `ToolsPresets` | `[]ToolsPreset` | Tools presets |

### Helper Functions

```go
// Pointer helpers for optional fields
types.String("value")     // *string
types.Int(10)             // *int
types.Float64(1.5)        // *float64
types.Bool(true)          // *bool
types.PermissionModePtr(types.PermissionModeAcceptEdits)  // *PermissionMode

// Schema helpers for MCP tools
sdkmcp.Schema(props, required)    // Create input schema
sdkmcp.StringProperty(desc)       // String property
sdkmcp.NumberProperty(desc)      // Number property
sdkmcp.BooleanProperty(desc)     // Boolean property
sdkmcp.ObjectProperty(props)      // Object property
sdkmcp.ArrayProperty(items)      // Array property
```

## Message Types

- `ResultMessage` - Final result of a query (includes cost, duration, stop_reason)
- `AssistantMessage` - Streaming text from Claude (includes model, usage)
- `UserMessage` - User message
- `SystemMessage` - System message (subtypes: task_started, task_progress, task_notification)
- `StreamEvent` - Streaming event
- `RateLimitEvent` - Rate limit event (Go SDK only)

### ResultMessage Fields

```go
type ResultMessage struct {
    Subtype        string
    DurationMS     int
    DurationAPIMS  int
    IsError        bool
    NumTurns       int
    SessionID      string
    StopReason     *string           // "stop", "early_stop", "error", etc.
    TotalCostUSD   *float64
    Usage          map[string]interface{}  // Per-turn usage
    Result         *string
    StructuredOutput interface{}
}
```

### Content Blocks

- `TextBlock` - Text content
- `ThinkingBlock` - Thinking content (extended thinking)
- `ToolUseBlock` - Tool use request (id, name, input)
- `ToolResultBlock` - Tool execution result (tool_use_id, content, is_error)
- `GenericContentBlock` - Unknown block type (Go SDK only)

## Error Handling

```go
import "github.com/unitsvc/claude-agent-sdk-golang/errors"

msgChan, err := client.Query(ctx, "Hello")
if err != nil {
    // Check for sentinel errors
    if errors.Is(err, claude.ErrNoAPIKey) {
        log.Fatal("API key not configured")
    }
    if errors.Is(err, claude.ErrNotInstalled) {
        log.Fatal("Claude CLI not installed")
    }

    // Check for specific error types
    var cliErr *errors.CLIError
    if errors.As(err, &cliErr) {
        log.Printf("CLI Error: %s (exit code: %d)", cliErr.Message, cliErr.ExitCode)
    }

    var connErr *errors.CLIConnectionError
    if errors.As(err, &connErr) {
        log.Printf("Connection Error: %s", connErr.Message)
    }

    log.Fatal(err)
}
```

**Available Sentinel Errors:**
- `ErrNoAPIKey` - No API key configured
- `ErrNotInstalled` - Claude CLI not installed
- `ErrConnectionFailed` - Connection failed
- `ErrTimeout` - Operation timed out
- `ErrInterrupted` - Operation interrupted

## Examples

See the [examples](examples/) directory for comprehensive usage examples:

| Example | Description |
|---------|-------------|
| `quick_start` | Basic usage examples |
| `streaming_mode` | Message streaming patterns |
| `streaming_interactive` | Interactive streaming |
| `streaming_goroutines` | Goroutine-based streaming |
| `hooks` | Hook system usage |
| `tool_permission` | Permission callbacks |
| `mcp_calculator` | MCP server example |
| `mcp_sdk_simple` | Simple SDK MCP server |
| `mcp_sdk_server` | Full SDK MCP server |
| `mcp_control` | MCP server control |
| `agents` | Custom agent definitions |
| `system_prompt` | Custom system prompts |
| `setting_sources` | Settings configuration |
| `budget` | Budget management |
| `include_partial_messages` | Partial message handling |
| `stderr_callback` | Stderr handling |
| `tools_option` | Tools configuration |
| `filesystem_agents` | Filesystem agents |
| `task_messages` | Task message handling |
| `plugin_example` | Plugin usage |

## Go SDK Advantages

1. **More Hook Events** - SessionStart, SessionEnd (not in Python)
2. **Better Schema Helpers** - Schema(), SimpleSchema(), StringProperty() etc.
3. **More Tests** - 350+ unit tests, 50+ E2E tests
4. **Type Safety** - Compile-time type checking
5. **Concurrency** - Goroutine + channel pattern
6. **Exported Transport** - Custom transport implementation support
7. **Sentinel Errors** - ErrNoAPIKey, ErrNotInstalled, etc.
8. **RateLimitEvent** - Parsed rate limit events (Go SDK only)
9. **GenericContentBlock** - Forward-compatible unknown block handling

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
go test ./client/...
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Version

**Current Version**: 0.1.48-971994c

Synced with Python SDK v0.1.48+.
