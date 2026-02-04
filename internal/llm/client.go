package llm

import (
	"context"
	"encoding/json"
	"fmt"
)

const defaultMaxIterations = 10

type Client struct {
	provider Provider
	tools    map[string]Tool
}

func New(provider Provider) *Client {
	return &Client{
		provider: provider,
		tools:    make(map[string]Tool),
	}
}

func (c *Client) RegisterTool(tool Tool) {
	c.tools[tool.Name()] = tool
}

func (c *Client) Complete(ctx context.Context, prompt string, opts ...CompletionOption) (*CompletionResponse, error) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: prompt},
		},
		MaxIterations: defaultMaxIterations,
	}

	for _, opt := range opts {
		opt(&req)
	}

	tools := make([]Tool, 0, len(c.tools))
	for _, t := range c.tools {
		tools = append(tools, t)
	}
	req.Tools = tools

	if len(req.Tools) == 0 || !c.provider.SupportsTools() {
		return c.provider.Complete(ctx, req)
	}

	return c.completeWithToolLoop(ctx, req)
}

type CompletionOption func(*CompletionRequest)

func WithSystemPrompt(prompt string) CompletionOption {
	return func(req *CompletionRequest) {
		req.SystemPrompt = prompt
	}
}

func WithTemperature(temp float64) CompletionOption {
	return func(req *CompletionRequest) {
		req.Temperature = temp
	}
}

func WithMaxIterations(max int) CompletionOption {
	return func(req *CompletionRequest) {
		req.MaxIterations = max
	}
}

func (c *Client) completeWithToolLoop(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	messages := make([]Message, len(req.Messages))
	copy(messages, req.Messages)

	if req.SystemPrompt != "" {
		systemMsg := Message{Role: RoleSystem, Content: req.SystemPrompt}
		messages = append([]Message{systemMsg}, messages...)
	}

	for i := 0; i < req.MaxIterations; i++ {
		resp, err := c.provider.Complete(ctx, CompletionRequest{
			Messages:    messages,
			Tools:       req.Tools,
			Temperature: req.Temperature,
		})
		if err != nil {
			return nil, err
		}

		toolCalls := resp.ToolCalls
		if len(toolCalls) == 0 {
			toolCalls = resp.Message.ToolCalls
		}
		if len(resp.Message.ToolCalls) == 0 && len(resp.ToolCalls) > 0 {
			resp.Message.ToolCalls = resp.ToolCalls
		}

		if len(toolCalls) == 0 {
			return resp, nil
		}

		messages = append(messages, resp.Message)

		for _, tc := range toolCalls {
			tool, ok := c.tools[tc.Function.Name]
			if !ok {
				messages = append(messages, Message{
					Role:       RoleTool,
					Content:    fmt.Sprintf(`{"error": "tool not found: %s"}`, tc.Function.Name),
					ToolCallID: tc.ID,
				})
				continue
			}

			result, err := tool.Execute(ctx, tc.Function.Arguments)
			var content string
			if err != nil {
				content = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			} else {
				resultJSON, err := json.Marshal(result)
				if err != nil {
					content = fmt.Sprintf(`{"error": "failed to marshal result: %s"}`, err.Error())
				} else {
					content = string(resultJSON)
				}
			}

			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    content,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	return nil, fmt.Errorf("max iterations (%d) reached", req.MaxIterations)
}
