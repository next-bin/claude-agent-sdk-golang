# Claude Agent SDK for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/unitsvc/claude-agent-sdk-golang.svg)](https://pkg.go.dev/github.com/unitsvc/claude-agent-sdk-golang)
[![Go Report Card](https://goreportcard.com/badge/github.com/unitsvc/claude-agent-sdk-golang)](https://goreportcard.com/report/github.com/unitsvc/claude-agent-sdk-golang)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[English Documentation](README.md)

一个用于构建 Claude AI 智能体的 Go SDK。本 SDK 提供了 [Claude Agent SDK](https://github.com/anthropics/claude-agent-sdk-python) 的 Go 语言实现，让你能够构建可以使用工具、处理权限、与 MCP 服务器交互的 AI 智能体。

## 目录

- [特性](#特性)
- [环境要求](#环境要求)
- [安装](#安装)
- [快速开始](#快速开始)
- [架构](#架构)
- [核心功能](#核心功能)
- [高级主题](#高级主题)
- [API 参考](#api-参考)
- [错误处理](#错误处理)
- [示例](#示例)
- [测试](#测试)
- [性能](#性能)
- [安全](#安全)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)
- [故障排除](#故障排除)
- [迁移指南](#迁移指南)
- [更新日志](#更新日志)

## 特性

| 特性 | 描述 |
|------|------|
| 🔄 **完整 API 兼容** | 与 Python SDK v0.1.50 完全兼容 |
| 📡 **流式消息** | 通过 Go channel 实现实时消息流 |
| 🔌 **MCP 服务器支持** | 支持 Stdio、SSE、HTTP 和进程内 SDK MCP 服务器 |
| 🪝 **Hook 系统** | 12 种 hook 事件用于工具生命周期管理 |
| 🔐 **权限控制** | 细粒度的工具权限回调 |
| 💾 **会话 API** | 列出、查询、重命名和标记对话会话 |
| 🎯 **类型安全** | Go 泛型提供编译时类型检查 |
| ⚡ **并发支持** | 原生 goroutine + channel 模式 |
| 📊 **费用追踪** | 内置使用量和费用追踪 |
| 🛠️ **自定义工具** | 使用 JSON Schema 验证定义自定义工具 |

## 环境要求

- **Go 1.21+**（支持泛型）
- **Claude Code CLI** 已安装并认证：

```bash
# 安装 Claude Code CLI
npm install -g @anthropic-ai/claude-code

# 登录 Anthropic
claude login
```

### 验证安装

```bash
# 检查 Go 版本
go version  # 应为 1.21 或更高

# 检查 Claude CLI
claude --version
```

## 安装

```bash
go get github.com/unitsvc/claude-agent-sdk-golang
```

### Go Modules

```go
import claude "github.com/unitsvc/claude-agent-sdk-golang"
```

### 版本固定

```go
// go.mod
require github.com/unitsvc/claude-agent-sdk-golang v0.1.50
```

## 快速开始

### 简单查询

与 Claude 交互的最简单方式：

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

    // 一次性查询 - 自动创建客户端、发送查询并关闭
    msgChan, err := claude.Query(ctx, "2+2等于多少？", nil)
    if err != nil {
        log.Fatal(err)
    }

    // 处理流式消息
    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.ResultMessage:
            if m.Result != nil {
                fmt.Printf("结果: %s\n", *m.Result)
            }
            fmt.Printf("耗时: %dms, 轮次: %d\n", m.DurationMS, m.NumTurns)
            if m.TotalCostUSD != nil {
                fmt.Printf("费用: $%.6f\n", *m.TotalCostUSD)
            }
        case *types.AssistantMessage:
            for _, block := range m.Content {
                if tb, ok := block.(types.TextBlock); ok {
                    fmt.Printf("助手: %s\n", tb.Text)
                }
            }
        }
    }
}
```

### 带配置的客户端

适用于需要多次查询和更多控制的场景：

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
    // 创建可在 Ctrl+C 时取消的上下文
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // 创建带自定义配置的客户端
    client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
        Model:        types.String(types.ModelSonnet),
        MaxTurns:     types.Int(5),
        MaxBudgetUSD: types.Float64(1.0),
    })
    defer client.Close()

    // 连接到 Claude
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    // 发送查询
    msgChan, err := client.Query(ctx, "写一首关于编程的俳句。")
    if err != nil {
        log.Fatal(err)
    }

    // 处理消息
    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.AssistantMessage:
            for _, block := range m.Content {
                if tb, ok := block.(types.TextBlock); ok {
                    fmt.Print(tb.Text)
                }
            }
        case *types.ResultMessage:
            fmt.Printf("\n\n---\n会话: %s\n", m.SessionID)
            if m.TotalCostUSD != nil {
                fmt.Printf("费用: $%.6f\n", *m.TotalCostUSD)
            }
        }
    }
}
```

## 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        你的应用程序                              │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   Query()   │  │   Client    │  │     Sessions API        │  │
│  │   (简单)    │  │   (高级)    │  │ ListSessions, 等        │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘  │
├─────────┴────────────────┴─────────────────────┴───────────────┤
│                        SDK 核心                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   消息      │  │    Hook     │  │    权限                 │  │
│  │   解析器    │  │   系统      │  │    管理器               │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                      传输层                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              SubprocessCLITransport                      │    │
│  │         (与 Claude CLI 通信)                             │    │
│  └─────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────┤
│                       Claude Code CLI                           │
│                    (Anthropic 官方 CLI)                         │
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

| 组件 | 描述 |
|------|------|
| **Query()** | 简单的一次性查询函数 |
| **Client** | 用于交互式会话的全功能客户端 |
| **消息解析器** | 解析来自 Claude CLI 的 JSONL 消息 |
| **Hook 系统** | 工具生命周期的事件驱动回调 |
| **权限管理器** | 控制工具执行权限 |
| **传输层** | 处理与 Claude CLI 的子进程通信 |
| **会话 API** | 管理对话历史 |

### 项目结构

```
claude-agent-sdk-golang/
├── client.go              # 客户端实现
├── query.go               # Query 函数
├── sdk.go                 # 公共 API 导出
├── types/
│   └── types.go           # 类型定义
├── errors/
│   └── errors.go          # 错误类型
├── internal/
│   ├── messageparser/     # JSONL 消息解析
│   ├── query/             # Query 实现
│   ├── sessions/          # 会话 API
│   └── transport/         # CLI 传输层
├── sdkmcp/
│   └── server.go          # SDK MCP 服务器
└── examples/
    └── ...                # 使用示例
```

## 核心功能

### 流式消息

SDK 使用 Go channel 实现实时消息流：

```go
for msg := range msgChan {
    switch m := msg.(type) {
    case *types.AssistantMessage:
        // 流式文本 - 可能分多个块到达
        for _, block := range m.Content {
            switch b := block.(type) {
            case types.TextBlock:
                fmt.Print(b.Text)  // 文本内容
            case types.ThinkingBlock:
                fmt.Printf("[思考: %s]\n", b.Thinking)  // 扩展思考
            case types.ToolUseBlock:
                fmt.Printf("[调用工具: %s]\n", b.Name)  // 工具调用
            }
        }
    case *types.ResultMessage:
        // 最终结果 - 包含摘要信息
        fmt.Printf("会话: %s\n", m.SessionID)
        fmt.Printf("耗时: %dms\n", m.DurationMS)
        fmt.Printf("轮次: %d\n", m.NumTurns)
        if m.TotalCostUSD != nil {
            fmt.Printf("费用: $%.6f\n", *m.TotalCostUSD)
        }
        if m.StopReason != nil {
            fmt.Printf("停止原因: %s\n", *m.StopReason)
        }
    }
}
```

### 权限处理

使用细粒度权限控制工具执行：

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
        // 记录工具使用
        log.Printf("工具: %s, 输入: %v", toolName, input)

        switch toolName {
        case "Bash":
            // 自动批准安全命令
            if cmd, ok := input["command"].(string); ok {
                if strings.HasPrefix(cmd, "git ") || strings.HasPrefix(cmd, "go ") {
                    return types.PermissionResultAllow{Behavior: "allow"}, nil
                }
            }
            // 其他命令需要确认
            return types.PermissionResultDeny{
                Behavior: "deny",
                Message:  "命令需要手动批准",
            }, nil

        case "Write":
            // 将写入重定向到沙箱目录
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
            // 允许所有读取
            return types.PermissionResultAllow{Behavior: "allow"}, nil

        default:
            // 拒绝未知工具
            return types.PermissionResultDeny{
                Behavior: "deny",
                Message:  fmt.Sprintf("工具 %s 不被允许", toolName),
            }, nil
        }
    },
})
```

### MCP 服务器

配置外部 MCP 服务器：

```go
// Stdio MCP 服务器（本地进程）
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

// SSE MCP 服务器（远程）
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

// HTTP MCP 服务器
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MCPServers: map[string]interface{}{
        "api": types.McpHttpServerConfig{
            URL: "https://api.example.com/mcp",
        },
    },
})
```

### SDK MCP 服务器（进程内）

无需外部进程创建自定义工具：

```go
import "github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"

// 定义计算器工具
calculator := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{
    sdkmcp.Tool("add", "两数相加",
        sdkmcp.Schema(map[string]interface{}{
            "a": sdkmcp.NumberProperty("第一个数"),
            "b": sdkmcp.NumberProperty("第二个数"),
        }, []string{"a", "b"}),
        func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
            a, _ := args["a"].(float64)
            b, _ := args["b"].(float64)
            return sdkmcp.TextResult(fmt.Sprintf("%.2f", a+b)), nil
        }),

    sdkmcp.Tool("multiply", "两数相乘",
        sdkmcp.Schema(map[string]interface{}{
            "a": sdkmcp.NumberProperty("第一个数"),
            "b": sdkmcp.NumberProperty("第二个数"),
        }, []string{"a", "b"}),
        func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
            a, _ := args["a"].(float64)
            b, _ := args["b"].(float64)
            return sdkmcp.TextResult(fmt.Sprintf("%.2f", a*b)), nil
        }),
})

// 配合客户端使用
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

为工具生命周期事件注册回调：

```go
// 定义 hook 回调
type LoggingHook struct{}

func (h *LoggingHook) Call(input interface{}) (interface{}, error) {
    switch i := input.(type) {
    case types.PreToolUseHookInput:
        log.Printf("[前置] 工具: %s", i.ToolName)
        log.Printf("[前置] 输入: %v", i.ToolInput)
    case types.PostToolUseHookInput:
        log.Printf("[后置] 工具: %s", i.ToolName)
        log.Printf("[后置] 结果: %v", i.ToolResult)
    }
    return nil, nil // 返回 nil 继续，返回 error 阻止
}

// 注册 hooks
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Hooks: map[types.HookEvent][]types.HookMatcher{
        types.HookEventPreToolUse: {
            {
                Matcher: "Bash",  // 正则模式
                Hooks: []types.HookCallback{&LoggingHook{}},
            },
        },
        types.HookEventPostToolUse: {
            {
                Matcher: ".*",  // 匹配所有工具
                Hooks: []types.HookCallback{&LoggingHook{}},
            },
        },
    },
})
```

#### Hook 事件

| 事件 | 触发时机 | 输入类型 |
|------|----------|----------|
| `HookEventPreToolUse` | 工具执行前 | `PreToolUseHookInput` |
| `HookEventPostToolUse` | 工具成功执行后 | `PostToolUseHookInput` |
| `HookEventPostToolUseFailure` | 工具执行失败后 | `PostToolUseFailureHookInput` |
| `HookEventUserPromptSubmit` | 用户提交提示时 | `UserPromptSubmitHookInput` |
| `HookEventStop` | 对话停止时 | `StopHookInput` |
| `HookEventSubagentStart` | 子代理启动时 | `SubagentStartHookInput` |
| `HookEventSubagentStop` | 子代理停止时 | `SubagentStopHookInput` |
| `HookEventPreCompact` | 上下文压缩前 | `PreCompactHookInput` |
| `HookEventNotification` | 通知事件 | `NotificationHookInput` |
| `HookEventPermissionRequest` | 权限请求时 | `PermissionRequestHookInput` |
| `HookEventSessionStart` | 会话开始时 | `SessionStartHookInput` |
| `HookEventSessionEnd` | 会话结束时 | `SessionEndHookInput` |

### 会话 API

管理对话历史：

```go
// 列出项目的会话
sessions, err := claude.ListSessions("/path/to/project", 10, true)
if err != nil {
    log.Fatal(err)
}

for _, sess := range sessions {
    fmt.Printf("会话: %s\n", sess.SessionID)
    fmt.Printf("  摘要: %s\n", sess.Summary)
    fmt.Printf("  修改时间: %s\n", time.UnixMilli(sess.LastModified).Format(time.RFC3339))

    if sess.CustomTitle != nil {
        fmt.Printf("  标题: %s\n", *sess.CustomTitle)
    }
    if sess.Tag != nil {
        fmt.Printf("  标签: %s\n", *sess.Tag)
    }
    if sess.CreatedAt != nil {
        fmt.Printf("  创建时间: %s\n", time.UnixMilli(*sess.CreatedAt).Format(time.RFC3339))
    }
}

// 获取单个会话信息（无需目录扫描）
info := claude.GetSessionInfo("550e8400-e29b-41d4-a716-446655440000", "/path/to/project")
if info != nil {
    fmt.Printf("会话: %s\n", info.Summary)
}

// 获取会话消息
messages, err := claude.GetSessionMessages(
    "550e8400-e29b-41d4-a716-446655440000",
    "/path/to/project",
    10,   // limit (0 = 无限制)
    0,    // offset
)
for _, msg := range messages {
    fmt.Printf("[%s] %v\n", msg.Type, msg.Message)
}

// 重命名会话
err := claude.RenameSession(
    "550e8400-e29b-41d4-a716-446655440000",
    "我的重要会话",
    "/path/to/project",
)

// 标记会话
err := claude.TagSession(
    "550e8400-e29b-41d4-a716-446655440000",
    "重要",
    "/path/to/project",
)
```

### 权限模式

| 模式 | 描述 | 使用场景 |
|------|------|----------|
| `PermissionModeDefault` | 提示权限确认 | 交互式应用 |
| `PermissionModeAcceptEdits` | 自动接受文件编辑 | 代码编辑工具 |
| `PermissionModePlan` | 规划模式 | 复杂多步任务 |
| `PermissionModeBypassPermissions` | 绕过所有权限 | 仅限可信环境 |

```go
// 自动接受编辑
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
})
```

### 自定义系统提示

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    SystemPrompt: `你是一个专注于 Go 语言的专业编程助手。

请遵循以下准则：
1. 始终使用惯用的 Go 代码
2. 尽可能使用标准库
3. 示例中包含错误处理
4. 为复杂逻辑添加注释`,
})
```

### 智能体

为不同任务定义专门的智能体：

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Agents: []types.AgentDefinition{
        {
            Name:        "go-expert",
            Description: "Go 编程专家，用于编写和审查代码",
            Prompt:      "你是一个精通标准库、最佳实践和常见模式的 Go 编程专家。",
            Tools:       []string{"Bash", "Read", "Write", "Edit", "Glob", "Grep"},
            Model:       types.String(types.ModelSonnet),
        },
        {
            Name:        "security-reviewer",
            Description: "安全专家代码审查员",
            Prompt:      "你是一个专注于识别漏洞和提出修复建议的安全专家。",
            Tools:       []string{"Read", "Grep"},
            Model:       types.String(types.ModelOpus),
        },
    },
})
```

### 细粒度工具流

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: types.Bool(true),
})
```

启用后，工具输入增量实时流式传输，支持渐进式 UI 更新。

## 高级主题

### 并发查询

同时处理多个查询：

```go
func processQuery(ctx context.Context, prompt string) error {
    msgChan, err := claude.Query(ctx, prompt, nil)
    if err != nil {
        return err
    }

    for msg := range msgChan {
        if m, ok := msg.(*types.ResultMessage); ok {
            fmt.Printf("结果: %s\n", *m.Result)
        }
    }
    return nil
}

// 并发运行多个查询
var wg sync.WaitGroup
prompts := []string{"1+1等于多少？", "2+2等于多少？", "3+3等于多少？"}

for _, p := range prompts {
    wg.Add(1)
    go func(prompt string) {
        defer wg.Done()
        if err := processQuery(ctx, prompt); err != nil {
            log.Printf("错误: %v", err)
        }
    }(p)
}
wg.Wait()
```

### 上下文取消

正确处理上下文取消：

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

msgChan, err := client.Query(ctx, "你好")
if err != nil {
    log.Fatal(err)
}

for {
    select {
    case msg, ok := <-msgChan:
        if !ok {
            return // 通道已关闭
        }
        // 处理消息
    case <-ctx.Done():
        log.Println("上下文已取消")
        client.Interrupt(context.Background())
        return
    }
}
```

### 自定义传输层

为测试或特殊需求实现自定义传输：

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

// 使用自定义传输
client := client.NewWithOptions(&types.ClaudeAgentOptions{
    Transport: &MockTransport{},
})
```

### 错误恢复

从错误中恢复并继续：

```go
for {
    msgChan, err := client.Query(ctx, prompt)
    if err != nil {
        if errors.Is(err, claude.ErrConnectionFailed) {
            // 尝试重新连接
            time.Sleep(time.Second)
            if err := client.Connect(ctx); err != nil {
                log.Printf("重连失败: %v", err)
                continue
            }
        }
        continue
    }

    // 处理消息...
    break
}
```

## API 参考

### 包函数

```go
// 简单查询（内部创建客户端）
msgChan, err := claude.Query(ctx, "提示", opts)

// 使用现有客户端查询
msgChan, err := claude.QueryWithClient(ctx, client, "提示")

// 创建客户端
client := claude.NewClient()
client := claude.NewClientWithOptions(opts)

// 会话 API
sessions, err := claude.ListSessions(directory, limit, includeWorktrees)
info := claude.GetSessionInfo(sessionID, directory)
messages, err := claude.GetSessionMessages(sessionID, directory, limit, offset)
err := claude.RenameSession(sessionID, title, directory)
err := claude.TagSession(sessionID, tag, directory)
```

### 客户端方法

```go
// 连接
client.Connect(ctx) error
client.Close() error

// 查询
client.Query(ctx, prompt) (<-chan Message, error)
client.ReceiveMessages(ctx) (<-chan Message, error)

// 控制
client.Interrupt(ctx) error
client.StopTask(ctx) error
client.SetPermissionMode(ctx, mode) error
client.SetModel(ctx, model) error

// MCP
client.ReconnectMCPServer(ctx, name) error
client.ToggleMCPServer(ctx, name, enabled) error
client.GetMCPStatus(ctx) (*McpStatusResponse, error)

// 信息
client.GetServerInfo() *ServerInfo
```

### 配置选项参考

| 选项 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| `Model` | `*string` | `"sonnet"` | AI 模型：`"opus"`、`"sonnet"`、`"haiku"` |
| `SystemPrompt` | `string` | `""` | 自定义系统提示 |
| `CWD` | `*string` | 当前目录 | 工作目录 |
| `MaxTurns` | `*int` | 无限制 | 最大对话轮次 |
| `MaxBudgetUSD` | `*float64` | 无限制 | 最大预算（美元） |
| `PermissionMode` | `*PermissionMode` | `default` | 权限处理模式 |
| `CanUseTool` | `func` | `nil` | 工具权限回调 |
| `Hooks` | `map` | `nil` | 事件 hooks |
| `MCPServers` | `map` | `nil` | MCP 服务器配置 |
| `AllowedTools` | `[]string` | 全部 | 允许的工具 |
| `DisallowedTools` | `[]string` | 无 | 禁止的工具 |
| `IncludePartialMessages` | `*bool` | `false` | 启用部分流式传输 |
| `Agents` | `[]AgentDefinition` | `nil` | 自定义智能体定义 |
| `CLIPath` | `*string` | 自动 | Claude CLI 路径 |
| `Env` | `map[string]string` | `nil` | 额外的环境变量 |

### 辅助函数

```go
// 指针辅助函数
types.String("value")           // *string
types.Int(10)                   // *int
types.Float64(1.5)              // *float64
types.Bool(true)                // *bool
types.PermissionModePtr(mode)   // *PermissionMode

// MCP 工具的 Schema 辅助函数
sdkmcp.Schema(props, required)      // 带必填字段的完整 schema
sdkmcp.SimpleSchema(props)          // 简单 schema（无必填字段）
sdkmcp.StringProperty(desc)         // 字符串属性
sdkmcp.NumberProperty(desc)         // 数字属性
sdkmcp.BooleanProperty(desc)        // 布尔属性
sdkmcp.ObjectProperty(props, req)   // 嵌套对象属性
sdkmcp.ArrayProperty(items)         // 数组属性
```

## 消息类型

### 顶级消息

| 类型 | 描述 | 关键字段 |
|------|------|----------|
| `ResultMessage` | 最终结果 | `Result`, `SessionID`, `TotalCostUSD`, `DurationMS`, `NumTurns`, `StopReason` |
| `AssistantMessage` | Claude 的响应 | `Content`, `Model`, `Usage` |
| `UserMessage` | 用户输入 | `Content` |
| `SystemMessage` | 系统事件 | `Subtype`, `Data` |
| `StreamEvent` | 流式事件 | `Type`, `Data` |
| `RateLimitEvent` | 速率限制信息 | `Type`, `Data` |

### 内容块

| 类型 | 描述 | 关键字段 |
|------|------|----------|
| `TextBlock` | 文本内容 | `Text` |
| `ThinkingBlock` | 扩展思考 | `Thinking` |
| `ToolUseBlock` | 工具请求 | `ID`, `Name`, `Input` |
| `ToolResultBlock` | 工具结果 | `ToolUseID`, `Content`, `IsError` |
| `GenericContentBlock` | 未知类型 | `Type`, `Raw` |

## 错误处理

```go
import (
    "errors"
    "log"

    claude "github.com/unitsvc/claude-agent-sdk-golang"
    sdkerrors "github.com/unitsvc/claude-agent-sdk-golang/errors"
)

msgChan, err := client.Query(ctx, "你好")
if err != nil {
    // 检查哨兵错误
    switch {
    case errors.Is(err, claude.ErrNoAPIKey):
        log.Fatal("API 密钥未配置。请运行: claude login")

    case errors.Is(err, claude.ErrNotInstalled):
        log.Fatal("Claude CLI 未安装。请运行: npm install -g @anthropic-ai/claude-code")

    case errors.Is(err, claude.ErrConnectionFailed):
        log.Fatal("连接失败。Claude CLI 是否正在运行？")

    case errors.Is(err, claude.ErrTimeout):
        log.Fatal("操作超时")

    case errors.Is(err, claude.ErrInterrupted):
        log.Println("操作被中断")
        return
    }

    // 检查错误类型
    var cliErr *sdkerrors.CLIError
    if errors.As(err, &cliErr) {
        log.Printf("CLI 错误: %s (退出码: %d)", cliErr.Message, cliErr.ExitCode)
        log.Printf("Stderr: %s", cliErr.Stderr)
    }

    var procErr *sdkerrors.ProcessError
    if errors.As(err, &procErr) {
        log.Printf("进程错误: %v", procErr)
    }

    log.Fatal(err)
}
```

## 示例

[examples](examples/) 目录包含完整的示例：

| 示例 | 描述 |
|------|------|
| `quick_start` | 基本用法模式 |
| `streaming_mode` | 消息流式技术 |
| `streaming_interactive` | 带上下文的交互式流式传输 |
| `streaming_goroutines` | 并发流式模式 |
| `hooks` | 包含所有事件的 Hook 系统 |
| `tool_permission` | 权限回调示例 |
| `mcp_calculator` | MCP 服务器集成 |
| `mcp_sdk_simple` | 简单的进程内 MCP 服务器 |
| `mcp_sdk_server` | 功能完整的 SDK MCP 服务器 |
| `mcp_control` | MCP 服务器运行时控制 |
| `agents` | 自定义智能体定义 |
| `system_prompt` | 系统提示配置 |
| `setting_sources` | 设置和配置 |
| `budget` | 预算管理 |
| `include_partial_messages` | 部分消息处理 |
| `stderr_callback` | Stderr 输出处理 |
| `tools_option` | 工具配置 |
| `filesystem_agents` | 文件系统操作 |
| `task_messages` | 任务事件处理 |
| `plugin_example` | 插件集成 |

## 测试

```bash
# 运行所有测试
go test ./...

# 运行覆盖率测试
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行短测试（跳过 E2E）
go test -short ./...

# 运行特定包
go test ./internal/sessions/... -v

# 运行竞态检测
go test -race ./...
```

## 性能

### 基准测试

```bash
# 运行基准测试
go test -bench=. ./...

# 内存分析
go test -bench=. -benchmem ./...
```

### 优化建议

1. **复用客户端**：创建一个客户端并复用于多个查询
2. **协程池**：使用工作池处理并发查询
3. **缓冲通道**：高吞吐场景使用带缓冲的通道
4. **上下文超时**：设置合理的超时防止阻塞

```go
// 推荐：复用客户端
client := claude.NewClientWithOptions(opts)
defer client.Close()
client.Connect(ctx)

for _, prompt := range prompts {
    msgChan, _ := client.Query(ctx, prompt)
    // 处理消息...
}

// 不推荐：每次创建新客户端
for _, prompt := range prompts {
    msgChan, _ := claude.Query(ctx, prompt, opts) // 每次创建新客户端
    // 处理消息...
}
```

## 安全

### 最佳实践

1. **不要硬编码 API 密钥** - 使用环境变量或安全存储
2. **验证输入** - 发送到 Claude 前清理用户输入
3. **限制权限** - 使用 `AllowedTools` 和 `CanUseTool` 限制工具访问
4. **沙箱写入** - 将文件写入重定向到安全目录
5. **审计日志** - 记录所有工具执行以便安全审计

```go
// 示例：安全配置
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    // 限制工具
    AllowedTools: []string{"Read", "Bash"},

    // 验证和沙箱
    CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
        // 审计日志
        log.Printf("工具请求: %s 来自用户", toolName)

        // 验证输入
        if toolName == "Bash" {
            if cmd, ok := input["command"].(string); ok {
                // 阻止危险命令
                if strings.Contains(cmd, "rm -rf") {
                    return types.PermissionResultDeny{
                        Behavior: "deny",
                        Message:  "不允许执行破坏性命令",
                    }, nil
                }
            }
        }

        return types.PermissionResultAllow{Behavior: "allow"}, nil
    },
})
```

### 环境变量

| 变量 | 描述 |
|------|------|
| `ANTHROPIC_API_KEY` | Anthropic API 密钥 |
| `CLAUDE_CONFIG_DIR` | 自定义配置目录 |
| `CLAUDE_CODE_ENTRYPOINT` | 入口点标识符 |

## 最佳实践

### 资源管理

```go
// 总是关闭客户端
client := claude.NewClientWithOptions(opts)
defer client.Close()

// 总是取消上下文
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 错误处理

```go
// 检查所有错误
msgChan, err := client.Query(ctx, prompt)
if err != nil {
    // 处理特定错误
    switch {
    case errors.Is(err, claude.ErrNoAPIKey):
        // 处理缺失 API 密钥
    case errors.Is(err, claude.ErrTimeout):
        // 处理超时
    default:
        // 处理其他错误
    }
}
```

### 并发

```go
// 使用 sync.WaitGroup 协调
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

## 常见问题

### 如何设置自定义工作目录？

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CWD: types.String("/path/to/project"),
})
```

### 如何限制对话轮次？

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MaxTurns: types.Int(5),
})
```

### 如何设置预算限制？

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    MaxBudgetUSD: types.Float64(1.0),  // 最大 $1.00
})
```

