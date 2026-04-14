# Changelog

All notable changes to this project will be documented in this file.

## [v0.1.59] - 2026-04-14

### Changed
- `DeleteSession()` now cascades to subagent transcript directory
  - Deletes the `{session_id}.jsonl` file and the sibling `{session_id}/` subdirectory
  - Matches Python SDK v0.1.59 behavior: `shutil.rmtree(path.parent / session_id, ignore_errors=True)`
- Updated bundled CLI version (2.1.104 → 2.1.107)

### Tests
- Added `TestDeleteSession_CascadeSubagentDir` unit test
- Added `TestDeleteSession_CascadeNoSubagentDir` unit test
- Added `TestDeleteSessionCascadeSubagentDir` e2e test

## [v0.1.58] - 2026-04-13

### Added
- `GetContextUsage()` method on Client for context window usage breakdown by category
- `ContextUsageCategory` type for individual usage categories
- `ContextUsageResponse` type with full context usage data
- `DeleteSession()` function to remove session files
- `ForkSession()` function to fork sessions with UUID remapping
- `ForkSessionResult` type for fork operation results
- `SystemPromptFile` type for file-based system prompts
- `TaskBudget` type for API-side token budget configuration
- `PermissionModeDontAsk` and `PermissionModeAuto` permission mode constants
- Transport support for `--task-budget` CLI argument
- Transport support for `--system-prompt-file` CLI argument
- E2E tests for session mutations (DeleteSession, ForkSession)
- Unit tests for sessions mutations (ForkSession, DeleteSession, Unicode sanitization)
- **Transport Middleware** - New middleware system for intercepting transport operations
  - `TransportMiddleware` interface with `InterceptWrite` and `InterceptRead` methods
  - `MiddlewareTransport` wrapper for applying middleware chain
  - `LoggingMiddleware` for logging write/read operations
  - `MetricsMiddleware` for collecting operation counts
- **Functional Options** - New option package for flexible configuration
  - `option.RequestConfig` for holding configuration
  - `option.RequestOption` functional option type
  - 20+ WithXXX option functions for all configuration fields
- `CLIVersion` constant in root package for bundled CLI version tracking

### Fixed
- Nil pointer dereference in `buildConversationChain` (sessions.go:1196,1209)
- Race condition in `pendingControlResponses` handling (queryimpl/query.go:266-280)
- Missing content-replacement handling in `ForkSession`

### Changed
- **Package Renames** - Internal packages renamed to avoid naming conflicts
  - `internal/query` → `internal/queryimpl`
  - `internal/transport` → `internal/transportimpl`
- Added complete Go doc comments for `client.New()` function

### Documentation
- Updated README.md with Middleware and Functional Options sections
- Updated README-zh.md with corresponding Chinese documentation
- Updated CLAUDE.md with API patterns for middleware and functional options
- Added middleware and functional options examples

### Tests
- Added `transport/transport_test.go` with middleware tests
- Added `option/option_test.go` with functional options tests
- Added `integration_test.go` for cross-package integration tests

## [v0.1.50] - 2026-03-23

### Added
- `GetSessionInfo()` for single-session metadata lookup
- Changed `SDKSessionInfo.FileSize` to optional for remote storage support

### Changed
- Updated SDK version to v0.1.50

## [v0.1.49]

### Added
- `RenameSession()` to append custom title entries
- `TagSession()` to append tag entries with Unicode sanitization

## [v0.1.48]

### Added
- `RateLimitEvent` message type
- `RateLimitInfo` and `RateLimitType` types
- Session mutations: `RenameSession()` and `TagSession()`
- AgentDefinition fields: `Skills`, `Memory`, `McpServers`
- `Usage` field in `AssistantMessage` for per-turn usage tracking

### Changed
- Optimized unit test performance (~10s to ~1.7s cached)
- Reduced mock sleep in timeout tests

## [v0.1.47] - 2026-03-06

### Changed
- Bundled CLI version update (2.1.69 to 2.1.70)
- No API changes

## [v0.1.46] - 2026-03-05

### Added
- Sessions API: `ListSessions()` and `GetSessionMessages()`
- `agent_id`/`agent_type` fields in tool-lifecycle hook inputs
- `parsePermissionRequestHookInput()` function
- `internal/sessions` package with full implementation
- `SDKSessionInfo` and `SessionMessage` types
- 7 unit tests for sessions package

## [v0.1.45]

### Added
- `TaskStartedMessage`, `TaskProgressMessage`, `TaskNotificationMessage` types
- `TaskUsage` and `TaskNotificationStatus` types
