// Example tool_permission demonstrates tool permission handling in the Claude Agent SDK for Go.
//
// The CanUseTool callback allows you to control which tools Claude can use and
// modify permissions dynamically. This is useful for:
// - Implementing custom security policies
// - Auto-approving certain tools
// - Modifying tool inputs for safety (e.g., redirecting file writes)
// - Providing user feedback when tools are denied
// - Updating permission rules based on context
//
// This example shows:
// 1. Basic permission handling with allow/deny decisions
// 2. PermissionResultAllow with updated_input (redirect file paths)
// 3. PermissionResultAllow with updated permissions
// 4. PermissionResultDeny with custom messages
// 5. Dangerous command detection
// 6. PermissionUpdate usage for dynamic rule changes
package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/unitsvc/claude-agent-sdk-golang/client"
	_ "github.com/unitsvc/claude-agent-sdk-golang/examples/internal"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

func main() {
	fmt.Println("=== Claude Agent SDK Go - Tool Permission Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates tool permission handling patterns:")
	fmt.Println("1. Basic permission allow/deny decisions")
	fmt.Println("2. PermissionResultAllow with updated_input (redirect file paths)")
	fmt.Println("3. PermissionResultAllow with updated permissions")
	fmt.Println("4. PermissionResultDeny with custom messages and interrupt")
	fmt.Println("5. Dangerous command detection")
	fmt.Println("6. Using PermissionUpdate to modify permission rules dynamically")
	fmt.Println()

	// Note: The SDK requires the Claude CLI to be installed.
	// This example shows the API structure. Actual usage requires:
	// 1. Install Claude CLI: npm install -g @anthropic-ai/claude-code
	// 2. Authenticate: claude login
	// 3. Run this program

	// Example 1: Basic permission handler with allow/deny decisions
	fmt.Println("--- Example 1: Basic Permission Handler ---")
	basicClient := createClientWithBasicPermissionHandler()
	defer basicClient.Close()
	fmt.Println("Client created with basic permission handler (allowing Read, denying Bash)")
	fmt.Println()

	// Example 2: Permission handler with input modification (redirect file paths)
	fmt.Println("--- Example 2: Permission Handler with Input Modification ---")
	inputModClient := createClientWithInputModification()
	defer inputModClient.Close()
	fmt.Println("Client created with input modification (redirects writes to safe directory)")
	fmt.Println()

	// Example 3: Permission handler with updated permissions
	fmt.Println("--- Example 3: Permission Handler with Updated Permissions ---")
	updateClient := createClientWithPermissionUpdates()
	defer updateClient.Close()
	fmt.Println("Client created with permission updates (adds rules on each allowed tool)")
	fmt.Println()

	// Example 4: Permission handler with deny messages
	fmt.Println("--- Example 4: Permission Handler with Deny Messages ---")
	denyClient := createClientWithDenyMessages()
	defer denyClient.Close()
	fmt.Println("Client created with detailed deny messages")
	fmt.Println()

	// Example 5: Comprehensive permission handler
	fmt.Println("--- Example 5: Comprehensive Permission Handler ---")
	comprehensiveClient := createClientWithComprehensiveHandler()
	defer comprehensiveClient.Close()
	fmt.Println("Client created with comprehensive permission logic")
	fmt.Println()

	// To actually run queries, uncomment the following:
	// ctx, cancel := internal.SetupSignalContext()
	// defer cancel()
	// runExampleQuery(ctx, comprehensiveClient)
}

// createClientWithBasicPermissionHandler creates a client with simple allow/deny logic.
// This demonstrates the most basic form of permission handling.
func createClientWithBasicPermissionHandler() *client.Client {
	return client.NewWithOptions(&types.ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
			fmt.Printf("[Permission Request] Tool: %s\n", toolName)

			// Define allowed tools
			allowedTools := map[string]bool{
				"Read":  true,
				"Write": true,
				"Glob":  true,
				"Grep":  true,
			}

			// Define denied tools
			deniedTools := map[string]bool{
				"Bash":  true, // Block shell commands
				"Edit":  true, // Block file edits
				"Skill": true, // Block skill execution
			}

			if allowedTools[toolName] {
				fmt.Printf("[ALLOWED] Tool %s is permitted\n", toolName)
				// Return a simple allow result
				return types.PermissionResultAllow{
					Behavior: "allow",
				}, nil
			}

			if deniedTools[toolName] {
				fmt.Printf("[DENIED] Tool %s is blocked\n", toolName)
				// Return a deny result with a message
				return types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   fmt.Sprintf("Tool %s is not allowed by security policy", toolName),
					Interrupt: false,
				}, nil
			}

			// Default: deny unknown tools
			fmt.Printf("[DENIED] Tool %s is not in allowed list\n", toolName)
			return types.PermissionResultDeny{
				Behavior:  "deny",
				Message:   fmt.Sprintf("Tool %s is not recognized or permitted", toolName),
				Interrupt: false,
			}, nil
		},
	})
}

