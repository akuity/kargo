package api

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestNewMinimalPromotion(t *testing.T) {
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stage",
			Namespace: "test-project",
		},
	}
	promo := NewMinimalPromotion(stage, "test-freight")
	require.NotNil(t, promo)
	require.Equal(t, "test-project", promo.Namespace)
	require.Equal(t, "promo-", promo.GenerateName)
	require.Empty(t, promo.Name)
	require.Equal(t, "test-stage", promo.Spec.Stage)
	require.Equal(t, "test-freight", promo.Spec.Freight)
	require.Empty(t, promo.Spec.Steps)
	require.Empty(t, promo.Spec.Vars)
}

func TestGeneratePromotionName(t *testing.T) {
	tests := []struct {
		name       string
		stageName  string
		freight    string
		assertions func(t *testing.T, result string)
	}{
		{
			name:      "standard input lengths",
			stageName: "dev",
			freight:   "abc123def456",
			assertions: func(t *testing.T, result string) {
				components := strings.Split(result, ".")
				require.Len(t, components, 3)
				require.Equal(t, "dev", components[0])
				require.Len(t, components[1], ulid.EncodedSize)
				require.Equal(t, "abc123d", components[2])
			},
		},
		{
			name:      "short freight",
			stageName: "prod",
			freight:   "abc",
			assertions: func(t *testing.T, result string) {
				components := strings.Split(result, ".")
				require.Len(t, components, 3)
				require.Equal(t, "prod", components[0])
				require.Len(t, components[1], ulid.EncodedSize)
				require.Equal(t, "abc", components[2])
			},
		},
		{
			name: "long stage name gets truncated",
			// nolint:lll
			stageName: "this-is-a-very-long-stage-name-that-exceeds-the-maximum-allowed-length-for-kubernetes-resources-and-should-be-truncated-to-fit-within-the-limits-set-by-the-api-server-which-is-253-characters-including-the-generated-suffix",
			freight:   "abc123def456",
			assertions: func(t *testing.T, result string) {
				require.Len(t, result, 253) // Kubernetes resource name limit
				require.Equal(
					t,
					maxStageNamePrefixForPromotionName,
					len(result[:strings.Index(result, ".")]),
				)
			},
		},
		{
			name:      "long freight gets truncated",
			stageName: "stage",
			freight:   "this-is-a-very-long-freight-hash-that-should-be-truncated",
			assertions: func(t *testing.T, result string) {
				shortHash := result[strings.LastIndex(result, ".")+1:]
				require.Len(t, shortHash, promotionShortHashLength)
			},
		},
		{
			name:      "all lowercase conversion",
			stageName: "DEV-STAGE",
			freight:   "ABC123DEF456",
			assertions: func(t *testing.T, result string) {
				require.Equal(t, "dev-stage", result[:len("dev-stage")])
				require.Equal(t, "abc123d", result[len(result)-7:])
			},
		},
		{
			name:      "empty inputs",
			stageName: "",
			freight:   "",
			assertions: func(t *testing.T, result string) {
				require.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(t, GeneratePromotionName(tt.stageName, tt.freight))
		})
	}
}

func TestGetPromotion(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	testCases := []struct {
		name       string
		client     client.Client
		assertions func(*testing.T, *kargoapi.Promotion, error)
	}{
		{
			name:   "not found",
			client: fake.NewClientBuilder().WithScheme(scheme).Build(),
			assertions: func(t *testing.T, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Nil(t, promo)
			},
		},

		{
			name: "found",
			client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-promotion",
						Namespace: "fake-namespace",
					},
				},
			).Build(),
			assertions: func(t *testing.T, promo *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-promotion", promo.Name)
				require.Equal(t, "fake-namespace", promo.Namespace)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			promo, err := GetPromotion(
				t.Context(),
				testCase.client,
				types.NamespacedName{
					Namespace: "fake-namespace",
					Name:      "fake-promotion",
				},
			)
			testCase.assertions(t, promo, err)
		})
	}
}

