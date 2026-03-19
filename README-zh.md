# Claude Agent SDK for Go (Go 版 Claude Agent SDK)

[English Version](README.md)

用于使用 Claude 构建 AI 代理的 Go SDK。本 SDK 提供了 Claude Agent SDK 的 Go 实现，使您能够构建可以使用工具、处理权限并与 MCP 服务器交互的 AI 代理。

## 前置要求

- Go 1.21 或更高版本
- Claude Code CLI 已安装并认证:
  ```bash
  npm install -g @anthropic-ai/claude-code
  claude login
  ```

## 安装

```bash
go get github.com/unitsvc/claude-agent-sdk-golang
```

## 快速开始

### 简单查询

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

    // 简单的单次查询
    msgChan, err := claude.Query(ctx, "2+2 等于几?", nil)
    if err != nil {
        log.Fatal(err)
    }

    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.ResultMessage:
            if m.Result != nil {
                fmt.Printf("结果: %s\n", *m.Result)
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

### 带选项的客户端

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

    // 创建带自定义选项的客户端
    client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
        Model: types.String(types.ModelSonnet),
    })
    defer client.Close()

    // 连接到 Claude
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    // 发送查询
    msgChan, err := client.Query(ctx, "讲个笑话")
    if err != nil {
        log.Fatal(err)
    }

    for msg := range msgChan {
        switch m := msg.(type) {
        case *types.ResultMessage:
            if m.Result != nil {
                fmt.Printf("结果: %s\n", *m.Result)
            }
        }
    }
}
```

## 功能特性

### 流式消息

SDK 在生成消息时流式传输:

```go
for msg := range msgChan {
    switch m := msg.(type) {
    case *types.AssistantMessage:
        // 来自助手的流式文本
        for _, block := range m.Content {
            if tb, ok := block.(types.TextBlock); ok {
                fmt.Print(tb.Text)
            }
            if tb, ok := block.(types.ThinkingBlock); ok {
                fmt.Printf("[思考: %s]\n", tb.Thinking)
            }
        }
    case *types.ResultMessage:
        // 最终结果
        if m.TotalCostUSD != nil {
            fmt.Printf("\n费用: $%.4f\n", *m.TotalCostUSD)
        }
        fmt.Printf("耗时: %dms\n", m.DurationMS)
    }
}
```

### 权限处理

通过回调控制工具权限:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
        if toolName == "Bash" {
            // 自动批准 Bash 命令
            return types.PermissionResultAllow{
                Behavior: "allow",
            }, nil
        }
        // 拒绝其他工具
        return types.PermissionResultDeny{
            Behavior: "deny",
            Message:  "权限被拒绝",
        }, nil
    },
})
```

### 自定义系统提示

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    SystemPrompt: "你是一个专门研究 Go 语言的编程助手.",
})
```

### 工作目录

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    CWD: types.String("/path/to/project"),
})
```

### MCP 服务器

配置 MCP (Model Context Protocol) 服务器:

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

### SDK MCP 服务器 (进程内)

创建带有自定义工具的进程内 MCP 服务器:

