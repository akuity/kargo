package external

import (
	"testing"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func Test_parseValuesAsList(t *testing.T) {
	tests := []struct {
		name      string
		values    *apiextensionsv1.JSON
		env       map[string]any
		expected  []string
		expectErr bool
	}{
		{
			name:     "nil values",
			values:   nil,
			env:      nil,
			expected: nil,
		},
		{
			name:      "invalid json",
			values:    &apiextensionsv1.JSON{Raw: []byte(`{invalid json}`)},
			env:       nil,
			expectErr: true,
		},
		{
			name:      "expression does not evaluate to []string",
			values:    &apiextensionsv1.JSON{Raw: []byte(`"42"`)},
			env:       nil,
			expectErr: true,
		},
		{
			name:     "list of strings",
			values:   &apiextensionsv1.JSON{Raw: []byte(`["a", "b", "c"]`)},
			env:      nil,
			expected: []string{"a", "b", "c"},
		},
		{
			name: "list of expressions returning 1 value each",
			values: &apiextensionsv1.JSON{
				Raw: []byte(
					`[
					"${{ request.body.repository.clone_url }}", 
					"${{ request.body.repository.other_url }}"
				]`,
				)},
			env: map[string]any{
				"request": map[string]any{
					"body": map[string]any{
						"repository": map[string]any{
							"clone_url": "https://example.com/repo.git",
							"other_url": "https://example.com/other.git",
						},
					},
				},
			},
			expected: []string{
				"https://example.com/repo.git",
				"https://example.com/other.git",
			},
		},
		{
			name: "list of expressions returning multiple value each",
			values: &apiextensionsv1.JSON{
				Raw: []byte(
					`[
						"${{ request.body.repository.branchOne.commits }}", 
						"${{ request.body.repository.branchTwo.commits }}"
					]`,
				),
			},
			env: map[string]any{
				"request": map[string]any{
					"body": map[string]any{
						"repository": map[string]any{
							"branchOne": map[string]any{
								"commits": []string{"abc123", "efg456"},
							},
							"branchTwo": map[string]any{
								"commits": []string{"ghi789", "jkl000"},
							},
						},
					},
				},
			},
			expected: []string{"abc123", "efg456", "ghi789", "jkl000"},
		},
		{
			name: "mixed static, expression single value, and expression multiple values",
			values: &apiextensionsv1.JSON{
				Raw: []byte(
					`[
						"my-static-value",
						"${{ request.body.repository.clone_url }}", 
						"${{ request.body.repository.branchOne.commits }}"
					]`,
				),
			},
			env: map[string]any{
				"request": map[string]any{
					"body": map[string]any{
						"repository": map[string]any{
							"clone_url": "https://example.com/repo.git",
							"branchOne": map[string]any{
								"commits": []string{"abc123", "efg456"},
							},
						},
					},
				},
			},
			expected: []string{
				"my-static-value",
				"https://example.com/repo.git",
				"abc123", "efg456",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseValuesAsList(tt.values, tt.env)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, got)
		})
	}
}
