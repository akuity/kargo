package kargomcp

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// noOutput is returned as the Out value by tool handlers with no typed
// structured output. The go-sdk omits StructuredContent when this is nil.
var noOutput any

// jsonAnyResult marshals an arbitrary value as structured content.
func jsonAnyResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(fmt.Errorf("marshal response: %w", err))
	}
	return &mcp.CallToolResult{
		StructuredContent: json.RawMessage(data),
	}, noOutput, nil
}

// errResult returns an error as a tool result (not a protocol-level error).
func errResult(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
		IsError: true,
	}, noOutput, nil
}

// mustOutputSchema derives a JSON Schema from T for use as Tool.OutputSchema.
// Panics at startup if schema derivation fails (programmer error).
func mustOutputSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("output schema for %T: %v", *new(T), err))
	}
	return s
}

// readOnly returns ToolAnnotations indicating a read-only, non-destructive tool.
func readOnly() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true}
}

// destructive returns ToolAnnotations indicating a write operation.
func destructive() *mcp.ToolAnnotations {
	t := true
	return &mcp.ToolAnnotations{DestructiveHint: &t}
}

// limitItems reverses a {"items":[...]} map to newest-first and truncates to
// limit entries (default 20 when limit <= 0). Pass the result of flattenFreightGroups
// or a similar helper that produces this shape.
func limitItems(m map[string]any, limit int) map[string]any {
	if limit <= 0 {
		limit = 20
	}
	raw, _ := m["items"].([]json.RawMessage)
	slices.Reverse(raw)
	if len(raw) > limit {
		raw = raw[:limit]
	}
	m["items"] = raw
	return m
}

// okResult returns a simple success message as a tool result.
func okResult(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, noOutput, nil
}