// createClientWithInputModification creates a client that demonstrates
// PermissionResultAllow with UpdatedInput to redirect file writes to a safe directory.
// This matches Python's "redirect writes to a safe directory" feature.
func createClientWithInputModification() *client.Client {
	return client.NewWithOptions(&types.ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
			fmt.Printf("[Permission Request] Tool: %s\n", toolName)

			// Always allow read operations
			if toolName == "Read" || toolName == "Glob" || toolName == "Grep" {
				fmt.Printf("[ALLOWED] Read-only tool %s is permitted\n", toolName)
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			}

			// Deny write operations to system directories
			if toolName == "Write" || toolName == "Edit" {
				filePath, _ := input["file_path"].(string)
				if filePath != "" {
					// Block writes to system directories
					if strings.HasPrefix(filePath, "/etc/") || strings.HasPrefix(filePath, "/usr/") {
						fmt.Printf("[DENIED] Cannot write to system directory: %s\n", filePath)
						return types.PermissionResultDeny{
							Behavior:  "deny",
							Message:   fmt.Sprintf("Cannot write to system directory: %s", filePath),
							Interrupt: false,
						}, nil
					}

					// Redirect writes to a safe directory (matching Python example)
					if !strings.HasPrefix(filePath, "/tmp/") && !strings.HasPrefix(filePath, "./") {
						// Extract filename and redirect to ./safe_output/
						safePath := "./safe_output/" + filepath.Base(filePath)
						fmt.Printf("[REDIRECT] Redirecting write from %s to %s\n", filePath, safePath)

						// Create modified input with the new path
						modifiedInput := make(map[string]interface{})
						for k, v := range input {
							modifiedInput[k] = v
						}
						modifiedInput["file_path"] = safePath

						return types.PermissionResultAllow{
							Behavior:     "allow",
							UpdatedInput: modifiedInput,
						}, nil
					}
				}

				fmt.Printf("[ALLOWED] Write tool permitted for path: %s\n", filePath)
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			}

			// Check dangerous bash commands (matching Python example)
			if toolName == "Bash" {
				command, _ := input["command"].(string)
				dangerousCommands := []string{"rm -rf", "sudo", "chmod 777", "dd if=", "mkfs"}

				for _, dangerous := range dangerousCommands {
					if strings.Contains(command, dangerous) {
						fmt.Printf("[DENIED] Dangerous command pattern detected: %s\n", dangerous)
						return types.PermissionResultDeny{
							Behavior:  "deny",
							Message:   fmt.Sprintf("Dangerous command pattern detected: %s", dangerous),
							Interrupt: false,
						}, nil
					}
				}

				fmt.Printf("[ALLOWED] Bash command appears safe: %s\n", command)
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			}

			// Default: deny
			return types.PermissionResultDeny{
				Behavior:  "deny",
				Message:   fmt.Sprintf("Tool %s requires explicit approval", toolName),
				Interrupt: false,
			}, nil
		},
	})
}

// createClientWithPermissionUpdates creates a client that returns PermissionResultAllow
// with updated permissions. This demonstrates how to dynamically add permission rules.
func createClientWithPermissionUpdates() *client.Client {
	return client.NewWithOptions(&types.ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
			fmt.Printf("[Permission Request] Tool: %s\n", toolName)

			// Check the suggestions from the CLI context
			if len(ctx.Suggestions) > 0 {
				fmt.Printf("[Suggestions] Received %d permission suggestions\n", len(ctx.Suggestions))
				for i, suggestion := range ctx.Suggestions {
					fmt.Printf("  Suggestion %d: Type=%s\n", i+1, suggestion.Type)
				}
			}

			// Allow specific tools with permission updates
			if toolName == "Read" || toolName == "Glob" || toolName == "Grep" {
				fmt.Printf("[ALLOWED] Tool %s with permission updates\n", toolName)

				// Create permission updates to add rules
				// This allows future calls to these tools to be automatically approved
				updatedPermissions := []types.PermissionUpdate{
					{
						Type: types.PermissionUpdateTypeAddRules,
						Rules: []types.PermissionRuleValue{
							{
								ToolName: toolName,
							},
						},
						Behavior:    permissionBehaviorPtr(types.PermissionBehaviorAllow),
						Destination: permissionUpdateDestinationPtr(types.PermissionUpdateDestinationSession),
					},
				}

				return types.PermissionResultAllow{
					Behavior:           "allow",
					UpdatedPermissions: updatedPermissions,
				}, nil
			}

			// For Bash tool, allow but add specific directory restrictions
			if toolName == "Bash" {
				fmt.Printf("[ALLOWED] Tool Bash with directory restrictions\n")

				// Add allowed directories for Bash commands
				updatedPermissions := []types.PermissionUpdate{
					{
						Type:        types.PermissionUpdateTypeAddDirectories,
						Directories: []string{"/home/user/safe-dir", "/tmp/workspace"},
						Destination: permissionUpdateDestinationPtr(types.PermissionUpdateDestinationSession),
					},
				}

				return types.PermissionResultAllow{
					Behavior:           "allow",
					UpdatedPermissions: updatedPermissions,
				}, nil
			}

			// Deny other tools
			return types.PermissionResultDeny{
				Behavior:  "deny",
				Message:   fmt.Sprintf("Tool %s requires explicit approval", toolName),
				Interrupt: false,
			}, nil
		},
	})
}

