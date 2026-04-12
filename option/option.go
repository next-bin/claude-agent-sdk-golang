// Package option provides functional options for the Claude Agent SDK.
//
// Functional options allow flexible configuration of SDK operations without
// requiring large parameter structs. Options can be composed and applied
// at different levels (client, request, method).
//
// Example:
//
//	client := claude.NewClient(
//		option.WithSystemPrompt("You are a helpful assistant"),
//		option.WithMaxTurns(5),
//		option.WithModel(claude.ModelSonnet),
//	)
package option

import (
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

// RequestOption is a function that modifies request configuration.
type RequestOption func(*RequestConfig) error

// RequestConfig holds configuration for SDK requests.
type RequestConfig struct {
	// Core options
	SystemPrompt         interface{} // string, SystemPromptPreset, or SystemPromptFile
	Model                *string
	FallbackModel        *string
	MaxTurns             *int
	MaxBudgetUSD         *float64
	PermissionMode       *types.PermissionMode
	ContinueConversation bool
	Resume               *string
	SessionID            *string

	// Tool options
	Tools           interface{} // []string or ToolsPreset
	AllowedTools    []string
	DisallowedTools []string

	// MCP options
	MCPServers interface{} // map, string, or Path

	// Hook options
	Hooks map[types.HookEvent][]types.HookMatcher

	// Other options
	CWD     interface{} // string or Path
	CLIPath interface{} // string or Path
	Env     map[string]string
	Betas   []types.SdkBeta

	// Thinking options
	Thinking types.ThinkingConfig
	Effort   *string

	// Output options
	OutputFormat map[string]interface{}

	// Advanced options
	Agents                  map[string]types.AgentDefinition
	SettingSources          []types.SettingSource
	Sandbox                 *types.SandboxSettings
	Plugins                 []types.SdkPluginConfig
	EnableFileCheckpointing bool
	TaskBudget              *types.TaskBudget
}

// Apply applies multiple options to the configuration.
func (c *RequestConfig) Apply(opts ...RequestOption) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return err
		}
	}
	return nil
}

// NewRequestConfig creates a new configuration with default values.
func NewRequestConfig(opts ...RequestOption) (*RequestConfig, error) {
	config := &RequestConfig{}
	if err := config.Apply(opts...); err != nil {
		return nil, err
	}
	return config, nil
}

// ============================================================================
// Core Options
// ============================================================================

// WithSystemPrompt sets the system prompt.
// Accepts string, SystemPromptPreset, or SystemPromptFile.
func WithSystemPrompt(prompt interface{}) RequestOption {
	return func(c *RequestConfig) error {
		c.SystemPrompt = prompt
		return nil
	}
}

// WithModel sets the AI model to use.
func WithModel(model string) RequestOption {
	return func(c *RequestConfig) error {
		c.Model = &model
		return nil
	}
}

// WithFallbackModel sets the fallback model if primary is unavailable.
func WithFallbackModel(model string) RequestOption {
	return func(c *RequestConfig) error {
		c.FallbackModel = &model
		return nil
	}
}

// WithMaxTurns sets the maximum number of conversation turns.
func WithMaxTurns(turns int) RequestOption {
	return func(c *RequestConfig) error {
		c.MaxTurns = &turns
		return nil
	}
}

// WithMaxBudgetUSD sets the maximum budget in USD.
func WithMaxBudgetUSD(budget float64) RequestOption {
	return func(c *RequestConfig) error {
		c.MaxBudgetUSD = &budget
		return nil
	}
}

// WithPermissionMode sets the permission mode.
func WithPermissionMode(mode types.PermissionMode) RequestOption {
	return func(c *RequestConfig) error {
		c.PermissionMode = &mode
		return nil
	}
}

// WithContinueConversation continues an existing conversation.
func WithContinueConversation() RequestOption {
	return func(c *RequestConfig) error {
		c.ContinueConversation = true
		return nil
	}
}

// WithResume resumes a specific session.
func WithResume(sessionID string) RequestOption {
	return func(c *RequestConfig) error {
		c.Resume = &sessionID
		return nil
	}
}

// WithSessionID sets a specific session ID.
func WithSessionID(sessionID string) RequestOption {
	return func(c *RequestConfig) error {
		c.SessionID = &sessionID
		return nil
	}
}

