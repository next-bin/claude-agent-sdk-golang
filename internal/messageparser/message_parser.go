// Package messageparser provides functionality to parse raw JSON messages
// from the Claude CLI into typed Message objects.
package messageparser

import (
	"log/slog"

	"github.com/unitsvc/claude-agent-sdk-golang/errors"
	"github.com/unitsvc/claude-agent-sdk-golang/types"
)

// ParseMessage parses a raw message dictionary from CLI output into a typed Message object.
// Returns nil for unrecognized message types (forward-compatible with newer CLI versions).
func ParseMessage(data map[string]interface{}) (types.Message, error) {
	if data == nil {
		return nil, errors.NewMessageParseError(
			"Invalid message data type (expected map, got nil)",
			nil,
		)
	}

	messageType, ok := data["type"].(string)
	if !ok || messageType == "" {
		return nil, errors.NewMessageParseError(
			"Message missing 'type' field",
			data,
		)
	}

	switch messageType {
	case "user":
		return parseUserMessage(data)
	case "assistant":
		return parseAssistantMessage(data)
	case "system":
		return parseSystemMessage(data)
	case "result":
		return parseResultMessage(data)
	case "stream_event":
		return parseStreamEvent(data)
	case "rate_limit_event":
		return parseRateLimitEvent(data)
	default:
		// Forward-compatible: skip unrecognized message types so newer
		// CLI versions don't crash older SDK versions.
		slog.Debug("Skipping unknown message type", "type", messageType)
		return nil, nil
	}
}

// parseTextBlock parses a text content block.
func parseTextBlock(data map[string]interface{}) (types.TextBlock, error) {
	text, ok := data["text"].(string)
	if !ok {
		return types.TextBlock{}, errors.NewMessageParseError(
			"TextBlock missing 'text' field",
			data,
		)
	}
	return types.TextBlock{
		Type: "text",
		Text: text,
	}, nil
}

// parseThinkingBlock parses a thinking content block.
func parseThinkingBlock(data map[string]interface{}) (types.ThinkingBlock, error) {
	thinking, ok := data["thinking"].(string)
	if !ok {
		return types.ThinkingBlock{}, errors.NewMessageParseError(
			"ThinkingBlock missing 'thinking' field",
			data,
		)
	}
	signature, _ := data["signature"].(string) // signature is optional
	return types.ThinkingBlock{
		Type:      "thinking",
		Thinking:  thinking,
		Signature: signature,
	}, nil
}

// parseToolUseBlock parses a tool use content block.
func parseToolUseBlock(data map[string]interface{}) (types.ToolUseBlock, error) {
	id, ok := data["id"].(string)
	if !ok {
		return types.ToolUseBlock{}, errors.NewMessageParseError(
			"ToolUseBlock missing 'id' field",
			data,
		)
	}
	name, ok := data["name"].(string)
	if !ok {
		return types.ToolUseBlock{}, errors.NewMessageParseError(
			"ToolUseBlock missing 'name' field",
			data,
		)
	}
	input, _ := data["input"].(map[string]interface{}) // input can be nil
	return types.ToolUseBlock{
		Type:  "tool_use",
		ID:    id,
		Name:  name,
		Input: input,
	}, nil
}

// parseToolResultBlock parses a tool result content block.
func parseToolResultBlock(data map[string]interface{}) (types.ToolResultBlock, error) {
	toolUseID, ok := data["tool_use_id"].(string)
	if !ok {
		return types.ToolResultBlock{}, errors.NewMessageParseError(
			"ToolResultBlock missing 'tool_use_id' field",
			data,
		)
	}

	content := data["content"] // can be string, []ContentBlock, or nil

	var isError *bool
	if isErr, ok := data["is_error"]; ok {
		if boolVal, ok := isErr.(bool); ok {
			isError = &boolVal
		}
	}

	return types.ToolResultBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}, nil
}

// parseContentBlocks parses an array of content blocks.
func parseContentBlocks(blocks []interface{}) ([]types.ContentBlock, error) {
	result := make([]types.ContentBlock, 0, len(blocks))

	for _, block := range blocks {
		blockMap, ok := block.(map[string]interface{})
		if !ok {
			continue
		}

		blockType, ok := blockMap["type"].(string)
		if !ok {
			continue
		}

		var contentBlock types.ContentBlock
		var err error

		switch blockType {
		case "text":
			var tb types.TextBlock
			tb, err = parseTextBlock(blockMap)
			contentBlock = tb
		case "thinking":
			var tb types.ThinkingBlock
			tb, err = parseThinkingBlock(blockMap)
			contentBlock = tb
		case "tool_use":
			var tb types.ToolUseBlock
			tb, err = parseToolUseBlock(blockMap)
			contentBlock = tb
		case "tool_result":
			var tb types.ToolResultBlock
			tb, err = parseToolResultBlock(blockMap)
			contentBlock = tb
		default:
			// Unknown block type - create a generic content block
			contentBlock = types.GenericContentBlock{Data: blockMap}
		}

		if err != nil {
			return nil, err
		}

		if contentBlock != nil {
			result = append(result, contentBlock)
		}
	}

	return result, nil
}

