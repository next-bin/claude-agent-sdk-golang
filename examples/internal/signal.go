// Package internal provides shared utilities for SDK examples.
package internal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalContext creates a context that is cancelled when SIGINT or SIGTERM is received.
// This allows examples to gracefully shutdown on Ctrl+C.
//
// Usage:
//
//	ctx, cancel := internal.SetupSignalContext()
//	defer cancel()
func SetupSignalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
		cancel()
	}()

	return ctx, cancel
}

// SetupSignalContextWithCancel creates a context that can be cancelled manually
// or when SIGINT/SIGTERM is received.
func SetupSignalContextWithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}