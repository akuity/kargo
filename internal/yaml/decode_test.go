package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func TestDecodeField(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		path       string
		assertions func(*testing.T, any, error)
	}{
		{
			name: "simple string",
			yaml: "key: value",
			path: "key",
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "value", result)
			},
		},
		{
			name: "nested map",
			yaml: "outer:\n  inner: nested value",
			path: "outer.inner",
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "nested value", result)
			},
		},
		{
			name: "array access",
			yaml: "array:\n  - item1\n  - item2",
			path: "array.[1]",
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "item2", result)
			},
		},
		{
			name: "complex nested structure",
			yaml: `
root:
  nested:
    array:
      - key: value1
      - key: value2
    map:
      key1: val1
      key2: val2
`,
			path: "root.nested.array.[1].key",
			assertions: func(t *testing.T, result any, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "value2", result)
			},
		},
		{
			name: "non-existent field",
			yaml: "key: value",
			path: "nonexistent",
			assertions: func(t *testing.T, _ any, err error) {
				assert.EqualError(t, err, "field 'nonexistent' not found")
			},
		},
		{
			name: "invalid array index",
			yaml: "array:\n  - item1\n  - item2",
			path: "array.[2]",
			assertions: func(t *testing.T, _ any, err error) {
				assert.EqualError(t, err, "index out of range: 2")
			},
		},
		{
			name: "access nested field on scalar",
			yaml: "key: value",
			path: "key.nested",
			assertions: func(t *testing.T, _ any, err error) {
				assert.EqualError(t, err, "cannot access nested field on scalar node")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			assert.NoError(t, err, "Failed to unmarshal YAML")

			var result any
			err = DecodeField(&node, tt.path, &result)

			tt.assertions(t, result, err)
		})
	}
}

func Test_findNode(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		path       []string
		assertions func(*testing.T, *yaml.Node, error)
	}{
		{
			name: "root document node",
			yaml: "key: value",
			path: []string{"key"},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "value", result.Value)
			},
		},
		{
			name: "nested mapping",
			yaml: "outer:\n  inner: nested value",
			path: []string{"outer", "inner"},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "nested value", result.Value)
			},
		},
		{
			name: "sequence access",
			yaml: "array:\n  - item1\n  - item2",
			path: []string{"array", "[1]"},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "item2", result.Value)
			},
		},
		{
			name: "empty path",
			yaml: "key: value",
			path: []string{},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, yaml.DocumentNode, result.Kind)
			},
		},
		{
			name: "non-existent field",
			yaml: "key: value",
			path: []string{"nonexistent"},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.EqualError(t, err, "field 'nonexistent' not found")
				assert.Nil(t, result)
			},
		},
		{
			name: "invalid sequence index",
			yaml: "array:\n  - item1\n  - item2",
			path: []string{"array", "invalid"},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid index format")
				assert.Nil(t, result)
			},
		},
		{
			name: "out of range sequence index",
			yaml: "array:\n  - item1\n  - item2",
			path: []string{"array", "[5]"},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.EqualError(t, err, "index out of range: 5")
				assert.Nil(t, result)
			},
		},
		{
			name: "access nested field on scalar",
			yaml: "key: value",
			path: []string{"key", "nested"},
			assertions: func(t *testing.T, result *yaml.Node, err error) {
				assert.EqualError(t, err, "cannot access nested field on scalar node")
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			assert.NoError(t, err)

			result, err := findNode(&node, tt.path)
			tt.assertions(t, result, err)
		})
	}
}
