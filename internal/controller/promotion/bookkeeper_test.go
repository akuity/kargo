package promotion

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/bookkeeper"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewBookkeeperMechanism(t *testing.T) {
	pm := newBookkeeperMechanism(
		&credentials.FakeDB{},
		bookkeeper.NewService(nil),
	)
	bpm, ok := pm.(*bookkeeperMechanism)
	require.True(t, ok)
	require.NotNil(t, bpm.doSingleUpdateFn)
	require.NotNil(t, bpm.getReadRefFn)
	require.NotNil(t, bpm.getCredentialsFn)
	require.NotNil(t, bpm.renderManifestsFn)
}

func TestBookkeeperGetName(t *testing.T) {
	require.NotEmpty(t, (&bookkeeperMechanism{}).GetName())
}

func TestBookkeeperPromote(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *bookkeeperMechanism
		stage      *api.Stage
		newState   api.StageState
		assertions func(newStateIn, newStateOut api.StageState, err error)
	}{
		{
			name:      "no updates",
			promoMech: &bookkeeperMechanism{},
			stage: &api.Stage{
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{},
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "error applying update",
			promoMech: &bookkeeperMechanism{
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ api.GitRepoUpdate,
					newState api.StageState,
					images []string,
				) (api.StageState, error) {
					require.Equal(t, []string{"fake-url:fake-tag"}, images)
					return newState, errors.New("something went wrong")
				},
			},
			stage: &api.Stage{
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						GitRepoUpdates: []api.GitRepoUpdate{
							{
								Bookkeeper: &api.BookkeeperPromotionMechanism{},
							},
						},
					},
				},
			},
			newState: api.StageState{
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "success",
			promoMech: &bookkeeperMechanism{
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ api.GitRepoUpdate,
					newState api.StageState,
					images []string,
				) (api.StageState, error) {
					require.Equal(t, []string{"fake-url:fake-tag"}, images)
					return newState, nil
				},
			},
			stage: &api.Stage{
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{
						GitRepoUpdates: []api.GitRepoUpdate{
							{
								Bookkeeper: &api.BookkeeperPromotionMechanism{},
							},
						},
					},
				},
			},
			newState: api.StageState{
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				require.Equal(t, newStateIn, newStateOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStateOut, err := testCase.promoMech.Promote(
				context.Background(),
				testCase.stage,
				testCase.newState,
			)
			testCase.assertions(testCase.newState, newStateOut, err)
		})
	}
}

func TestBookkeeperDoSingleUpdate(t *testing.T) {
	const testRef = "fake-ref"
	testCases := []struct {
		name       string
		promoMech  *bookkeeperMechanism
		update     api.GitRepoUpdate
		assertions func(newStateIn, newStateOut api.StageState, err error)
	}{
		{
			name: "error getting readref",
			promoMech: &bookkeeperMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return "", 0, errors.New("something went wrong")
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "error getting repo credentials",
			promoMech: &bookkeeperMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{},
						false,
						errors.New("something went wrong")
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "error rendering manifests",
			promoMech: &bookkeeperMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "fake-username",
						Password: "fake-personal-access-token",
					}, true, nil
				},
				renderManifestsFn: func(
					context.Context,
					bookkeeper.RenderRequest,
				) (bookkeeper.RenderResponse, error) {
					return bookkeeper.RenderResponse{}, errors.New("something went wrong")
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error rendering manifests for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "success -- no action",
			promoMech: &bookkeeperMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "fake-username",
						Password: "fake-personal-access-token",
					}, true, nil
				},
				renderManifestsFn: func(
					context.Context,
					bookkeeper.RenderRequest,
				) (bookkeeper.RenderResponse, error) {
					return bookkeeper.RenderResponse{
						ActionTaken: bookkeeper.ActionTakenNone,
						CommitID:    "fake-commit-id",
					}, nil
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					"fake-commit-id",
					newStateOut.Commits[0].HealthCheckCommit,
				)
				// The newState is otherwise unaltered
				newStateIn.Commits[0].HealthCheckCommit = ""
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "success -- commit",
			promoMech: &bookkeeperMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "fake-username",
						Password: "fake-personal-access-token",
					}, true, nil
				},
				renderManifestsFn: func(
					context.Context,
					bookkeeper.RenderRequest,
				) (bookkeeper.RenderResponse, error) {
					return bookkeeper.RenderResponse{
						ActionTaken: bookkeeper.ActionTakenPushedDirectly,
						CommitID:    "fake-commit-id",
					}, nil
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					"fake-commit-id",
					newStateOut.Commits[0].HealthCheckCommit,
				)
				// The newState is otherwise unaltered
				newStateIn.Commits[0].HealthCheckCommit = ""
				require.Equal(t, newStateIn, newStateOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStateIn := api.StageState{
				Commits: []api.GitCommit{{}},
			}
			newStateOut, err := testCase.promoMech.doSingleUpdate(
				context.Background(),
				"fake-namespace",
				testCase.update,
				newStateIn,
				nil, // Images
			)
			testCase.assertions(newStateIn, newStateOut, err)
		})
	}
}