```go
import "github.com/unitsvc/claude-agent-sdk-golang/sdkmcp"

// 定义一个工具
addTool := sdkmcp.Tool("add", "将两个数字相加",
    sdkmcp.Schema(map[string]interface{}{
        "a": sdkmcp.NumberProperty("第一个数字"),
        "b": sdkmcp.NumberProperty("第二个数字"),
    }, []string{"a", "b"}),
    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
        a, _ := args["a"].(float64)
        b, _ := args["b"].(float64)
        return sdkmcp.TextResult(fmt.Sprintf("结果: %.2f", a+b)), nil
    })

// 创建服务器
calcServer := sdkmcp.CreateSdkMcpServer("calculator", []*sdkmcp.SdkMcpTool{addTool})

// 与客户端一起使用
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

### Hooks (钩子)

注册工具事件钩子:

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

**钩子**是 Claude Code _应用程序_ (不是 Claude) 在 Claude 代理循环的特定点调用的 Go 函数。钩子可以提供确定性处理和自动化反馈。了解更多请参阅 [Claude Code Hooks 参考文档](https://docs.anthropic.com/en/docs/claude-code/hooks)。

**可用的钩子事件:**
- `HookEventPreToolUse` - 工具执行前
- `HookEventPostToolUse` - 工具执行成功后
- `HookEventPostToolUseFailure` - 工具执行失败后
- `HookEventUserPromptSubmit` - 用户提交提示时
- `HookEventStop` - 对话停止时
- `HookEventSubagentStart` - 子代理启动时
- `HookEventSubagentStop` - 子代理停止时
- `HookEventPreCompact` - 上下文压缩前
- `HookEventNotification` - 通知
- `HookEventPermissionRequest` - 权限请求
- `HookEventSessionStart` - 会话开始时 (仅 Go SDK)
- `HookEventSessionEnd` - 会话结束时 (仅 Go SDK)

完整的示例请参阅 [examples/hooks/main.go](examples/hooks/main.go)。

### 权限模式

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    PermissionMode: types.PermissionModePtr(types.PermissionModeAcceptEdits),
})
```

可用模式:
- `PermissionModeDefault` - 默认行为
- `PermissionModeAcceptEdits` - 自动接受文件编辑
- `PermissionModePlan` - 规划模式
- `PermissionModeBypassPermissions` - 绕过所有权限

### Sessions API (会话 API)

列出和检索会话历史:

```go
// 列出所有会话
sessions, err := claude.ListSessions(ctx)
for _, sess := range sessions {
    fmt.Printf("会话: %s (%s)\n", sess.SessionID, sess.CustomTitle)
}

// 从特定会话获取消息
messages, err := claude.GetSessionMessages(ctx, sessionID)
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.Role, msg.Content)
}
```

### 会话变更

重命名和标记会话:

```go
// 重命名会话
err := claude.RenameSession(ctx, sessionID, "我的新标题")

// 标记会话
err := claude.TagSession(ctx, sessionID, "重要", "golang")
```

### 客户端控制方法

```go
// 重新连接 MCP 服务器
err := client.ReconnectMCPServer(ctx, "myServer")

// 切换 MCP 服务器开关
err := client.ToggleMCPServer(ctx, "myServer", false)

// 停止当前任务
err := client.StopTask(ctx)
```

### 细粒度工具流

通过 `include_partial_messages` 启用详细工具流:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: true,
})
```

启用后,SDK 会自动设置 `CLAUDE_CODE_ENABLE_FINE_GRAINED_TOOL_STREAMING=1` 以获取详细的工具输入增量。

### Agents (代理)

定义具有特定配置的自定义代理:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    Agents: []types.AgentDefinition{
        {
            Description: "Go 专家",
            Prompt:      "你是一个 Go 编程专家.",
            Tools:       []string{"Bash", "Read", "Write"},
            Model:       types.String(types.ModelSonnet),
            Skills:      []string{"golang"},
        },
    },
})
```

### 流式传输模式

SDK 支持多种流式传输模式。查看 [examples/streaming_mode/main.go](examples/streaming_mode/main.go) 获取完整演示:

```go
// 带消息类型过滤的交互式流式传输
for msg := range msgChan {
    switch m := msg.(type) {
    case *types.AssistantMessage:
        // 处理流式文本
    case *types.ResultMessage:
        // 处理最终结果
    }
}
```

### 包含部分消息

在流式传输时接收部分消息内容:

```go
client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
    IncludePartialMessages: true,
})
```

## API 参考

### `Query(ctx, prompt, options)`

简单的单次查询函数。

```go
msgChan, err := claude.Query(ctx, "2+2 等于几?", nil)
```