func TestAbortPromotion(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.SchemeBuilder.AddToScheme(scheme))

	t.Run("not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		err := AbortPromotion(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		}, kargoapi.AbortActionTerminate)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("already in a terminal phase", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-promotion",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
		).Build()

		err := AbortPromotion(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		}, kargoapi.AbortActionTerminate)
		require.NoError(t, err)

		promotion, err := GetPromotion(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		})
		require.NoError(t, err)
		_, ok := promotion.Annotations[kargoapi.AnnotationKeyAbort]
		require.False(t, ok)
	})

	t.Run("success", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
					Name:      "fake-promotion",
				},
			},
		).Build()

		err := AbortPromotion(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		}, kargoapi.AbortActionTerminate)
		require.NoError(t, err)

		stage, err := GetPromotion(t.Context(), c, types.NamespacedName{
			Namespace: "fake-namespace",
			Name:      "fake-promotion",
		})
		require.NoError(t, err)
		require.Equal(t, (&kargoapi.AbortPromotionRequest{
			Action: kargoapi.AbortActionTerminate,
		}).String(), stage.Annotations[kargoapi.AnnotationKeyAbort])
	})
}

func Test_ComparePromotionByPhaseAndCreationTime(t *testing.T) {
	now := time.Date(2024, time.April, 10, 0, 0, 0, 0, time.UTC)
	ulidEarlier := ulid.MustNew(ulid.Timestamp(now.Add(-time.Hour)), nil)
	ulidLater := ulid.MustNew(ulid.Timestamp(now.Add(time.Hour)), nil)

	tests := []struct {
		name     string
		a        kargoapi.Promotion
		b        kargoapi.Promotion
		expected int
	}{
		{
			name: "Running before Terminated",
			a: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseRunning,
				},
			},
			b: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			expected: -1,
		},
		{
			name: "Pending before Terminated",
			a: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			b: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			expected: -1,
		},
		{
			name: "Pending after Running",
			a: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			b: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseRunning,
				},
			},
			expected: 1,
		},
		{
			name: "Terminated after Running",
			a: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseFailed,
				},
			},
			b: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseRunning,
				},
			},
			expected: 1,
		},
		{
			name: "Earlier ULID first if both Running",
			a: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidEarlier.String(),
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseRunning,
				},
			},
			b: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidLater.String(),
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseRunning,
				},
			},
			expected: -1,
		},
		{
			name: "Later ULID first if both Terminated",
			a: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidLater.String(),
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseErrored,
				},
			},
			b: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "promotion." + ulidEarlier.String(),
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			expected: -1,
		},
		{
			name: "Equal promotions",
			a: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "promotion-a",
					CreationTimestamp: metav1.Time{Time: now},
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			b: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "promotion-a",
					CreationTimestamp: metav1.Time{Time: now},
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			expected: 0,
		},
		{
			name: "Nil creation timestamps",
			a: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			b: kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComparePromotionByPhaseAndCreationTime(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestComparePromotionPhase(t *testing.T) {
	tests := []struct {
		name     string
		a        kargoapi.PromotionPhase
		b        kargoapi.PromotionPhase
		expected int
	}{
		{
			name:     "Running before Terminated",
			a:        kargoapi.PromotionPhaseRunning,
			b:        kargoapi.PromotionPhaseSucceeded,
			expected: -1,
		},
		{
			name:     "Terminated after Running",
			a:        kargoapi.PromotionPhaseFailed,
			b:        kargoapi.PromotionPhaseRunning,
			expected: 1,
		},
		{
			name:     "Running before other phase",
			a:        kargoapi.PromotionPhaseRunning,
			b:        kargoapi.PromotionPhasePending,
			expected: -1,
		},
		{
			name:     "Other phase after Running",
			a:        "",
			b:        kargoapi.PromotionPhaseRunning,
			expected: 1,
		},
		{
			name:     "Pending before Terminated",
			a:        kargoapi.PromotionPhasePending,
			b:        kargoapi.PromotionPhaseErrored,
			expected: -1,
		},
		{
			name:     "Pending after Running",
			a:        kargoapi.PromotionPhasePending,
			b:        kargoapi.PromotionPhaseRunning,
			expected: 1,
		},
		{
			name:     "Equal Running phases",
			a:        kargoapi.PromotionPhaseRunning,
			b:        kargoapi.PromotionPhaseRunning,
			expected: 0,
		},
		{
			name: "Equal Terminated phases",
			a:    kargoapi.PromotionPhaseSucceeded,
			b:    kargoapi.PromotionPhaseFailed,
		},
		{
			name:     "Equal other phases",
			a:        kargoapi.PromotionPhasePending,
			b:        kargoapi.PromotionPhasePending,
			expected: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, ComparePromotionPhase(tt.a, tt.b))
		})
	}
}

