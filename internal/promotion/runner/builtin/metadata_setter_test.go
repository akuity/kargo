package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/promotion"
)

func Test_metadataSetter_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "missing updates field",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): updates is required",
			},
		},
		{
			name:   "invalid config type",
			config: promotion.Config{"updates": "not-an-array"},
			expectedProblems: []string{
				"updates: Invalid type. Expected: array, given: string",
			},
		},
		{
			name: "missing required fields in updates",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						// missing kind, name, values
					},
				},
			},
			expectedProblems: []string{
				"updates.0: kind is required",
				"updates.0: name is required",
				"updates.0: values is required",
			},
		},
		{
			name: "invalid kind",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind":   "InvalidKind",
						"name":   "test",
						"values": map[string]any{"key": "value"},
					},
				},
			},
			expectedProblems: []string{
				"updates.0.kind: updates.0.kind must be one of the following: \"Stage\", \"Freight\"",
			},
		},
		{
			name: "empty name",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind":   "Stage",
						"name":   "",
						"values": map[string]any{"key": "value"},
					},
				},
			},
			expectedProblems: []string{
				"name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "empty values",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind":   "Stage",
						"name":   "test",
						"values": map[string]any{},
					},
				},
			},
			expectedProblems: []string{
				"values: Must have at least 1 properties",
			},
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"updates": []map[string]any{
					{
						"kind": "Stage",
						"name": "test-stage",
						"values": map[string]any{
							"string":  "value",
							"number":  42,
							"bool":    true,
							"nullKey": nil,
							"object": map[string]any{
								"nested": "value",
								"array":  []any{"item1", "item2"},
								"deep": map[string]any{
									"foo":  "bar",
									"nums": []any{1, 2, 3},
								},
							},
						},
					},
					{
						"kind": "Freight",
						"name": "test-freight",
						"values": map[string]any{
							"deployed": true,
							"version":  "1.0.0",
						},
					},
				},
			},
			expectedProblems: nil,
		},
	}

	r := newMetadataSetter(nil)
	runner, ok := r.(*metadataSetter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}
