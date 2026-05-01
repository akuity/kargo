package kargomcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectItems(t *testing.T) {
	t.Parallel()
	type in struct {
		V int `json:"v"`
	}
	raw := func(v int) json.RawMessage {
		b, _ := json.Marshal(in{V: v})
		return b
	}

	testCases := []struct {
		name   string
		raws   []json.RawMessage
		assert func(*testing.T, []int)
	}{
		{
			name:   "nil input",
			raws:   nil,
			assert: func(t *testing.T, got []int) { require.Empty(t, got) },
		},
		{
			name: "reverses order (newest-first)",
			raws: []json.RawMessage{raw(1), raw(2), raw(3)},
			assert: func(t *testing.T, got []int) {
				require.Equal(t, []int{3, 2, 1}, got)
			},
		},
		{
			name: "skips invalid JSON",
			raws: []json.RawMessage{raw(1), []byte("bad"), raw(3)},
			assert: func(t *testing.T, got []int) {
				require.Equal(t, []int{3, 1}, got)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := projectItems(tc.raws, func(i in) int { return i.V })
			tc.assert(t, got)
		})
	}
}

func TestSanitizeResource(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  any
		assert func(*testing.T, any)
	}{
		{
			name: "removes managedFields and generateName",
			input: map[string]any{
				"metadata": map[string]any{
					"name":            "foo",
					"generateName":    "foo-",
					"managedFields":   []any{map[string]any{"manager": "kubectl"}},
				},
			},
			assert: func(t *testing.T, got any) {
				m := got.(map[string]any)
				meta := m["metadata"].(map[string]any)
				require.Equal(t, "foo", meta["name"])
				require.NotContains(t, meta, "generateName")
				require.NotContains(t, meta, "managedFields")
			},
		},
		{
			name: "removes last-applied annotation, drops empty annotations map",
			input: map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/last-applied-configuration": `{"spec":{}}`,
					},
				},
			},
			assert: func(t *testing.T, got any) {
				meta := got.(map[string]any)["metadata"].(map[string]any)
				require.NotContains(t, meta, "annotations")
			},
		},
		{
			name: "keeps other annotations",
			input: map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"kubectl.kubernetes.io/last-applied-configuration": `{"spec":{}}`,
						"example.com/keep": "yes",
					},
				},
			},
			assert: func(t *testing.T, got any) {
				anns := got.(map[string]any)["metadata"].(map[string]any)["annotations"].(map[string]any)
				require.Equal(t, "yes", anns["example.com/keep"])
				require.NotContains(t, anns, "kubectl.kubernetes.io/last-applied-configuration")
			},
		},
		{
			name: "drops null values",
			input: map[string]any{
				"metadata": map[string]any{"name": "x"},
				"spec":     map[string]any{"field": nil, "other": "val"},
			},
			assert: func(t *testing.T, got any) {
				spec := got.(map[string]any)["spec"].(map[string]any)
				require.NotContains(t, spec, "field")
				require.Equal(t, "val", spec["other"])
			},
		},
		{
			name:  "no metadata passes through",
			input: map[string]any{"kind": "Foo"},
			assert: func(t *testing.T, got any) {
				require.Equal(t, "Foo", got.(map[string]any)["kind"])
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.assert(t, sanitizeResource(tc.input))
		})
	}
}

func TestFlattenFreightGroups(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  any
		assert func(*testing.T, []json.RawMessage)
	}{
		{
			name:   "nil payload",
			input:  nil,
			assert: func(t *testing.T, got []json.RawMessage) { require.Empty(t, got) },
		},
		{
			name: "single default group",
			input: map[string]any{
				"groups": map[string]any{
					"": map[string]any{
						"items": []any{
							map[string]any{"name": "a"},
							map[string]any{"name": "b"},
						},
					},
				},
			},
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Len(t, got, 2)
			},
		},
		{
			name: "multiple groups are merged",
			input: map[string]any{
				"groups": map[string]any{
					"g1": map[string]any{"items": []any{map[string]any{"name": "a"}}},
					"g2": map[string]any{"items": []any{map[string]any{"name": "b"}}},
				},
			},
			assert: func(t *testing.T, got []json.RawMessage) {
				require.Len(t, got, 2)
			},
		},
		{
			name: "empty groups yields empty slice",
			input: map[string]any{
				"groups": map[string]any{},
			},
			assert: func(t *testing.T, got []json.RawMessage) { require.Empty(t, got) },
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := flattenFreightGroups(tc.input)
			tc.assert(t, got)
		})
	}
}
