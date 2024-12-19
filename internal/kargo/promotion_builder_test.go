package kargo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/user"
)

func TestPromotionBuilder_Build(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(s))

	tests := []struct {
		name       string
		stage      kargoapi.Stage
		freight    string
		userInfo   user.Info
		assertions func(*testing.T, *kargoapi.Promotion, error)
	}{
		{
			name: "empty stage name returns error",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-project",
				},
			},
			freight: "abc123",
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				assert.ErrorContains(t, err, "stage is required")
				assert.Nil(t, promotion)
			},
		},
		{
			name: "empty freight returns error",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "test-project",
				},
				Spec: kargoapi.StageSpec{
					PromotionTemplate: &kargoapi.PromotionTemplate{},
				},
			},
			freight: "",
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				assert.ErrorContains(t, err, "freight is required")
				assert.Nil(t, promotion)
			},
		},
		{
			name: "missing promotion template returns error",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "test-project",
				},
			},
			freight: "abc123",
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				assert.ErrorContains(t, err, "has no promotion template")
				assert.Nil(t, promotion)
			},
		},
		{
			name: "successful build with direct steps",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "test-project",
				},
				Spec: kargoapi.StageSpec{
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Vars: []kargoapi.PromotionVariable{
								{Name: "key1", Value: "value1"},
							},
							Steps: []kargoapi.PromotionStep{
								{
									As:   "step1",
									Uses: "fake-step",
								},
							},
						},
					},
				},
			},
			freight: "abc123",
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.NotNil(t, promotion)

				// Check basic metadata
				assert.Equal(t, "test-project", promotion.Namespace)
				assert.Equal(t, "test-stage", promotion.Spec.Stage)
				assert.Equal(t, "abc123", promotion.Spec.Freight)

				// Check vars
				assert.Equal(t, []kargoapi.PromotionVariable{
					{
						Name:  "key1",
						Value: "value1",
					},
				}, promotion.Spec.Vars)

				// Check steps
				require.Len(t, promotion.Spec.Steps, 1)
				assert.Equal(t, "step1", promotion.Spec.Steps[0].As)
				assert.Equal(t, "fake-step", promotion.Spec.Steps[0].Uses)

				// Check name format
				assert.Contains(t, promotion.Name, "test-stage")
				assert.Contains(t, promotion.Name, "abc123"[:6])
			},
		},
		{
			name: "successful build with user info",
			stage: kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-stage",
					Namespace: "test-project",
				},
				Spec: kargoapi.StageSpec{
					PromotionTemplate: &kargoapi.PromotionTemplate{
						Spec: kargoapi.PromotionTemplateSpec{
							Steps: []kargoapi.PromotionStep{
								{
									As: "task-step",
									Task: &kargoapi.PromotionTaskReference{
										Name: "test-task",
									},
									Vars: []kargoapi.PromotionVariable{
										{Name: "input1", Value: "value1"},
									},
								},
							},
						},
					},
				},
			},
			freight: "abc123",
			userInfo: user.Info{
				IsAdmin: true,
			},
			assertions: func(t *testing.T, promotion *kargoapi.Promotion, err error) {
				require.NoError(t, err)
				require.NotNil(t, promotion)

				// Check metadata including user annotation
				assert.Equal(t, kargoapi.EventActorAdmin, promotion.Annotations[kargoapi.AnnotationKeyCreateActor])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := user.ContextWithInfo(context.Background(), tt.userInfo)

			c := fake.NewClientBuilder().
				WithScheme(s).
				Build()

			b := NewPromotionBuilder(c)
			promotion, err := b.Build(ctx, tt.stage, tt.freight)
			tt.assertions(t, promotion, err)
		})
	}
}

