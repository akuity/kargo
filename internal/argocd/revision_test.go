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
		freight []kargoapi.FreightReference
		want    []string
	}{
		{
			name: "no application",
			want: nil,
		},
		{
			name: "no sources",
			app:  &argocdapi.Application{},
			want: nil,
		},
		{
			name: "multisource",
			app: &argocdapi.Application{
				Spec: argocdapi.ApplicationSpec{
					Sources: []argocdapi.ApplicationSource{
						{
							// This has no repoURL. This probably cannot actually happen, but
							// our logic says we'll have an empty string (no desired revision)
							// in this case.
						},
						{
							// This has an update and a matching artifact in the Freight. We
							// should know what revision we want.
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
						{
							// This has no matching update, but does have a matching artifact
							// in the Freight. We should know what revision we want.
							RepoURL: "https://example.com",
							Chart:   "another-fake-chart",
						},
						{
							// This has no matching update, but does have a matching artifact
							// in the Freight. We should know what revision we want.
							//
							// OCI is a special case.
							RepoURL: "example.com",
							Chart:   "fake-chart",
						},
						{
							// This has no matching artifact in the Freight. We should not
							// know what revision we want.
							RepoURL: "https://example.com",
							Chart:   "yet-another-fake-chart",
						},
						{
							// This has an update and a matching artifact in the Freight. We
							// should know what revision we want.
							RepoURL: "https://github.com/universe/42",
						},
						{
							// This has no matching update, but does have a matching artifact
							// in the Freight. We should know what revision we want.
							RepoURL: "https://github.com/another-universe/42",
						},
						{
							// This has no matching artifact in the Freight. We should not
							// know what revision we want.
							RepoURL: "https://github.com/yet-another-universe/42",
						},
					},
				},
			},
			freight: []kargoapi.FreightReference{
				{
					Origin: testOrigin,
					Charts: []kargoapi.Chart{
						{
							RepoURL: "https://example.com",
							Name:    "fake-chart",
							Version: "v2.0.0",
						},
						{
							RepoURL: "https://example.com",
							Name:    "another-fake-chart",
							Version: "v1.0.0",
						},
						{
							RepoURL: "oci://example.com/fake-chart",
							Version: "v3.0.0",
						},
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "https://github.com/universe/42",
							ID:      "fake-commit",
						},
						{
							RepoURL: "https://github.com/another-universe/42",
							ID:      "another-fake-commit",
						},
					},
				},
			},
			want: []string{"", "v2.0.0", "v1.0.0", "v3.0.0", "", "fake-commit", "another-fake-commit", ""},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stage := &kargoapi.Stage{
				Spec: kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						ArgoCDAppUpdates: []kargoapi.ArgoCDAppUpdate{{
							Origin: &testOrigin,
							SourceUpdates: []kargoapi.ArgoCDSourceUpdate{
								{
									RepoURL: "https://example.com",
									Chart:   "fake-chart",
								},
								{
									RepoURL: "https://github.com/universe/42",
								},
							},
						}},
					},
				},
			}
			revisions, err := GetDesiredRevisions(
				context.Background(),
				nil, // No client is needed as long as we're always explicit about origins
				stage,
				&stage.Spec.PromotionMechanisms.ArgoCDAppUpdates[0], // nolint: staticcheck
				testCase.app,
				testCase.freight,
			)
			require.NoError(t, err)
			require.Equal(t, testCase.want, revisions)
		})
	}
}
