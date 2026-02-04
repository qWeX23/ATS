package llm

import (
	"context"
	"encoding/json"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type SchemaType string

const (
	SchemaTypeObject  SchemaType = "object"
	SchemaTypeString  SchemaType = "string"
	SchemaTypeInteger SchemaType = "integer"
	SchemaTypeNumber  SchemaType = "number"
	SchemaTypeBoolean SchemaType = "boolean"
	SchemaTypeArray   SchemaType = "array"
)

type Schema struct {
	Type        SchemaType         `json:"type"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
}

type Tool interface {
	Name() string
	Description() string
	Parameters() *Schema
	Execute(ctx context.Context, args json.RawMessage) (any, error)
}

type CompletionRequest struct {
	Messages      []Message
	Tools         []Tool
	SystemPrompt  string
	Temperature   float64
	MaxIterations int
}

type CompletionResponse struct {
	Message      Message
	ToolCalls    []ToolCall
	FinishReason string
}
