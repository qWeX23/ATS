package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type funcTool struct {
	name        string
	description string
	fn          reflect.Value
	inputType   reflect.Type
	hasContext  bool
}

func (t *funcTool) Name() string {
	return t.name
}

func (t *funcTool) Description() string {
	return t.description
}

func (t *funcTool) Parameters() *Schema {
	return typeToSchema(t.inputType)
}

func (t *funcTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	inputValue := reflect.New(t.inputType)
	if err := json.Unmarshal(args, inputValue.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal arguments: %w", err)
	}

	callArgs := make([]reflect.Value, 0, 2)
	if t.hasContext {
		callArgs = append(callArgs, reflect.ValueOf(ctx))
	}
	callArgs = append(callArgs, inputValue.Elem())

	results := t.fn.Call(callArgs)

	if len(results) == 0 {
		return nil, nil
	}

	if len(results) == 1 {
		if err, ok := results[0].Interface().(error); ok {
			return nil, err
		}
		return results[0].Interface(), nil
	}

	result := results[0].Interface()
	if errVal := results[1].Interface(); errVal != nil {
		return result, errVal.(error)
	}
	return result, nil
}

func FuncTool(name, description string, fn any) Tool {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		panic("FuncTool: fn must be a function")
	}

	if fnType.NumIn() != 1 && fnType.NumIn() != 2 {
		panic("FuncTool: fn must take 1-2 arguments: (input) or (context.Context, input)")
	}

	if fnType.NumOut() > 2 {
		panic("FuncTool: fn must return 0-2 values: () or (error) or (result) or (result, error)")
	}

	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	hasContext := false
	inputIndex := 0
	if fnType.NumIn() == 2 {
		if fnType.In(0) != ctxType {
			panic("FuncTool: when 2 args, first must be context.Context")
		}
		hasContext = true
		inputIndex = 1
	}

	inputType := fnType.In(inputIndex)

	return &funcTool{
		name:        name,
		description: description,
		fn:          fnValue,
		inputType:   inputType,
		hasContext:  hasContext,
	}
}

func typeToSchema(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		return structToSchema(t)
	case reflect.String:
		return &Schema{Type: SchemaTypeString}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: SchemaTypeInteger}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: SchemaTypeNumber}
	case reflect.Bool:
		return &Schema{Type: SchemaTypeBoolean}
	case reflect.Slice, reflect.Array:
		return &Schema{
			Type:  SchemaTypeArray,
			Items: typeToSchema(t.Elem()),
		}
	default:
		return &Schema{Type: SchemaTypeString}
	}
}

func structToSchema(t reflect.Type) *Schema {
	schema := &Schema{
		Type:       SchemaTypeObject,
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			name = parts[0]
			if name == "" {
				name = field.Name
			}
		}

		fieldSchema := typeToSchema(field.Type)
		descTag := field.Tag.Get("desc")
		if descTag != "" {
			fieldSchema.Description = descTag
		}

		schema.Properties[name] = fieldSchema

		if !strings.Contains(jsonTag, "omitempty") {
			schema.Required = append(schema.Required, name)
		}
	}

	return schema
}

func ToolToSchema(t Tool) map[string]any {
	params := t.Parameters()
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  SchemaToMap(params),
		},
	}
}

func SchemaToMap(s *Schema) map[string]any {
	m := map[string]any{
		"type": s.Type,
	}

	if s.Description != "" {
		m["description"] = s.Description
	}

	if len(s.Properties) > 0 {
		props := make(map[string]any)
		for k, v := range s.Properties {
			props[k] = SchemaToMap(v)
		}
		m["properties"] = props
	}

	if len(s.Required) > 0 {
		m["required"] = s.Required
	}

	if s.Items != nil {
		m["items"] = SchemaToMap(s.Items)
	}

	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}

	return m
}