func TestPromotionBuilder_InflateSteps(t *testing.T) {
	s := runtime.NewScheme()
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
				assert.ErrorContains(t, err, "not found")
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

				assert.Equal(t, "direct-step", steps[0].As)
				assert.Equal(t, "fake-step", steps[0].Uses)
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
							Vars: []kargoapi.PromotionVariable{
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
						Vars: []kargoapi.PromotionVariable{
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

				// Check direct step
				assert.Equal(t, "direct-step", steps[0].As)
				assert.Equal(t, "fake-step", steps[0].Uses)

				// Check inflated task step
				assert.Equal(t, "task-step::sub-step", steps[1].As)
				assert.Equal(t, "other-fake-step", steps[1].Uses)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
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
							Vars: []kargoapi.PromotionVariable{
								{Name: "input1", Value: "value1"},
							},
						},
						{
							As: "task2",
							Task: &kargoapi.PromotionTaskReference{
								Kind: "ClusterPromotionTask",
								Name: "test-task-2",
							},
							Vars: []kargoapi.PromotionVariable{
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
						Vars: []kargoapi.PromotionVariable{
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
						Vars: []kargoapi.PromotionVariable{
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

				assert.Equal(t, "task1::step1", steps[0].As)
				assert.Equal(t, "fake-step", steps[0].Uses)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
					{Name: "input1", Value: "value1"},
				}, steps[0].Vars)

				assert.Equal(t, "task2::step2", steps[1].As)
				assert.Equal(t, "other-fake-step", steps[1].Uses)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
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

			b := NewPromotionBuilder(c)
			p := tt.promo.DeepCopy()
			err := b.InflateSteps(context.Background(), p)
			tt.assertions(t, p.Spec.Steps, err)
		})
	}
}

