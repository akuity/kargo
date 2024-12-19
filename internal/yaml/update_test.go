package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func TestUpdateField(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		path       string
		value      any
		assertions func(*testing.T, *yaml.Node, string, error)
	}{
		{
			name:  "update existing simple field",
			yaml:  "key: old value",
			path:  "key",
			value: "new value",
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "new value", node.Content[0].Content[1].Value)
				assert.Equal(t, `key: new value
`, result)
			},
		},
		{
			name: "update nested field",
			yaml: `root:
    nested: old value`,
			path:  "root.nested",
			value: "new value",
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "new value", node.Content[0].Content[1].Content[1].Value)
				assert.Equal(t, `root:
    nested: new value
`, result)
			},
		},
		{
			name: "update nested field with a number for a key",
			yaml: `root:
    0: old value`,
			path:  "root.0",
			value: "new value",
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "new value", node.Content[0].Content[1].Content[1].Value)
				assert.Equal(t, `root:
    0: new value
`, result)
			},
		},
		{
			name: "update nested field with a number for a key (with colon)",
			yaml: `root:
    0: old value`,
			path:  "root.:0",
			value: "new value",
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "new value", node.Content[0].Content[1].Content[1].Value)
				assert.Equal(t, `root:
    0: new value
`, result)
			},
		},
		{
			name:  "create new nested field",
			yaml:  `root:`,
			path:  "root.new.nested",
			value: "new value",
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "new value", node.Content[0].Content[1].Content[1].Content[1].Value)
				assert.Equal(t, `root:
    new:
        nested: new value
`, result)
			},
		},
		{
			name: "add new scalar to end of sequence (legacy syntax)",
			yaml: `
root:
  array:
    - item1
    - item2`,
			path:  "root.array.[2]",
			value: "item3",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - item2
        - item3
`, result)
			},
		},
		{
			name: "add new scalar to end of sequence",
			yaml: `
root:
  array:
    - item1
    - item2`,
			path:  "root.array.-1",
			value: "item3",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - item2
        - item3
`, result)
			},
		},
		{
			name: "add new mapping to end of sequence (legacy syntax)",
			yaml: `
root:
  array:
    - item1
    - item2`,
			path:  "root.array.[2].key",
			value: "value",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - item2
        - key: value
`, result)
			},
		},
		{
			name: "add new mapping to end of sequence",
			yaml: `
root:
  array:
    - item1
    - item2`,
			path:  "root.array.-1.key",
			value: "value",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - item2
        - key: value
`, result)
			},
		},
		{
			name: "add new item beyond current length (legacy syntax)",
			yaml: `
root:
  array:
    - item1`,
			path:  "root.array.[5]",
			value: "item2",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - item2
`, result)
			},
		},
		{
			name: "add new item beyond current length",
			yaml: `
root:
  array:
    - item1`,
			path:  "root.array.-1",
			value: "item2",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - item2
`, result)
			},
		},
		{
			name: "update existing item in sequence (legacy syntax)",
			yaml: `
root:
  array:
    - item1
    - item2
    - item3`,
			path:  "root.array.[1]",
			value: "updated_item",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - updated_item
        - item3
`, result)
			},
		},
		{
			name: "update existing item in sequence",
			yaml: `
root:
  array:
    - item1
    - item2
    - item3`,
			path:  "root.array.1",
			value: "updated_item",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - updated_item
        - item3
`, result)
			},
		},
		{
			name: "add new sequence to end of sequence (legacy syntax)",
			yaml: `
root:
  array:
    - item1`,
			path:  "root.array.[1]",
			value: []string{"sub1", "sub2"},
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - - sub1
          - sub2
`, result)
			},
		},
		{
			name: "add new sequence to end of sequence",
			yaml: `
root:
  array:
    - item1`,
			path:  "root.array.-1",
			value: []string{"sub1", "sub2"},
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `root:
    array:
        - item1
        - - sub1
          - sub2
`, result)
			},
		},
		{
			name: "update with complex value",
			yaml: "key: old value",
			path: "key",
			value: map[string]any{
				"array":  []int{1, 2, 3},
				"nested": "value",
			},
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, yaml.MappingNode, node.Content[0].Content[1].Kind)
				assert.Equal(t, `key:
    array:
        - 1
        - 2
        - 3
    nested: value
`, result)
			},
		},
		{
			name:  "error on empty path",
			yaml:  "key: value",
			path:  "",
			value: "new value",
			assertions: func(t *testing.T, _ *yaml.Node, _ string, err error) {
				assert.EqualError(t, err, "empty key")
			},
		},
		{
			name: "error on empty path during recursive update",
			yaml: `root:
    nested:
        key: value`,
			path:  "root.nested.", // Note the trailing dot, which will result in an empty part
			value: "new value",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				assert.ErrorContains(t, err, "empty key part in key")
				assert.Empty(t, result)
			},
		},
		{
			name:  "error on invalid array index",
			yaml:  "array:\n  - item1",
			path:  "array.invalid",
			value: "new item",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				assert.EqualError(t, err, "invalid index format: invalid")
				assert.Empty(t, result)
			},
		},
		{
			name: "error on negative array index (legacy syntax)",
			yaml: `
root:
  array:
    - item1
    - item2
    - item3`,
			path:  "root.array.[-2]",
			value: "new value",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.Error(t, err)
				assert.Equal(t, "invalid negative index: -2", err.Error())
				assert.Empty(t, result)
			},
		},
		{
			name: "error on negative array index",
			yaml: `
root:
  array:
    - item1
    - item2
    - item3`,
			path:  "root.array.-2",
			value: "new value",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.Error(t, err)
				assert.Equal(t, "invalid negative index: -2", err.Error())
				assert.Empty(t, result)
			},
		},
		{
			name: "update through alias node",
			yaml: `
anchor: &anchor
  key: old_value
alias: *anchor`,
			path:  "alias.key",
			value: "new_value",
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `anchor: &anchor
    key: new_value
alias: *anchor
`, result)

				// Verify that both the anchor and alias reflect the update
				anchorNode := node.Content[0].Content[1]
				aliasNode := node.Content[0].Content[3]
				assert.Equal(t, "new_value", anchorNode.Content[1].Value)
				assert.Equal(t, anchorNode, aliasNode.Alias)
			},
		},
		{
			name: "update alias node itself",
			yaml: `
anchor: &anchor
  key: value
alias: *anchor`,
			path:  "alias",
			value: "new_value",
			assertions: func(t *testing.T, node *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `anchor: &anchor
    key: value
alias: new_value
`, result)

				// Verify that the alias has been replaced with a scalar
				aliasNode := node.Content[0].Content[3]
				assert.Equal(t, yaml.ScalarNode, aliasNode.Kind)
				assert.Equal(t, "new_value", aliasNode.Value)
			},
		},
		{
			name: "update nested field preserving comments",
			yaml: `# Root comment
root:
    # Nested comment
    nested:
        key1: value1 # Inline comment
        key2: value2
    # Array comment
    array:
        - item1 # Array item comment
        - item2`,
			path:  "root.nested.key1",
			value: "new value",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `# Root comment
root:
    # Nested comment
    nested:
        key1: new value # Inline comment
        key2: value2
    # Array comment
    array:
        - item1 # Array item comment
        - item2
`, result)
			},
		},
		{
			name: "update array item preserving comments",
			yaml: `# Root comment
root:
    # Nested comment
    nested:
        key1: value1 # Inline comment
        key2: value2
    # Array comment
    array:
        - item1 # Array item comment
        - item2`,
			path:  "root.array.[0]",
			value: "new item",
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `# Root comment
root:
    # Nested comment
    nested:
        key1: value1 # Inline comment
        key2: value2
    # Array comment
    array:
        - new item # Array item comment
        - item2
`, result)
			},
		},
		{
			name: "replace entire mapping node preserving comments",
			yaml: `# Root comment
root:
    # Nested comment
    nested:
        key1: value1 # Inline comment 1
        key2: value2 # Inline comment 2
    other: value`,
			path: "root.nested",
			value: map[string]any{
				"newKey1": "newValue1",
				"newKey2": "newValue2",
				"key1":    "updatedValue1",
			},
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `# Root comment
root:
    # Nested comment
    nested:
        key1: updatedValue1 # Inline comment 1
        newKey1: newValue1
        newKey2: newValue2
    other: value
`, result)
			},
		},
		{
			name: "replace entire sequence node preserving comments",
			yaml: `# Root comment
root:
    # Array comment
    array:
        - item1 # Comment 1
        - item2 # Comment 2
        - item3 # Comment 3
    other: value`,
			path:  "root.array",
			value: []string{"newItem1", "newItem2", "newItem3", "newItem4"},
			assertions: func(t *testing.T, _ *yaml.Node, result string, err error) {
				require.NoError(t, err)
				assert.Equal(t, `# Root comment
root:
    # Array comment
    array:
        - newItem1 # Comment 1
        - newItem2 # Comment 2
        - newItem3 # Comment 3
        - newItem4
    other: value
`, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.yaml), &node)
			require.NoError(t, err)

			err = UpdateField(&node, tt.path, tt.value)
			var result string
			if err == nil {
				output, marshalErr := yaml.Marshal(&node)
				require.NoError(t, marshalErr)
				result = string(output)
			}
			tt.assertions(t, &node, result, err)
		})
	}
}

func TestSplitKey(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		expected    []string
		errContains string
	}{
		{
			name:        "empty key",
			errContains: "empty key",
		},
		{
			name:        "starts with dot",
			key:         ".foo",
			errContains: "empty key part in key",
		},
		{
			name:        "ends with dot",
			key:         "foo.",
			errContains: "empty key part in key",
		},
		{
			name:        "double dots",
			key:         "foo..bar",
			errContains: "empty key part in key",
		},
		{
			name:        "invalid escape sequence",
			key:         `foo\nbar`,
			errContains: "invalid escape sequence",
		},
		{
			name:        "invalid use of colon",
			key:         `foo:bar`,
			errContains: "unexpected colon in key",
		},
		{
			name:     "basic split",
			key:      "foo.bar.bat.baz",
			expected: []string{"foo", "bar", "bat", "baz"},
		},
		{
			name:     "split key with escaped dots",
			key:      `foo\.bar.bat\.baz`,
			expected: []string{"foo.bar", "bat.baz"},
		},
		{
			name:     "split key with escaped colon",
			key:      `foo.bar\:bat.baz`,
			expected: []string{"foo", "bar:bat", "baz"},
		},
		{
			name:     "split key with unescaped colon",
			key:      `foo.bar.:2.baz`,
			expected: []string{"foo", "bar", "2", "baz"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parts, err := splitKey(testCase.key)
			if testCase.errContains != "" {
				require.ErrorContains(t, err, testCase.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expected, parts)
			}
		})
	}
}
