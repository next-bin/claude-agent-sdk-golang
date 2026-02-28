// Example agents demonstrates how to configure and use custom agents with the Claude Agent SDK for Go.
//
// Agents allow you to define specialized subagents with their own prompts, tools, and models.
// The main agent can delegate tasks to these subagents by invoking them via the Agent tool.
//
// This example shows:
// 1. Using the Agents option in ClaudeAgentOptions
// 2. Defining AgentDefinition with description, prompt, tools, and model
// 3. How subagents are invoked by the main agent
// 4. Different agent configurations for various use cases
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	fmt.Println("=== Claude Agent SDK Go - Agents Example ===")
	fmt.Println()

	// Example 1: Basic agent configuration with a single specialized agent
	fmt.Println("--- Example 1: Basic Agent Configuration ---")
	basicAgentExample(ctx)

	// Example 2: Multiple specialized agents
	fmt.Println("\n--- Example 2: Multiple Specialized Agents ---")
	multipleAgentsExample(ctx)

	// Example 3: Agent with custom model
	fmt.Println("\n--- Example 3: Agent with Custom Model ---")
	customModelAgentExample(ctx)

	// Example 4: Full workflow with agents
	fmt.Println("\n--- Example 4: Full Workflow with Agents ---")
	workflowAgentExample(ctx)
}

// basicAgentExample demonstrates the simplest way to configure an agent.
func basicAgentExample(ctx context.Context) {
	// Define a simple code review agent
	// This agent specializes in reviewing code for issues and improvements
	codeReviewAgent := types.AgentDefinition{
		Description: "Reviews code for best practices and potential issues",
		Prompt:      "You are a code reviewer. Analyze code for bugs, performance issues, security vulnerabilities, and adherence to best practices. Provide constructive feedback.",
		Tools:       []string{"Read", "Grep"}, // Agent can only use file reading tools
		Model:       types.String("sonnet"),
	}

	// Configure ClaudeAgentOptions with the agent
	// The agent name (key in the map) is used to invoke the agent
	options := &types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Agents: map[string]types.AgentDefinition{
			"code-reviewer": codeReviewAgent,
		},
		MaxTurns: types.Int(2),
	}

	// Create client with agent configuration
	client := claude.NewClientWithOptions(options)
	defer client.Close()

	fmt.Println("Configured code-reviewer agent with Read, Grep tools")
	fmt.Println()

	// Uncomment to run an actual query:
	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	msgChan, err := client.Query(ctx, "Use the code-reviewer agent to review the code in main.go")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	for msg := range msgChan {
		handleMessage(msg)
	}
}

// multipleAgentsExample shows how to configure multiple specialized agents.
func multipleAgentsExample(ctx context.Context) {
	// Define multiple agents, each with a specific role

	// 1. Analyzer agent - examines code structure
	analyzerAgent := types.AgentDefinition{
		Description: "Analyzes code structure and patterns",
		Prompt:      "You are a code analyzer. Examine code structure, patterns, and architecture.",
		Tools:       []string{"Read", "Grep", "Glob"},
	}

	// 2. Tester agent - creates and runs tests
	testerAgent := types.AgentDefinition{
		Description: "Creates and runs tests",
		Prompt:      "You are a testing expert. Write comprehensive tests and ensure code quality.",
		Tools:       []string{"Read", "Write", "Bash"},
		Model:       types.String("sonnet"),
	}

	// Configure all agents in ClaudeAgentOptions
	options := &types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Agents: map[string]types.AgentDefinition{
			"analyzer": analyzerAgent,
			"tester":   testerAgent,
		},
		SettingSources: []types.SettingSource{types.SettingSourceUser, types.SettingSourceProject},
		MaxTurns:       types.Int(2),
	}

	client := claude.NewClientWithOptions(options)
	defer client.Close()

	fmt.Println("Configured 2 specialized agents:")
	fmt.Println("  - analyzer: for code structure analysis")
	fmt.Println("  - tester: for test creation and verification")
	fmt.Println()

	// Uncomment to run an actual query:
	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	msgChan, err := client.Query(ctx, "Use the analyzer agent to find all Go files in the current directory")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	for msg := range msgChan {
		handleMessage(msg)
	}
}

// customModelAgentExample demonstrates agents with different model configurations.
func customModelAgentExample(ctx context.Context) {
	// Agent models can be: "sonnet", "opus", "haiku", or "inherit"
	// "inherit" means the agent uses the same model as the parent

	// High-capability agent using Opus for complex reasoning
	architectAgent := types.AgentDefinition{
		Description: "System architect for complex design decisions and architecture planning",
		Prompt:      "You are a senior system architect. Provide detailed architectural guidance, design patterns, and scalability recommendations.",
		Tools:       []string{"Read", "Bash"},
		Model:       types.String("opus"), // Use Opus for complex reasoning
	}

	// Fast agent using Haiku for quick tasks
	formatterAgent := types.AgentDefinition{
		Description: "Code formatter and linter for quick code quality checks",
		Prompt:      "You are a code formatter. Quickly format and lint code according to standard style guidelines. Be fast and efficient.",
		Tools:       []string{"Read", "Edit"},
		Model:       types.String("haiku"), // Use Haiku for fast, simple tasks
	}

	// Agent inheriting parent model
	documentationAgent := types.AgentDefinition{
		Description: "Documentation writer that creates and updates documentation",
		Prompt:      "You are a technical writer. Create clear, comprehensive documentation. Include examples and explain complex concepts simply.",
		Tools:       []string{"Read", "Write", "Glob"},
		Model:       types.String("inherit"), // Inherits from parent agent
	}

	options := &types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Agents: map[string]types.AgentDefinition{
			"architect":     architectAgent,
			"formatter":     formatterAgent,
			"documentation": documentationAgent,
		},
	}

	client := claude.NewClientWithOptions(options)
	defer client.Close()

	fmt.Println("Configured agents with different models:")
	fmt.Println("  - architect: uses Opus for complex reasoning")
	fmt.Println("  - formatter: uses Haiku for fast tasks")
	fmt.Println("  - documentation: inherits parent model (Sonnet)")
	_ = ctx
}