func TestIsCurrentStepRunningRunning(t *testing.T) {
	tests := []struct {
		name           string
		promotion      *kargoapi.Promotion
		expectedResult bool
	}{
		{
			name: "promotion is running",
			promotion: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{{
						Status: kargoapi.PromotionStepStatusRunning,
					}},
					CurrentStep: 0,
					Phase:       kargoapi.PromotionPhaseRunning,
				},
			},
			expectedResult: true,
		},
		{
			name: "promotion is not running",
			promotion: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					StepExecutionMetadata: []kargoapi.StepExecutionMetadata{{
						Status: kargoapi.PromotionStepStatusSucceeded,
					}},
					CurrentStep: 0,
					Phase:       kargoapi.PromotionPhasePending,
				},
			},
			expectedResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expectedResult, IsCurrentStepRunning(tt.promotion))
		})
	}
}

func TestInflateSteps(t *testing.T) {
	s := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(s))

	tests := []struct {
		name       string
		promo      kargoapi.Promotion
		objects    []client.Object
		assertions func(*testing.T, []kargoapi.PromotionStep, error)
	}{
		{
			name: "task not found returns error",
			promo: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-project",
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{
							Task: &kargoapi.PromotionTaskReference{
								Name: "missing-task",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, _ []kargoapi.PromotionStep, err error) {
				require.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "single direct step",
			promo: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-project",
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{
							As:   "direct-step",
							Uses: "fake-step",
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.NoError(t, err)
				require.Len(t, steps, 1)
				require.Equal(t, "direct-step", steps[0].As)
				require.Equal(t, "fake-step", steps[0].Uses)
			},
		},
		{
			name: "mix of direct and task steps",
			promo: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-project",
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{
							As:   "direct-step",
							Uses: "fake-step",
						},
						{
							As: "task-step",
							Task: &kargoapi.PromotionTaskReference{
								Name: "test-task",
							},
							Vars: []kargoapi.ExpressionVariable{
								{Name: "input1", Value: "value1"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1"},
						},
						Steps: []kargoapi.PromotionStep{
							{
								As:   "sub-step",
								Uses: "other-fake-step",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.NoError(t, err)
				require.Len(t, steps, 2)
				require.Equal(t, "direct-step", steps[0].As)
				require.Equal(t, "fake-step", steps[0].Uses)
				require.Equal(t, "task-step::sub-step", steps[1].As)
				require.Equal(t, "other-fake-step", steps[1].Uses)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "value1"},
				}, steps[1].Vars)
			},
		},
		{
			name: "multiple task steps",
			promo: kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-promotion",
					Namespace: "test-project",
				},
				Spec: kargoapi.PromotionSpec{
					Steps: []kargoapi.PromotionStep{
						{
							As: "task1",
							Task: &kargoapi.PromotionTaskReference{
								Name: "test-task-1",
							},
							Vars: []kargoapi.ExpressionVariable{
								{Name: "input1", Value: "value1"},
							},
						},
						{
							As: "task2",
							Task: &kargoapi.PromotionTaskReference{
								Kind: "ClusterPromotionTask",
								Name: "test-task-2",
							},
							Vars: []kargoapi.ExpressionVariable{
								{Name: "input2", Value: "value2"},
							},
						},
					},
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task-1",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1"},
						},
						Steps: []kargoapi.PromotionStep{
							{
								As:   "step1",
								Uses: "fake-step",
							},
						},
					},
				},
				&kargoapi.ClusterPromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-task-2",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input2"},
						},
						Steps: []kargoapi.PromotionStep{
							{
								As:   "step2",
								Uses: "other-fake-step",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.NoError(t, err)
				require.Len(t, steps, 2)
				require.Equal(t, "task1::step1", steps[0].As)
				require.Equal(t, "fake-step", steps[0].Uses)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "value1"},
				}, steps[0].Vars)
				require.Equal(t, "task2::step2", steps[1].As)
				require.Equal(t, "other-fake-step", steps[1].Uses)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input2", Value: "value2"},
				}, steps[1].Vars)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			p := tt.promo.DeepCopy()
			err := InflateSteps(t.Context(), c, p)
			tt.assertions(t, p.Spec.Steps, err)
		})
	}
}

