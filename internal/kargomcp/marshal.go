package kargomcp

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

// toUnstructured converts any swagger-generated payload to *unstructured.Unstructured
// via a JSON round trip. On unmarshal failure the returned object is empty.
func toUnstructured(payload any) *unstructured.Unstructured {
	data, _ := json.Marshal(payload)
	u := &unstructured.Unstructured{}
	_ = json.Unmarshal(data, &u.Object)
	return u
}

// sanitizeResource strips noisy Kubernetes bookkeeping fields from an
// Unstructured object in-place before returning it to the LLM:
//   - metadata.managedFields (GC bookkeeping, no semantic value)
//   - metadata.generateName (template prefix, redundant with name)
//   - metadata.annotations["kubectl.kubernetes.io/last-applied-configuration"]
//     (full JSON-string duplicate of the spec)
//
// Also recursively drops null-valued fields via dropNulls.
func sanitizeResource(u *unstructured.Unstructured) *unstructured.Unstructured {
	u.SetManagedFields(nil)
	if meta, ok := u.Object["metadata"].(map[string]any); ok {
		delete(meta, "generateName")
	}
	anns := u.GetAnnotations()
	delete(anns, "kubectl.kubernetes.io/last-applied-configuration")
	if len(anns) == 0 {
		anns = nil
	}
	u.SetAnnotations(anns)
	if m, ok := dropNulls(u.Object).(map[string]any); ok {
		u.Object = m
	}
	return u
}

// dropNulls recursively removes nil values from maps and nil elements from
// slices, so the LLM doesn't see fields like "artifacts":null or "vars":null.
// Empty maps are intentionally left intact — their presence can be meaningful
// (e.g. "step-6":{} in status.state signals the step executed).
func dropNulls(v any) any {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			if child == nil {
				delete(val, k)
			} else {
				val[k] = dropNulls(child)
			}
		}
		return val
	case []any:
		for i, elem := range val {
			if elem != nil {
				val[i] = dropNulls(elem)
			}
		}
		return slices.DeleteFunc(val, func(e any) bool { return e == nil })
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
