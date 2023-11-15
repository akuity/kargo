package promotion

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	render "github.com/akuity/kargo/internal/kargo-render"
)

func TestNewKargoRenderMechanism(t *testing.T) {
	pm := newKargoRenderMechanism(&credentials.FakeDB{})
	krpm, ok := pm.(*kargoRenderMechanism)
	require.True(t, ok)
	require.NotNil(t, krpm.doSingleUpdateFn)
	require.NotNil(t, krpm.getReadRefFn)
	require.NotNil(t, krpm.getCredentialsFn)
	require.NotNil(t, krpm.renderManifestsFn)
}

func TestKargoRenderGetName(t *testing.T) {
	require.NotEmpty(t, (&kargoRenderMechanism{}).GetName())
}

func TestKargoRenderPromote(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *kargoRenderMechanism
		stage      *kargoapi.Stage
		newFreight kargoapi.SimpleFreight
		assertions func(
			newFreightIn kargoapi.SimpleFreight,
			newFreightOut kargoapi.SimpleFreight,
			err error,
		)
	}{
		{
			name:      "no updates",
			promoMech: &kargoRenderMechanism{},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{},
				},
			},
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error applying update",
			promoMech: &kargoRenderMechanism{
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ kargoapi.GitRepoUpdate,
					newFreight kargoapi.SimpleFreight,
					images []string,
				) (kargoapi.SimpleFreight, error) {
					require.Equal(t, []string{"fake-url:fake-tag"}, images)
					return newFreight, errors.New("something went wrong")
				},
			},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						GitRepoUpdates: []kargoapi.GitRepoUpdate{
							{
								Render: &kargoapi.KargoRenderPromotionMechanism{},
							},
						},
					},
				},
			},
			newFreight: kargoapi.SimpleFreight{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "success",
			promoMech: &kargoRenderMechanism{
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ kargoapi.GitRepoUpdate,
					newFreight kargoapi.SimpleFreight,
					images []string,
				) (kargoapi.SimpleFreight, error) {
					require.Equal(t, []string{"fake-url:fake-tag"}, images)
					return newFreight, nil
				},
			},
			stage: &kargoapi.Stage{
				Spec: &kargoapi.StageSpec{
					PromotionMechanisms: &kargoapi.PromotionMechanisms{
						GitRepoUpdates: []kargoapi.GitRepoUpdate{
							{
								Render: &kargoapi.KargoRenderPromotionMechanism{},
							},
						},
					},
				},
			},
			newFreight: kargoapi.SimpleFreight{
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
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

func TestKargoRenderDoSingleUpdate(t *testing.T) {
	const testRef = "fake-ref"
	testCases := []struct {
		name       string
		promoMech  *kargoRenderMechanism
		update     kargoapi.GitRepoUpdate
		assertions func(
			newFreightIn kargoapi.SimpleFreight,
			newFreightOut kargoapi.SimpleFreight,
			err error,
		)
	}{
		{
			name: "error getting readref",
			promoMech: &kargoRenderMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
				) (string, int, error) {
					return "", 0, errors.New("something went wrong")
				},
			},
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error getting repo credentials",
			promoMech: &kargoRenderMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
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
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
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
			promoMech: &kargoRenderMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
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
				renderManifestsFn: func(render.Request) (render.Response, error) {
					return render.Response{}, errors.New("something went wrong")
				},
			},
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
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
			promoMech: &kargoRenderMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
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
				renderManifestsFn: func(render.Request) (render.Response, error) {
					return render.Response{
						ActionTaken: render.ActionTakenNone,
						CommitID:    "fake-commit-id",
					}, nil
				},
			},
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
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
			promoMech: &kargoRenderMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
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
				renderManifestsFn: func(render.Request) (render.Response, error) {
					return render.Response{
						ActionTaken: render.ActionTakenPushedDirectly,
						CommitID:    "fake-commit-id",
					}, nil
				},
			},
			assertions: func(
				newFreightIn kargoapi.SimpleFreight,
				newFreightOut kargoapi.SimpleFreight,
				err error,
			) {
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
			newFreightIn := kargoapi.SimpleFreight{
				Commits: []kargoapi.GitCommit{{}},
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