// createClientWithDenyMessages creates a client that demonstrates
// PermissionResultDeny with detailed messages and interrupt behavior.
func createClientWithDenyMessages() *client.Client {
	return client.NewWithOptions(&types.ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
			fmt.Printf("[Permission Request] Tool: %s, Input: %v\n", toolName, input)

			// Critical tools that should interrupt the conversation
			criticalDeniedTools := map[string]string{
				"Skill":        "Skill execution is disabled for security reasons. This action cannot be undone.",
				"NotebookEdit": "Notebook editing requires elevated permissions. Contact administrator.",
			}

			// Standard tools that are denied but don't interrupt
			standardDeniedTools := map[string]string{
				"Edit":  "File editing is restricted. Use Read-only operations instead.",
				"Write": "File writing is restricted. Submit changes for review.",
			}

			// Check critical denied tools - interrupt the conversation
			if message, denied := criticalDeniedTools[toolName]; denied {
				fmt.Printf("[CRITICAL DENY] Tool %s is blocked with interrupt\n", toolName)
				return types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   message,
					Interrupt: true, // Interrupt the entire conversation
				}, nil
			}

			// Check standard denied tools - don't interrupt
			if message, denied := standardDeniedTools[toolName]; denied {
				fmt.Printf("[DENIED] Tool %s is blocked\n", toolName)
				return types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   message,
					Interrupt: false, // Don't interrupt, let Claude try alternatives
				}, nil
			}

			// Allow read-only tools with detailed logging
			readOnlyTools := []string{"Read", "Glob", "Grep"}
			for _, allowed := range readOnlyTools {
				if toolName == allowed {
					fmt.Printf("[ALLOWED] Read-only tool %s is permitted\n", toolName)
					return types.PermissionResultAllow{
						Behavior: "allow",
					}, nil
				}
			}

			// Default deny with helpful message
			return types.PermissionResultDeny{
				Behavior:  "deny",
				Message:   fmt.Sprintf("Tool %s is not in the permitted list. Allowed tools: Read, Glob, Grep", toolName),
				Interrupt: false,
			}, nil
		},
	})
}

