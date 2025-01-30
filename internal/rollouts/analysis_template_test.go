package rollouts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

func Test_flattenTemplates(t *testing.T) {
	tests := []struct {
		name          string
		templateSpecs []*rolloutsapi.AnalysisTemplateSpec
		assertions    func(*testing.T, *rolloutsapi.AnalysisTemplate, error)
	}{
		{
			name:          "handle nil templates",
			templateSpecs: nil,
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				require.NotNil(t, template)
				require.Empty(t, template.Spec.Metrics)
				require.Empty(t, template.Spec.Args)
				require.Empty(t, template.Spec.DryRun)
				require.Empty(t, template.Spec.MeasurementRetention)
			},
		},
		{
			name:          "handle empty list",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{},
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				require.Empty(t, template.Spec.Metrics)
				require.Empty(t, template.Spec.Args)
			},
		},
		{
			name: "no changes on single template",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Metrics: []rolloutsapi.Metric{
						{
							Name:             "foo",
							SuccessCondition: "{{args.test}}",
						},
					},
					Args: []rolloutsapi.Argument{
						{
							Name:  "test",
							Value: ptr.To("true"),
						},
					},
				},
			},
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)
				assert.Equal(t, rolloutsapi.Metric{
					Name:             "foo",
					SuccessCondition: "{{args.test}}",
				}, template.Spec.Metrics[0])
				assert.Equal(t, rolloutsapi.Argument{
					Name:  "test",
					Value: ptr.To("true"),
				}, template.Spec.Args[0])
			},
		},
		{
			name: "merge multiple metrics",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Metrics: []rolloutsapi.Metric{
						{
							Name:             "foo",
							SuccessCondition: "true",
						},
					},
					DryRun: []rolloutsapi.DryRun{
						{
							MetricName: "foo",
						},
					},
					MeasurementRetention: []rolloutsapi.MeasurementRetention{
						{
							MetricName: "foo",
							Limit:      int32(5),
						},
					},
				},
				{
					Metrics: []rolloutsapi.Metric{
						{
							Name:             "bar",
							SuccessCondition: "true",
						},
					},
					DryRun: []rolloutsapi.DryRun{
						{
							MetricName: "bar",
						},
					},
					MeasurementRetention: []rolloutsapi.MeasurementRetention{
						{
							MetricName: "bar",
							Limit:      int32(10),
						},
					},
				},
			},
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.NoError(t, err)

				require.Len(t, template.Spec.Metrics, 2)
				assert.Equal(t, rolloutsapi.Metric{
					Name:             "foo",
					SuccessCondition: "true",
				}, template.Spec.Metrics[0])
				assert.Equal(t, rolloutsapi.Metric{
					Name:             "bar",
					SuccessCondition: "true",
				}, template.Spec.Metrics[1])

				require.Len(t, template.Spec.DryRun, 2)
				assert.Equal(t, rolloutsapi.DryRun{
					MetricName: "foo",
				}, template.Spec.DryRun[0])
				assert.Equal(t, rolloutsapi.DryRun{
					MetricName: "bar",
				}, template.Spec.DryRun[1])

				require.Len(t, template.Spec.MeasurementRetention, 2)
				assert.Equal(t, rolloutsapi.MeasurementRetention{
					MetricName: "foo",
					Limit:      int32(5),
				}, template.Spec.MeasurementRetention[0])
				assert.Equal(t, rolloutsapi.MeasurementRetention{
					MetricName: "bar",
					Limit:      int32(10),
				}, template.Spec.MeasurementRetention[1])
			},
		},
		{
			name: "merge fail with metric name collision",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Metrics: []rolloutsapi.Metric{
						{
							Name:             "foo",
							SuccessCondition: "true",
						},
					},
				},
				{
					Metrics: []rolloutsapi.Metric{
						{
							Name:             "foo",
							SuccessCondition: "false",
						},
					},
				},
			},
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.ErrorContains(t, err, "duplicate metric name")
				require.Nil(t, template)
			},
		},
		{
			name: "merge fail with dry-run name collision",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Metrics: []rolloutsapi.Metric{
						{
							Name:             "foo",
							SuccessCondition: "true",
						},
					},
					DryRun: []rolloutsapi.DryRun{
						{
							MetricName: "metric1",
						},
					},
				},
				{
					Metrics: []rolloutsapi.Metric{
						{
							Name:             "bar",
							SuccessCondition: "true",
						},
					},
					DryRun: []rolloutsapi.DryRun{
						{
							MetricName: "metric1",
						},
					},
				},
			},
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.ErrorContains(t, err, "duplicate dry-run metric name")
				require.Nil(t, template)
			},
		},
		{
			name: "merge fail with measurement retention name collision",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					MeasurementRetention: []rolloutsapi.MeasurementRetention{
						{
							MetricName: "metric1",
							Limit:      int32(0),
						},
					},
				},
				{
					MeasurementRetention: []rolloutsapi.MeasurementRetention{
						{
							MetricName: "metric1",
							Limit:      int32(0),
						},
					},
				},
			},
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.ErrorContains(t, err, "duplicate measurement retention metric name")
				require.Nil(t, template)
			},
		},
		{
			name: "merge fail with argument error",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("value1"),
						},
					},
				},
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("value2"),
						},
					},
				},
			},
			assertions: func(t *testing.T, template *rolloutsapi.AnalysisTemplate, err error) {
				require.ErrorContains(t, err, "flatten arguments")
				require.ErrorContains(t, err, "conflicting values for argument")
				require.Nil(t, template)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := flattenTemplates(tt.templateSpecs)
			tt.assertions(t, result, err)
		})
	}
}

