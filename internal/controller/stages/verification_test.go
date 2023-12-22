package stages

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

func TestStarVerification(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(*kargoapi.VerificationInfo, error)
	}{
		{
			name: "error listing AnalysisRuns",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(vi *kargoapi.VerificationInfo, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error listing AnalysisRuns for Stage")
			},
		},
		{
			name: "Analysis run already exists",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				listAnalysisRunsFn: func(
					_ context.Context,
					objList client.ObjectList,
					_ ...client.ListOption,
				) error {
					analysisRuns, ok := objList.(*rollouts.AnalysisRunList)
					require.True(t, ok)
					analysisRuns.Items = []rollouts.AnalysisRun{{}}
					return nil
				},
			},
			assertions: func(_ *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "error getting AnalysisTemplate",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *kargoapi.VerificationInfo, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error getting AnalysisTemplate")
			},
		},
		{
			name: "AnalysisTemplate not found",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return nil, nil
				},
			},
			assertions: func(_ *kargoapi.VerificationInfo, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "AnalysisTemplate")
				require.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "error building AnalysisRun",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				buildAnalysisRunFn: func(
					*kargoapi.Stage,
					[]*rollouts.AnalysisTemplate,
				) (*rollouts.AnalysisRun, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(_ *kargoapi.VerificationInfo, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error building AnalysisRun for Stage")
			},
		},
		{
			name: "error creating AnalysisRun",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				buildAnalysisRunFn: func(
					*kargoapi.Stage,
					[]*rollouts.AnalysisTemplate,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{}, nil
				},
				createAnalysisRunFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(_ *kargoapi.VerificationInfo, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error creating AnalysisRun")
			},
		},
		{
			name: "success",
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					Verification: &kargoapi.Verification{
						AnalysisTemplates: []kargoapi.AnalysisTemplateReference{{}},
					},
				},
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						ID: "fake-id",
					},
				},
			},
			reconciler: &reconciler{
				listAnalysisRunsFn: func(
					context.Context,
					client.ObjectList,
					...client.ListOption,
				) error {
					return nil
				},
				getAnalysisTemplateFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisTemplate, error) {
					return &rollouts.AnalysisTemplate{}, nil
				},
				buildAnalysisRunFn: func(
					*kargoapi.Stage,
					[]*rollouts.AnalysisTemplate,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-run",
							Namespace: "fake-namespace",
						},
					}, nil
				},
				createAnalysisRunFn: func(
					context.Context,
					client.Object,
					...client.CreateOption,
				) error {
					return nil
				},
			},
			assertions: func(ver *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						AnalysisRun: kargoapi.AnalysisRunReference{
							Name:      "fake-run",
							Namespace: "fake-namespace",
						},
					},
					ver,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.startVerification(
					context.Background(),
					testCase.stage,
				),
			)
		})
	}
}

func TestGetVerificationInfo(t *testing.T) {
	testCases := []struct {
		name       string
		stage      *kargoapi.Stage
		reconciler *reconciler
		assertions func(*kargoapi.VerificationInfo, error)
	}{
		{
			name: "error getting AnalysisRun",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						VerificationInfo: &kargoapi.VerificationInfo{
							AnalysisRun: kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(vi *kargoapi.VerificationInfo, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error getting AnalysisRun")
			},
		},
		{
			name: "AnalysisRun not found",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						VerificationInfo: &kargoapi.VerificationInfo{
							AnalysisRun: kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return nil, nil
				},
			},
			assertions: func(vi *kargoapi.VerificationInfo, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "AnalysisRun")
				require.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "success",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					CurrentFreight: &kargoapi.SimpleFreight{
						VerificationInfo: &kargoapi.VerificationInfo{
							AnalysisRun: kargoapi.AnalysisRunReference{
								Name:      "fake-run",
								Namespace: "fake-namespace",
							},
						},
					},
				},
			},
			reconciler: &reconciler{
				getAnalysisRunFn: func(
					context.Context,
					client.Client,
					types.NamespacedName,
				) (*rollouts.AnalysisRun, error) {
					return &rollouts.AnalysisRun{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fake-run",
							Namespace: "fake-namespace",
						},
						Status: rollouts.AnalysisRunStatus{
							Phase: rollouts.AnalysisPhaseSuccessful,
						},
					}, nil
				},
			},
			assertions: func(ver *kargoapi.VerificationInfo, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&kargoapi.VerificationInfo{
						AnalysisRun: kargoapi.AnalysisRunReference{
							Name:      "fake-run",
							Namespace: "fake-namespace",
							Phase:     string(rollouts.AnalysisPhaseSuccessful),
						},
					},
					ver,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				testCase.reconciler.getVerificationInfo(
					context.Background(),
					testCase.stage,
				),
			)
		})
	}
}

