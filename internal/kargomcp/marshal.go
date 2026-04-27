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

// projectItems unmarshals each raw JSON item into T, reverses the slice
// (newest-first), and applies project to produce a summary S.
// Items that fail to unmarshal are skipped.
func projectItems[T, S any](raws []json.RawMessage, project func(T) S) []S {
	slices.Reverse(raws)
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


// sanitizeResource strips noisy Kubernetes bookkeeping fields from a resource
// before returning it to the LLM:
//   - metadata.managedFields (GC bookkeeping, no semantic value)
//   - metadata.resourceVersion (internal Kubernetes state)
//   - metadata.generateName (template prefix, redundant with name)
//   - metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"]
//     (full JSON-string duplicate of the spec)
func sanitizeResource(payload any) any {
	data, _ := json.Marshal(payload)
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return payload
	}
	meta, _ := m["metadata"].(map[string]any)
	if meta == nil {
		return m
	}
	delete(meta, "managedFields")
	delete(meta, "resourceVersion")
	delete(meta, "generateName")
	if anns, ok := meta["annotations"].(map[string]any); ok {
		delete(anns, "kubectl.kubernetes.io/last-applied-configuration")
		if len(anns) == 0 {
			delete(meta, "annotations")
		}
	}
	return dropNulls(m)
}

// dropNulls recursively removes nil values and empty maps from the JSON tree
// so the LLM doesn't see noise like "artifacts":null, "retry":{}, "task":{}.
func dropNulls(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			if child == nil {
				delete(val, k)
				continue
			}
			cleaned := dropNulls(child)
			if m, ok := cleaned.(map[string]any); ok && len(m) == 0 {
				delete(val, k)
			} else {
				val[k] = cleaned
			}
		}
		return val
	case []any:
		out := val[:0]
		for _, elem := range val {
			if elem == nil {
				continue
			}
			cleaned := dropNulls(elem)
			if m, ok := cleaned.(map[string]any); ok && len(m) == 0 {
				continue
			}
			out = append(out, cleaned)
		}
		return out
	default:
		return v
	}
}

// okResult returns a simple success message as a tool result.
func okResult(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, noOutput, nil
}