// createClientWithComprehensiveHandler creates a client with a comprehensive
// permission handler that demonstrates all features together.
func createClientWithComprehensiveHandler() *client.Client {
	// Track approved tools for session-level caching
	approvedTools := make(map[string]bool)

	return client.NewWithOptions(&types.ClaudeAgentOptions{
		CanUseTool: func(toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (types.PermissionResult, error) {
			fmt.Printf("\n[Permission Check] ===================\n")
			fmt.Printf("Tool: %s\n", toolName)

			// Log input details for debugging (be careful with sensitive data)
			if input != nil {
				// Only log safe fields
				if path, ok := input["file_path"]; ok {
					fmt.Printf("File path: %v\n", path)
				}
				if cmd, ok := input["command"]; ok {
					fmt.Printf("Command: %v\n", cmd)
				}
			}

			// Check if already approved in this session
			if approvedTools[toolName] {
				fmt.Printf("[CACHED] Tool %s was previously approved\n", toolName)
				return types.PermissionResultAllow{
					Behavior: "allow",
				}, nil
			}

			// Process CLI suggestions if available
			if len(ctx.Suggestions) > 0 {
				fmt.Printf("[Suggestions] %d suggestions from CLI:\n", len(ctx.Suggestions))
				for _, suggestion := range ctx.Suggestions {
					fmt.Printf("  - Type: %s", suggestion.Type)
					if suggestion.Destination != nil {
						fmt.Printf(", Destination: %s", *suggestion.Destination)
					}
					fmt.Println()
				}
			}

			// Define tool categories
			alwaysAllowed := map[string]bool{
				"Read": true,
				"Glob": true,
				"Grep": true,
			}

			requiresApproval := map[string]bool{
				"Bash":  true,
				"Edit":  true,
				"Write": true,
			}

			alwaysDenied := map[string]bool{
				"Skill":        true,
				"NotebookEdit": true,
			}

			// Handle always allowed tools
			if alwaysAllowed[toolName] {
				fmt.Printf("[DECISION] ALLOW - Tool %s is in always-allowed list\n", toolName)
				approvedTools[toolName] = true

				// Add session-level permission rule
				return types.PermissionResultAllow{
					Behavior: "allow",
					UpdatedPermissions: []types.PermissionUpdate{
						{
							Type: types.PermissionUpdateTypeAddRules,
							Rules: []types.PermissionRuleValue{
								{ToolName: toolName},
							},
							Behavior:    permissionBehaviorPtr(types.PermissionBehaviorAllow),
							Destination: permissionUpdateDestinationPtr(types.PermissionUpdateDestinationSession),
						},
					},
				}, nil
			}

			// Handle always denied tools
			if alwaysDenied[toolName] {
				fmt.Printf("[DECISION] DENY - Tool %s is in always-denied list\n", toolName)
				return types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   fmt.Sprintf("Tool %s is permanently disabled by security policy", toolName),
					Interrupt: true,
				}, nil
			}

			// Handle tools requiring approval
			if requiresApproval[toolName] {
				// For Bash, check the command
				if toolName == "Bash" {
					if cmd, ok := input["command"].(string); ok {
						// Allow safe commands
						safeCommands := []string{"ls", "pwd", "echo", "cat", "head", "tail"}
						for _, safeCmd := range safeCommands {
							if len(cmd) >= len(safeCmd) && cmd[:len(safeCmd)] == safeCmd {
								fmt.Printf("[DECISION] ALLOW - Bash command appears safe: %s\n", safeCmd)
								approvedTools[toolName] = true
								return types.PermissionResultAllow{
									Behavior: "allow",
								}, nil
							}
						}

						// Deny dangerous commands
						dangerousPatterns := []string{"rm -rf", "sudo", "chmod 777", "> /dev/sd"}
						for _, pattern := range dangerousPatterns {
							if contains(cmd, pattern) {
								fmt.Printf("[DECISION] DENY - Dangerous command pattern detected: %s\n", pattern)
								return types.PermissionResultDeny{
									Behavior:  "deny",
									Message:   fmt.Sprintf("Command contains dangerous pattern: %s", pattern),
									Interrupt: false,
								}, nil
							}
						}
					}

					// Default: ask for approval on other bash commands
					fmt.Printf("[DECISION] DENY - Bash command requires approval\n")
					return types.PermissionResultDeny{
						Behavior:  "deny",
						Message:   "Bash command requires manual approval. Please review and approve.",
						Interrupt: false,
					}, nil
				}

				// For Edit/Write, require explicit approval
				fmt.Printf("[DECISION] DENY - Tool %s requires explicit approval\n", toolName)
				return types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   fmt.Sprintf("Tool %s requires explicit user approval", toolName),
					Interrupt: false,
				}, nil
			}

			// Default: deny unknown tools
			fmt.Printf("[DECISION] DENY - Unknown tool: %s\n", toolName)
			return types.PermissionResultDeny{
				Behavior:  "deny",
				Message:   fmt.Sprintf("Tool %s is not recognized. Known tools: Read, Glob, Grep, Bash, Edit, Write", toolName),
				Interrupt: false,
			}, nil
		},
	})
}

// runExampleQuery demonstrates how to run a query with permission handling.
func runExampleQuery(ctx context.Context, c *client.Client) {
	// Connect to Claude
	if err := c.Connect(ctx); err != nil {
		log.Printf("Failed to connect: %v", err)
		return
	}

	// Send a query that will trigger tool permission checks
	msgChan, err := c.Query(ctx, "List the files in the current directory using Bash")
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}

	// Process messages
	for msg := range msgChan {
		fmt.Printf("Message: %v\n", msg)
	}
}

// Helper functions

// permissionBehaviorPtr returns a pointer to a PermissionBehavior.
func permissionBehaviorPtr(b types.PermissionBehavior) *types.PermissionBehavior {
	return &b
}

// permissionUpdateDestinationPtr returns a pointer to a PermissionUpdateDestination.
func permissionUpdateDestinationPtr(d types.PermissionUpdateDestination) *types.PermissionUpdateDestination {
	return &d
}

// contains checks if a string contains a substring (case-sensitive).
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
