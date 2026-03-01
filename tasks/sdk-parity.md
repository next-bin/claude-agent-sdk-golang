# SDK Parity Tracking

## Version Reference
- Python SDK: 0.1.44-a58d3ab
- Go SDK: Target parity with above
- Last Updated: 2026-03-01

## Feature Status

### Core Types

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| PermissionMode | ✅ | ✅ | Done | Modes: default, acceptEdits, plan, bypassPermissions |
| SettingSource | ✅ | ✅ | Done | Sources: user, project, local |
| SdkBeta | ✅ | ✅ | Done | Beta headers for features |
| Model constants | ✅ | ✅ | Done | opus, sonnet, haiku |

### Message Types

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| UserMessage | ✅ | ✅ | Done | User message type |
| AssistantMessage | ✅ | ✅ | Done | Assistant message type |
| SystemMessage | ✅ | ✅ | Done | System message type |
| ResultMessage | ✅ | ✅ | Done | Final result message |

### Content Blocks

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| TextBlock | ✅ | ✅ | Done | Text content |
| ThinkingBlock | ✅ | ✅ | Done | Thinking content |
| ToolUseBlock | ✅ | ✅ | Done | Tool use request |
| ToolResultBlock | ✅ | ✅ | Done | Tool execution result |

### Client & Query

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| query() | ✅ | ✅ | Done | One-shot query |
| Client | ✅ | ✅ | Done | Interactive client |
| ClaudeAgentOptions | ✅ | ✅ | Done | Configuration options |
| Streaming | ✅ | ✅ | Done | Message streaming via channels |

### Tool Permission System

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| PermissionResult | ✅ | ✅ | Done | Allow/Deny results |
| PermissionUpdate | ✅ | ✅ | Done | Permission updates |
| ToolPermissionContext | ✅ | ✅ | Done | Permission context |
| CanUseTool callback | ✅ | ✅ | Done | Tool permission callback |

### Hooks System

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| HookCallback | ✅ | ✅ | Done | Hook function type |
| HookContext | ✅ | ✅ | Done | Hook execution context |
| HookMatcher | ✅ | ✅ | Done | Hook matching rules |
| PreToolUseHookInput | ✅ | ✅ | Done | Pre-tool hook |
| PostToolUseHookInput | ✅ | ✅ | Done | Post-tool hook |
| PostToolUseFailureHookInput | ✅ | ✅ | Done | Tool failure hook |
| UserPromptSubmitHookInput | ✅ | ✅ | Done | Prompt submit hook |
| StopHookInput | ✅ | ✅ | Done | Stop hook |
| SubagentStartHookInput | ✅ | ✅ | Done | Subagent start hook |
| SubagentStopHookInput | ✅ | ✅ | Done | Subagent stop hook |
| PreCompactHookInput | ✅ | ✅ | Done | Pre-compact hook |
| NotificationHookInput | ✅ | ✅ | Done | Notification hook |
| PermissionRequestHookInput | ✅ | ✅ | Done | Permission request hook |

### MCP Server Support

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| McpServerConfig | ✅ | ✅ | Done | Base MCP config |
| McpStdioServerConfig | ✅ | ✅ | Done | Stdio MCP server |
| McpSSEServerConfig | ✅ | ✅ | Done | SSE MCP server |
| McpHttpServerConfig | ✅ | ✅ | Done | HTTP MCP server |
| McpSdkServerConfig | ✅ | ✅ | Done | In-process SDK server |
| create_sdk_mcp_server() | ✅ | ✅ | Done | SDK MCP server factory |
| @tool decorator | ✅ | ✅ | Done | Tool definition helper |

### Agent Support

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| AgentDefinition | ✅ | ✅ | Done | Agent definition |

### Sandbox Support

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| SandboxSettings | ✅ | ✅ | Done | Sandbox configuration |
| SandboxNetworkConfig | ✅ | ✅ | Done | Network sandbox settings |
| SandboxIgnoreViolations | ✅ | ✅ | Done | Violation ignore settings |

### Thinking Config

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| ThinkingConfig | ✅ | ✅ | Done | Thinking configuration |
| ThinkingConfigEnabled | ✅ | ✅ | Done | Enabled thinking |
| ThinkingConfigDisabled | ✅ | ✅ | Done | Disabled thinking |
| ThinkingConfigAdaptive | ✅ | ✅ | Done | Adaptive thinking |

### Plugin Support

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| SdkPluginConfig | ✅ | ✅ | Done | Plugin configuration |

### Errors

| Feature | Python | Go | Status | Notes |
|---------|--------|-----|--------|-------|
| ClaudeSDKError | ✅ | ✅ | Done | Base SDK error |
| CLIConnectionError | ✅ | ✅ | Done | Connection error |
| CLINotFoundError | ✅ | ✅ | Done | CLI not found error |
| ProcessError | ✅ | ✅ | Done | Process error |
| CLIJSONDecodeError | ✅ | ✅ | Done | JSON decode error |

## Priority Order for Remaining Work

1. ✅ Core types (Message, ContentBlock, etc.) - DONE
2. ✅ Client/Query functions - DONE
3. ✅ Tool handling - DONE
4. ✅ Hooks system - DONE
5. ✅ MCP server support - DONE
6. ✅ Sandbox settings - DONE
7. 🔄 Ongoing: Maintain parity with Python SDK updates