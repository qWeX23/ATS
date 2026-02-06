package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"ats/internal/llm"
)

type Client struct {
	baseURL string
	model   string
	client  *http.Client
}

func New(baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Client{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{},
	}
}

func (c *Client) SupportsTools() bool {
	return true
}

func (c *Client) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	messages := make([]Message, 0, len(req.Messages)+1)

	if req.SystemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: req.SystemPrompt})
	}

	for _, m := range req.Messages {
		om := Message{
			Role:    string(m.Role),
			Content: m.Content,
		}
		if m.Role == llm.RoleTool {
			om.ToolName = m.Name
		} else {
			om.Name = m.Name
		}
		if len(m.ToolCalls) > 0 {
			om.ToolCalls = make([]ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				om.ToolCalls = append(om.ToolCalls, ToolCall{
					Function: ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		messages = append(messages, om)
	}

	tools := make([]Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, Tool{
			Type: "function",
			Function: Function{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  llm.SchemaToMap(t.Parameters()),
			},
		})
	}

	chatReq := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	if req.Temperature != 0 {
		chatReq.Options = map[string]any{
			"temperature": req.Temperature,
		}
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	toolCalls := make([]llm.ToolCall, 0, len(chatResp.Message.ToolCalls))
	for _, tc := range chatResp.Message.ToolCalls {
		args := tc.Function.Arguments
		// Ollama may return arguments as either a JSON object or a string.
		// Try unmarshaling as string first; if successful, re-unmarshal the string content.
		var argString string
		if err := json.Unmarshal(args, &argString); err == nil {
			args = json.RawMessage(argString)
		}

		toolCalls = append(toolCalls, llm.ToolCall{
			ID: tc.ID,
			Function: llm.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: args,
			},
		})
	}

	return &llm.CompletionResponse{
		Message: llm.Message{
			Role:      llm.Role(chatResp.Message.Role),
			Content:   chatResp.Message.Content,
			ToolCalls: toolCalls,
		},
		ToolCalls:    toolCalls,
		FinishReason: chatResp.DoneReason,
	}, nil
}
