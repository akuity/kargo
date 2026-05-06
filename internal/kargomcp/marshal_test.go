package kargomcp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestSanitizeResource(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		input  any
		assert func(*testing.T, *unstructured.Unstructured)
	}{
		{
			name: "removes managedFields and generateName",
			input: map[string]any{
				"metadata": map[string]any{
					"name":          "foo",
					"generateName":  "foo-",
					"managedFields": []any{map[string]any{"manager": "kubectl"}},
				},
			},
			assert: func(t *testing.T, got *unstructured.Unstructured) {
				meta, ok := got.Object["metadata"].(map[string]any)
				require.True(t, ok)
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
			assert: func(t *testing.T, got *unstructured.Unstructured) {
				meta, ok := got.Object["metadata"].(map[string]any)
				require.True(t, ok)
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
			assert: func(t *testing.T, got *unstructured.Unstructured) {
				meta, ok := got.Object["metadata"].(map[string]any)
				require.True(t, ok)
				anns, ok := meta["annotations"].(map[string]any)
				require.True(t, ok)
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
			assert: func(t *testing.T, got *unstructured.Unstructured) {
				spec, ok := got.Object["spec"].(map[string]any)
				require.True(t, ok)
				require.NotContains(t, spec, "field")
				require.Equal(t, "val", spec["other"])
			},
		},
		{
			name:  "no metadata passes through",
			input: map[string]any{"kind": "Foo"},
			assert: func(t *testing.T, got *unstructured.Unstructured) {
				require.Equal(t, "Foo", got.Object["kind"])
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.assert(t, sanitizeResource(toUnstructured(tc.input)))
		})
	}
}
