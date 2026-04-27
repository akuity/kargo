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

// projectItems unmarshals each raw JSON item into T, applies project to
// produce a summary S, reverses the slice (newest-first), and truncates to
// limit (default 20). Items that fail to unmarshal are skipped.
func projectItems[T, S any](raws []json.RawMessage, limit int, project func(T) S) []S {
	if limit <= 0 {
		limit = 20
	}
	slices.Reverse(raws)
	if len(raws) > limit {
		raws = raws[:limit]
	}
	out := make([]S, 0, len(raws))
	for _, raw := range raws {
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		out = append(out, project(item))
	}
	return out
}


// okResult returns a simple success message as a tool result.
func okResult(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, noOutput, nil
}
