package directives

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestConfig_ToJSON(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name:   "empty config",
			config: Config{},
			want:   "{}",
		},
		{
			name: "simple config",
			config: Config{
				"key1": "value1",
				"key2": int64(42),
				"key3": true,
			},
			want: `{"key1":"value1","key2":42,"key3":true}`,
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
			want: `{"key1":"value1","key2":{"nested1":"nestedValue1","nested2":99},"key3":[1,2,3]}`,
		},
		{
			name: "config with nil value",
			config: Config{
				"key1": nil,
				"key2": "value2",
			},
			want: `{"key1":null,"key2":"value2"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.ToJSON()
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}