### `QueryWithClient(ctx, client, prompt)`

使用现有客户端进行查询。

```go
msgChan, err := claude.QueryWithClient(ctx, client, "你好!")
```

### `Client`

用于交互式对话的全功能客户端。

#### 方法:
- `Connect(ctx)` - 建立连接
- `Query(ctx, prompt)` - 发送查询
- `ReceiveMessages(ctx)` - 接收消息直到 ResultMessage
- `Interrupt(ctx)` - 中断当前操作
- `SetPermissionMode(ctx, mode)` - 更改权限模式
- `SetModel(ctx, model)` - 更改 AI 模型
- `ReconnectMCPServer(ctx, name)` - 重新连接 MCP 服务器
- `ToggleMCPServer(ctx, name, enabled)` - 切换 MCP 服务器
- `StopTask(ctx)` - 停止运行中的任务
- `GetServerInfo()` - 获取服务器初始化信息
- `GetMCPStatus()` - 获取 MCP 服务器状态
- `Close()` - 关闭连接

### Sessions API 函数

- `ListSessions(ctx)` - 列出所有会话
- `GetSessionMessages(ctx, sessionID)` - 从会话中获取消息
- `RenameSession(ctx, sessionID, title)` - 重命名会话
- `TagSession(ctx, sessionID, tags...)` - 标记会话

### 选项

| 选项 | 类型 | 描述 |
|--------|------|-------------|
| `Model` | `*string` | AI 模型 (`"opus"`, `"sonnet"`, `"haiku"`, `"inherit"`) |
| `SystemPrompt` | `string` | 自定义系统提示 |
| `CWD` | `*string` | 工作目录 |
| `MaxTurns` | `*int` | 最大对话轮次 |
| `MaxBudgetUSD` | `*float64` | 最大预算 (USD) |
| `PermissionMode` | `*PermissionMode` | 权限处理模式 |
| `CanUseTool` | `func` | 权限回调函数 |
| `Hooks` | `map` | 事件钩子 |
| `MCPServers` | `map` | MCP 服务器配置 |
| `AllowedTools` | `[]string` | 允许的工具 |
| `DisallowedTools` | `[]string` | 禁止的工具 |
| `IncludePartialMessages` | `*bool` | 启用部分消息流式传输 |
| `Agents` | `[]AgentDefinition` | 代理定义 |
| `SystemPromptPresets` | `[]SystemPromptPreset` | 系统提示预设 |
| `ToolsPresets` | `[]ToolsPreset` | 工具预设 |

### 辅助函数

```go
// 可选字段的指针辅助函数
types.String("value")     // *string
types.Int(10)             // *int
types.Float64(1.5)        // *float64
types.Bool(true)          // *bool
types.PermissionModePtr(types.PermissionModeAcceptEdits)  // *PermissionMode

// MCP 工具的 Schema 辅助函数
sdkmcp.Schema(props, required)    // 创建输入 schema
sdkmcp.StringProperty(desc)       // 字符串属性
sdkmcp.NumberProperty(desc)      // 数字属性
sdkmcp.BooleanProperty(desc)     // 布尔属性
sdkmcp.ObjectProperty(props)      // 对象属性
sdkmcp.ArrayProperty(items)      // 数组属性
```

## 消息类型

- `ResultMessage` - 查询的最终结果 (包含费用、耗时、stop_reason)
- `AssistantMessage` - Claude 的流式文本 (包含模型、使用量)
- `UserMessage` - 用户消息
- `SystemMessage` - 系统消息 (子类型: task_started, task_progress, task_notification)
- `StreamEvent` - 流式事件
- `RateLimitEvent` - 速率限制事件 (仅 Go SDK)

### ResultMessage 字段