### 如何使用特定模型？

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Model: types.String(types.ModelOpus),  // "opus", "sonnet", "haiku"
})
```

### 如何优雅地处理 Ctrl+C？

```go
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer cancel()

client := claude.NewClientWithOptions(opts)
defer client.Close()

if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}

// 所有操作使用 ctx - 会在 Ctrl+C 时取消
msgChan, err := client.Query(ctx, "你好")
```

### 如何在查询后获取会话 ID？

```go
for msg := range msgChan {
    if m, ok := msg.(*types.ResultMessage); ok {
        fmt.Printf("会话 ID: %s\n", m.SessionID)
    }
}
```

### 如何流式传输部分消息？

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: types.Bool(true),
})
```

### 如何使用多个 MCP 服务器？

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

## 故障排除

### "Claude CLI not installed"

```bash
# 安装 Claude CLI
npm install -g @anthropic-ai/claude-code

# 验证安装
claude --version
```

### "API key not configured"

```bash
# 登录 Anthropic
claude login

# 或设置环境变量
export ANTHROPIC_API_KEY=your-key-here
```

### "Connection failed"

1. 检查 Claude CLI 是否在 PATH 中
2. 尝试在终端直接运行 `claude`
3. 检查 CLI 更新：`npm update -g @anthropic-ai/claude-code`

