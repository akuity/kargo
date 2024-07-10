package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	render "github.com/akuity/kargo/internal/kargo-render"
)

func TestNewKargoRenderMechanism(t *testing.T) {
	pm := newKargoRenderMechanism(
		fake.NewFakeClient(),
		&credentials.FakeDB{},
	)
	kpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.Equal(t, "Kargo Render promotion mechanism", kpm.name)
	require.NotNil(t, kpm.client)
	require.NotNil(t, kpm.selectUpdatesFn)
	require.NotNil(t, kpm.applyConfigManagementFn)
}

func TestSelectKargoRenderUpdates(t *testing.T) {
	testCases := []struct {
		name       string
		updates    []kargoapi.GitRepoUpdate
		assertions func(*testing.T, []kargoapi.GitRepoUpdate)
	}{
		{
			name: "no updates",
			assertions: func(t *testing.T, selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "no kargo render updates",
			updates: []kargoapi.GitRepoUpdate{
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(t *testing.T, selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Empty(t, selectedUpdates)
			},
		},
		{
			name: "some kargo render updates",
			updates: []kargoapi.GitRepoUpdate{
				{
					RepoURL: "fake-url",
					Render:  &kargoapi.KargoRenderPromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
					Helm:    &kargoapi.HelmPromotionMechanism{},
				},
				{
					RepoURL: "fake-url",
				},
			},
			assertions: func(t *testing.T, selectedUpdates []kargoapi.GitRepoUpdate) {
				require.Len(t, selectedUpdates, 1)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, selectKargoRenderUpdates(testCase.updates))
		})
	}
}

func TestKargoRenderApply(t *testing.T) {
	testRenderedManifestName := "fake-filename"
	testRenderedManifest := []byte("fake-rendered-manifest")
	testSourceCommitID := "fake-commit-id"
	testOrigin := kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKindWarehouse,
		Name: "fake-warehouse",
	}
	testCases := []struct {
		name       string
		update     kargoapi.GitRepoUpdate
		newFreight []kargoapi.FreightReference
		renderer   *renderer
		assertions func(t *testing.T, changes []string, workDir string, err error)
	}{
		{
			name: "error running Kargo Render",
			update: kargoapi.GitRepoUpdate{
				Render: &kargoapi.KargoRenderPromotionMechanism{},
			},
			renderer: &renderer{
				renderManifestsFn: func(render.Request) error {
					return errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ []string, _ string, err error) {
				require.ErrorContains(t, err, "error rendering manifests via Kargo Render")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "update doesn't specify images",
			update: kargoapi.GitRepoUpdate{
				Render: &kargoapi.KargoRenderPromotionMechanism{
					Origin: &testOrigin,
				},
			},
			newFreight: []kargoapi.FreightReference{{
				Origin: testOrigin,
				Images: []kargoapi.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
						Digest:  "fake-digest",
					},
				},
			}},
			renderer: &renderer{
				renderManifestsFn: func(req render.Request) error {
					if err := os.MkdirAll(req.LocalOutPath, 0755); err != nil {
						return err
					}
					return os.WriteFile(
						filepath.Join(req.LocalOutPath, testRenderedManifestName),
						testRenderedManifest,
						0600,
					)
				},
			},
			assertions: func(t *testing.T, changeSummary []string, workDir string, err error) {
				require.NoError(t, err)
				// The work directory should contain the rendered manifest
				files, err := os.ReadDir(workDir)
				require.NoError(t, err)
				require.Len(t, files, 1)
				require.Equal(t, testRenderedManifestName, files[0].Name())
				contents, err := os.ReadFile(filepath.Join(workDir, testRenderedManifestName))
				require.NoError(t, err)
				require.Equal(t, testRenderedManifest, contents)
				// Inspect the change summary
				require.Equal(
					t,
					[]string{
						fmt.Sprintf("rendered manifests from commit %s", testSourceCommitID[:7]),
						"updated manifests to use image fake-url:fake-tag",
					},
					changeSummary,
				)
			},
		},
		// {
		// 	name: "update specifies images",
		// 	update: kargoapi.GitRepoUpdate{
		// 		Render: &kargoapi.KargoRenderPromotionMechanism{
		// 			Origin: &testOrigin,
		// 			Images: []kargoapi.KargoRenderImageUpdate{
		// 				{
		// 					Image: "fake-url",
		// 				},
		// 				{
		// 					Image:     "another-fake-url",
		// 					UseDigest: true,
		// 				},
		// 			},
		// 		},
		// 	},
		// 	newFreight: []kargoapi.FreightReference{{
		// 		Origin: testOrigin,
		// 		Images: []kargoapi.Image{
		// 			{
		// 				RepoURL: "fake-url",
		// 				Tag:     "fake-tag",
		// 				Digest:  "fake-digest",
		// 			},
		// 			{
		// 				RepoURL: "another-fake-url",
		// 				Tag:     "another-fake-tag",
		// 				Digest:  "another-fake-digest",
		// 			},
		// 		},
		// 	}},
		// 	renderer: &renderer{
		// 		renderManifestsFn: func(req render.Request) error {
		// 			if err := os.MkdirAll(req.LocalOutPath, 0755); err != nil {
		// 				return err
		// 			}
		// 			return os.WriteFile(
		// 				filepath.Join(req.LocalOutPath, testRenderedManifestName),
		// 				testRenderedManifest,
		// 				0600,
		// 			)
		// 		},
		// 	},
		// 	assertions: func(t *testing.T, changeSummary []string, workDir string, err error) {
		// 		require.NoError(t, err)
		// 		// The work directory should contain the rendered manifest
		// 		files, err := os.ReadDir(workDir)
		// 		require.NoError(t, err)
		// 		require.Len(t, files, 1)
		// 		require.Equal(t, testRenderedManifestName, files[0].Name())
		// 		contents, err := os.ReadFile(filepath.Join(workDir, testRenderedManifestName))
		// 		require.NoError(t, err)
		// 		require.Equal(t, testRenderedManifest, contents)
		// 		// Inspect the change summary
		// 		require.Equal(
		// 			t,
		// 			[]string{
		// 				fmt.Sprintf("rendered manifests from commit %s", testSourceCommitID[:7]),
		// 				"updated manifests to use image another-fake-url@another-fake-digest",
		// 				"updated manifests to use image fake-url:fake-tag",
		// 			},
		// 			changeSummary,
		// 		)
		// 	},
		// },
	}
	for _, testCase := range testCases {
		stage := &kargoapi.Stage{
			Spec: kargoapi.StageSpec{
				PromotionMechanisms: &kargoapi.PromotionMechanisms{
					GitRepoUpdates: []kargoapi.GitRepoUpdate{testCase.update},
				},
			},
		}
		testWorkDir := t.TempDir()
		t.Run(testCase.name, func(t *testing.T) {
			changes, err := testCase.renderer.apply(
				context.Background(),
				stage,
				&stage.Spec.PromotionMechanisms.GitRepoUpdates[0],
				testCase.newFreight,
				testSourceCommitID,
				"", // Home directory is not used by this implementation
				testWorkDir,
				git.RepoCredentials{},
			)
			testCase.assertions(t, changes, testWorkDir, err)
		})
	}
}
