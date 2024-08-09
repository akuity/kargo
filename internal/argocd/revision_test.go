package argocd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocdapi "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func TestGetDesiredRevisions(t *testing.T) {
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name    string
		app     *argocdapi.Application
		freight kargoapi.FreightReference
		want    []string
	}{
		{
			name: "no application",
			want: []string{},
		},
		{
			name: "no application source",
			app:  &argocdapi.Application{},
			want: []string{},
		},
		{
			name: "no source repo URL",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{},
				},
			},
			want: []string{},
		},
		{
			name: "chart source",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{
						RepoURL: "https://example.com",
						Chart:   "fake-chart",
					},
				},
			},
			freight: kargoapi.FreightReference{
				Origin: testOrigin,
				Charts: []kargoapi.Chart{
					{
						RepoURL: "https://example.com",
						Name:    "other-fake-chart",
						Version: "v1.0.0",
					},
					{
						RepoURL: "https://example.com",
						Name:    "fake-chart",
						Version: "v2.0.0",
					},
				},
			},
			want: []string{"v2.0.0"},
		},
		{
			name: "chart sources",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Sources: argocdapi.ApplicationSources{
						{
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
						{
							RepoURL: "https://example.com",
							Chart:   "other-fake-chart",
						},
					},
				},
			},
			freight: kargoapi.FreightReference{
				Origin: testOrigin,
				Charts: []kargoapi.Chart{
					{
						RepoURL: "https://example.com",
						Name:    "fake-chart",
						Version: "v1.0.0",
					},
					{
						RepoURL: "https://example.com",
						Name:    "other-fake-chart",
						Version: "v2.0.0",
					},
				},
			},
			want: []string{"v1.0.0", "v2.0.0"},
		},
		{
			name: "git source",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{
						RepoURL: "https://github.com/universe/42",
					},
				},
			},
			freight: kargoapi.FreightReference{
				Origin: testOrigin,
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "https://github.com/bad/41",
						ID:      "bad-revision",
					},
					{
						RepoURL: "https://github.com/universe/42",
						ID:      "fake-revision",
					},
				},
			},
			want: []string{"fake-revision"},
		},
		{
			name: "git sources",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Sources: argocdapi.ApplicationSources{
						{
							RepoURL: "https://github.com/universe/42",
						},
						{
							RepoURL: "https://github.com/universe/43",
						},
					},
				},
			},
			freight: kargoapi.FreightReference{
				Origin: testOrigin,
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "https://github.com/universe/42",
						ID:      "fake-revision",
					},
					{
						RepoURL: "https://github.com/universe/43",
						ID:      "other-fake-revision",
					},
				},
			},
			want: []string{"fake-revision", "other-fake-revision"},
		},
		{
			name: "mixed sources",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Sources: argocdapi.ApplicationSources{
						{
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
						{
							RepoURL: "https://github.com/universe/42",
						},
					},
				},
			},
			freight: kargoapi.FreightReference{
				Origin: testOrigin,
				Charts: []kargoapi.Chart{
					{
						RepoURL: "https://example.com",
						Name:    "fake-chart",
						Version: "v1.0.0",
					},
				},
				Commits: []kargoapi.GitCommit{
					{
						RepoURL: "https://github.com/universe/42",
						ID:      "fake-revision",
					},
				},
			},
			want: []string{"v1.0.0", "fake-revision"},
		},
		{
			name: "git source with health check commit",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Source: &argocdapi.ApplicationSource{
						RepoURL: "https://github.com/universe/42",
					},
				},
			},
			freight: kargoapi.FreightReference{
				Origin: testOrigin,
				Commits: []kargoapi.GitCommit{
					{
						RepoURL:           "https://github.com/universe/42",
						HealthCheckCommit: "fake-revision",
						ID:                "bad-revision",
					},
				},
			},
			want: []string{"fake-revision"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stage := &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
							Origin: &testOrigin,
						}},
					},
				},
				Status: kargoapi.StageStatus{
					FreightHistory: kargoapi.FreightHistory{{
						Freight: map[string]kargoapi.FreightReference{
							testOrigin.String(): testCase.freight,
						},
					}},
				},
			}
			revisions, err := GetDesiredRevisions(
				context.Background(),
				nil, // No client is needed as long as we're always explicit about origins
				stage,
				&stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0],
				testCase.app,
				stage.Status.FreightHistory.Current().References(),
			)
			require.NoError(t, err)
			require.Equal(t, testCase.want, revisions)
		})
	}
}