### "Tool not found"

MCP 工具的命名格式为 `mcp__<server>__<tool>`：

```go
// 正确的工具名格式
AllowedTools: []string{"mcp__calc__add", "mcp__calc__multiply"}
```

### "Context deadline exceeded"

为长时间运行的查询增加超时：

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### "Permission denied"

检查 `CanUseTool` 回调或 `PermissionMode`：

```go
// 方式一：使用绕过模式（生产环境不推荐）
PermissionMode: types.PermissionModePtr(types.PermissionModeBypassPermissions)

// 方式二：将工具添加到允许列表
AllowedTools: []string{"Bash", "Read", "Write"}

// 方式三：实现 CanUseTool 回调
CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
    return types.PermissionResultAllow{Behavior: "allow"}, nil
}
```

## 迁移指南

### 从 Python SDK 迁移

| Python | Go |
|--------|-----|
| `from claude_agent_sdk import Query` | `import claude "github.com/unitsvc/claude-agent-sdk-golang"` |
| `async for msg in query(...)` | `for msg := range claude.Query(...)` |
| `options=ClaudeAgentOptions(...)` | `&types.ClaudeAgentOptions{...}` |
| `permission_result_allow()` | `types.PermissionResultAllow{Behavior: "allow"}` |
| `@tool` 装饰器 | `sdkmcp.Tool(...)` |