func TestPromotionBuilder_inflateTaskSteps(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(s))

	tests := []struct {
		name       string
		project    string
		taskAlias  string
		promoVars  []kargoapi.PromotionVariable
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
				assert.True(t, apierrors.IsNotFound(err))
				assert.Nil(t, steps)
			},
		},
		{
			name:    "invalid config for task variables",
			project: "test-project",

			taskStep: kargoapi.PromotionStep{
				Task: &kargoapi.PromotionTaskReference{
					Name: "test-task",
				},
				// Missing values
				Vars: nil,
			},
			objects: []client.Object{
				&kargoapi.PromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-task",
						Namespace: "test-project",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.PromotionVariable{
							{Name: "input1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				assert.ErrorContains(t, err, "missing value for variable")
				assert.Nil(t, steps)
			},
		},
		{
			name:      "successful task step inflation",
			project:   "test-project",
			taskAlias: "task-1",
			promoVars: []kargoapi.PromotionVariable{
				{Name: "input3", Value: "value1"},
			},
			taskStep: kargoapi.PromotionStep{
				Task: &kargoapi.PromotionTaskReference{
					Name: "test-task",
				},
				Vars: []kargoapi.PromotionVariable{
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
						Vars: []kargoapi.PromotionVariable{
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
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, steps []kargoapi.PromotionStep, err error) {
				require.NoError(t, err)
				require.Len(t, steps, 2)

				assert.Equal(t, "task-1::step1", steps[0].As)
				assert.Equal(t, "fake-step", steps[0].Uses)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
					{Name: "input1", Value: "value1"},
					{Name: "input2", Value: "value2"},
				}, steps[0].Vars)

				assert.Equal(t, "task-1::step2", steps[1].As)
				assert.Equal(t, "other-fake-step", steps[1].Uses)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
					{Name: "input1", Value: "value1"},
					{Name: "input2", Value: "value2"},
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
				Vars: []kargoapi.PromotionVariable{
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
						Vars: []kargoapi.PromotionVariable{
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

				assert.Equal(t, "custom-alias::step-0", steps[0].As)
				assert.Equal(t, "custom-alias::step-1", steps[1].As)
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
				Vars: []kargoapi.PromotionVariable{
					{Name: "input1", Value: "value1"},
				},
			},
			objects: []client.Object{
				&kargoapi.ClusterPromotionTask{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-cluster-task",
					},
					Spec: kargoapi.PromotionTaskSpec{
						Vars: []kargoapi.PromotionVariable{
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
				assert.Equal(t, "task-0::custom-alias", steps[0].As)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.objects...).
				Build()

			b := NewPromotionBuilder(c)
			steps, err := b.inflateTaskSteps(context.Background(), tt.project, tt.taskAlias, tt.promoVars, tt.taskStep)
			tt.assertions(t, steps, err)
		})
	}
}

func TestPromotionBuilder_getTaskSpec(t *testing.T) {
	s := runtime.NewScheme()
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
				assert.ErrorContains(t, err, "missing task reference")
				assert.Nil(t, result)
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
				assert.ErrorContains(t, err, "unknown task reference kind")
				assert.Nil(t, result)
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
				assert.ErrorContains(t, err, "something went wrong")
				assert.Nil(t, result)
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
				assert.ErrorContains(t, err, "something went wrong")
				assert.Nil(t, result)
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
						Vars: []kargoapi.PromotionVariable{
							{Name: "input1", Value: "value1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				assert.Len(t, result.Vars, 1)
				assert.Equal(t, "input1", result.Vars[0].Name)
				assert.Equal(t, "value1", result.Vars[0].Value)
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
						Vars: []kargoapi.PromotionVariable{
							{Name: "input1", Value: "value1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				assert.Len(t, result.Vars, 1)
				assert.Equal(t, "input1", result.Vars[0].Name)
				assert.Equal(t, "value1", result.Vars[0].Value)
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
						Vars: []kargoapi.PromotionVariable{
							{Name: "input1", Value: "value1"},
						},
					},
				},
			},
			assertions: func(t *testing.T, result *kargoapi.PromotionTaskSpec, err error) {
				require.NoError(t, err)
				require.NotNil(t, result)

				assert.Len(t, result.Vars, 1)
				assert.Equal(t, "input1", result.Vars[0].Name)
				assert.Equal(t, "value1", result.Vars[0].Value)
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

			b := NewPromotionBuilder(c)
			result, err := b.getTaskSpec(context.Background(), tt.project, tt.ref)
			tt.assertions(t, result, err)
		})
	}
}

func Test_generatePromotionName(t *testing.T) {
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
				assert.Len(t, components, 3)
				assert.Equal(t, "dev", components[0])
				assert.Len(t, components[1], ulidLength)
				assert.Equal(t, "abc123d", components[2])
			},
		},
		{
			name:      "short freight",
			stageName: "prod",
			freight:   "abc",
			assertions: func(t *testing.T, result string) {
				components := strings.Split(result, ".")
				assert.Len(t, components, 3)
				assert.Equal(t, "prod", components[0])
				assert.Len(t, components[1], ulidLength)
				assert.Equal(t, "abc", components[2])
			},
		},
		{
			name: "long stage name gets truncated",
			// nolint:lll
			stageName: "this-is-a-very-long-stage-name-that-exceeds-the-maximum-allowed-length-for-kubernetes-resources-and-should-be-truncated-to-fit-within-the-limits-set-by-the-api-server-which-is-253-characters-including-the-generated-suffix",
			freight:   "abc123def456",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, len(result), 253) // Kubernetes resource name limit
				assert.Equal(t, maxStageNamePrefixLength, len(result[:strings.Index(result, ".")]))
			},
		},
		{
			name:      "long freight gets truncated",
			stageName: "stage",
			freight:   "this-is-a-very-long-freight-hash-that-should-be-truncated",
			assertions: func(t *testing.T, result string) {
				shortHash := result[strings.LastIndex(result, ".")+1:]
				assert.Equal(t, shortHashLength, len(shortHash))
			},
		},
		{
			name:      "all lowercase conversion",
			stageName: "DEV-STAGE",
			freight:   "ABC123DEF456",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "dev-stage", result[:len("dev-stage")])
				assert.Equal(t, "abc123d", result[len(result)-7:])
			},
		},
		{
			name:      "empty inputs",
			stageName: "",
			freight:   "",
			assertions: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePromotionName(tt.stageName, tt.freight)
			tt.assertions(t, result)
		})
	}
}

