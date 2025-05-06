package indexer

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/argocd"
)

func TestEventsByInvolvedObjectAPIGroup(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		event      *corev1.Event
		assertions func(*testing.T, []string)
	}{
		{
			name: "Event has no involved object",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-event",
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name: "Event has involved object with no API group",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-event",
				},
				InvolvedObject: corev1.ObjectReference{},
			},
			assertions: func(t *testing.T, keys []string) {
				require.Nil(t, keys)
			},
		},
		{
			name: "Event has involved object with API group",
			event: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-event",
				},
				InvolvedObject: corev1.ObjectReference{
					APIVersion: "fake-group/fake-version",
				},
			},
			assertions: func(t *testing.T, keys []string) {
				require.Equal(
					t,
					[]string{
						"fake-group",
					},
					keys,
				)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assertions(t, EventsByInvolvedObjectAPIGroup(tc.event))
		})
	}
}

func TestStagesByAnalysisRun(t *testing.T) {
	const testShardName = "test-shard"
	t.Parallel()
	testCases := []struct {
		name                string
		controllerShardName string
		stage               *kargoapi.Stage
		assertions          func(*testing.T, []string)
	}{
		{
			name:                "Stage belongs to another shard",
			controllerShardName: testShardName,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.ShardLabelKey: "another-shard",
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name:                "Stage belongs to this shard",
			controllerShardName: testShardName,
			stage: &kargoapi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.ShardLabelKey: testShardName,
					},
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: kargoapi.VerificationInfoStack{
								{
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Namespace: "fake-namespace",
										Name:      "fake-analysis-run",
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Equal(
					t,
					[]string{
						"fake-namespace:fake-analysis-run",
					},
					res,
				)
			},
		},
		{
			name:                "Stage is unlabeled and this is not the default controller",
			controllerShardName: testShardName,
			stage:               &kargoapi.Stage{},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name:                "Stage is unlabeled and this is the default controller",
			controllerShardName: "",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: kargoapi.VerificationInfoStack{
								{
									AnalysisRun: &kargoapi.AnalysisRunReference{
										Namespace: "fake-namespace",
										Name:      "fake-analysis-run",
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Equal(
					t,
					[]string{
						"fake-namespace:fake-analysis-run",
					},
					res,
				)
			},
		},
		{
			name: "Stage does not have any Freight history",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name: "Stage does not have any Verification history",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: nil,
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
		{
			name: "Stage does not have any AnalysisRun references",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {},
							},
							VerificationHistory: kargoapi.VerificationInfoStack{
								{
									AnalysisRun: nil,
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, res []string) {
				require.Nil(t, res)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := StagesByAnalysisRun(tc.controllerShardName)(tc.stage)
			tc.assertions(t, res)
		})
	}
}

func TestPromotionsByStage(t *testing.T) {
	testCases := map[string]struct {
		input      *kargoapi.Promotion
		predicates []func(*kargoapi.Promotion) bool
		expected   []string
	}{
		"empty predicates/terminal phase": {
			input: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			expected: []string{"fake-stage"},
		},
		"empty predicates/non-terminal phase": {
			input: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			expected: []string{"fake-stage"},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			actual := PromotionsByStage(tc.input)
			require.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func TestRunningPromotionsByArgoCDApplications(t *testing.T) {
	const testShardName = "test-shard"

	testCases := []struct {
		name      string
		obj       client.Object
		stage     client.Object
		shardName string
		expected  []string
	}{
		{
			name:     "Object is not a Promotion",
			obj:      &kargoapi.Stage{},
			expected: nil,
		},
		{
			name: "Promotion is not running",
			obj: &kargoapi.Promotion{
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			expected: nil,
		},
		{
			name: "Promotion belongs to another shard",
			obj: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.ShardLabelKey: "another",
					},
				},
			},
			shardName: testShardName,
			expected:  nil,
		},
		{
			name: "Promotion belongs to this shard",
			obj: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						kargoapi.ShardLabelKey: testShardName,
					},
				},
			},
			shardName: testShardName,
			expected:  nil,
		},
		{
			name: "Promotion has directive steps",
			obj: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
					Vars: []kargoapi.ExpressionVariable{
						{
							Name:  "app",
							Value: "fake-app-from-var",
						},
					},
					Steps: []kargoapi.PromotionStep{
						{
							Uses: "argocd-update",
							Config: &apiextensionsv1.JSON{
								Raw: []byte(`{"apps":[{"namespace":"fake-namespace","name":"fake-app"}]}`),
							},
						},
						{
							Uses: "fake-directive",
						},
						{
							Uses: "argocd-update",
							Config: &apiextensionsv1.JSON{
								// Note that this uses an expression
								Raw: []byte(`{"apps":[{"name":"fake-app-${{ ctx.stage }}"}]}`),
							},
						},
						{
							Uses: "argocd-update",
							Config: &apiextensionsv1.JSON{
								// Note that this uses a variable within the expression
								Raw: []byte(`{"apps":[{"name":"${{ vars.app }}"}]}`),
							},
						},
						{
							Uses: "argocd-update",
							Vars: []kargoapi.ExpressionVariable{
								{
									Name:  "app",
									Value: "fake-app-from-step-var",
								},
							},
							Config: &apiextensionsv1.JSON{
								// Note that this uses a step-level variable within the expression
								Raw: []byte(`{"apps":[{"name":"${{ vars.app }}"}]}`),
							},
						},
						{
							Uses: "argocd-update",
							Config: &apiextensionsv1.JSON{
								// Note that this uses output from a (fake) previous step within the expression
								Raw: []byte(`{"apps":[{"name":"fake-app-${{ outputs.push.branch }}"}]}`),
							},
						},
						{
							Uses: "argocd-update",
							Vars: []kargoapi.ExpressionVariable{
								{
									Name:  "input",
									Value: "${{ outputs.composition.name }}",
								},
							},
							Config: &apiextensionsv1.JSON{
								// Note that this uses output from a previous step through a variable
								Raw: []byte(`{"apps":[{"name":"fake-app-${{ vars.input }}"}]}`),
							},
						},
						{
							Uses: "argocd-update",
							As:   "task-1::update",
							Config: &apiextensionsv1.JSON{
								// Note that this uses output from a "task" step within the expression
								Raw: []byte(`{"apps":[{"name":"fake-app-${{ task.outputs.fake.name }}"}]}`),
							},
						},
					},
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseRunning,
					State: &apiextensionsv1.JSON{
						// Mock the output of the previous steps
						// nolint:lll
						Raw: []byte(`{"push":{"branch":"from-branch"},"composition":{"name":"from-composition"},"task-1::fake":{"name":"from-task"}}`),
					},
					CurrentStep: 7, // Ensure all steps above are considered
				},
			},
			expected: []string{
				"fake-namespace:fake-app",
				fmt.Sprintf("%s:%s", argocd.Namespace(), "fake-app-fake-stage"),
				fmt.Sprintf("%s:%s", argocd.Namespace(), "fake-app-from-var"),
				fmt.Sprintf("%s:%s", argocd.Namespace(), "fake-app-from-step-var"),
				fmt.Sprintf("%s:%s", argocd.Namespace(), "fake-app-from-branch"),
				fmt.Sprintf("%s:%s", argocd.Namespace(), "fake-app-from-composition"),
				fmt.Sprintf("%s:%s", argocd.Namespace(), "fake-app-from-task"),
			},
		},
		{
			name: "Promotion has directive steps without Applications",
			obj: &kargoapi.Promotion{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "fake-namespace",
				},
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
					Steps: []kargoapi.PromotionStep{
						{
							Uses: "fake-directive",
						},
						{
							Uses: "fake-directive",
						},
					},
				},
				Status: kargoapi.PromotionStatus{
					Phase:       kargoapi.PromotionPhaseRunning,
					CurrentStep: 1, // Ensure all steps above are considered
				},
			},
			expected: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				RunningPromotionsByArgoCDApplications(
					context.TODO(),
					fake.NewClientBuilder().Build(),
					testCase.shardName,
				)(testCase.obj),
			)
		})
	}
}