func TestFlattenTemplates(t *testing.T) {
	metric := func(name, successCondition string) rollouts.Metric {
		return rollouts.Metric{
			Name:             name,
			SuccessCondition: successCondition,
		}
	}
	arg := func(name string, value *string) rollouts.Argument {
		return rollouts.Argument{
			Name:  name,
			Value: value,
		}
	}
	t.Run("Handle empty list", func(t *testing.T) {
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{})
		require.Nil(t, err)
		require.Len(t, template.Spec.Metrics, 0)
		require.Len(t, template.Spec.Args, 0)

	})
	t.Run("No changes on single template", func(t *testing.T) {
		orig := &rollouts.AnalysisTemplate{
			Spec: rollouts.AnalysisTemplateSpec{
				Metrics: []rollouts.Metric{metric("foo", "{{args.test}}")},
				Args:    []rollouts.Argument{arg("test", ptr.To("true"))},
			},
		}
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{orig})
		require.Nil(t, err)
		require.Equal(t, orig.Spec, template.Spec)
	})
	t.Run("Merge multiple metrics successfully", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					DryRun: []rollouts.DryRun{{
						MetricName: "foo",
					}},
					MeasurementRetention: []rollouts.MeasurementRetention{{
						MetricName: "foo",
					}},
					Args: nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					DryRun: []rollouts.DryRun{{
						MetricName: "bar",
					}},
					MeasurementRetention: []rollouts.MeasurementRetention{{
						MetricName: "bar",
					}},
					Args: nil,
				},
			},
		})
		require.Nil(t, err)
		require.Nil(t, template.Spec.Args)
		require.Len(t, template.Spec.Metrics, 2)
		require.Equal(t, fooMetric, template.Spec.Metrics[0])
		require.Equal(t, barMetric, template.Spec.Metrics[1])
	})
	t.Run("Merge analysis templates successfully", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "foo",
						},
					},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			},
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "bar",
						},
					},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "bar",
						},
					},
					Args: nil,
				},
			},
		})
		require.Nil(t, err)
		require.Nil(t, template.Spec.Args)
		require.Len(t, template.Spec.Metrics, 2)
		require.Equal(t, fooMetric, template.Spec.Metrics[0])
		require.Equal(t, barMetric, template.Spec.Metrics[1])
	})
	t.Run("Merge fail with name collision", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					Args:    nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					Args:    nil,
				},
			},
		})
		require.Nil(t, template)
		require.Equal(t, err, fmt.Errorf("two metrics have the same name 'foo'"))
	})
	t.Run("Merge fail with dry-run name collision", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					DryRun: []rollouts.DryRun{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			},
		})
		require.Nil(t, template)
		require.Equal(t, err, fmt.Errorf("two Dry-Run metric rules have the same name 'foo'"))
	})
	t.Run("Merge fail with measurement retention metrics name collision", func(t *testing.T) {
		fooMetric := metric("foo", "true")
		barMetric := metric("bar", "true")
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{fooMetric},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: []rollouts.Metric{barMetric},
					MeasurementRetention: []rollouts.MeasurementRetention{
						{
							MetricName: "foo",
						},
					},
					Args: nil,
				},
			},
		})
		require.Nil(t, template)
		require.Equal(t, err, fmt.Errorf("two Measurement Retention metric rules have the same name 'foo'"))
	})
	t.Run("Merge multiple args successfully", func(t *testing.T) {
		fooArgs := arg("foo", ptr.To("true"))
		barArgs := arg("bar", ptr.To("true"))
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgs},
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{barArgs},
				},
			},
		})
		require.Nil(t, err)
		require.Len(t, template.Spec.Args, 2)
		require.Equal(t, fooArgs, template.Spec.Args[0])
		require.Equal(t, barArgs, template.Spec.Args[1])
	})
	t.Run(" Merge args with same name but only one has value", func(t *testing.T) {
		fooArgsValue := arg("foo", ptr.To("true"))
		fooArgsNoValue := arg("foo", nil)
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgsValue},
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgsNoValue},
				},
			},
		})
		require.Nil(t, err)
		require.Len(t, template.Spec.Args, 1)
		require.Contains(t, template.Spec.Args, fooArgsValue)
	})
	t.Run("Error: merge args with same name and both have values", func(t *testing.T) {
		fooArgs := arg("foo", ptr.To("true"))
		fooArgsWithDiffValue := arg("foo", ptr.To("false"))
		template, err := flattenTemplates([]*rollouts.AnalysisTemplate{
			{
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgs},
				},
			}, {
				Spec: rollouts.AnalysisTemplateSpec{
					Metrics: nil,
					Args:    []rollouts.Argument{fooArgsWithDiffValue},
				},
			},
		})
		require.Equal(t, fmt.Errorf("Argument `foo` specified multiple times with different values: 'true', 'false'"), err)
		require.Nil(t, template)
	})
}