func Test_mergeArgs(t *testing.T) {
	tests := []struct {
		name         string
		incomingArgs []rolloutsapi.Argument
		templateArgs []rolloutsapi.Argument
		assertions   func(*testing.T, []rolloutsapi.Argument, error)
	}{
		{
			name:         "nil lists",
			incomingArgs: nil,
			templateArgs: nil,
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Empty(t, args)
			},
		},
		{
			name:         "empty lists",
			incomingArgs: []rolloutsapi.Argument{},
			templateArgs: []rolloutsapi.Argument{},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Empty(t, args)
			},
		},
		{
			name:         "use template defaults",
			incomingArgs: nil,
			templateArgs: []rolloutsapi.Argument{
				{
					Name:  "foo",
					Value: ptr.To("bar"),
				},
				{
					Name: "secret",
					ValueFrom: &rolloutsapi.ValueFrom{
						SecretKeyRef: &rolloutsapi.SecretKeyRef{
							Name: "secret-name",
							Key:  "secret-key",
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 2)

				assert.Equal(t, rolloutsapi.Argument{
					Name:  "foo",
					Value: ptr.To("bar"),
				}, args[0])
				assert.Equal(t, rolloutsapi.Argument{
					Name: "secret",
					ValueFrom: &rolloutsapi.ValueFrom{
						SecretKeyRef: &rolloutsapi.SecretKeyRef{
							Name: "secret-name",
							Key:  "secret-key",
						},
					},
				}, args[1])
			},
		},
		{
			name: "incoming args override template defaults",
			incomingArgs: []rolloutsapi.Argument{
				{
					Name:  "foo",
					Value: ptr.To("override"),
				},
			},
			templateArgs: []rolloutsapi.Argument{
				{
					Name:  "foo",
					Value: ptr.To("default"),
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 1)

				assert.Equal(t, rolloutsapi.Argument{
					Name:  "foo",
					Value: ptr.To("override"),
				}, args[0])
			},
		},
		{
			name: "unresolved argument error",
			incomingArgs: []rolloutsapi.Argument{
				{
					Name: "foo",
				},
			},
			templateArgs: []rolloutsapi.Argument{
				{
					Name: "foo",
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.ErrorContains(t, err, "unresolved argument")
				require.Nil(t, args)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mergeArgs(tt.incomingArgs, tt.templateArgs)
			tt.assertions(t, result, err)
		})
	}
}

func Test_flattenArgs(t *testing.T) {
	tests := []struct {
		name          string
		templateSpecs []*rolloutsapi.AnalysisTemplateSpec
		assertions    func(*testing.T, []rolloutsapi.Argument, error)
	}{
		{
			name: "merge multiple args",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("true"),
						},
					},
				},
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "bar",
							Value: ptr.To("false"),
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 2)
				assert.Equal(t, rolloutsapi.Argument{
					Name:  "foo",
					Value: ptr.To("true"),
				}, args[0])
				assert.Equal(t, rolloutsapi.Argument{
					Name:  "bar",
					Value: ptr.To("false"),
				}, args[1])
			},
		},
		{
			name: "merge args with same name but only one has value",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("value"),
						},
					},
				},
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: nil,
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 1)
				assert.Equal(t, rolloutsapi.Argument{
					Name:  "foo",
					Value: ptr.To("value"),
				}, args[0])
			},
		},
		{
			name: "error when merging args with same name but different values",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("true"),
						},
					},
				},
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("false"),
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.ErrorContains(t, err, "conflicting values for argument")
				require.Nil(t, args)
			},
		},
		{
			name: "nil args in templateSpecs",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: nil,
				},
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("value"),
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 1)
				assert.Equal(t, rolloutsapi.Argument{
					Name:  "foo",
					Value: ptr.To("value"),
				}, args[0])
			},
		},
		{
			name: "empty args slice in templateSpecs",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: []rolloutsapi.Argument{},
				},
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("value"),
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 1)
				assert.Equal(t, rolloutsapi.Argument{
					Name:  "foo",
					Value: ptr.To("value"),
				}, args[0])
			},
		},
		{
			name: "handle argument with both value and valueFrom",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: ptr.To("value"),
							ValueFrom: &rolloutsapi.ValueFrom{
								SecretKeyRef: &rolloutsapi.SecretKeyRef{
									Name: "secret1",
									Key:  "key1",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 1)
				assert.Equal(t, rolloutsapi.Argument{
					Name:  "foo",
					Value: ptr.To("value"),
					ValueFrom: &rolloutsapi.ValueFrom{
						SecretKeyRef: &rolloutsapi.SecretKeyRef{
							Name: "secret1",
							Key:  "key1",
						},
					},
				}, args[0])
			},
		},
		{
			name: "merge args with ValueFrom",
			templateSpecs: []*rolloutsapi.AnalysisTemplateSpec{
				{
					Args: []rolloutsapi.Argument{
						{
							Name: "foo",
							ValueFrom: &rolloutsapi.ValueFrom{
								SecretKeyRef: &rolloutsapi.SecretKeyRef{
									Name: "secret1",
									Key:  "key1",
								},
							},
						},
					},
				},
				{
					Args: []rolloutsapi.Argument{
						{
							Name:  "foo",
							Value: nil,
						},
					},
				},
			},
			assertions: func(t *testing.T, args []rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Len(t, args, 1)
				assert.Equal(t, rolloutsapi.Argument{
					Name: "foo",
					ValueFrom: &rolloutsapi.ValueFrom{
						SecretKeyRef: &rolloutsapi.SecretKeyRef{
							Name: "secret1",
							Key:  "key1",
						},
					},
				}, args[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := flattenArgs(tt.templateSpecs)
			tt.assertions(t, result, err)
		})
	}
}