### 主要区别

1. **异步 vs Channel**：Python 使用 `async/await`，Go 使用 channel
2. **选项**：Go 使用结构体指针表示可选字段
3. **错误**：Go 将错误作为值返回，Python 抛出异常
4. **Context**：Go 需要显式的 context 用于取消

## 更新日志

### v0.1.50 (2026-03-23)

- 新增 `GetSessionInfo()` 用于单会话查询
- `SDKSessionInfo.FileSize` 改为可选，支持远程存储
- 更新至 Python SDK v0.1.50

### v0.1.49

- 新增 `RenameSession()` 和 `TagSession()` 函数
- `SDKSessionInfo` 新增 `Tag` 和 `CreatedAt` 字段
- 修复会话标题和摘要链提取

### v0.1.48

- 新增细粒度工具流式支持
- `AssistantMessage` 新增 `Usage` 字段
- 修复优雅的子进程关闭

### v0.1.46

- 新增会话 API：`ListSessions()`、`GetSessionMessages()`
- Hook 输入新增代理上下文字段

### v0.1.45

- `ResultMessage` 新增 `StopReason` 字段
- 新增任务消息类型
- 新增 MCP 控制方法：`ReconnectMCPServer()`、`ToggleMCPServer()`、`StopTask()`

详见 [CHANGELOG.md](CHANGELOG.md)。

