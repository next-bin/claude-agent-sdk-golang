// Package main demonstrates functional options usage.
package main

import (
	"fmt"
	"log"

	"github.com/next-bin/claude-agent-sdk-golang/option"
	"github.com/next-bin/claude-agent-sdk-golang/types"
)

func main() {
	fmt.Println("Functional Options Example")
	fmt.Println("==========================")
	fmt.Println()

	// Example 1: Basic options
	fmt.Println("Example 1: Basic configuration")
	config1, err := option.NewRequestConfig(
		option.WithSystemPrompt("You are a helpful assistant"),
		option.WithModel(types.ModelSonnet),
		option.WithMaxTurns(3),
	)
	if err != nil {
		log.Fatal(err)
	}
	printConfig(config1, "Basic")

	// Example 2: Tools configuration
	fmt.Println("\nExample 2: Tools configuration")
	config2, err := option.NewRequestConfig(
		option.WithTools([]string{"Read", "Write", "Bash"}),
		option.WithAllowedTools([]string{"Read", "Write"}),
		option.WithPermissionMode(types.PermissionModeAcceptEdits),
	)
	if err != nil {
		log.Fatal(err)
	}
	printConfig(config2, "Tools")

	// Example 3: Hooks configuration
	fmt.Println("\nExample 3: Hooks configuration")
	config3, err := option.NewRequestConfig(
		option.WithMaxTurns(5),
		option.WithHooks(map[types.HookEvent][]types.HookMatcher{
			types.HookEventPreToolUse: []types.HookMatcher{
				{Matcher: "Bash"},
			},
			types.HookEventPostToolUse: []types.HookMatcher{
				{Matcher: "*"},
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	printConfig(config3, "Hooks")

	// Example 4: Composing options
	fmt.Println("\nExample 4: Composing options")
	baseOptions := []option.RequestOption{
		option.WithSystemPrompt("Base system prompt"),
		option.WithMaxTurns(5),
	}

	extraOptions := []option.RequestOption{
		option.WithModel(types.ModelOpus),
		option.WithPermissionMode(types.PermissionModeDefault),
	}

	// Combine options
	allOptions := append(baseOptions, extraOptions...)
	config4, err := option.NewRequestConfig(allOptions...)
	if err != nil {
		log.Fatal(err)
	}
	printConfig(config4, "Composed")

	// Example 5: Using with client (demonstration)
	fmt.Println("\nExample 5: Client creation pattern")
	fmt.Println("  Functional options can be used with client.NewWithOptions:")
	fmt.Println("  client := client.NewWithOptions(optionsFromConfig(config))")
}

func printConfig(config *option.RequestConfig, name string) {
	fmt.Printf("  %s config:\n", name)

	if config.SystemPrompt != nil {
		fmt.Printf("    SystemPrompt: %v\n", config.SystemPrompt)
	}

	if config.Model != nil {
		fmt.Printf("    Model: %s\n", *config.Model)
	}

	if config.MaxTurns != nil {
		fmt.Printf("    MaxTurns: %d\n", *config.MaxTurns)
	}

	if config.PermissionMode != nil {
		fmt.Printf("    PermissionMode: %s\n", *config.PermissionMode)
	}

	if config.Tools != nil {
		fmt.Printf("    Tools: %v\n", config.Tools)
	}

	if len(config.Hooks) > 0 {
		fmt.Printf("    Hooks: %d events configured\n", len(config.Hooks))
	}
}