func TestPromotionsByStageAndFreight(t *testing.T) {
	promo := &kargoapi.Promotion{
		Spec: kargoapi.PromotionSpec{
			Stage:   "fake-stage",
			Freight: "fake-freight",
		},
	}
	res := PromotionsByStageAndFreight(promo)
	require.Equal(t, []string{"fake-stage:fake-freight"}, res)
}

func TestFreightByWarehouse(t *testing.T) {
	testCases := []struct {
		name     string
		freight  *kargoapi.Freight
		expected []string
	}{
		{
			name:     "Freight has no Warehouse origin",
			freight:  &kargoapi.Freight{},
			expected: nil,
		},
		{
			name: "Freight has a Warehouse origin",
			freight: &kargoapi.Freight{
				Origin: kargoapi.FreightOrigin{
					Kind: kargoapi.FreightOriginKindWarehouse,
					Name: "fake-warehouse",
				},
			},
			expected: []string{"fake-warehouse"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				FreightByWarehouse(testCase.freight),
			)
		})
	}
}

func TestFreightByCurrentStages(t *testing.T) {
	testCases := []struct {
		name     string
		freight  *kargoapi.Freight
		expected []string
	}{
		{
			name:     "Freight is not currently in use by any Stages",
			freight:  &kargoapi.Freight{},
			expected: []string{},
		},
		{
			name: "Freight is currently in use by a Stage",
			freight: &kargoapi.Freight{
				Status: kargoapi.FreightStatus{
					CurrentlyIn: map[string]kargoapi.CurrentStage{"fake-stage": {}},
				},
			},
			expected: []string{"fake-stage"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Run(testCase.name, func(t *testing.T) {
				require.Equal(
					t,
					testCase.expected,
					FreightByCurrentStages(testCase.freight),
				)
			})
		})
	}
}

