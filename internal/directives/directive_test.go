package directives

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestState_Set(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() State
		key      string
		value    any
		expected any
	}{
		{
			name:     "Set string value",
			setup:    func() State { return make(State) },
			key:      "key1",
			value:    "value1",
			expected: "value1",
		},
		{
			name:     "Set integer value",
			setup:    func() State { return make(State) },
			key:      "key2",
			value:    42,
			expected: 42,
		},
		{
			name:     "Set slice value",
			setup:    func() State { return make(State) },
			key:      "key3",
			value:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Set map value",
			setup:    func() State { return make(State) },
			key:      "key4",
			value:    map[string]int{"a": 1, "b": 2},
			expected: map[string]int{"a": 1, "b": 2},
		},
		{
			name:     "Set nil value",
			setup:    func() State { return make(State) },
			key:      "key5",
			value:    nil,
			expected: nil,
		},
		{
			name: "Overwrite existing value",
			setup: func() State {
				s := make(State)
				s["key"] = "initial_value"
				return s
			},
			key:      "key",
			value:    "new_value",
			expected: "new_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.setup()
			state.Set(tt.key, tt.value)
			assert.Equal(t, tt.expected, state[tt.key])
		})
	}
}

func TestState_Get(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() State
		key      string
		expected any
		exists   bool
	}{
		{
			name: "Get existing string value",
			setup: func() State {
				s := make(State)
				s["key1"] = "value1"
				return s
			},
			key:      "key1",
			expected: "value1",
			exists:   true,
		},
		{
			name: "Get existing integer value",
			setup: func() State {
				s := make(State)
				s["key2"] = 42
				return s
			},
			key:      "key2",
			expected: 42,
			exists:   true,
		},
		{
			name: "Get existing slice value",
			setup: func() State {
				s := make(State)
				s["key3"] = []string{"a", "b", "c"}
				return s
			},
			key:      "key3",
			expected: []string{"a", "b", "c"},
			exists:   true,
		},
		{
			name: "Get existing map value",
			setup: func() State {
				s := make(State)
				s["key4"] = map[string]int{"a": 1, "b": 2}
				return s
			},
			key:      "key4",
			expected: map[string]int{"a": 1, "b": 2},
			exists:   true,
		},
		{
			name: "Get existing nil value",
			setup: func() State {
				s := make(State)
				s["key5"] = nil
				return s
			},
			key:      "key5",
			expected: nil,
			exists:   true,
		},
		{
			name: "Get non-existent key",
			setup: func() State {
				return make(State)
			},
			key:      "non_existent",
			expected: nil,
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := tt.setup()
			value, ok := state.Get(tt.key)

			assert.Equal(t, tt.expected, value)
			assert.Equal(t, tt.exists, ok)
		})
	}
}

func TestConfig_DeepCopy(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		assertions func(*testing.T, Config, Config)
	}{
		{
			name:   "nil config",
			config: nil,
			assertions: func(t *testing.T, _, copied Config) {
				assert.Nil(t, copied, "Expected nil result for nil input")
			},
		},
		{
			name:   "empty config",
			config: Config{},
			assertions: func(t *testing.T, original, copied Config) {
				assert.Empty(t, copied, "Expected empty result for empty input")
				assert.NotSame(t, original, copied, "Expected a new instance, not the same reference")
			},
		},
		{
			name: "simple config",
			config: Config{
				"key1": "value1",
				"key2": int64(42),
				"key3": true,
			},
			assertions: func(t *testing.T, original, copied Config) {
				assert.Equal(t, original, copied, "Expected equal content")
				assert.NotSame(t, original, copied, "Expected a new instance, not the same reference")

				// Modify original to ensure deep copy
				original["key1"] = "modified"
				assert.NotEqual(t, original, copied, "Modifying original should not affect the copy")
			},
		},
		{
			name: "nested config",
			config: Config{
				"key1": "value1",
				"key2": map[string]any{
					"nested1": "nestedValue1",
					"nested2": int64(99),
				},
				"key3": []any{int64(1), int64(2), int64(3)},
			},
			assertions: func(t *testing.T, original, copied Config) {
				assert.Equal(t, original, copied, "Expected equal content")
				assert.NotSame(t, original, copied, "Expected a new instance, not the same reference")

				// Check nested map
				originalNested := original["key2"].(map[string]any) // nolint: forcetypeassert
				copiedNested := copied["key2"].(map[string]any)     // nolint: forcetypeassert
				assert.Equal(t, originalNested, copiedNested, "Expected equal nested content")
				assert.NotSame(t, originalNested, copiedNested, "Expected a new instance for nested map")

				// Modify original nested map
				originalNested["nested1"] = "modified"
				assert.NotEqual(t, originalNested, copiedNested, "Modifying original nested map should not affect the copy")

				// Check slice
				originalSlice := original["key3"].([]any) // nolint: forcetypeassert
				copiedSlice := copied["key3"].([]any)     // nolint: forcetypeassert
				assert.Equal(t, originalSlice, copiedSlice, "Expected equal slice content")
				assert.NotSame(t, originalSlice, copiedSlice, "Expected a new instance for slice")

				// Modify original slice
				originalSlice[0] = 999
				assert.NotEqual(t, originalSlice, copiedSlice, "Modifying original slice should not affect the copy")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, tt.config, tt.config.DeepCopy())
		})
	}
}