// workflowAgentExample shows a complete workflow using agents.
func workflowAgentExample(ctx context.Context) {
	// Define a practical set of agents for a development workflow

	// Security reviewer agent - focused on security issues
	securityAgent := types.AgentDefinition{
		Description: "Security specialist that identifies vulnerabilities and suggests fixes",
		Prompt: `You are a security expert. Review code for:
- SQL injection vulnerabilities
- XSS vulnerabilities
- Authentication and authorization issues
- Sensitive data exposure
- Security misconfigurations

Provide specific remediation steps for each finding.`,
		Tools: []string{"Read", "Grep", "Glob"},
		Model: types.String("sonnet"),
	}

	// Performance analyzer agent
	performanceAgent := types.AgentDefinition{
		Description: "Performance analyst that identifies bottlenecks and optimization opportunities",
		Prompt: `You are a performance optimization specialist. Analyze code for:
- Algorithm complexity issues
- Memory inefficiencies
- Database query optimization
- Caching opportunities
- Concurrency improvements

Provide specific optimization recommendations with expected impact.`,
		Tools: []string{"Read", "Bash", "Grep"},
		Model: types.String("sonnet"),
	}

	options := &types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Agents: map[string]types.AgentDefinition{
			"security-reviewer":   securityAgent,
			"performance-analyst": performanceAgent,
		},
		// Optional: Set permission mode for automation
		// Use with caution - bypassPermissions disables all permission prompts
		// PermissionMode: (*types.PermissionMode)(types.String("bypassPermissions")),
	}

	client := claude.NewClientWithOptions(options)
	defer client.Close()

	fmt.Println("Configured workflow agents:")
	fmt.Println("  - security-reviewer: identifies security vulnerabilities")
	fmt.Println("  - performance-analyst: finds optimization opportunities")
	fmt.Println()

	// Example of how to run a query that might use agents
	// The main agent can invoke subagents using the "Agent" tool
	// For example, a query like "Review this code for security issues"
	// would trigger the security-reviewer agent

	// Uncomment the following to actually run a query:
	/*
		fmt.Println("Running query that may invoke agents...")
		msgChan, err := client.Query(ctx, "Review the code in main.go for any security issues")
		if err != nil {
			log.Printf("Query failed: %v", err)
			return
		}

		for msg := range msgChan {
			handleMessage(msg)
		}
	*/
}

// handleMessage processes different message types from the agent.
func handleMessage(msg types.Message) {
	switch m := msg.(type) {
	case *types.AssistantMessage:
		// Assistant messages contain the agent's response
		for _, block := range m.Content {
			switch b := block.(type) {
			case types.TextBlock:
				fmt.Printf("Assistant: %s\n", b.Text)
			case types.ToolUseBlock:
				// Check if this is an Agent tool invocation
				if b.Name == "Agent" {
					fmt.Printf("Invoking subagent: %+v\n", b.Input)
				} else {
					fmt.Printf("Tool Use: %s\n", b.Name)
				}
			}
		}
	case *types.ResultMessage:
		// Final result message
		if m.Result != nil {
			fmt.Printf("Result: %s\n", *m.Result)
		}
		fmt.Printf("Session ID: %s, Duration: %dms\n", m.SessionID, m.DurationMs)
	case *types.StreamEvent:
		// Stream events for partial messages during streaming
		fmt.Printf("Stream Event: %+v\n", m.Event)
	}
}

// Example: Running a query that invokes agents
// This function demonstrates how the main agent delegates to subagents.
func runAgentQueryExample() {
	ctx := context.Background()

	// Define agents for a code review workflow
	options := &types.ClaudeAgentOptions{
		Model: types.String(types.ModelSonnet),
		Agents: map[string]types.AgentDefinition{
			"code-reviewer": {
				Description: "Reviews code for quality and issues",
				Prompt:      "Review code thoroughly. Report issues with severity levels.",
				Tools:       []string{"Read", "Grep"},
				Model:       types.String("sonnet"),
			},
		},
	}

	client := claude.NewClientWithOptions(options)
	defer client.Close()

	// Connect to Claude
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Query that triggers agent usage
	// The main agent may invoke the code-reviewer agent using the Agent tool
	query := `Please review the authentication module. Use the code-reviewer agent to analyze:
1. Security vulnerabilities
2. Code quality issues
3. Potential improvements`

	msgChan, err := client.Query(ctx, query)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Process the response stream
	for msg := range msgChan {
		handleMessage(msg)
	}
}
