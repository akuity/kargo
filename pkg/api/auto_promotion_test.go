package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestFindMatchingPromotionPolicy(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, kargoapi.AddToScheme(scheme))

	stageMeta := metav1.ObjectMeta{
		Name:      "fake-stage",
		Namespace: "fake-project",
	}
	testCases := []struct {
		name        string
		objects     []runtime.Object
		interceptor interceptor.Funcs
		assert      func(*testing.T, *kargoapi.PromotionPolicy, error)
	}{
		{
			name: "nil without ProjectConfig",
			assert: func(t *testing.T, policy *kargoapi.PromotionPolicy, err error) {
				require.NoError(t, err)
				require.Nil(t, policy)
			},
		},
		{
			name: "error getting ProjectConfig",
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
			assert: func(t *testing.T, policy *kargoapi.PromotionPolicy, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, policy)
			},
		},
		{
			name: "nil when no policy matches",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							StageSelector: &kargoapi.PromotionPolicySelector{Name: "other-stage"},
						}},
					},
				},
			},
			assert: func(t *testing.T, policy *kargoapi.PromotionPolicy, err error) {
				require.NoError(t, err)
				require.Nil(t, policy)
			},
		},
		{
			name: "returns first matching policy even when a later one also matches",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								StageSelector:        &kargoapi.PromotionPolicySelector{Name: "other-stage"},
								AutoPromotionEnabled: false,
							},
							{
								StageSelector:        &kargoapi.PromotionPolicySelector{Name: "fake-stage"},
								AutoPromotionEnabled: true,
							},
							{
								StageSelector:        &kargoapi.PromotionPolicySelector{Name: "fake-stage"},
								AutoPromotionEnabled: false,
							},
						},
					},
				},
			},
			assert: func(t *testing.T, policy *kargoapi.PromotionPolicy, err error) {
				require.NoError(t, err)
				require.NotNil(t, policy)
				require.True(t, policy.AutoPromotionEnabled)
			},
		},
		{
			name: "matches by deprecated Stage field",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage: "fake-stage", // nolint:staticcheck
						}},
					},
				},
			},
			assert: func(t *testing.T, policy *kargoapi.PromotionPolicy, err error) {
				require.NoError(t, err)
				require.NotNil(t, policy)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(testCase.objects...).
				WithInterceptorFuncs(testCase.interceptor).
				Build()
			policy, err := FindMatchingPromotionPolicy(t.Context(), c, stageMeta)
			testCase.assert(t, policy, err)
		})
	}
}

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
		name        string
		objects     []runtime.Object
		interceptor interceptor.Funcs
		assert      func(*testing.T, bool, error)
	}{
		{
			name: "disabled without ProjectConfig",
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.False(t, enabled)
			},
		},
		{
			name: "error getting ProjectConfig",
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
			assert: func(t *testing.T, enabled bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, enabled)
			},
		},
		{
			name: "disabled with empty promotion policies",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{},
					},
				},
			},
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.False(t, enabled)
			},
		},
		{
			name: "enabled by deprecated Stage field",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{{
							Stage:                "fake-stage", // nolint:staticcheck
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
			name: "returns first matching policy for the Stage",
			objects: []runtime.Object{
				&kargoapi.ProjectConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-project",
						Namespace: "fake-project",
					},
					Spec: kargoapi.ProjectConfigSpec{
						PromotionPolicies: []kargoapi.PromotionPolicy{
							{
								StageSelector:        &kargoapi.PromotionPolicySelector{Name: "other-stage"},
								AutoPromotionEnabled: false,
							},
							{
								StageSelector:        &kargoapi.PromotionPolicySelector{Name: "fake-stage"},
								AutoPromotionEnabled: true,
							},
							{
								StageSelector:        &kargoapi.PromotionPolicySelector{Name: "fake-stage"},
								AutoPromotionEnabled: false,
							},
						},
					},
				},
			},
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.True(t, enabled)
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
				WithInterceptorFuncs(testCase.interceptor).
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
	newestFreightStage := &kargoapi.Stage{
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
	matchUpstreamStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-stage",
			Namespace: "fake-project",
		},
		Spec: kargoapi.StageSpec{
			RequestedFreight: []kargoapi.FreightRequest{{
				Origin: warehouseOrigin,
				Sources: kargoapi.FreightSources{
					Stages: []string{"upstream"},
					AutoPromotionOptions: &kargoapi.AutoPromotionOptions{
						SelectionPolicy: kargoapi.AutoPromotionSelectionPolicyMatchUpstream,
					},
				},
			}},
		},
	}

	testCases := []struct {
		name             string
		stage            *kargoapi.Stage
		availableFreight []kargoapi.Freight
		assert           func(*testing.T, map[string]kargoapi.Freight, error)
	}{
		{
			name:  "newest Freight selected per origin",
			stage: newestFreightStage,
			availableFreight: []kargoapi.Freight{
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
			assert: func(t *testing.T, candidates map[string]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Len(t, candidates, 2)
				require.Equal(t, "newer-freight", candidates[warehouseOrigin.String()].Name)
				require.Equal(t, "other-freight", candidates[otherOrigin.String()].Name)
			},
		},
		{
			name:  "creation-time tie broken by lexically greater name",
			stage: newestFreightStage,
			availableFreight: []kargoapi.Freight{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "aaa-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: warehouseOrigin,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "bbb-freight",
						CreationTimestamp: metav1.Time{Time: now},
					},
					Origin: warehouseOrigin,
				},
			},
			assert: func(t *testing.T, candidates map[string]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, "bbb-freight", candidates[warehouseOrigin.String()].Name)
			},
		},
		{
			name:  "matchUpstream selects the single available Freight",
			stage: matchUpstreamStage,
			availableFreight: []kargoapi.Freight{{
				ObjectMeta: metav1.ObjectMeta{Name: "upstream-current-freight"},
				Origin:     warehouseOrigin,
			}},
			assert: func(t *testing.T, candidates map[string]kargoapi.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, "upstream-current-freight", candidates[warehouseOrigin.String()].Name)
			},
		},
		{
			name:  "matchUpstream with multiple available Freight errors",
			stage: matchUpstreamStage,
			availableFreight: []kargoapi.Freight{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "first-freight"},
					Origin:     warehouseOrigin,
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "second-freight"},
					Origin:     warehouseOrigin,
				},
			},
			assert: func(t *testing.T, _ map[string]kargoapi.Freight, err error) {
				require.ErrorContains(t, err, "unexpectedly found 2 available Freight")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidates, err := SelectAutoPromotionCandidates(
				testCase.stage,
				testCase.availableFreight,
			)
			testCase.assert(t, candidates, err)
		})
	}
}
