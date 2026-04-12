// Example filesystem_agents demonstrates filesystem-focused agent configurations in the Claude Agent SDK for Go.
//
// This example shows:
// 1. Creating a read-only filesystem agent
// 2. Creating a code analysis agent
// 3. Creating a documentation agent
// 4. Working with file system tools
//
// Prerequisites:
// - Claude CLI installed: npm install -g @anthropic-ai/claude-code
// - Authenticated: claude login
package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/next-bin/claude-agent-sdk-golang"
	"github.com/next-bin/claude-agent-sdk-golang/examples/internal"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func main() {
	ctx, cancel := internal.SetupSignalContext()
	defer cancel()

	fmt.Println("=== Claude Agent SDK Go - Filesystem Agents Example ===")
	fmt.Println()

	// Example 1: Read-only filesystem agent
	readOnlyExample(ctx)

	// Example 2: Code analysis agent
	codeAnalysisExample(ctx)

	// Example 3: Documentation generator agent
	documentationExample(ctx)

	// Example 4: Full filesystem agent with workspace
	workspaceExample(ctx)
}

// readOnlyExample demonstrates a read-only agent that can only read files.
func readOnlyExample(ctx context.Context) {
	fmt.Println("--- Example 1: Read-Only Filesystem Agent ---")
	fmt.Println("This agent can only read and search files, not modify them.")
	fmt.Println()

	// Create a read-only agent by only allowing Read, Glob, and Grep tools
	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		AllowedTools: []string{"Read", "Glob", "Grep"},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Read-only agent configured with tools: Read, Glob, Grep")
	fmt.Println("(Connect to use this agent)")
	fmt.Println()
}

// codeAnalysisExample demonstrates an agent for code analysis.
func codeAnalysisExample(ctx context.Context) {
	fmt.Println("--- Example 2: Code Analysis Agent ---")
	fmt.Println("This agent analyzes code for quality, patterns, and potential issues.")
	fmt.Println()

	// Create a code analysis agent with read and search tools
	codeReviewAgent := types.AgentDefinition{
		Description: "A code analysis specialist that reviews code for quality, patterns, and potential issues.",
		Prompt: `You are a code analysis expert. When reviewing code:
1. Identify potential bugs or issues
2. Check for code style and best practices
3. Look for security vulnerabilities
4. Suggest improvements
Be thorough but concise in your analysis.`,
		Tools: []string{"Read", "Glob", "Grep"},
		Model: types.String("sonnet"),
	}

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		AllowedTools: []string{"Read", "Glob", "Grep", "Task"},
		Agents: map[string]types.AgentDefinition{
			"code-analyzer": codeReviewAgent,
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Code analysis agent configured with:")
	fmt.Println("  - Primary tools: Read, Glob, Grep, Task")
	fmt.Println("  - Subagent: code-analyzer (specialized for code review)")
	fmt.Println("(Connect to use this agent)")
	fmt.Println()
}

// documentationExample demonstrates an agent for documentation generation.
func documentationExample(ctx context.Context) {
	fmt.Println("--- Example 3: Documentation Generator Agent ---")
	fmt.Println("This agent creates and updates documentation files.")
	fmt.Println()

	// Create a documentation agent
	docsAgent := types.AgentDefinition{
		Description: "A documentation specialist that creates clear, comprehensive documentation.",
		Prompt: `You are a documentation expert. When creating documentation:
1. Write clear, concise explanations
2. Include code examples where appropriate
3. Use proper markdown formatting
4. Create README files, API docs, and guides
Focus on making complex concepts easy to understand.`,
		Tools: []string{"Read", "Write", "Glob", "Grep"},
		Model: types.String("sonnet"),
	}

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		AllowedTools: []string{"Read", "Write", "Glob", "Grep", "Task"},
		Agents: map[string]types.AgentDefinition{
			"docs-writer": docsAgent,
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Documentation agent configured with:")
	fmt.Println("  - Primary tools: Read, Write, Glob, Grep, Task")
	fmt.Println("  - Subagent: docs-writer (specialized for documentation)")
	fmt.Println("(Connect to use this agent)")
	fmt.Println()
}

// workspaceExample demonstrates a full filesystem agent with workspace configuration.
func workspaceExample(ctx context.Context) {
	fmt.Println("--- Example 4: Full Workspace Agent ---")
	fmt.Println("This agent has full filesystem access within a specified directory.")
	fmt.Println()

	// Create a workspace agent with full file access
	workspaceAgent := types.AgentDefinition{
		Description: "A workspace manager that can read, edit, and organize files.",
		Prompt: `You are a workspace management expert. You can:
1. Read and analyze existing files
2. Create new files and directories
3. Edit and refactor existing code
4. Organize project structure
Work carefully and always back up before major changes.`,
		Tools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
		Model: types.String("sonnet"),
	}

	// Configure with sandbox for safer operations
	sandboxEnabled := true
	autoAllowBash := true

	client := claude.NewClientWithOptions(&types.ClaudeAgentOptions{
		AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "Task"},
		Agents: map[string]types.AgentDefinition{
			"workspace-manager": workspaceAgent,
		},
		Sandbox: &types.SandboxSettings{
			Enabled:                  &sandboxEnabled,
			AutoAllowBashIfSandboxed: &autoAllowBash,
		},
	})
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	fmt.Println("Workspace agent configured with:")
	fmt.Println("  - Primary tools: Read, Write, Edit, Bash, Glob, Grep, Task")
	fmt.Println("  - Subagent: workspace-manager (full filesystem access)")
	fmt.Println("  - Sandbox: enabled for safer operations")
	fmt.Println("(Connect to use this agent)")
	fmt.Println()
}
