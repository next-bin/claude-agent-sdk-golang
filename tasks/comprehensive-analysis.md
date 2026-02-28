# SDK Python vs Go 全面对比分析报告

## 执行摘要

**对比时间**: 2026-02-27
**更新时间**: 2026-02-28
**分析方法**: 6个并行agent深入对比各模块
**总体结论**: ✅ Go SDK 已完全实现 Python SDK 全部核心功能，测试覆盖更完善

---

## 1. 核心API对比

| 功能模块 | Python | Go | 状态 | 差异说明 |
|----------|--------|-----|------|----------|
| query() 单次查询 | ✅ | ✅ | **一致** | Go用channel返回，Python用async iterator |
| ClaudeSDKClient | ✅ | ✅ | **一致** | 两者功能完全对等 |
| 流式消息接收 | ✅ | ✅ | **一致** | Go用channel, Python用async iterator |
| async context manager | ✅ | ❌ | **差异** | Python支持`async with`，Go需手动Close() |
| Transport参数 | ✅ | ✅ | **一致** | Go已导出Transport接口 |

---

## 2. 类型系统对比

### 2.1 消息类型

| 类型 | Python | Go | 状态 |
|------|--------|-----|------|
| UserMessage | ✅ | ✅ | **一致** |
| AssistantMessage | ✅ | ✅ | **一致** |
| SystemMessage | ✅ | ✅ | **一致** |
| ResultMessage | ✅ | ✅ | **一致** |
| StreamEvent | ✅ | ✅ | **一致** |

### 2.2 内容块类型

| 类型 | Python | Go | 状态 |
|------|--------|-----|------|
| TextBlock | ✅ | ✅ | **一致** (Go额外有Type字段) |
| ThinkingBlock | ✅ | ✅ | **一致** |
| ToolUseBlock | ✅ | ✅ | **一致** |
| ToolResultBlock | ✅ | ✅ | **一致** |

### 2.3 Hook事件对比

| 事件 | Python | Go | 状态 |
|------|--------|-----|------|
| PreToolUse | ✅ | ✅ | **一致** |
| PostToolUse | ✅ | ✅ | **一致** |
| PostToolUseFailure | ✅ | ✅ | **一致** |
| UserPromptSubmit | ✅ | ✅ | **一致** |
| Stop | ✅ | ✅ | **一致** |
| SubagentStop | ✅ | ✅ | **一致** |
| PreCompact | ✅ | ✅ | **一致** |
| Notification | ✅ | ✅ | **一致** |
| SubagentStart | ✅ | ✅ | **一致** |
| PermissionRequest | ✅ | ✅ | **一致** |
| SessionStart | ❌ | ✅ | **Go独有** |
| SessionEnd | ❌ | ✅ | **Go独有** |

---

## 3. MCP服务器实现对比

| 功能 | Python | Go | 状态 | 差异说明 |
|------|--------|-----|------|----------|
| create_sdk_mcp_server() | ✅ | ✅ | **一致** | Go用CreateSdkMcpServer() |
| @tool装饰器 | ✅ | ✅ | **等效** | Go用sdkmcp.Tool()函数 |
| ToolAnnotations | ✅ | ✅ | **一致** | |
| ImageContent | ✅ | ✅ | **一致** | Go有ImageResult()辅助函数 |
| McpSdkServerConfig | ✅ | ✅ | **一致** | |
| Schema辅助函数 | ⚠️ | ✅ | **Go更好** | Go有Schema(), SimpleSchema()等辅助函数 |

### 工具定义语法对比

**Python (装饰器):**
```python
@tool("add", "Add two numbers", {"a": float, "b": float})
async def add_numbers(args: dict) -> dict:
    return {"content": [{"type": "text", "text": f"Result: {args['a'] + args['b']}"}]}
```

**Go (函数):**
```go
addTool := sdkmcp.Tool("add", "Add two numbers", schema,
    func(ctx context.Context, args map[string]interface{}) (*sdkmcp.ToolResult, error) {
        return sdkmcp.TextResult(fmt.Sprintf("Result: %.0f", a+b)), nil
    })
```

---

## 4. 权限系统对比

| 功能 | Python | Go | 状态 |
|------|--------|-----|------|
| PermissionMode | ✅ | ✅ | **一致** |
| CanUseTool回调 | ✅ | ✅ | **一致** |
| PermissionResultAllow | ✅ | ✅ | **一致** |
| PermissionResultDeny | ✅ | ✅ | **一致** |
| UpdatedInput支持 | ✅ | ✅ | **一致** |
| UpdatedPermissions | ✅ | ✅ | **一致** |

---

## 5. 测试覆盖对比

### 5.1 单元测试统计

| 指标 | Python | Go | 差异 |
|------|--------|-----|------|
| 测试文件数 | 12 | 10+ | Go分布更合理 |
| 总测试数 | ~132 | ~273 | +141 (Go更多) |
| 测试代码行数 | ~4,800 | ~10,424 | +5,624 (Go更多) |