func TestFreightByVerifiedStages(t *testing.T) {
	testCases := []struct {
		name     string
		freight  *kargoapi.Freight
		expected []string
	}{
		{
			name:     "Freight is not verified in any Stages",
			freight:  &kargoapi.Freight{},
			expected: []string{},
		},
		{
			name: "Freight is verified in a Stage",
			freight: &kargoapi.Freight{
				Status: kargoapi.FreightStatus{
					VerifiedIn: map[string]kargoapi.VerifiedStage{
						"fake-stage": {},
					},
				},
			},
			expected: []string{"fake-stage"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Run(testCase.name, func(t *testing.T) {
				require.Equal(
					t,
					testCase.expected,
					FreightByVerifiedStages(testCase.freight),
				)
			})
		})
	}
}

func TestFreightApprovedForStages(t *testing.T) {
	testCases := []struct {
		name     string
		freight  *kargoapi.Freight
		expected []string
	}{
		{
			name:     "Freight is not approved for any Stages",
			freight:  &kargoapi.Freight{},
			expected: []string{},
		},
		{
			name: "Freight is approved for a Stage",
			freight: &kargoapi.Freight{
				Status: kargoapi.FreightStatus{
					ApprovedFor: map[string]kargoapi.ApprovedStage{
						"fake-stage": {},
					},
				},
			},
			expected: []string{"fake-stage"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Run(testCase.name, func(t *testing.T) {
				require.Equal(
					t,
					testCase.expected,
					FreightApprovedForStages(testCase.freight),
				)
			})
		})
	}
}

