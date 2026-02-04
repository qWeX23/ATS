package llm

import (
	"context"
	"encoding/json"
	"testing"
)

type mockProvider struct {
	responses []*CompletionResponse
	callCount int
}

func (m *mockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if m.callCount >= len(m.responses) {
		return &CompletionResponse{
			Message: Message{
				Role:    RoleAssistant,
				Content: "default response",
			},
		}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func (m *mockProvider) SupportsTools() bool {
	return true
}

func TestClientWithoutTools(t *testing.T) {
	mock := &mockProvider{
		responses: []*CompletionResponse{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "Hello!",
				},
			},
		},
	}

	client := New(mock)
	resp, err := client.Complete(context.Background(), "Hi there")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message.Content != "Hello!" {
		t.Errorf("expected 'Hello!', got %s", resp.Message.Content)
	}
}

func TestClientWithTool(t *testing.T) {
	mock := &mockProvider{
		responses: []*CompletionResponse{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "",
					ToolCalls: []ToolCall{
						{
							ID: "call_1",
							Function: ToolCallFunction{
								Name:      "add",
								Arguments: json.RawMessage(`{"x": 5, "y": 3}`),
							},
						},
					},
				},
				ToolCalls: []ToolCall{
					{
						ID: "call_1",
						Function: ToolCallFunction{
							Name:      "add",
							Arguments: json.RawMessage(`{"x": 5, "y": 3}`),
						},
					},
				},
			},
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "The result is 8",
				},
			},
		},
	}

	client := New(mock)
	client.RegisterTool(FuncTool("add", "Add two numbers", func(ctx context.Context, args struct {
		X int `json:"x"`
		Y int `json:"y"`
	}) int {
		return args.X + args.Y
	}))

	resp, err := client.Complete(context.Background(), "What is 5 + 3?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message.Content != "The result is 8" {
		t.Errorf("expected 'The result is 8', got %s", resp.Message.Content)
	}

	if mock.callCount != 2 {
		t.Errorf("expected 2 calls, got %d", mock.callCount)
	}
}

func TestClientWithMessageToolCallsOnly(t *testing.T) {
	mock := &mockProvider{
		responses: []*CompletionResponse{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "",
					ToolCalls: []ToolCall{
						{
							ID: "call_1",
							Function: ToolCallFunction{
								Name:      "add",
								Arguments: json.RawMessage(`{"x": 2, "y": 4}`),
							},
						},
					},
				},
			},
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "The result is 6",
				},
			},
		},
	}

	client := New(mock)
	client.RegisterTool(FuncTool("add", "Add two numbers", func(ctx context.Context, args struct {
		X int `json:"x"`
		Y int `json:"y"`
	}) int {
		return args.X + args.Y
	}))

	resp, err := client.Complete(context.Background(), "What is 2 + 4?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message.Content != "The result is 6" {
		t.Errorf("expected 'The result is 6', got %s", resp.Message.Content)
	}

	if mock.callCount != 2 {
		t.Errorf("expected 2 calls, got %d", mock.callCount)
	}
}

func TestClientWithSystemPrompt(t *testing.T) {
	mock := &mockProvider{
		responses: []*CompletionResponse{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "I am helpful",
				},
			},
		},
	}

	client := New(mock)
	_, err := client.Complete(context.Background(), "Who are you?", WithSystemPrompt("You are a helpful assistant"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientMaxIterations(t *testing.T) {
	mock := &mockProvider{
		responses: []*CompletionResponse{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "",
					ToolCalls: []ToolCall{
						{
							Function: ToolCallFunction{
								Name:      "add",
								Arguments: json.RawMessage(`{"x": 1, "y": 1}`),
							},
						},
					},
				},
				ToolCalls: []ToolCall{
					{
						Function: ToolCallFunction{
							Name:      "add",
							Arguments: json.RawMessage(`{"x": 1, "y": 1}`),
						},
					},
				},
			},
			// Infinite loop - always returns tool call
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "",
					ToolCalls: []ToolCall{
						{
							Function: ToolCallFunction{
								Name:      "add",
								Arguments: json.RawMessage(`{"x": 2, "y": 2}`),
							},
						},
					},
				},
				ToolCalls: []ToolCall{
					{
						Function: ToolCallFunction{
							Name:      "add",
							Arguments: json.RawMessage(`{"x": 2, "y": 2}`),
						},
					},
				},
			},
		},
	}

	client := New(mock)
	client.RegisterTool(FuncTool("add", "Add", func(ctx context.Context, args struct {
		X int `json:"x"`
		Y int `json:"y"`
	}) int {
		return args.X + args.Y
	}))

	_, err := client.Complete(context.Background(), "Test", WithMaxIterations(2))
	if err == nil {
		t.Error("expected error for max iterations")
	}

	if mock.callCount != 2 {
		t.Errorf("expected 2 calls, got %d", mock.callCount)
	}
}

func TestClientUnknownTool(t *testing.T) {
	mock := &mockProvider{
		responses: []*CompletionResponse{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "",
					ToolCalls: []ToolCall{
						{
							Function: ToolCallFunction{
								Name:      "unknown",
								Arguments: json.RawMessage(`{}`),
							},
						},
					},
				},
				ToolCalls: []ToolCall{
					{
						Function: ToolCallFunction{
							Name:      "unknown",
							Arguments: json.RawMessage(`{}`),
						},
					},
				},
			},
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "Error handled",
				},
			},
		},
	}

	client := New(mock)
	// Register a tool so the client enters the tool loop
	client.RegisterTool(FuncTool("known", "A known tool", func(ctx context.Context, args struct{ X int }) int {
		return args.X
	}))

	resp, err := client.Complete(context.Background(), "Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Message.Content != "Error handled" {
		t.Errorf("expected 'Error handled', got %s", resp.Message.Content)
	}
}