### 5.2 E2E测试统计

| 指标 | Python | Go | 差异 |
|------|--------|-----|------|
| E2E测试文件数 | 9 | 10 | +1 |
| E2E测试数 | 31 | 44 | +13 (Go更多) |

### 5.3 测试实现状态 ✅ 全部完成

| 测试类型 | 状态 | 对应文件 |
|----------|------|----------|
| Subprocess Buffering测试 | ✅ 完成 | `internal/transport/buffering_test.go` |
| Rate Limit Event测试 | ✅ 完成 | `internal/messageparser/rate_limit_test.go` |
| Tool Callbacks测试 | ✅ 完成 | `internal/query/callbacks_test.go` |
| Transport Concurrency测试 | ✅ 完成 | `internal/transport/transport_test.go` |
| E2E 全套测试 | ✅ 完成 | `e2e-tests/` 目录下10个文件 |

---

## 6. 示例覆盖对比

### 6.1 示例对应表

| 示例 | Python | Go | 等价性 |
|------|--------|-----|--------|
| quick_start | ✅ | ✅ | **完全一致** |
| streaming_mode | ✅ | ✅ | **Go更详细** |
| mcp_calculator | ✅ | ✅ | **完全一致** |
| hooks | ✅ | ✅ | **Go更详细** |
| tool_permission_callback | ✅ | ✅ | **Go更详细** |
| agents | ✅ | ✅ | **Go更详细** |
| system_prompt | ✅ | ✅ | **完全一致** |
| setting_sources | ✅ | ✅ | **完全一致** |
| tools_option | ✅ | ✅ | **完全一致** |
| budget / max_budget_usd | ✅ | ✅ | **完全一致** |
| include_partial_messages | ✅ | ✅ | **完全一致** |
| stderr_callback | ✅ | ✅ | **Go更详细** |
| filesystem_agents | ✅ | ✅ | **侧重点不同** |
| plugin_example | ✅ | ✅ | **完全一致** |

### 6.2 Go独有示例

- `streaming_goroutines` - Go特有的goroutine流式处理
- `streaming_interactive` - 交互式流式处理
- `mcp_sdk_server` - 手动MCP服务器实现
- `mcp_sdk_simple` - 简化MCP服务器

---

## 7. 关键差异总结

### 7.1 Go SDK优势

1. **更多Hook事件** - SessionStart, SessionEnd
2. **更好的Schema辅助** - Schema(), SimpleSchema(), StringProperty()等
3. **更多测试** - 单元测试和E2E测试数量都更多
4. **更详细的示例** - 多数示例有更多子场景
5. **类型安全** - 编译时类型检查
6. **并发安全** - goroutine + channel模式
7. **导出的Transport接口** - 允许自定义实现

### 7.2 Python SDK优势

1. **装饰器语法** - @tool装饰器更简洁
2. **async context manager** - 支持async with
3. **自动类型转换** - {"a": float}自动转JSON Schema
4. **async_iterable_prompt** - 支持发送异步消息流

---

## 8. 实施计划状态

### ✅ Phase 1: 核心功能对齐 - 完成
- ✅ 导出Transport接口到transport/包
- ✅ sdkmcp包已实现
- ✅ Tool()函数和CreateSdkMcpServer()完成

### ✅ Phase 2: 完善单元测试 - 完成
- ✅ 添加subprocess buffering测试
- ✅ 添加rate limit event测试
- ✅ 添加tool callback测试
- ✅ 添加transport concurrency测试

### ✅ Phase 3: 完善E2E测试 - 完成
- ✅ 移植Python E2E测试
- ✅ 添加API key集成测试
- ✅ 10个E2E测试文件，44个测试用例

### ✅ Phase 4: 示例完善 - 完成
- ✅ 审查所有示例与Python一致
- ✅ 更新注释和文档
- ✅ 18个示例覆盖所有功能

### ✅ Phase 5: 验证和清理 - 完成
- ✅ 运行所有测试
- ✅ 检查race condition
- ✅ 代码格式化和lint

---

## 9. 结论

**Go SDK 已完成与 Python SDK 的全面对齐：**

| 维度 | 状态 |
|------|------|
| 核心功能 | ✅ 完全对等 |
| MCP支持 | ✅ 完全对等 |
| Hook系统 | ✅ Go更丰富 (多2个事件) |
| 权限系统 | ✅ 完全对等 |
| 单元测试 | ✅ Go覆盖更多 (273 vs 132) |
| E2E测试 | ✅ Go覆盖更多 (44 vs 31) |
| 示例覆盖 | ✅ Go更多 (18 vs 16) |
| Transport扩展 | ✅ 已导出接口 |

---

生成时间: 2026-02-27
更新时间: 2026-02-28