func TestMergeArgs(t *testing.T) {
	{
		// nil list
		args, err := mergeArgs(nil, nil)
		require.NoError(t, err)
		require.Nil(t, args)
	}
	{
		// empty list
		args, err := mergeArgs(nil, []rollouts.Argument{})
		require.NoError(t, err)
		require.Equal(t, []rollouts.Argument{}, args)
	}
	{
		// use defaults
		args, err := mergeArgs(
			nil, []rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("bar"),
				},
				{
					Name: "my-secret",
					ValueFrom: &rollouts.ValueFrom{
						SecretKeyRef: &rollouts.SecretKeyRef{
							Name: "name",
							Key:  "key",
						},
					},
				},
			})
		require.NoError(t, err)
		require.Len(t, args, 2)
		require.Equal(t, "foo", args[0].Name)
		require.Equal(t, "bar", *args[0].Value)
		require.Equal(t, "my-secret", args[1].Name)
		require.NotNil(t, args[1].ValueFrom)
	}
	{
		// overwrite defaults
		args, err := mergeArgs(
			[]rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("overwrite"),
				},
			}, []rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("bar"),
				},
			})
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, "foo", args[0].Name)
		require.Equal(t, "overwrite", *args[0].Value)
	}
	{
		// not resolved
		args, err := mergeArgs(
			[]rollouts.Argument{
				{
					Name: "foo",
				},
			}, []rollouts.Argument{
				{
					Name: "foo",
				},
			})
		require.EqualError(t, err, "args.foo was not resolved")
		require.Nil(t, args)
	}
	{
		// extra arg
		args, err := mergeArgs(
			[]rollouts.Argument{
				{
					Name:  "foo",
					Value: ptr.To("my-value"),
				},
				{
					Name:  "extra-arg",
					Value: ptr.To("extra-value"),
				},
			}, []rollouts.Argument{
				{
					Name: "foo",
				},
			})
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, "foo", args[0].Name)
		require.Equal(t, "my-value", *args[0].Value)
	}
}
