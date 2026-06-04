package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestIsAutoPromotionEnabled(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	stageMeta := metav1.ObjectMeta{
		Name:      "fake-stage",
		Namespace: "fake-project",
		Labels: map[string]string{
			"tier": "prod",
		},
	}
	testCases := []struct {
		name    string
		objects []runtime.Object
		assert  func(*testing.T, bool, error)
	}{
		{
			name: "disabled without ProjectConfig",
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.False(t, enabled)
			},
		},
		{
			name: "enabled by matching name selector",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							StageSelector:        &kargoapi.PromotionPolicySelector{Name: "fake-stage"},
							AutoPromotionEnabled: true,
						}},
					},
				},
			},
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.True(t, enabled)
			},
		},
		{
			name: "enabled by matching label selector",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							StageSelector: &kargoapi.PromotionPolicySelector{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"tier": "prod"},
								},
							},
							AutoPromotionEnabled: true,
						}},
					},
				},
			},
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.True(t, enabled)
			},
		},
		{
			name: "disabled by non-matching selector",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							StageSelector:        &kargoapi.PromotionPolicySelector{Name: "other-stage"},
							AutoPromotionEnabled: true,
						}},
					},
				},
			},
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.False(t, enabled)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(testCase.objects...).
				Build()
			enabled, err := IsAutoPromotionEnabled(t.Context(), c, stageMeta)
			testCase.assert(t, enabled, err)
		})
	}
}

func TestSelectAutoPromotionCandidates(t *testing.T) {
	now := time.Now()
	warehouseOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	otherOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "other-warehouse",
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: "fake-project",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin: warehouseOrigin,
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
				{
					Origin: otherOrigin,
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
			},
		},
	}

	candidates, err := SelectAutoPromotionCandidates(
		stage,
		[]kargoapi.Freight{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "older-freight",
					CreationTimestamp: metav1.Time{Time: now.Add(-time.Hour)},
				},
				Origin: warehouseOrigin,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "newer-freight",
					CreationTimestamp: metav1.Time{Time: now},
				},
				Origin: warehouseOrigin,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "other-freight",
					CreationTimestamp: metav1.Time{Time: now.Add(-30 * time.Minute)},
				},
				Origin: otherOrigin,
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	require.Equal(t, "newer-freight", candidates[warehouseOrigin.String()].Name)
	require.Equal(t, "other-freight", candidates[otherOrigin.String()].Name)
}

func TestSelectAutoPromotionCandidatesSkipsRejectedFreight(t *testing.T) {
	now := time.Now()
	origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	otherOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "other-warehouse",
	}
	stage := &kargoapi.Stage{
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{
				{
					Origin: origin,
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
				{
					Origin: otherOrigin,
					Sources: kargoapi.FreightSources{
						Direct: true,
					},
				},
			},
		},
	}

	candidates, err := SelectAutoPromotionCandidates(
		stage,
		[]kargoapi.Freight{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "older-freight",
					CreationTimestamp: metav1.Time{Time: now.Add(-time.Hour)},
				},
				Origin: origin,
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "newer-rejected-freight",
					CreationTimestamp: metav1.Time{Time: now},
				},
				Origin: origin,
				Status: kargoapi.FreightStatus{
					Rejected: &kargoapi.FreightRejection{},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "other-rejected-freight",
					CreationTimestamp: metav1.Time{Time: now},
				},
				Origin: otherOrigin,
				Status: kargoapi.FreightStatus{
					Rejected: &kargoapi.FreightRejection{},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "older-freight", candidates[origin.String()].Name)
	require.NotContains(t, candidates, otherOrigin.String())
}

func TestSelectAutoPromotionCandidatesForMatchUpstream(t *testing.T) {
	origin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: "fake-project",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: origin,
				Sources: kargoapi.FreightSources{
					Stages: []string{"upstream"},
					AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
						SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
					},
				},
			}},
		},
	}

	candidates, err := SelectAutoPromotionCandidates(
		stage,
		[]kargoapi.Freight{{
			ObjectMeta: metav1.ObjectMeta{Name: "upstream-current-freight"},
			Origin:     origin,
		}},
	)
	require.NoError(t, err)
	require.Equal(t, "upstream-current-freight", candidates[origin.String()].Name)

	_, err = SelectAutoPromotionCandidates(
		stage,
		[]kargoapi.Freight{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "first-freight"},
				Origin:     origin,
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "second-freight"},
				Origin:     origin,
			},
		},
	)
	require.ErrorContains(t, err, "unexpectedly found 2 available Freight")
}