```go
type ResultMessage struct {
    Subtype        string
    DurationMS     int
    DurationAPIMS  int
    IsError        bool
    NumTurns       int
    SessionID      string
    StopReason     *string           // "stop", "early_stop", "error" 等
    TotalCostUSD   *float64
    Usage          map[string]interface{}  // 每轮使用量
    Result         *string
    StructuredOutput interface{}
}
```

### 内容块

- `TextBlock` - 文本内容
- `ThinkingBlock` - 思考内容 (扩展思考)
- `ToolUseBlock` - 工具使用请求 (id, name, input)
- `ToolResultBlock` - 工具执行结果 (tool_use_id, content, is_error)
- `GenericContentBlock` - 未知块类型 (仅 Go SDK)

## 错误处理

```go
import "github.com/unitsvc/claude-agent-sdk-golang/errors"

msgChan, err := client.Query(ctx, "你好")
if err != nil {
    // 检查哨兵错误
    if errors.Is(err, claude.ErrNoAPIKey) {
        log.Fatal("未配置 API 密钥")
    }
    if errors.Is(err, claude.ErrNotInstalled) {
        log.Fatal("Claude CLI 未安装")
    }

    // 检查特定错误类型
    var cliErr *errors.CLIError
    if errors.As(err, &cliErr) {
        log.Printf("CLI 错误: %s (退出代码: %d)", cliErr.Message, cliErr.ExitCode)
    }

    var connErr *errors.CLIConnectionError
    if errors.As(err, &connErr) {
        log.Printf("连接错误: %s", connErr.Message)
    }

    log.Fatal(err)
}
```

**可用的哨兵错误:**
- `ErrNoAPIKey` - 未配置 API 密钥
- `ErrNotInstalled` - Claude CLI 未安装
- `ErrConnectionFailed` - 连接失败
- `ErrTimeout` - 操作超时
- `ErrInterrupted` - 操作被中断

## 示例

查看 [examples](examples/) 目录获取详细的使用示例:

| 示例 | 描述 |
|---------|-------------|
| `quick_start` | 基本用法示例 |
| `streaming_mode` | 消息流式传输模式 |
| `streaming_interactive` | 交互式流式传输 |
| `streaming_goroutines` | 基于 Goroutine 的流式传输 |
| `hooks` | Hook 系统用法 |
| `tool_permission` | 权限回调 |
| `mcp_calculator` | MCP 服务器示例 |
| `mcp_sdk_simple` | 简单 SDK MCP 服务器 |
| `mcp_sdk_server` | 完整 SDK MCP 服务器 |
| `mcp_control` | MCP 服务器控制 |
| `agents` | 自定义代理定义 |
| `system_prompt` | 自定义系统提示 |
| `setting_sources` | 设置配置 |
| `budget` | 预算管理 |
| `include_partial_messages` | 部分消息处理 |
| `stderr_callback` | Stderr 处理 |
| `tools_option` | 工具配置 |
| `filesystem_agents` | 文件系统代理 |
| `task_messages` | 任务消息处理 |
| `plugin_example` | 插件用法 |

## Go SDK 优势

1. **更多钩子事件** - SessionStart, SessionEnd (Python 版没有)
2. **更好的 Schema 辅助函数** - Schema(), SimpleSchema(), StringProperty() 等
3. **更多测试** - 350+ 单元测试, 50+ E2E 测试
4. **类型安全** - 编译时类型检查
5. **并发** - Goroutine + channel 模式
6. **导出的传输层** - 支持自定义传输实现
7. **哨兵错误** - ErrNoAPIKey, ErrNotInstalled 等
8. **RateLimitEvent** - 解析速率限制事件 (仅 Go SDK)
9. **GenericContentBlock** - 前向兼容的未知块处理

## 测试

```bash
# 运行所有测试
go test ./...

# 带覆盖率运行
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行简短测试 (跳过 E2E)
go test -short ./...

# 运行特定包
go test ./client/...
```

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE)。

## 版本

**当前版本**: 0.1.48-971994c

与 Python SDK v0.1.48+ 同步。
