# Claude Agent SDK for Golang

<p align="center">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="License: MIT">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go 1.26+">
  <a href="https://pkg.go.dev/github.com/next-bin/claude-agent-sdk-golang"><img src="https://pkg.go.dev/badge/github.com/next-bin/claude-agent-sdk-golang.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/next-bin/claude-agent-sdk-golang"><img src="https://goreportcard.com/badge/github.com/next-bin/claude-agent-sdk-golang" alt="Go Report Card"></a>
</p>

<p align="center">
  <a href="README.md">English</a>
</p>

用于构建 Claude 智能体的 Go SDK。提供查询、交互式会话、自定义工具、钩子和会话管理等高级 API。

## 目录

- **入门**
  - [安装](#安装)
  - [快速开始](#快速开始)
- **核心概念**
  - [基本用法](#基本用法)
    - [工具权限](#工具权限)
    - [工作目录](#工作目录)
  - [交互式会话](#交互式会话)
  - [错误处理](#错误处理)
- **高级功能**
  - [自定义工具](#自定义工具)
    - [混合服务器](#混合服务器)
  - [钩子](#钩子)
    - [可用钩子事件](#可用钩子事件)
  - [会话 API](#会话-api)
  - [动态控制](#动态控制)
- **资源**
  - [示例](#示例)
  - [贡献](#贡献)
  - [相关项目](#相关项目)
- [错误处理](#错误处理)
- [示例](#示例)
- [贡献](#贡献)

## 安装

```bash
go get github.com/next-bin/claude-agent-sdk-golang
```

**依赖：**

| 依赖                | 说明                                                                   |
| ------------------- | ---------------------------------------------------------------------- |
| **Go**              | 1.26 或更高版本                                                        |
| **Claude Code CLI** | 已安装并认证（[安装指南](https://code.claude.com/docs/en/quickstart)） |

## 快速开始

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

    msgChan, err := claude.Query(ctx, "2 + 2 等于几？", nil)
    if err != nil {
        log.Fatal(err)
    }

    for msg := range msgChan {
        fmt.Printf("%v\n", msg)
    }
}
```

## 基本用法

### 简单查询

```go
ctx := context.Background()
msgChan, err := claude.Query(ctx, "你好 Claude", nil)
```

### 配置选项

```go
import "github.com/next-bin/claude-agent-sdk-golang/types"

opts := &types.ClaudeAgentOptions{
    SystemPrompt: types.String("你是一个有用的助手"),
    MaxTurns:     types.Int(1),
}

msgChan, err := claude.Query(ctx, "讲个笑话", opts)
```

### 工作目录

```go
opts := &types.ClaudeAgentOptions{
    CWD: "/path/to/project",
}
```

### 工具权限

默认情况下，Claude 拥有完整的 [Claude Code 工具集](https://code.claude.com/docs/en/settings#tools-available-to-claude)。`AllowedTools` 是自动批准列表，未列出的工具会走 `PermissionMode` 和 `CanUseTool` 决策流程。

```go
opts := &types.ClaudeAgentOptions{
    AllowedTools:   []string{"Read", "Write", "Bash"},
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
}
```

## 交互式会话

对于需要后续消息的对话，使用 `client.Client`：

```go
import "github.com/next-bin/claude-agent-sdk-golang/client"

c := client.NewWithOptions(&types.ClaudeAgentOptions{
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
})
defer c.Close()

// 连接并发送初始消息
err := c.Connect(ctx, "你好 Claude")

// 读取响应
for msg := range c.ReceiveResponse(ctx) {
    fmt.Printf("%T: %v\n", msg, msg)
}

// 发送后续消息
err = c.Query(ctx, "能再详细解释一下吗？")
```

## 自定义工具

将自定义工具定义为进程内 MCP 服务器，无需管理子进程。

```go
import "github.com/next-bin/claude-agent-sdk-golang/sdkmcp"

greetTool := sdkmcp.Tool(
    "greet",
    "打招呼",
    sdkmcp.SimpleSchema(map[string]string{"name": "string"}),
    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
        name := args["name"].(string)
        return sdkmcp.TextResult(fmt.Sprintf("你好，%s！", name)), nil
    },
)

server := sdkmcp.CreateSdkMcpServer("my-tools", []*sdkmcp.SdkMcpTool{greetTool})

opts := &types.ClaudeAgentOptions{
    MCPServers:   map[string]types.McpServerConfig{"tools": server},
    AllowedTools: []string{"mcp__tools__greet"},
}
```

## 钩子

钩子是在智能体循环特定节点由 Claude Code 应用程序调用的函数。

```go
type bashHook struct{}

func (h *bashHook) Execute(input types.HookInput, toolUseID *string, ctx types.HookContext) (types.HookJSONOutput, error) {
    hookInput, ok := input.(types.PreToolUseHookInput)
    if !ok {
        return types.SyncHookJSONOutput{Continue_: types.Bool(true)}, nil
    }

    command, _ := hookInput.ToolInput["command"].(string)
    if strings.Contains(command, "rm -rf") {
        reason := "危险命令已被钩子拦截"
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
```

### 可用钩子事件

| 钩子                 | 说明           |
| -------------------- | -------------- |
| `PreToolUse`         | 工具执行前     |
| `PostToolUse`        | 工具执行后     |
| `PostToolUseFailure` | 工具失败时     |
| `UserPromptSubmit`   | 用户提交消息时 |
| `Stop`               | 智能体停止时   |
| `SubagentStart`      | 子智能体启动时 |
| `SubagentStop`       | 子智能体停止时 |
| `PreCompact`         | 上下文压缩前   |
| `Notification`       | 通知           |
| `PermissionRequest`  | 请求权限时     |

## 会话 API

以编程方式管理对话会话。

```go
import claude "github.com/next-bin/claude-agent-sdk-golang"

// 列出会话
sessions, err := claude.ListSessions("/path/to/project", 10, true)

// 获取会话消息
messages, err := claude.GetSessionMessages(sessionID, "/path/to/project", 0, 0)

// 获取单个会话信息
info := claude.GetSessionInfo(sessionID, "/path/to/project")

// 会话操作
err = claude.RenameSession(sessionID, "新标题", "/path/to/project")
err = claude.TagSession(sessionID, "实验", "/path/to/project")
err = claude.DeleteSession(sessionID, "/path/to/project")
result, err := claude.ForkSession(sessionID, "/path/to/project", nil, nil)
```

## 动态控制

在运行时控制活跃会话。

```go
err := c.Connect(ctx)

// 切换权限模式
err = c.SetPermissionMode(ctx, "acceptEdits")

// 切换模型
err = c.SetModel(ctx, "claude-sonnet-4-6")

// 获取上下文使用率
usage, err := c.GetContextUsage(ctx)
fmt.Printf("已使用 %.1f%% 上下文\n", usage.Percentage)

// 获取 MCP 服务器状态
status, err := c.GetMCPStatus(ctx)

// 中断对话
err = c.Interrupt(ctx)
```

## 错误处理

```go
import claude "github.com/next-bin/claude-agent-sdk-golang"

msgChan, err := claude.Query(ctx, "Hello", nil)
if err != nil {
    switch {
    case claude.ErrNotInstalled:
        fmt.Println("请安装 Claude Code")
    case claude.ErrConnectionFailed:
        fmt.Println("连接失败")
    case claude.ErrTimeout:
        fmt.Println("查询超时")
    default:
        fmt.Printf("错误: %v\n", err)
    }
}
```

## 示例

| 示例                                         | 说明         |
| -------------------------------------------- | ------------ |
| [quick_start](examples/quick_start/)         | 基本查询     |
| [streaming_mode](examples/streaming_mode/)   | 交互式客户端 |
| [mcp_sdk_server](examples/mcp_sdk_server/)   | 自定义工具   |
| [hooks](examples/hooks/)                     | 钩子系统     |
| [tool_permission](examples/tool_permission/) | 权限回调     |
| [agents](examples/agents/)                   | 自定义智能体 |

## 贡献

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 为新功能添加测试
4. 提交 Pull Request

### 开发

```bash
git clone https://github.com/next-bin/claude-agent-sdk-golang.git
cd claude-agent-sdk-golang
go mod download
go test ./...
go vet ./...
```

## 相关项目

- [Claude Code 文档](https://code.claude.com/docs/en) — Claude Code 文档
- [MCP 规范](https://modelcontextprotocol.io/) — Model Context Protocol
- [Anthropic API](https://docs.anthropic.com/) — Anthropic API 文档
