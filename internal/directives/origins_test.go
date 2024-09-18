package directives

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetDesiredOrigin(t *testing.T) {
	testOrigin := &AppFromOrigin{
		Kind: "Foo",
		Name: "bar",
	}
	expectedOrigin := &kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKind(testOrigin.Kind),
		Name: testOrigin.Name,
	}
	testCases := []struct {
		name  string
		setup func() (any, any)
	}{
		{
			name: "ArgoCDUpdateConfig",
			setup: func() (any, any) {
				m := &ArgoCDUpdateConfig{
					FromOrigin: testOrigin,
				}
				return m, m
			},
		},
		{
			name: "ArgoCDAppUpdate can inherit from ArgoCDUpdateConfig",
			setup: func() (any, any) {
				m := &ArgoCDUpdateConfig{
					FromOrigin: testOrigin,
					Apps:       []ArgoCDAppUpdate{{}},
				}
				return m, &m.Apps[0]
			},
		},
		{
			name: "ArgoCDAppUpdate can override ArgoCDUpdateConfig",
			setup: func() (any, any) {
				m := &ArgoCDUpdateConfig{
					Apps: []ArgoCDAppUpdate{{
						FromOrigin: testOrigin,
					}},
				}
				return m, &m.Apps[0]
			},
		},
		{
			name: "ArgoCDAppSourceUpdate can inherit from ArgoCDAppUpdate",
			setup: func() (any, any) {
				m := &ArgoCDAppUpdate{
					FromOrigin: testOrigin,
					Sources:    []ArgoCDAppSourceUpdate{{}},
				}
				return m, &m.Sources[0]
			},
		},
		{
			name: "ArgoCDAppSourceUpdate can override ArgoCDAppUpdate",
			setup: func() (any, any) {
				m := &ArgoCDAppUpdate{
					Sources: []ArgoCDAppSourceUpdate{{
						FromOrigin: testOrigin,
					}},
				}
				return m, &m.Sources[0]
			},
		},
		{
			name: "ArgoCDKustomizeImageUpdates can inherit from ArgoCDAppSourceUpdate",
			setup: func() (any, any) {
				m := &ArgoCDAppSourceUpdate{
					FromOrigin: testOrigin,
					Kustomize:  &ArgoCDKustomizeImageUpdates{},
				}
				return m, m.Kustomize
			},
		},
		{
			name: "ArgoCDKustomizeImageUpdates can override ArgoCDAppSourceUpdate",
			setup: func() (any, any) {
				m := &ArgoCDAppSourceUpdate{
					Kustomize: &ArgoCDKustomizeImageUpdates{
						FromOrigin: testOrigin,
					},
				}
				return m, m.Kustomize
			},
		},
		{
			name: "ArgoCDKustomizeImageUpdate can inherit from ArgoCDKustomizeImageUpdates",
			setup: func() (any, any) {
				m := &ArgoCDKustomizeImageUpdates{
					FromOrigin: testOrigin,
					Images:     []ArgoCDKustomizeImageUpdate{{}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "ArgoCDKustomizeImageUpdate can override ArgoCDKustomizeImageUpdates",
			setup: func() (any, any) {
				m := &ArgoCDKustomizeImageUpdates{
					Images: []ArgoCDKustomizeImageUpdate{{
						FromOrigin: testOrigin,
					}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "ArgoCDHelmParameterUpdates can inherit from ArgoCDAppSourceUpdate",
			setup: func() (any, any) {
				m := &ArgoCDAppSourceUpdate{
					FromOrigin: testOrigin,
					Helm:       &ArgoCDHelmParameterUpdates{},
				}
				return m, m.Helm
			},
		},
		{
			name: "ArgoCDHelmParameterUpdates can override ArgoCDAppSourceUpdate",
			setup: func() (any, any) {
				m := &ArgoCDAppSourceUpdate{
					Helm: &ArgoCDHelmParameterUpdates{
						FromOrigin: testOrigin,
					},
				}
				return m, m.Helm
			},
		},
		{
			name: "ArgoCDHelmImageUpdate can inherit from ArgoCDHelmParameterUpdates",
			setup: func() (any, any) {
				m := &ArgoCDHelmParameterUpdates{
					FromOrigin: testOrigin,
					Images:     []ArgoCDHelmImageUpdate{{}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "ArgoCDHelmImageUpdate can override ArgoCDHelmParameterUpdates",
			setup: func() (any, any) {
				m := &ArgoCDHelmParameterUpdates{
					Images: []ArgoCDHelmImageUpdate{{
						FromOrigin: testOrigin,
					}},
				}
				return m, &m.Images[0]
			},
		},
		{
			name: "transitive inheritance",
			setup: func() (any, any) {
				m := &ArgoCDUpdateConfig{
					FromOrigin: testOrigin,
					Apps: []ArgoCDAppUpdate{{
						Sources: []ArgoCDAppSourceUpdate{{
							Kustomize: &ArgoCDKustomizeImageUpdates{
								Images: []ArgoCDKustomizeImageUpdate{{}},
							},
						}},
					}},
				}
				return m, &m.Apps[0].Sources[0].Kustomize.Images[0]
			},
		},
		{
			name: "override transitive inheritance",
			setup: func() (any, any) {
				m := &ArgoCDUpdateConfig{
					Apps: []ArgoCDAppUpdate{{
						Sources: []ArgoCDAppSourceUpdate{{
							Kustomize: &ArgoCDKustomizeImageUpdates{
								Images: []ArgoCDKustomizeImageUpdate{{
									FromOrigin: testOrigin,
								}},
							},
						}},
					}},
				}
				return m, &m.Apps[0].Sources[0].Kustomize.Images[0]
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mechanism, targetMechanism := tc.setup()
			actual := getDesiredOrigin(mechanism, targetMechanism)
			require.Equal(t, expectedOrigin, actual)
		})
	}
}
