package directives

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocdapi "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

func Test_argoCDUpdater_getDesiredRevisions(t *testing.T) {
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
							// This has a matching artifact in the Freight, but no update
							// that specifies the desired revision.
							//
							// Before v1.1, we would have inferred the desired revision from
							// the Freight.
							//
							// Beginning with v1.1, we make no attempt to infer the desired
							// revision when it is not explicitly specified.
							//
							// This case is here purely as validation of the updated behavior.
							RepoURL: "https://example.com",
							Chart:   "fake-chart",
						},
						{
							// This has an update that directly specifies the desired
							// revision.
							RepoURL: "https://example.com",
							Chart:   "another-fake-chart",
						},
						{
							// This has a matching artifact in the Freight, but no update
							// that specifies the desired revision.
							//
							// Before v1.1, we would have inferred the desired revision from
							// the Freight.
							//
							// Beginning with v1.1, we make no attempt to infer the desired
							// revision when it is not explicitly specified.
							//
							// This case is here purely as validation of the updated behavior.
							RepoURL: "https://github.com/universe/42",
						},
						{
							// This has an update that directly specifies the desired
							// revision.
							RepoURL: "https://github.com/another-universe/42",
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
							Version: "fake-version",
						},
					},
					Commits: []kargoapi.GitCommit{
						{
							RepoURL: "https://github.com/universe/42",
							ID:      "fake-commit",
						},
					},
				},
			},
			want: []string{"", "", "another-fake-version", "", "another-fake-commit"},
		},
	}

	runner := &argocdUpdater{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			stepCtx := &PromotionStepContext{
				Freight: kargoapi.FreightCollection{},
			}
			for _, freight := range testCase.freight {
				stepCtx.Freight.UpdateOrPush(freight)
			}
			revisions, err := runner.getDesiredRevisions(
				stepCtx,
				&ArgoCDAppUpdate{
					FromOrigin: &AppFromOrigin{
						Kind: Kind(testOrigin.Kind),
						Name: testOrigin.Name,
					},
					Sources: []ArgoCDAppSourceUpdate{
						{
							RepoURL:         "https://example.com",
							Chart:           "another-fake-chart",
							DesiredRevision: "another-fake-version",
						},
						{
							RepoURL:         "https://github.com/another-universe/42",
							DesiredRevision: "another-fake-commit",
						},
					},
				},
				testCase.app,
			)
			require.NoError(t, err)
			require.Equal(t, testCase.want, revisions)
		})
	}
}
