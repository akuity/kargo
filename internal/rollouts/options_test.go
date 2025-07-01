package rollouts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestAnalysisRunOptions(t *testing.T) {
	tests := []struct {
		name       string
		options    []AnalysisRunOption
		assertions func(*testing.T, *AnalysisRunOptions)
	}{
		{
			name: "name prefix with normal length",
			options: []AnalysisRunOption{
				WithNamePrefix("test-prefix"),
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Equal(t, "test-prefix", opts.NamePrefix)
			},
		},
		{
			name: "name prefix truncates long prefix",
			options: []AnalysisRunOption{
				WithNamePrefix("a" + stringWithLength(maxNamePrefixLength+10)),
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Len(t, opts.NamePrefix, maxNamePrefixLength)
			},
		},
		{
			name: "name suffix with normal length",
			options: []AnalysisRunOption{
				WithNameSuffix("suffix1"),
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Equal(t, "suffix1", opts.NameSuffix)
			},
		},
		{
			name: "name suffix truncates long suffix",
			options: []AnalysisRunOption{
				WithNameSuffix("a" + stringWithLength(maxNameSuffixLength+10)),
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Len(t, opts.NameSuffix, maxNameSuffixLength)
			},
		},
		{
			name: "extra labels: single set",
			options: []AnalysisRunOption{
				WithExtraLabels{"key1": "value1", "key2": "value2"},
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Equal(t, map[string]string{
					"key1": "value1",
					"key2": "value2",
				}, opts.ExtraLabels)
			},
		},
		{
			name: "extra labels: multiple sets are merged",
			options: []AnalysisRunOption{
				WithExtraLabels{"key1": "value1"},
				WithExtraLabels{"key2": "value2"},
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Equal(t, map[string]string{
					"key1": "value1",
					"key2": "value2",
				}, opts.ExtraLabels)
			},
		},
		{
			name: "extra annotations: single set",
			options: []AnalysisRunOption{
				WithExtraAnnotations{"key1": "value1", "key2": "value2"},
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Equal(t, map[string]string{
					"key1": "value1",
					"key2": "value2",
				}, opts.ExtraAnnotations)
			},
		},
		{
			name: "extra annotations: multiple sets are merged",
			options: []AnalysisRunOption{
				WithExtraAnnotations{"key1": "value1"},
				WithExtraAnnotations{"key2": "value2"},
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Equal(t, map[string]string{
					"key1": "value1",
					"key2": "value2",
				}, opts.ExtraAnnotations)
			},
		},
		{
			name: "single owner",
			options: []AnalysisRunOption{
				WithOwner(Owner{
					APIVersion:    "v1",
					Kind:          "Pod",
					Reference:     types.NamespacedName{Name: "pod1", Namespace: "default"},
					BlockDeletion: true,
				}),
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Len(t, opts.Owners, 1)
				assert.Equal(t, opts.Owners[0], Owner{
					APIVersion:    "v1",
					Kind:          "Pod",
					Reference:     types.NamespacedName{Name: "pod1", Namespace: "default"},
					BlockDeletion: true,
				})
			},
		},
		{
			name: "multiple owners",
			options: []AnalysisRunOption{
				WithOwner(Owner{
					APIVersion: "v1",
					Kind:       "Pod",
					Reference:  types.NamespacedName{Name: "pod1", Namespace: "default"},
				}),
				WithOwner(Owner{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Reference:  types.NamespacedName{Name: "deploy1", Namespace: "default"},
				}),
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Len(t, opts.Owners, 2)
				assert.Equal(t, "Pod", opts.Owners[0].Kind)
				assert.Equal(t, "Deployment", opts.Owners[1].Kind)
			},
		},
		{
			name: "argument evaluation config",
			options: []AnalysisRunOption{
				WithArgumentEvaluationConfig{
					Env: map[string]any{
						"key": "value",
					},
					Vars: []kargoapi.ExpressionVariable{
						{Name: "pokemon_1", Value: "pikachu"},
						{Name: "pokemon_2", Value: "charizard"},
					},
				},
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.NotNil(t, opts.ExpressionConfig)
				assert.Equal(t, map[string]any{"key": "value"}, opts.ExpressionConfig.Env)
				assert.Equal(
					t,
					[]kargoapi.ExpressionVariable{
						{Name: "pokemon_1", Value: "pikachu"},
						{Name: "pokemon_2", Value: "charizard"},
					},
					opts.ExpressionConfig.Vars,
				)
			},
		},
		{
			name: "combined options",
			options: []AnalysisRunOption{
				WithNamePrefix("prefix"),
				WithNameSuffix("suffix"),
				WithExtraLabels{"key": "value"},
				WithOwner(Owner{
					APIVersion: "v1",
					Kind:       "Pod",
					Reference:  types.NamespacedName{Name: "pod1", Namespace: "default"},
				}),
				WithArgumentEvaluationConfig{
					Env: map[string]any{
						"key": "value",
					},
					Vars: []kargoapi.ExpressionVariable{
						{Name: "pokemon_1", Value: "pikachu"},
						{Name: "pokemon_2", Value: "charizard"},
					},
				},
			},
			assertions: func(t *testing.T, opts *AnalysisRunOptions) {
				assert.Equal(t, "prefix", opts.NamePrefix)
				assert.Equal(t, "suffix", opts.NameSuffix)
				assert.Equal(t, map[string]string{"key": "value"}, opts.ExtraLabels)
				assert.Len(t, opts.Owners, 1)
				assert.NotNil(t, opts.ExpressionConfig)
				assert.Equal(
					t,
					[]kargoapi.ExpressionVariable{
						{Name: "pokemon_1", Value: "pikachu"},
						{Name: "pokemon_2", Value: "charizard"},
					},
					opts.ExpressionConfig.Vars,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &AnalysisRunOptions{}
			opts.Apply(tt.options...)
			tt.assertions(t, opts)
		})
	}
}

func stringWithLength(length int) string {
	result := make([]rune, length)
	for i := range result {
		result[i] = 'x'
	}
	return string(result)
}