// parseUserMessage parses a user message.
func parseUserMessage(data map[string]interface{}) (*types.UserMessage, error) {
	message, ok := data["message"].(map[string]interface{})
	if !ok {
		return nil, errors.NewMessageParseError(
			"UserMessage missing 'message' field",
			data,
		)
	}

	content, ok := message["content"]
	if !ok {
		return nil, errors.NewMessageParseError(
			"UserMessage missing 'content' field",
			data,
		)
	}

	var uuid *string
	if u, ok := data["uuid"].(string); ok {
		uuid = &u
	}

	var parentToolUseID *string
	if p, ok := data["parent_tool_use_id"].(string); ok {
		parentToolUseID = &p
	}

	var toolUseResult map[string]interface{}
	if t, ok := data["tool_use_result"].(map[string]interface{}); ok {
		toolUseResult = t
	}

	// Handle content - can be string or []ContentBlock
	switch c := content.(type) {
	case string:
		return &types.UserMessage{
			Content:         c,
			UUID:            uuid,
			ParentToolUseID: parentToolUseID,
			ToolUseResult:   toolUseResult,
		}, nil
	case []interface{}:
		contentBlocks, err := parseContentBlocks(c)
		if err != nil {
			return nil, err
		}
		return &types.UserMessage{
			Content:         contentBlocks,
			UUID:            uuid,
			ParentToolUseID: parentToolUseID,
			ToolUseResult:   toolUseResult,
		}, nil
	default:
		return nil, errors.NewMessageParseError(
			"UserMessage content has invalid type",
			data,
		)
	}
}

// parseAssistantMessage parses an assistant message.
func parseAssistantMessage(data map[string]interface{}) (*types.AssistantMessage, error) {
	message, ok := data["message"].(map[string]interface{})
	if !ok {
		return nil, errors.NewMessageParseError(
			"AssistantMessage missing 'message' field",
			data,
		)
	}

	contentRaw, ok := message["content"]
	if !ok {
		return nil, errors.NewMessageParseError(
			"AssistantMessage missing 'content' field",
			data,
		)
	}

	contentArray, ok := contentRaw.([]interface{})
	if !ok {
		return nil, errors.NewMessageParseError(
			"AssistantMessage content is not an array",
			data,
		)
	}

	contentBlocks, err := parseContentBlocks(contentArray)
	if err != nil {
		return nil, err
	}

	model, ok := message["model"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"AssistantMessage missing 'model' field",
			data,
		)
	}

	var parentToolUseID *string
	if p, ok := data["parent_tool_use_id"].(string); ok {
		parentToolUseID = &p
	}

	var assistantErr *types.AssistantMessageError
	if e, ok := data["error"].(string); ok {
		err := types.AssistantMessageError(e)
		assistantErr = &err
	}

	// Parse usage field (v0.1.48+)
	var usage map[string]interface{}
	if u, ok := data["usage"].(map[string]interface{}); ok {
		usage = u
	}

	return &types.AssistantMessage{
		Content:         contentBlocks,
		Model:           model,
		ParentToolUseID: parentToolUseID,
		Error:           assistantErr,
		Usage:           usage,
	}, nil
}

// parseSystemMessage parses a system message.
func parseSystemMessage(data map[string]interface{}) (types.Message, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"SystemMessage missing 'subtype' field",
			data,
		)
	}

	// Check for specific task message subtypes
	switch subtype {
	case "task_started":
		return parseTaskStartedMessage(data)
	case "task_progress":
		return parseTaskProgressMessage(data)
	case "task_notification":
		return parseTaskNotificationMessage(data)
	default:
		// Generic system message
		return &types.SystemMessage{
			Subtype: subtype,
			Data:    data,
		}, nil
	}
}