func Test_generatePromotionTaskStepName(t *testing.T) {
	tests := []struct {
		name       string
		taskAlias  string
		stepAlias  string
		assertions func(t *testing.T, result string)
	}{
		{
			name:      "standard aliases",
			taskAlias: "deploy",
			stepAlias: "apply",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "deploy::apply", result)
			},
		},
		{
			name:      "empty task alias",
			taskAlias: "",
			stepAlias: "apply",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "::apply", result)
			},
		},
		{
			name:      "empty step alias",
			taskAlias: "deploy",
			stepAlias: "",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "deploy::", result)
			},
		},
		{
			name:      "both aliases empty",
			taskAlias: "",
			stepAlias: "",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "::", result)
			},
		},
		{
			name:      "aliases with special characters",
			taskAlias: "deploy-task",
			stepAlias: "apply_config",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "deploy-task::apply_config", result)
			},
		},
		{
			name:      "aliases containing separator",
			taskAlias: "deploy::task",
			stepAlias: "apply::config",
			assertions: func(t *testing.T, result string) {
				assert.Equal(t, "deploy::task::apply::config", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePromotionTaskStepAlias(tt.taskAlias, tt.stepAlias)
			tt.assertions(t, result)
		})
	}
}

func Test_promotionTaskVarsToStepVars(t *testing.T) {
	tests := []struct {
		name       string
		taskVars   []kargoapi.PromotionVariable
		promoVars  []kargoapi.PromotionVariable
		stepVars   []kargoapi.PromotionVariable
		assertions func(t *testing.T, result []kargoapi.PromotionVariable, err error)
	}{
		{
			name:     "nil inputs returns nil map and no error",
			taskVars: nil,
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.PromotionVariable, err error) {
				require.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name:     "empty inputs returns nil map and no error",
			taskVars: []kargoapi.PromotionVariable{},
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.PromotionVariable, err error) {
				require.NoError(t, err)
				assert.Nil(t, result)
			},
		},
		{
			name: "missing required variable returns error",
			taskVars: []kargoapi.PromotionVariable{
				{Name: "input1"},
			},
			stepVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: ""},
			},
			assertions: func(t *testing.T, result []kargoapi.PromotionVariable, err error) {
				assert.ErrorContains(t, err, "missing value for variable \"input1\"")
				assert.Nil(t, result)
			},
		},
		{
			name: "default value used when config value not provided",
			taskVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: "default1"},
			},
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.PromotionVariable, err error) {
				require.NoError(t, err)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
					{Name: "input1", Value: "default1"},
				}, result)
			},
		},
		{
			name: "step value overrides default value",
			taskVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: "default1"},
			},
			stepVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: "override1"},
			},
			assertions: func(t *testing.T, result []kargoapi.PromotionVariable, err error) {
				require.NoError(t, err)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
					{Name: "input1", Value: "override1"},
				}, result)
			},
		},
		{
			name: "promotion variable overrides default value",
			taskVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: "default1"},
			},
			promoVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: "override1"},
			},
			stepVars: nil,
			assertions: func(t *testing.T, result []kargoapi.PromotionVariable, err error) {
				require.NoError(t, err)
				// Variable is set by engine at runtime
				assert.Empty(t, result)
			},
		},
		{
			name: "multiple inputs processed correctly",
			taskVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: "default1"},
				{Name: "input2", Value: "default2"},
				{Name: "input3"},
			},
			stepVars: []kargoapi.PromotionVariable{
				{Name: "input1", Value: "override1"},
				{Name: "input3", Value: "value3"},
			},
			assertions: func(t *testing.T, result []kargoapi.PromotionVariable, err error) {
				require.NoError(t, err)
				assert.ElementsMatch(t, []kargoapi.PromotionVariable{
					{Name: "input1", Value: "override1"},
					{Name: "input2", Value: "default2"},
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
