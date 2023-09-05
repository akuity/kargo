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
		newFreight api.Freight
		assertions func(newFreightIn, newFreightOut api.Freight, err error)
	}{
		{
			name:      "no updates",
			promoMech: &bookkeeperMechanism{},
			stage: &api.Stage{
				Spec: &api.StageSpec{
					PromotionMechanisms: &api.PromotionMechanisms{},
				},
			},
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error applying update",
			promoMech: &bookkeeperMechanism{
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ api.GitRepoUpdate,
					newFreight api.Freight,
					images []string,
				) (api.Freight, error) {
					require.Equal(t, []string{"fake-url:fake-tag"}, images)
					return newFreight, errors.New("something went wrong")
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
			newFreight: api.Freight{
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "success",
			promoMech: &bookkeeperMechanism{
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ api.GitRepoUpdate,
					newFreight api.Freight,
					images []string,
				) (api.Freight, error) {
					require.Equal(t, []string{"fake-url:fake-tag"}, images)
					return newFreight, nil
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
			newFreight: api.Freight{
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newFreightOut, err := testCase.promoMech.Promote(
				context.Background(),
				testCase.stage,
				testCase.newFreight,
			)
			testCase.assertions(testCase.newFreight, newFreightOut, err)
		})
	}
}

func TestBookkeeperDoSingleUpdate(t *testing.T) {
	const testRef = "fake-ref"
	testCases := []struct {
		name       string
		promoMech  *bookkeeperMechanism
		update     api.GitRepoUpdate
		assertions func(newFreightIn, newFreightOut api.Freight, err error)
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
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
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
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, newFreightIn, newFreightOut)
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
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error rendering manifests for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
				require.Equal(t, newFreightIn, newFreightOut)
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
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					"fake-commit-id",
					newFreightOut.Commits[0].HealthCheckCommit,
				)
				// The newFreight is otherwise unaltered
				newFreightIn.Commits[0].HealthCheckCommit = ""
				require.Equal(t, newFreightIn, newFreightOut)
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
			assertions: func(newFreightIn, newFreightOut api.Freight, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					"fake-commit-id",
					newFreightOut.Commits[0].HealthCheckCommit,
				)
				// The newFreight is otherwise unaltered
				newFreightIn.Commits[0].HealthCheckCommit = ""
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newFreightIn := api.Freight{
				Commits: []api.GitCommit{{}},
			}
			newFreightOut, err := testCase.promoMech.doSingleUpdate(
				context.Background(),
				"fake-namespace",
				testCase.update,
				newFreightIn,
				nil, // Images
			)
			testCase.assertions(newFreightIn, newFreightOut, err)
		})
	}
}
