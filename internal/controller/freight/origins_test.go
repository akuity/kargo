package freight

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetDesiredOrigin(t *testing.T) {
	testOrigin := &kargoapi.FreightOrigin{
		Kind: "Foo",
		Name: "bar",
	}
	testCases := []struct {
		name  string
		setup func() (any, any)
	}{
		{
			name: "PromotionMechanisms",
			setup: func() (any, any) {
				m := &kargoapi.PromotionMechanisms{
					Origin: testOrigin,
				}
				return m, m
			},
		},
		{
			name: "GitRepoUpdate can inherit from PromotionMechanisms",
			setup: func() (any, any) {
				m := &kargoapi.PromotionMechanisms{
					Origin:         testOrigin,
					GitRepoUpdates: []kargoapi.GitRepoUpdate{{}},
				}
				return m, &m.GitRepoUpdates[0]
			},
		},
		{
			name: "GitRepoUpdate can override PromotionMechanisms",
			setup: func() (any, any) {
				m := &kargoapi.PromotionMechanisms{
					GitRepoUpdates: []kargoapi.GitRepoUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.GitRepoUpdates[0]
			},
		},
		{
			name: "KustomizePromotionMechanism can inherit from GitRepoUpdate",
			setup: func() (any, any) {
				m := &kargoapi.GitRepoUpdate{
					Origin:    testOrigin,
					Kustomize: &kargoapi.KustomizePromotionMechanism{},
				}
				return m, m.Kustomize
			},
		},
		{
			name: "KustomizePromotionMechanism can override GitRepoUpdate",
			setup: func() (any, any) {
				m := &kargoapi.GitRepoUpdate{
					Kustomize: &kargoapi.KustomizePromotionMechanism{
						Origin: testOrigin,
					},
				}
				return m, m.Kustomize
			},
		},
		{
			name: "KustomizeImageUpdate can inherit from KustomizePromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.KustomizePromotionMechanism{
					Origin: testOrigin,
					Images: []kargoapi.KustomizeImageUpdate{{}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "KustomizeImageUpdate can override KustomizePromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.KustomizePromotionMechanism{
					Images: []kargoapi.KustomizeImageUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "HelmPromotionMechanism can inherit from GitRepoUpdate",
			setup: func() (any, any) {
				m := &kargoapi.GitRepoUpdate{
					Origin: testOrigin,
					Helm:   &kargoapi.HelmPromotionMechanism{},
				}
				return m, m.Helm
			},
		},
		{
			name: "HelmPromotionMechanism can override GitRepoUpdate",
			setup: func() (any, any) {
				m := &kargoapi.GitRepoUpdate{
					Helm: &kargoapi.HelmPromotionMechanism{
						Origin: testOrigin,
					},
				}
				return m, m.Helm
			},
		},
		{
			name: "HelmImageUpdate can inherit from HelmPromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.HelmPromotionMechanism{
					Origin: testOrigin,
					Images: []kargoapi.HelmImageUpdate{{}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "HelmImageUpdate can override HelmPromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.HelmPromotionMechanism{
					Images: []kargoapi.HelmImageUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "HelmChartDependencyUpdate can inherit from HelmPromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.HelmPromotionMechanism{
					Origin: testOrigin,
					Charts: []kargoapi.HelmChartDependencyUpdate{{}},
				}
				return m, &m.Charts[0]
			},
		},
		{
			name: "HelmChartDependencyUpdate can override HelmPromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.HelmPromotionMechanism{
					Charts: []kargoapi.HelmChartDependencyUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.Charts[0]
			},
		},
		{
			name: "KargoRenderPromotionMechanism can inherit from GitRepoUpdate",
			setup: func() (any, any) {
				m := &kargoapi.GitRepoUpdate{
					Origin: testOrigin,
					Render: &kargoapi.KargoRenderPromotionMechanism{},
				}
				return m, m.Render
			},
		},
		{
			name: "KargoRenderPromotionMechanism can override GitRepoUpdate",
			setup: func() (any, any) {
				m := &kargoapi.GitRepoUpdate{
					Render: &kargoapi.KargoRenderPromotionMechanism{
						Origin: testOrigin,
					},
				}
				return m, m.Render
			},
		},
		{
			name: "KargoRenderImageUpdate can inherit from KargoRenderPromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.KargoRenderPromotionMechanism{
					Origin: testOrigin,
					Images: []kargoapi.KargoRenderImageUpdate{{}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "KargoRenderImageUpdate can override KargoRenderPromotionMechanism",
			setup: func() (any, any) {
				m := &kargoapi.KargoRenderPromotionMechanism{
					Images: []kargoapi.KargoRenderImageUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "ArgoCDAppUpdate can inherit from PromotionMechanisms",
			setup: func() (any, any) {
				m := &kargoapi.PromotionMechanisms{
					Origin:           testOrigin,
					ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{}},
				}
				return m, &m.ArgoCDAppUpdates[0]
			},
		},
		{
			name: "ArgoCDAppUpdate can override PromotionMechanisms",
			setup: func() (any, any) {
				m := &kargoapi.PromotionMechanisms{
					Origin:           testOrigin,
					ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{}},
				}
				return m, &m.ArgoCDAppUpdates[0]
			},
		},
		{
			name: "ArgoCDSourceUpdate can inherit from ArgoCDAppUpdate",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDAppUpdate{
					Origin:        testOrigin,
					SourceUpdates: []kargoapi.ArgoCDSourceUpdate{{}},
				}
				return m, &m.SourceUpdates[0]
			},
		},
		{
			name: "ArgoCDSourceUpdate can override ArgoCDAppUpdate",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDAppUpdate{
					SourceUpdates: []kargoapi.ArgoCDSourceUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.SourceUpdates[0]
			},
		},
		{
			name: "ArgoCDKustomize can inherit from ArgoCDSourceUpdate",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDSourceUpdate{
					Origin:    testOrigin,
					Kustomize: &kargoapi.ArgoCDKustomize{},
				}
				return m, m.Kustomize
			},
		},
		{
			name: "ArgoCDKustomize can override ArgoCDSourceUpdate",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDSourceUpdate{
					Kustomize: &kargoapi.ArgoCDKustomize{
						Origin: testOrigin,
					},
				}
				return m, m.Kustomize
			},
		},
		{
			name: "ArgoCDKustomizeImageUpdate can inherit from ArgoCDKustomize",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDKustomize{
					Origin: testOrigin,
					Images: []kargoapi.ArgoCDKustomizeImageUpdate{{}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "ArgoCDKustomizeImageUpdate can override ArgoCDKustomize",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDKustomize{
					Images: []kargoapi.ArgoCDKustomizeImageUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "ArgoCDHelm can inherit from ArgoCDSourceUpdate",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDSourceUpdate{
					Origin: testOrigin,
					Helm:   &kargoapi.ArgoCDHelm{},
				}
				return m, m.Helm
			},
		},
		{
			name: "ArgoCDHelm can override ArgoCDSourceUpdate",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDSourceUpdate{
					Helm: &kargoapi.ArgoCDHelm{
						Origin: testOrigin,
					},
				}
				return m, m.Helm
			},
		},
		{
			name: "ArgoCDHelmImageUpdate can inherit from ArgoCDHelm",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDHelm{
					Origin: testOrigin,
					Images: []kargoapi.ArgoCDHelmImageUpdate{{}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "ArgoCDHelmImageUpdate can override ArgoCDHelm",
			setup: func() (any, any) {
				m := &kargoapi.ArgoCDHelm{
					Images: []kargoapi.ArgoCDHelmImageUpdate{{
						Origin: testOrigin,
					}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "transitive inheritance",
			setup: func() (any, any) {
				m := &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionMechanisms: &kargoapi.PromotionMechanisms{
							Origin: testOrigin,
							GitRepoUpdates: []kargoapi.GitRepoUpdate{{
								Kustomize: &kargoapi.KustomizePromotionMechanism{
									Images: []kargoapi.KustomizeImageUpdate{{}},
								},
							}},
						},
					},
				}
				return m, &m.Spec.PromotionMechanisms.GitRepoUpdates[0].Kustomize.Images[0]
			},
		},
		{
			name: "override transitive inheritance",
			setup: func() (any, any) {
				m := &kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionMechanisms: &kargoapi.PromotionMechanisms{
							GitRepoUpdates: []kargoapi.GitRepoUpdate{{
								Kustomize: &kargoapi.KustomizePromotionMechanism{
									Images: []kargoapi.KustomizeImageUpdate{{
										Origin: testOrigin,
									}},
								},
							}},
						},
					},
				}
				return m, &m.Spec.PromotionMechanisms.GitRepoUpdates[0].Kustomize.Images[0]
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mechanism, targetMechanism := tc.setup()
			actual := GetDesiredOrigin(mechanism, targetMechanism)
			require.Same(t, testOrigin, actual)
		})
	}
}