func Test_inflateTaskSteps(t *testing.T) {
	s := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(s))

	tests := []struct {
		name       string
		project    string
		taskAlias  string
		promoVars  []kargoapi.ExpressionVariable
		taskStep   kargoapi.PromotionStep
		objects    []client.Object
		assertions func(*testing.T, []kargoapi.PromotionStep, error)
	}{
		{
			name:      "task not found",
			project:   "test-project",
			taskAlias: "deploy",
			taskStep: kargoapi.PromotionStep{
				Task: &kargoapi.PromotionTaskReference{
					Name: "missing-task",
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.True(t, apierrors.IsNotFound(err))
				require.Nil(t, steps)
			},
		},
		{
			name:    "invalid config for task variables",
			project: "test-project",
			taskStep: kargoapi.PromotionStep{
				Task: &kargoapi.PromotionTaskReference{
					Name: "test-task",
				},
				Vars: nil,
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.ErrorContains(t, err, "missing value for variable")
				require.Nil(t, steps)
			},
		},
		{
			name:      "successful task step inflation",
			project:   "test-project",
			taskAlias: "task-1",
			promoVars: []kargoapi.ExpressionVariable{
				{Name: "input3", Value: "value1"},
			},
			taskStep: kargoapi.PromotionStep{
				Task: &kargoapi.PromotionTaskReference{
					Name: "test-task",
				},
				Vars: []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "value1"},
					{Name: "input2", Value: "value2"},
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1"},
							{Name: "input2", Value: "default2"},
							{Name: "input3"},
						},
						Steps: []kargoapi.PromotionStep{
							{
								As:   "step1",
								Uses: "fake-step",
							},
							{
								As:   "step2",
								Uses: "other-fake-step",
								Vars: []kargoapi.ExpressionVariable{
									{Name: "input4", Value: "value4"},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.NoError(t, err)
				require.Len(t, steps, 2)
				require.Equal(t, "task-1::step1", steps[0].As)
				require.Equal(t, "fake-step", steps[0].Uses)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input2", Value: "default2"},
					{Name: "input1", Value: "value1"},
					{Name: "input2", Value: "value2"},
				}, steps[0].Vars)
				require.Equal(t, "task-1::step2", steps[1].As)
				require.Equal(t, "other-fake-step", steps[1].Uses)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input2", Value: "default2"},
					{Name: "input1", Value: "value1"},
					{Name: "input2", Value: "value2"},
					{Name: "input4", Value: "value4"},
				}, steps[1].Vars)
			},
		},
		{
			name:      "task steps with default alias",
			project:   "test-project",
			taskAlias: "custom-alias",
			taskStep: kargoapi.PromotionStep{
				Task: &kargoapi.PromotionTaskReference{
					Name: "test-task",
				},
				Vars: []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "value1"},
				},
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1"},
						},
						Steps: []kargoapi.PromotionStep{
							{
								Uses: "fake-step",
							},
							{
								Uses: "other-fake-step",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.NoError(t, err)
				require.Len(t, steps, 2)
				require.Equal(t, "custom-alias::step-1", steps[0].As)
				require.Equal(t, "custom-alias::step-2", steps[1].As)
			},
		},
		{
			name:      "cluster task with steps",
			project:   "test-project",
			taskAlias: "task-0",
			taskStep: kargoapi.PromotionStep{
				Task: &kargoapi.PromotionTaskReference{
					Kind: "ClusterPromotionTask",
					Name: "test-cluster-task",
				},
				Vars: []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "value1"},
				},
			},
			objects: []client.Object{
				&kargoapi.ClusterPromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-task",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1"},
						},
						Steps: []kargoapi.PromotionStep{
							{
								As:   "custom-alias",
								Uses: "fake-step",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.NoError(t, err)
				require.Len(t, steps, 1)
				require.Equal(t, "task-0::custom-alias", steps[0].As)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()
			steps, err := inflateTaskSteps(
				t.Context(), c, tt.project, tt.taskAlias, tt.promoVars, tt.taskStep,
			)
			tt.assertions(t, steps, err)
		})
	}
}

