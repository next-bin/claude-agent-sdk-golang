// Package main demonstrates transport middleware usage.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/next-bin/claude-agent-sdk-golang/transport"
)

func main() {
	fmt.Println("Middleware Example")
	fmt.Println("==================")
	fmt.Println()
	fmt.Println("Middleware can be used for:")
	fmt.Println("  - Logging requests and responses")
	fmt.Println("  - Collecting metrics")
	fmt.Println("  - Filtering messages")
	fmt.Println("  - Transforming data")
	fmt.Println()

	// Create logging middleware
	loggingMiddleware := transport.NewLoggingMiddleware(
		func(ctx context.Context, data string) {
			log.Printf("[WRITE] %s", truncate(data, 100))
		},
		func(ctx context.Context, msg map[string]interface{}) {
			msgType, _ := msg["type"].(string)
			log.Printf("[READ] type=%s", msgType)
		},
	)

	// Create metrics middleware
	metricsMiddleware := transport.NewMetricsMiddleware()

	// Simulate middleware usage
	simulateMiddlewareUsage(loggingMiddleware, metricsMiddleware)
}

func simulateMiddlewareUsage(loggingMiddleware, metricsMiddleware transport.TransportMiddleware) {
	ctx := context.Background()

	// Simulate write operation
	data := "{\"type\":\"user_message\",\"content\":\"Hello\"}"
	for _, m := range []transport.TransportMiddleware{loggingMiddleware, metricsMiddleware} {
		modified, err := m.InterceptWrite(ctx, data)
		if err != nil {
			log.Printf("Write error: %v", err)
			return
		}
		data = modified
	}

	// Simulate read operation
	msg := map[string]interface{}{
		"type":    "assistant",
		"content": "Hello! How can I help?",
	}
	for _, m := range []transport.TransportMiddleware{metricsMiddleware, loggingMiddleware} {
		result, err := m.InterceptRead(ctx, msg)
		if err != nil || result == nil {
			return
		}
		msg = result
	}

	// Cast metricsMiddleware to access GetWriteCount/GetReadCount
	if mm, ok := metricsMiddleware.(*transport.MetricsMiddleware); ok {
		fmt.Println("\nMetrics:")
		fmt.Printf("  Write count: %d\n", mm.GetWriteCount())
		fmt.Printf("  Read count: %d\n", mm.GetReadCount())
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
