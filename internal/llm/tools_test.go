package llm

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

type AddArgs struct {
	X int `json:"x" desc:"First number"`
	Y int `json:"y" desc:"Second number"`
}

func TestFuncTool(t *testing.T) {
	tool := FuncTool("add", "Add two numbers", func(ctx context.Context, args AddArgs) int {
		return args.X + args.Y
	})

	if tool.Name() != "add" {
		t.Errorf("expected name 'add', got %s", tool.Name())
	}

	if tool.Description() != "Add two numbers" {
		t.Errorf("expected description 'Add two numbers', got %s", tool.Description())
	}

	schema := tool.Parameters()
	if schema.Type != SchemaTypeObject {
		t.Errorf("expected type object, got %s", schema.Type)
	}

	if _, ok := schema.Properties["x"]; !ok {
		t.Error("expected property 'x' to exist")
	}

	if _, ok := schema.Properties["y"]; !ok {
		t.Error("expected property 'y' to exist")
	}
}

func TestFuncToolExecute(t *testing.T) {
	tool := FuncTool("add", "Add two numbers", func(ctx context.Context, args AddArgs) int {
		return args.X + args.Y
	})

	args := json.RawMessage(`{"x": 5, "y": 3}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if result != 8 {
		t.Errorf("expected 8, got %v", result)
	}
}

func TestStructToSchema(t *testing.T) {
	type TestStruct struct {
		Name     string `json:"name" desc:"The name"`
		Count    int    `json:"count"`
		Enabled  bool   `json:"enabled,omitempty"`
		Optional string `json:",omitempty"`
	}

	schema := structToSchema(reflect.TypeOf(TestStruct{}))

	if schema.Type != SchemaTypeObject {
		t.Errorf("expected object type, got %s", schema.Type)
	}

	if len(schema.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(schema.Required))
	}

	nameSchema, ok := schema.Properties["name"]
	if !ok {
		t.Fatal("expected 'name' property")
	}
	if nameSchema.Type != SchemaTypeString {
		t.Errorf("expected string type for name, got %s", nameSchema.Type)
	}
	if nameSchema.Description != "The name" {
		t.Errorf("expected description 'The name', got %s", nameSchema.Description)
	}

	if _, ok := schema.Properties["Optional"]; !ok {
		t.Error("expected property 'Optional' to exist")
	}
}

func TestToolWithError(t *testing.T) {
	tool := FuncTool("fail", "Always fails", func(ctx context.Context, args AddArgs) (int, error) {
		return 0, errors.New("something went wrong")
	})

	args := json.RawMessage(`{"x": 1, "y": 2}`)
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