// ============================================================================
// Tool Options
// ============================================================================

// WithTools sets the allowed tools.
// Accepts []string or ToolsPreset.
func WithTools(tools interface{}) RequestOption {
	return func(c *RequestConfig) error {
		c.Tools = tools
		return nil
	}
}

// WithAllowedTools sets specific allowed tools.
func WithAllowedTools(tools []string) RequestOption {
	return func(c *RequestConfig) error {
		c.AllowedTools = tools
		return nil
	}
}

// WithDisallowedTools sets disallowed tools.
func WithDisallowedTools(tools []string) RequestOption {
	return func(c *RequestConfig) error {
		c.DisallowedTools = tools
		return nil
	}
}

// ============================================================================
// MCP Options
// ============================================================================

// WithMCPServers sets MCP server configurations.
func WithMCPServers(servers interface{}) RequestOption {
	return func(c *RequestConfig) error {
		c.MCPServers = servers
		return nil
	}
}

// ============================================================================
// Hook Options
// ============================================================================

// WithHooks sets hook configurations.
func WithHooks(hooks map[types.HookEvent][]types.HookMatcher) RequestOption {
	return func(c *RequestConfig) error {
		c.Hooks = hooks
		return nil
	}
}

// ============================================================================
// Environment Options
// ============================================================================

// WithCWD sets the working directory.
func WithCWD(dir interface{}) RequestOption {
	return func(c *RequestConfig) error {
		c.CWD = dir
		return nil
	}
}

// WithCLIPath sets a custom CLI path.
func WithCLIPath(path interface{}) RequestOption {
	return func(c *RequestConfig) error {
		c.CLIPath = path
		return nil
	}
}

// WithEnv sets environment variables.
func WithEnv(env map[string]string) RequestOption {
	return func(c *RequestConfig) error {
		c.Env = env
		return nil
	}
}

// WithBetas enables beta features.
func WithBetas(betas []types.SdkBeta) RequestOption {
	return func(c *RequestConfig) error {
		c.Betas = betas
		return nil
	}
}

// ============================================================================
// Thinking Options
// ============================================================================

// WithThinking sets the thinking configuration.
func WithThinking(config types.ThinkingConfig) RequestOption {
	return func(c *RequestConfig) error {
		c.Thinking = config
		return nil
	}
}

// WithEffort sets the effort level (low, medium, high, max).
func WithEffort(effort string) RequestOption {
	return func(c *RequestConfig) error {
		c.Effort = &effort
		return nil
	}
}

// ============================================================================
// Output Options
// ============================================================================

// WithOutputFormat sets the structured output format.
func WithOutputFormat(format map[string]interface{}) RequestOption {
	return func(c *RequestConfig) error {
		c.OutputFormat = format
		return nil
	}
}

// ============================================================================
// Advanced Options
// ============================================================================

// WithAgents sets custom agent definitions.
func WithAgents(agents map[string]types.AgentDefinition) RequestOption {
	return func(c *RequestConfig) error {
		c.Agents = agents
		return nil
	}
}

// WithSettingSources sets configuration setting sources.
func WithSettingSources(sources []types.SettingSource) RequestOption {
	return func(c *RequestConfig) error {
		c.SettingSources = sources
		return nil
	}
}

// WithSandbox sets sandbox configuration.
func WithSandbox(sandbox *types.SandboxSettings) RequestOption {
	return func(c *RequestConfig) error {
		c.Sandbox = sandbox
		return nil
	}
}

// WithPlugins sets plugin configurations.
func WithPlugins(plugins []types.SdkPluginConfig) RequestOption {
	return func(c *RequestConfig) error {
		c.Plugins = plugins
		return nil
	}
}

// WithFileCheckpointing enables file checkpointing.
func WithFileCheckpointing() RequestOption {
	return func(c *RequestConfig) error {
		c.EnableFileCheckpointing = true
		return nil
	}
}

// WithTaskBudget sets the API-side token budget.
func WithTaskBudget(budget *types.TaskBudget) RequestOption {
	return func(c *RequestConfig) error {
		c.TaskBudget = budget
		return nil
	}
}
