package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// fakeGetter is a ProjectConfigGetter backed by a map keyed by Project name.
type fakeGetter map[string]*kargoapi.ProjectConfig

func (f fakeGetter) Get(
	_ context.Context,
	project string,
) (*kargoapi.ProjectConfig, error) {
	return f[project], nil
}

// errGetter is a ProjectConfigGetter that always fails.
type errGetter struct{}

func (errGetter) Get(
	context.Context,
	string,
) (*kargoapi.ProjectConfig, error) {
	return nil, errors.New("something went wrong")
}

func TestFindMatchingPromotionPolicy(t *testing.T) {
	stageMeta := metav1.ObjectMeta{
		Name:      "fake-stage",
		Namespace: "fake-project",
	}
	testCases := []struct {
		name   string
		getter ProjectConfigGetter
		assert func(*testing.T, *kargoapi.PromotionPolicy, error)
	}{
		{
			name: "nil without ProjectConfig",
			assert: func(t *testing.T, policy *kargoapi.PromotionPolicy, err error) {
				require.NoError(t, err)
				require.Nil(t, policy)
			},
		},
		{
			name:   "error getting ProjectConfig",
			getter: errGetter{},
			assert: func(t *testing.T, policy *kargoapi.PromotionPolicy, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.Nil(t, policy)
			},
		},
		{
			name: "nil when no policy matches",
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			if testCase.getter == nil {
				testCase.getter = fakeGetter{}
			}
			policy, err := FindMatchingPromotionPolicy(t.Context(), testCase.getter, stageMeta)
			testCase.assert(t, policy, err)
		})
	}
}

func TestIsAutoPromotionEnabled(t *testing.T) {
	stageMeta := metav1.ObjectMeta{
		Name:      "fake-stage",
		Namespace: "fake-project",
		Labels: map[string]string{
			"tier": "prod",
		},
	}
	testCases := []struct {
		name   string
		getter ProjectConfigGetter
		assert func(*testing.T, bool, error)
	}{
		{
			name: "disabled without ProjectConfig",
			assert: func(t *testing.T, enabled bool, err error) {
				require.NoError(t, err)
				require.False(t, enabled)
			},
		},
		{
			name:   "error getting ProjectConfig",
			getter: errGetter{},
			assert: func(t *testing.T, enabled bool, err error) {
				require.ErrorContains(t, err, "something went wrong")
				require.False(t, enabled)
			},
		},
		{
			name: "disabled with empty promotion policies",
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			getter: fakeGetter{
				"fake-project": &kargoapi.ProjectConfig{
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
			if testCase.getter == nil {
				testCase.getter = fakeGetter{}
			}
			enabled, err := IsAutoPromotionEnabled(t.Context(), testCase.getter, stageMeta)
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
		assert           func(*testing.T, map[string]kargoapi.Freight)
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
			assert: func(t *testing.T, candidates map[string]kargoapi.Freight) {
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
			assert: func(t *testing.T, candidates map[string]kargoapi.Freight) {
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
			assert: func(t *testing.T, candidates map[string]kargoapi.Freight) {
				require.Equal(t, "upstream-current-freight", candidates[warehouseOrigin.String()].Name)
			},
		},
		{
			name:  "matchUpstream skips origin when multiple Freight transiently available",
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
			assert: func(t *testing.T, candidates map[string]kargoapi.Freight) {
				require.Empty(t, candidates)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidates := SelectAutoPromotionCandidates(
				context.Background(),
				testCase.stage,
				testCase.availableFreight,
			)
			testCase.assert(t, candidates)
		})
	}
}

func TestSetAutoPromotionHoldAnnotation(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p",
			Annotations: map[string]string{
				kargoapi.AnnotationKeyAutoPromotionResume: "Warehouse/old",
			},
		},
	}
	SetAutoPromotionHoldAnnotation(promo, origin)
	require.Equal(t, "Warehouse/fake-warehouse", promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold])
	require.NotContains(t, promo.Annotations, kargoapi.AnnotationKeyAutoPromotionResume)
}

func TestSetAutoPromotionResumeAnnotation(t *testing.T) {
	origin := kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "fake-warehouse"}
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p",
			Annotations: map[string]string{
				kargoapi.AnnotationKeyAutoPromotionHold: "Warehouse/old",
			},
		},
	}
	SetAutoPromotionResumeAnnotation(promo, origin)
	require.Equal(t, "Warehouse/fake-warehouse", promo.Annotations[kargoapi.AnnotationKeyAutoPromotionResume])
	require.NotContains(t, promo.Annotations, kargoapi.AnnotationKeyAutoPromotionHold)
}