func Test_getPromotionTaskSpec(t *testing.T) {
	s := k8sruntime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(s))

	tests := []struct {
		name        string
		project     string
		ref         *kargoapi.PromotionTaskReference
		objects     []client.Object
		interceptor interceptor.Funcs
		assertions  func(*testing.T, *kargoapi.PromotionTaskSpec, error)
	}{
		{
			name:    "nil reference returns error",
			project: "test-project",
			ref:     nil,
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.ErrorContains(t, err, "missing task reference")
				require.Nil(t, result)
			},
		},
		{
			name:    "unknown task kind returns error",
			project: "test-project",
			ref: &kargoapi.PromotionTaskReference{
				Kind: "UnknownKind",
				Name: "test-task",
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.ErrorContains(t, err, "unknown task reference kind")
				require.Nil(t, result)
			},
		},
		{
			name:    "PromotionTask not found returns error",
			project: "test-project",
			ref: &kargoapi.PromotionTaskReference{
				Kind: "PromotionTask",
				Name: "missing-task",
			},
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, result)
			},
		},
		{
			name:    "ClusterPromotionTask not found returns error",
			project: "test-project",
			ref: &kargoapi.PromotionTaskReference{
				Kind: "ClusterPromotionTask",
				Name: "missing-cluster-task",
			},
			interceptor: interceptor.Funcs{
				Get: func(
					context.Context,
					client.WithWatch,
					client.ObjectKey,
					client.Object,
					...client.GetOption,
				) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, result)
			},
		},
		{
			name:    "successfully retrieves PromotionTask",
			project: "test-project",
			ref: &kargoapi.PromotionTaskReference{
				Kind: "PromotionTask",
				Name: "test-task",
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1", Value: "value1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Vars, 1)
				require.Equal(t, "input1", result.Vars[0].Name)
				require.Equal(t, "value1", result.Vars[0].Value)
			},
		},
		{
			name:    "successfully retrieves ClusterPromotionTask",
			project: "test-project",
			ref: &kargoapi.PromotionTaskReference{
				Kind: "ClusterPromotionTask",
				Name: "test-cluster-task",
			},
			objects: []client.Object{
				&kargoapi.ClusterPromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-task",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1", Value: "value1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Vars, 1)
				require.Equal(t, "input1", result.Vars[0].Name)
				require.Equal(t, "value1", result.Vars[0].Value)
			},
		},
		{
			name:    "empty kind defaults to PromotionTask",
			project: "test-project",
			ref: &kargoapi.PromotionTaskReference{
				Kind: "",
				Name: "test-task",
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.ExpressionVariable{
							{Name: "input1", Value: "value1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Vars, 1)
				require.Equal(t, "input1", result.Vars[0].Name)
				require.Equal(t, "value1", result.Vars[0].Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				WithInterceptorFuncs(tt.interceptor).
				Build()
			result, err := getPromotionTaskSpec(t.Context(), c, tt.project, tt.ref)
			tt.assertions(t, result, err)
		})
	}
}

func Test_generatePromotionTaskStepAlias(t *testing.T) {
	tests := []struct {
		name      string
		taskAlias string
		stepAlias string
		expected  string
	}{
		{name: "standard aliases", taskAlias: "deploy", stepAlias: "apply", expected: "deploy::apply"},
		{name: "empty task alias", taskAlias: "", stepAlias: "apply", expected: "::apply"},
		{name: "empty step alias", taskAlias: "deploy", stepAlias: "", expected: "deploy::"},
		{name: "both aliases empty", taskAlias: "", stepAlias: "", expected: "::"},
		{
			name:      "aliases with special characters",
			taskAlias: "deploy-task",
			stepAlias: "apply_config",
			expected:  "deploy-task::apply_config",
		},
		{
			name:      "aliases containing separator",
			taskAlias: "deploy::task",
			stepAlias: "apply::config",
			expected:  "deploy::task::apply::config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, generatePromotionTaskStepAlias(tt.taskAlias, tt.stepAlias))
		})
	}
}

