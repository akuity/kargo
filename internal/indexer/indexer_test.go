package indexer

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/argocd"
)

func TestIndexEventsByInvolvedObjectAPIGroup(t *testing.T) {
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
			tc.assertions(t, indexEventsByInvolvedObjectAPIGroup(tc.event))
		})
	}
}

func TestIndexStagesByAnalysisRun(t *testing.T) {
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
			res := indexStagesByAnalysisRun(tc.controllerShardName)(tc.stage)
			tc.assertions(t, res)
		})
	}
}

func TestIndexPromotionsByStage(t *testing.T) {
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
		"isPromotionPhaseNonTerminal excludes Promotions in terminal phases": {
			input: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseSucceeded,
				},
			},
			predicates: []func(*kargoapi.Promotion) bool{
				isPromotionPhaseNonTerminal,
			},
			expected: nil,
		},
		"isPromotionPhaseNonTerminal selects Promotions in non-terminal phases": {
			input: &kargoapi.Promotion{
				Spec: kargoapi.PromotionSpec{
					Stage: "fake-stage",
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhasePending,
				},
			},
			predicates: []func(*kargoapi.Promotion) bool{
				isPromotionPhaseNonTerminal,
			},
			expected: []string{"fake-stage"},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			actual := indexPromotionsByStage(tc.predicates...)(tc.input)
			require.ElementsMatch(t, tc.expected, actual)
		})
	}
}

func TestIndexRunningPromotionsByArgoCDApplications(t *testing.T) {
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
								Raw: []byte(`{"apps":[{"name":"fake-app-2"}]}`),
							},
						},
					},
				},
				Status: kargoapi.PromotionStatus{
					Phase: kargoapi.PromotionPhaseRunning,
				},
			},
			expected: []string{
				"fake-namespace:fake-app",
				fmt.Sprintf("%s:%s", argocd.Namespace(), "fake-app-2"),
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
					Phase: kargoapi.PromotionPhaseRunning,
				},
			},
			expected: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			require.Equal(
				t,
				testCase.expected,
				indexRunningPromotionsByArgoCDApplications(
					context.TODO(),
					testCase.shardName,
				)(testCase.obj),
			)
		})
	}
}

func TestIndexPromotionsByStageAndFreight(t *testing.T) {
	promo := &kargoapi.Promotion{
		Spec: kargoapi.PromotionSpec{
			Stage:   "fake-stage",
			Freight: "fake-freight",
		},
	}
	res := indexPromotionsByStageAndFreight(promo)
	require.Equal(t, []string{"fake-stage:fake-freight"}, res)
}

func TestFreightByWarehouseIndexer(t *testing.T) {
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
				FreightByWarehouseIndexer(testCase.freight),
			)
		})
	}
}

func TestFreightByVerifiedStagesIndexer(t *testing.T) {
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
					FreightByVerifiedStagesIndexer(testCase.freight),
				)
			})
		})
	}
}

func TestFreightApprovedForStagesIndexer(t *testing.T) {
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
					FreightApprovedForStagesIndexer(testCase.freight),
				)
			})
		})
	}
}

func TestIndexStagesByFreight(t *testing.T) {
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
				indexStagesByFreight(testCase.stage),
			)
		})
	}

}

func TestStagesByUpstreamStagesIndexer(t *testing.T) {
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
				StagesByUpstreamStagesIndexer(testCase.stage),
			)
		})
	}
}

func TestStagesByWarehouseIndexer(t *testing.T) {
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
				StagesByWarehouseIndexer(testCase.stage),
			)
		})
	}
}

func TestIndexServiceAccountsByOIDCClaims(t *testing.T) {
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
				indexServiceAccountsByOIDCClaims(testCase.sa),
			)
		})
	}
}
