# Claude Agent SDK for Go

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
        case *claude.ResultMessage:
            fmt.Printf("Result: %s\n", m.Result)
        case *claude.AssistantMessage:
            fmt.Printf("Assistant: %v\n", m.Content)
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
        Model: types.String("claude-sonnet-4-20250514"),
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
        case *claude.ResultMessage:
            fmt.Printf("Result: %s\n", m.Result)
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
    case *claude.AssistantMessage:
        // Streaming text from the assistant
        for _, block := range m.Content {
            if text, ok := block.(claude.TextBlock); ok {
                fmt.Print(text.Text)
            }
        }
    case *claude.ResultMessage:
        // Final result
        fmt.Printf("\nCost: $%.4f\n", m.CostUSD)
        fmt.Printf("Duration: %.2fs\n", m.DurationMS/1000)
    case *claude.ToolUseMessage:
        // Tool use started
        fmt.Printf("Using tool: %s\n", m.ToolName)
    case *claude.ToolResultMessage:
        // Tool completed
        fmt.Printf("Tool result: %s\n", m.ToolResult)
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
            return types.PermissionResultAllow{}, nil
        }
        // Ask for other tools
        return types.PermissionResultDeny{
            Message: "Permission denied",
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

### Hooks

Register hooks for tool events:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Hooks: map[types.HookEvent][]types.HookMatcher{
        types.HookEventPreToolUse: {
            {
                Matcher: "Bash",
                Hooks: []types.HookCallback{
                    MyHookCallback{},
                },
            },
        },
    },
})
```

A **hook** is a Go function that the Claude Code _application_ (_not_ Claude) invokes at specific points of the Claude agent loop. Hooks can provide deterministic processing and automated feedback for Claude. Read more in [Claude Code Hooks Reference](https://docs.anthropic.com/en/docs/claude-code/hooks).

For comprehensive examples, see [examples/hooks/main.go](examples/hooks/main.go).

## API Reference

### `Query(ctx, prompt, options)`

Simple one-shot query function.

### `Client`

Full-featured client for interactive conversations.

#### Methods:
- `Connect(ctx)`: Establish connection
- `Query(ctx, prompt)`: Send a query
- `Interrupt(ctx)`: Interrupt current operation
- `SetPermissionMode(ctx, mode)`: Change permission mode
- `SetModel(ctx, model)`: Change AI model
- `Close()`: Close the connection

### Options

| Option | Type | Description |
|--------|------|-------------|
| `Model` | `string` | AI model to use |
| `SystemPrompt` | `string` | Custom system prompt |
| `CWD` | `string` | Working directory |
| `MaxTurns` | `int` | Maximum conversation turns |
| `PermissionMode` | `PermissionMode` | Permission handling mode |
| `CanUseTool` | `func` | Permission callback |
| `Hooks` | `map` | Event hooks |
| `MCPServers` | `map` | MCP server configurations |

## Message Types

- `ResultMessage`: Final result of a query
- `AssistantMessage`: Streaming text from Claude
- `ToolUseMessage`: Tool use started
- `ToolResultMessage`: Tool execution result
- `PermissionRequestMessage`: Permission request

## Available Tools

See the [Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code/settings#tools-available-to-claude) for a complete list of available tools.

## Error Handling

```go
msgChan, err := client.Query(ctx, "Hello")
if err != nil {
    var cliErr *errors.CLIError
    if errors.As(err, &cliErr) {
        // Handle CLI-specific errors
    }
    log.Fatal(err)
}
```

## License

MIT License - see [LICENSE](LICENSE) for details.