func Test_validateAndUpdateArg(t *testing.T) {
	tests := []struct {
		name       string
		existing   rolloutsapi.Argument
		new        rolloutsapi.Argument
		assertions func(*testing.T, rolloutsapi.Argument, error)
	}{
		{
			name: "existing has no value, new has value",
			existing: rolloutsapi.Argument{
				Name:  "foo",
				Value: nil,
			},
			new: rolloutsapi.Argument{
				Name:  "foo",
				Value: ptr.To("value"),
			},
			assertions: func(t *testing.T, result rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Equal(t, "value", *result.Value)
			},
		},
		{
			name: "both have same value",
			existing: rolloutsapi.Argument{
				Name:  "foo",
				Value: ptr.To("value"),
			},
			new: rolloutsapi.Argument{
				Name:  "foo",
				Value: ptr.To("value"),
			},
			assertions: func(t *testing.T, result rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Equal(t, "value", *result.Value)
			},
		},
		{
			name: "both have different values",
			existing: rolloutsapi.Argument{
				Name:  "foo",
				Value: ptr.To("value1"),
			},
			new: rolloutsapi.Argument{
				Name:  "foo",
				Value: ptr.To("value2"),
			},
			assertions: func(t *testing.T, result rolloutsapi.Argument, err error) {
				require.ErrorContains(t, err, "conflicting values for argument")
				require.Equal(t, "value1", *result.Value)
			},
		},
		{
			name: "existing has valueFrom, new has different valueFrom",
			existing: rolloutsapi.Argument{
				Name: "foo",
				ValueFrom: &rolloutsapi.ValueFrom{
					SecretKeyRef: &rolloutsapi.SecretKeyRef{
						Name: "secret1",
						Key:  "key1",
					},
				},
			},
			new: rolloutsapi.Argument{
				Name: "foo",
				ValueFrom: &rolloutsapi.ValueFrom{
					SecretKeyRef: &rolloutsapi.SecretKeyRef{
						Name: "secret2",
						Key:  "key2",
					},
				},
			},
			assertions: func(t *testing.T, result rolloutsapi.Argument, err error) {
				require.NoError(t, err)
				require.Equal(t, "secret1", result.ValueFrom.SecretKeyRef.Name)
				require.Equal(t, "key1", result.ValueFrom.SecretKeyRef.Key)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAndUpdateArg(&tt.existing, tt.new)
			tt.assertions(t, tt.existing, err)
		})
	}
}