// parseTaskStartedMessage parses a task_started system message.
func parseTaskStartedMessage(data map[string]interface{}) (*types.TaskStartedMessage, error) {
	taskID, ok := data["task_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskStartedMessage missing 'task_id' field",
			data,
		)
	}

	description, ok := data["description"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskStartedMessage missing 'description' field",
			data,
		)
	}

	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskStartedMessage missing 'uuid' field",
			data,
		)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskStartedMessage missing 'session_id' field",
			data,
		)
	}

	var toolUseID *string
	if t, ok := data["tool_use_id"].(string); ok {
		toolUseID = &t
	}

	var taskType *string
	if t, ok := data["task_type"].(string); ok {
		taskType = &t
	}

	return &types.TaskStartedMessage{
		SystemMessage: types.SystemMessage{
			Subtype: "task_started",
			Data:    data,
		},
		TaskID:      taskID,
		Description: description,
		UUID:        uuid,
		SessionID:   sessionID,
		ToolUseID:   toolUseID,
		TaskType:    taskType,
	}, nil
}

// parseTaskProgressMessage parses a task_progress system message.
func parseTaskProgressMessage(data map[string]interface{}) (*types.TaskProgressMessage, error) {
	taskID, ok := data["task_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskProgressMessage missing 'task_id' field",
			data,
		)
	}

	description, ok := data["description"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskProgressMessage missing 'description' field",
			data,
		)
	}

	usageData, ok := data["usage"].(map[string]interface{})
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskProgressMessage missing 'usage' field",
			data,
		)
	}

	usage, err := parseTaskUsage(usageData)
	if err != nil {
		return nil, errors.NewMessageParseError(
			"TaskProgressMessage has invalid 'usage' field",
			data,
		)
	}

	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskProgressMessage missing 'uuid' field",
			data,
		)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskProgressMessage missing 'session_id' field",
			data,
		)
	}

	var toolUseID *string
	if t, ok := data["tool_use_id"].(string); ok {
		toolUseID = &t
	}

	var lastToolName *string
	if t, ok := data["last_tool_name"].(string); ok {
		lastToolName = &t
	}

	return &types.TaskProgressMessage{
		SystemMessage: types.SystemMessage{
			Subtype: "task_progress",
			Data:    data,
		},
		TaskID:       taskID,
		Description:  description,
		Usage:        usage,
		UUID:         uuid,
		SessionID:    sessionID,
		ToolUseID:    toolUseID,
		LastToolName: lastToolName,
	}, nil
}

// parseTaskNotificationMessage parses a task_notification system message.
func parseTaskNotificationMessage(data map[string]interface{}) (*types.TaskNotificationMessage, error) {
	taskID, ok := data["task_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskNotificationMessage missing 'task_id' field",
			data,
		)
	}

	status, ok := data["status"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskNotificationMessage missing 'status' field",
			data,
		)
	}

	outputFile, ok := data["output_file"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskNotificationMessage missing 'output_file' field",
			data,
		)
	}

	summary, ok := data["summary"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskNotificationMessage missing 'summary' field",
			data,
		)
	}

	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskNotificationMessage missing 'uuid' field",
			data,
		)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"TaskNotificationMessage missing 'session_id' field",
			data,
		)
	}

	var toolUseID *string
	if t, ok := data["tool_use_id"].(string); ok {
		toolUseID = &t
	}

	var usage *types.TaskUsage
	if usageData, ok := data["usage"].(map[string]interface{}); ok {
		if parsedUsage, err := parseTaskUsage(usageData); err == nil {
			usage = &parsedUsage
		}
	}

	return &types.TaskNotificationMessage{
		SystemMessage: types.SystemMessage{
			Subtype: "task_notification",
			Data:    data,
		},
		TaskID:     taskID,
		Status:     types.TaskNotificationStatus(status),
		OutputFile: outputFile,
		Summary:    summary,
		UUID:       uuid,
		SessionID:  sessionID,
		ToolUseID:  toolUseID,
		Usage:      usage,
	}, nil
}

// parseTaskUsage parses task usage statistics.
func parseTaskUsage(data map[string]interface{}) (types.TaskUsage, error) {
	totalTokens, _ := data["total_tokens"].(float64) // JSON numbers are float64
	toolUses, _ := data["tool_uses"].(float64)
	durationMs, _ := data["duration_ms"].(float64)

	return types.TaskUsage{
		TotalTokens: int(totalTokens),
		ToolUses:    int(toolUses),
		DurationMs:  int(durationMs),
	}, nil
}