func TestStagesByFreight(t *testing.T) {
	testCases := []struct {
		name     string
		stage    *kargoapi.Stage
		expected []string
	}{
		{
			name:     "Stage has no current Freight",
			stage:    &kargoapi.Stage{},
			expected: nil,
		},
		{
			name: "Stage has no Freight history",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "Stage has Freight",
			stage: &kargoapi.Stage{
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{
						{
							Freight: map[string]kargoapi.FreightReference{
								"fake-warehouse": {
									Name: "fake-freight",
								},
								"another-fake-warehouse": {
									Name: "another-fake-freight",
								},
							},
						},
					},
				},
			},
			expected: []string{"another-fake-freight", "fake-freight"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				StagesByFreight(testCase.stage),
			)
		})
	}

}

func TestStagesByUpstreamStages(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name     string
		stage    *kargoapi.Stage
		expected []string
	}{
		{
			name: "Stage has no upstream Stages",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: testOrigin,
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "Stage has upstream stages",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: testOrigin,
							Sources: kargoapi.FreightSources{
								Stages: []string{
									"fake-stage",
									"another-fake-stage",
								},
							},
						},
					},
				},
			},
			expected: []string{"another-fake-stage", "fake-stage"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				StagesByUpstreamStages(testCase.stage),
			)
		})
	}
}

func TestStagesByWarehouse(t *testing.T) {
	testCases := []struct {
		name     string
		stage    *kargoapi.Stage
		expected []string
	}{
		{
			name:     "Stage has no Warehouse origin",
			stage:    &kargoapi.Stage{},
			expected: nil,
		},
		{
			name: "Stage has Warehouse origins",
			stage: &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					RequestedFreight: []kargoapi.FreightRequest{
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse-indirect",
							},
							Sources: kargoapi.FreightSources{
								Direct: false,
							},
						},
						{
							Origin: kargoapi.FreightOrigin{
								Kind: kargoapi.FreightOriginKindWarehouse,
								Name: "fake-warehouse-2",
							},
							Sources: kargoapi.FreightSources{
								Direct: true,
							},
						},
					},
				},
			},
			expected: []string{"fake-warehouse", "fake-warehouse-2"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				StagesByWarehouse(testCase.stage),
			)
		})
	}
}

func TestServiceAccountsByOIDCClaims(t *testing.T) {
	testCases := []struct {
		name     string
		sa       *corev1.ServiceAccount
		expected []string
	}{
		{
			name: "ServiceAccount has no OIDC email",
			sa:   &corev1.ServiceAccount{},
		},
		{
			name: "ServiceAccount has OIDC email",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						rbacapi.AnnotationKeyOIDCClaimNamePrefix + "email": "fake-email, fake-email-2",
					},
				},
			},
			expected: []string{"email/fake-email", "email/fake-email-2"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				ServiceAccountsByOIDCClaims(testCase.sa),
			)
		})
	}
}

func TestWarehousesByRepoURL(t *testing.T) {
	for _, test := range []struct {
		name      string
		warehouse client.Object
		expected  []string
	}{
		{
			name: "simple",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					Subscriptions: []kargoapi.RepoSubscription{
						{
							Git: &kargoapi.GitSubscription{
								RepoURL: "https://github.com/username/repo",
							},
							Image: &kargoapi.ImageSubscription{
								RepoURL: "https://registry.hub.docker.com/u/svendowideit/testhook/",
							},
							Chart: &kargoapi.ChartSubscription{
								RepoURL: "https://example.com/charts/alpine-0.1.2",
							},
						},
					},
				},
			},
			expected: []string{
				"https://github.com/username/repo",
				"https://example.com/charts/alpine-0.1.2",
				"https://registry.hub.docker.com/u/svendowideit/testhook/",
			},
		},
		{
			name:      "not a warehouse",
			warehouse: &kargoapi.Freight{},
			expected:  nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t,
				test.expected,
				WarehousesByRepoURL(test.warehouse),
			)

		})
	}
}
