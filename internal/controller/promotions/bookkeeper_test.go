package promotions

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/akuityio/bookkeeper"
	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/credentials"
)

func TestApplyBookkeeperUpdate(t *testing.T) {
	testCases := []struct {
		name              string
		newState          api.EnvironmentState
		update            api.GitRepoUpdate
		credentialsDB     credentials.Database
		bookkeeperService bookkeeper.Service
		assertions        func(inState, outState api.EnvironmentState, err error)
	}{
		{
			name: "update doesn't actually use Bookkeeper",
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Equal(t, inState, outState)
			},
		},

		{
			name: "target branch is unspecified",
			update: api.GitRepoUpdate{
				Bookkeeper: &api.BookkeeperPromotionMechanism{},
			},
			assertions: func(_, _ api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no target branch is specified")
			},
		},

		{
			name: "state doesn't contain any information about the repo",
			update: api.GitRepoUpdate{
				Branch:     "env/fake",
				Bookkeeper: &api.BookkeeperPromotionMechanism{},
			},
			assertions: func(_, _ api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"environment does not subscribe to repo",
				)
			},
		},

		{
			name: "error getting Git repo credentials",
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-git-url",
						ID:      "fake-commit",
					},
				},
				Images: []api.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
				},
			},
			update: api.GitRepoUpdate{
				RepoURL:    "fake-git-url",
				Branch:     "env/fake",
				Bookkeeper: &api.BookkeeperPromotionMechanism{},
			},
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false,
						errors.New("something went wrong")
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, inState, outState)
			},
		},

		{
			name: "error rendering manifests",
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-git-url",
						ID:      "fake-commit",
					},
				},
				Images: []api.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
				},
			},
			update: api.GitRepoUpdate{
				RepoURL:    "fake-git-url",
				Branch:     "env/fake",
				Bookkeeper: &api.BookkeeperPromotionMechanism{},
			},
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			bookkeeperService: &fakeBookkeeperService{
				renderManifestsFn: func(
					context.Context,
					bookkeeper.RenderRequest,
				) (bookkeeper.RenderResponse, error) {
					return bookkeeper.RenderResponse{}, errors.New("something went wrong")
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error rendering manifests via Bookkeeper",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, inState, outState)
			},
		},

		{
			name: "success",
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-git-url",
						ID:      "fake-commit",
					},
				},
				Images: []api.Image{
					{
						RepoURL: "fake-image-url",
						Tag:     "fake-tag",
					},
				},
			},
			update: api.GitRepoUpdate{
				RepoURL:    "fake-git-url",
				Branch:     "env/fake",
				Bookkeeper: &api.BookkeeperPromotionMechanism{},
			},
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			bookkeeperService: &fakeBookkeeperService{
				renderManifestsFn: func(
					context.Context,
					bookkeeper.RenderRequest,
				) (bookkeeper.RenderResponse, error) {
					return bookkeeper.RenderResponse{
						ActionTaken: bookkeeper.ActionTakenPushedDirectly,
						CommitID:    "new-fake-commit",
					}, nil
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Len(t, outState.Commits, 1)
				require.Equal(
					t, "new-fake-commit",
					outState.Commits[0].HealthCheckCommit,
				)
				// inState and outState should otherwise match
				outState.Commits[0].HealthCheckCommit = ""
				require.Equal(t, inState, outState)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := reconciler{
				credentialsDB:     testCase.credentialsDB,
				bookkeeperService: testCase.bookkeeperService,
			}
			newState, err := reconciler.applyBookkeeperUpdate(
				context.Background(),
				"fake-namespace",
				testCase.newState,
				testCase.update,
			)
			testCase.assertions(testCase.newState, newState, err)
		})
	}
}

type fakeBookkeeperService struct {
	renderManifestsFn func(
		context.Context,
		bookkeeper.RenderRequest,
	) (bookkeeper.RenderResponse, error)
}

func (f *fakeBookkeeperService) RenderManifests(
	ctx context.Context,
	req bookkeeper.RenderRequest,
) (bookkeeper.RenderResponse, error) {
	return f.renderManifestsFn(ctx, req)
}