func Test_promotionTaskVarsToStepVars(t *testing.T) {
	tests := []struct {
		name       string
		taskVars   []kargoapi.ExpressionVariable
		promoVars  []kargoapi.ExpressionVariable
		stepVars   []kargoapi.ExpressionVariable
		assertions func(t *testing.T, result []kargoapi.ExpressionVariable, err error)
	}{
		{
			name:     "nil inputs returns nil map and no error",
			taskVars: nil,
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.ExpressionVariable, err error) {
				require.NoError(t, err)
				require.Nil(t, result)
			},
		},
		{
			name:     "empty inputs returns nil map and no error",
			taskVars: []kargoapi.ExpressionVariable{},
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.ExpressionVariable, err error) {
				require.NoError(t, err)
				require.Nil(t, result)
			},
		},
		{
			name: "missing required variable returns error",
			taskVars: []kargoapi.ExpressionVariable{
				{Name: "input1"},
			},
			stepVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: ""},
			},
			assertions: func(t *testing.T, result []kargoapi.ExpressionVariable, err error) {
				require.ErrorContains(t, err, "missing value for variable \"input1\"")
				require.Nil(t, result)
			},
		},
		{
			name: "default value used when config value not provided",
			taskVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: "default1"},
			},
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.ExpressionVariable, err error) {
				require.NoError(t, err)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "default1"},
				}, result)
			},
		},
		{
			name: "step value appends task default value",
			taskVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: "default1"},
			},
			stepVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: "override1"},
			},
			assertions: func(t *testing.T, result []kargoapi.ExpressionVariable, err error) {
				require.NoError(t, err)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "default1"},
					{Name: "input1", Value: "override1"},
				}, result)
			},
		},
		{
			name: "promotion variable overrides default value",
			taskVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: "default1"},
			},
			promoVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: "override1"},
			},
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.ExpressionVariable, err error) {
				require.NoError(t, err)
				require.Empty(t, result)
			},
		},
		{
			name: "multiple inputs processed correctly",
			taskVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: "default1"},
				{Name: "input2", Value: "default2"},
				{Name: "input3"},
			},
			stepVars: []kargoapi.ExpressionVariable{
				{Name: "input1", Value: "override1"},
				{Name: "input3", Value: "value3"},
			},
			assertions: func(t *testing.T, result []kargoapi.ExpressionVariable, err error) {
				require.NoError(t, err)
				require.ElementsMatch(t, []kargoapi.ExpressionVariable{
					{Name: "input1", Value: "default1"},
					{Name: "input2", Value: "default2"},
					{Name: "input1", Value: "override1"},
					{Name: "input3", Value: "value3"},
				}, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := promotionTaskVarsToStepVars(tt.taskVars, tt.promoVars, tt.stepVars)
			tt.assertions(t, result, err)
		})
	}
}