## Go SDK 优势

| 功能 | Python SDK | Go SDK |
|------|-----------|--------|
| Hook 事件 | 10 | 12 (+SessionStart, SessionEnd) |
| 单元测试 | 153 | 360+ |
| E2E 测试 | 32 | 55+ |
| Schema 辅助函数 | 有限 | 完整（Schema、SimpleSchema、Property helpers） |
| Transport | 内部 | 导出接口 |
| 速率限制事件 | 无 | 有（RateLimitEvent） |
| 通用内容块 | 无 | 有（GenericContentBlock） |
| 并发安全 | 部分 | 是（基于 channel） |
| 内存占用 | 较高 | 较低 |

## 版本

**当前版本**: 0.1.50-a7fd631

与 [Python SDK v0.1.50](https://github.com/anthropics/claude-agent-sdk-python) 同步。

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE)。

## 贡献

欢迎贡献！请：

1. Fork 仓库
2. 创建功能分支
3. 为新功能添加测试
4. 提交 Pull Request

### 开发环境配置

```bash
# 克隆仓库
git clone https://github.com/unitsvc/claude-agent-sdk-golang.git
cd claude-agent-sdk-golang

# 安装依赖
go mod download

# 运行测试
go test ./...

# 运行代码检查
go vet ./...
```

## 相关项目

- [Claude Agent SDK (Python)](https://github.com/anthropics/claude-agent-sdk-python) - 官方 Python SDK
- [Claude Code](https://github.com/anthropics/claude-code) - 官方 CLI 工具
- [MCP 规范](https://modelcontextprotocol.io/) - 模型上下文协议
- [Anthropic API](https://docs.anthropic.com/) - Anthropic API 文档