// parseResultMessage parses a result message.
func parseResultMessage(data map[string]interface{}) (*types.ResultMessage, error) {
	subtype, ok := data["subtype"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"ResultMessage missing 'subtype' field",
			data,
		)
	}

	durationMs, ok := data["duration_ms"].(float64) // JSON numbers are float64
	if !ok {
		return nil, errors.NewMessageParseError(
			"ResultMessage missing 'duration_ms' field",
			data,
		)
	}

	durationAPIMs, ok := data["duration_api_ms"].(float64)
	if !ok {
		return nil, errors.NewMessageParseError(
			"ResultMessage missing 'duration_api_ms' field",
			data,
		)
	}

	isError, ok := data["is_error"].(bool)
	if !ok {
		return nil, errors.NewMessageParseError(
			"ResultMessage missing 'is_error' field",
			data,
		)
	}

	numTurns, ok := data["num_turns"].(float64)
	if !ok {
		return nil, errors.NewMessageParseError(
			"ResultMessage missing 'num_turns' field",
			data,
		)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"ResultMessage missing 'session_id' field",
			data,
		)
	}

	var totalCostUSD *float64
	if t, ok := data["total_cost_usd"].(float64); ok {
		totalCostUSD = &t
	}

	var usage map[string]interface{}
	if u, ok := data["usage"].(map[string]interface{}); ok {
		usage = u
	}

	var result *string
	if r, ok := data["result"].(string); ok {
		result = &r
	}

	structuredOutput := data["structured_output"]

	return &types.ResultMessage{
		Subtype:          subtype,
		DurationMs:       int(durationMs),
		DurationAPIMs:    int(durationAPIMs),
		IsError:          isError,
		NumTurns:         int(numTurns),
		SessionID:        sessionID,
		TotalCostUSD:     totalCostUSD,
		Usage:            usage,
		Result:           result,
		StructuredOutput: structuredOutput,
	}, nil
}

// parseStreamEvent parses a stream event.
func parseStreamEvent(data map[string]interface{}) (*types.StreamEvent, error) {
	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"StreamEvent missing 'uuid' field",
			data,
		)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"StreamEvent missing 'session_id' field",
			data,
		)
	}

	event, ok := data["event"].(map[string]interface{})
	if !ok {
		return nil, errors.NewMessageParseError(
			"StreamEvent missing 'event' field",
			data,
		)
	}

	var parentToolUseID *string
	if p, ok := data["parent_tool_use_id"].(string); ok {
		parentToolUseID = &p
	}

	return &types.StreamEvent{
		UUID:            uuid,
		SessionID:       sessionID,
		Event:           event,
		ParentToolUseID: parentToolUseID,
	}, nil
}

// parseRateLimitEvent parses a rate_limit_event message.
func parseRateLimitEvent(data map[string]interface{}) (*types.RateLimitEvent, error) {
	uuid, ok := data["uuid"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"RateLimitEvent missing 'uuid' field",
			data,
		)
	}

	sessionID, ok := data["session_id"].(string)
	if !ok {
		return nil, errors.NewMessageParseError(
			"RateLimitEvent missing 'session_id' field",
			data,
		)
	}

	infoData, ok := data["rate_limit_info"].(map[string]interface{})
	if !ok {
		return nil, errors.NewMessageParseError(
			"RateLimitEvent missing 'rate_limit_info' field",
			data,
		)
	}

	info, err := parseRateLimitInfo(infoData)
	if err != nil {
		return nil, err
	}

	return &types.RateLimitEvent{
		RateLimitInfo: info,
		UUID:          uuid,
		SessionID:     sessionID,
	}, nil
}

// parseRateLimitInfo parses rate limit info from the CLI response.
func parseRateLimitInfo(data map[string]interface{}) (types.RateLimitInfo, error) {
	status, ok := data["status"].(string)
	if !ok {
		return types.RateLimitInfo{}, errors.NewMessageParseError(
			"RateLimitInfo missing 'status' field",
			data,
		)
	}

	info := types.RateLimitInfo{
		Status: types.RateLimitStatus(status),
		Raw:    data,
	}

	// Parse optional fields
	if resetsAt, ok := data["resetsAt"].(float64); ok {
		v := int(resetsAt)
		info.ResetsAt = &v
	}

	if rateLimitType, ok := data["rateLimitType"].(string); ok {
		t := types.RateLimitType(rateLimitType)
		info.RateLimitType = &t
	}

	if utilization, ok := data["utilization"].(float64); ok {
		info.Utilization = &utilization
	}

	if overageStatus, ok := data["overageStatus"].(string); ok {
		s := types.RateLimitStatus(overageStatus)
		info.OverageStatus = &s
	}

	if overageResetsAt, ok := data["overageResetsAt"].(float64); ok {
		v := int(overageResetsAt)
		info.OverageResetsAt = &v
	}

	if overageDisabledReason, ok := data["overageDisabledReason"].(string); ok {
		info.OverageDisabledReason = &overageDisabledReason
	}

	return info, nil
}
