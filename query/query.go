// Package query provides a one-shot query function for Claude.
//
// The query package offers a simple interface for sending a single query
// to Claude and receiving a response, without managing a long-running session.
package query

import (
	"context"

	"github.com/unitsvc/claude-agent-sdk-golang/client"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// Query sends a one-shot query to Claude and returns messages through a channel.
//
// This is the simplest way to interact with Claude. Pass a prompt and
// optional configuration, and receive messages through a channel.
//
// Example:
//
//	msgChan, err := query.Query(ctx, "What is 2+2?", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for msg := range msgChan {
//	    fmt.Printf("%v\n", msg)
//	}
func Query(ctx context.Context, prompt string, opts *types.ClaudeAgentOptions) (<-chan types.Message, error) {
	c := client.NewWithOptions(opts)
	if err := c.Connect(ctx, prompt); err != nil {
		return nil, err
	}

	// Return a channel that will receive messages
	output := make(chan types.Message)
	go func() {
		defer close(output)
		defer c.Disconnect(ctx)

		msgChan := c.ReceiveMessages(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				output <- msg
			}
		}
	}()

	return output, nil
}

// QueryWithClient sends a query using an existing client.
//
// Use this when you have a pre-configured client with custom options.
//
// Example:
//
//	c := client.NewWithOptions(&types.ClaudeAgentOptions{
//	    Model: types.String("claude-sonnet-4-20250514"),
//	})
//	msgChan, err := query.QueryWithClient(ctx, c, "Hello!")
func QueryWithClient(ctx context.Context, c *client.Client, prompt string) (<-chan types.Message, error) {
	if err := c.Connect(ctx, prompt); err != nil {
		return nil, err
	}

	// Return a channel that will receive messages
	output := make(chan types.Message)
	go func() {
		defer close(output)
		defer c.Disconnect(ctx)

		msgChan := c.ReceiveMessages(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				output <- msg
			}
		}
	}()

	return output, nil
}
