package controller

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	api "github.com/akuityio/kargo/api/v1alpha1"
)

func TestApplyKustomize(t *testing.T) {
	testCases := []struct {
		name                string
		newState            api.EnvironmentState
		update              api.KustomizePromotionMechanism
		kustomizeSetImageFn func(dir, repo, tag string) error
		assertions          func(error)
	}{
		{
			name: "error setting image",
			newState: api.EnvironmentState{
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			update: api.KustomizePromotionMechanism{
				Images: []api.KustomizeImageUpdate{
					{
						Image: "fake-url",
						Path:  "/fake/path",
					},
				},
			},
			kustomizeSetImageFn: func(string, string, string) error {
				return errors.New("something went wrong")
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error updating image")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			newState: api.EnvironmentState{
				Images: []api.Image{
					{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
				},
			},
			update: api.KustomizePromotionMechanism{
				Images: []api.KustomizeImageUpdate{
					{
						Image: "fake-url",
						Path:  "/fake/path",
					},
				},
			},
			kustomizeSetImageFn: func(string, string, string) error {
				return nil
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reconciler := promotionReconciler{
				kustomizeSetImageFn: testCase.kustomizeSetImageFn,
			}
			testCase.assertions(
				reconciler.applyKustomize(
					testCase.newState,
					testCase.update,
					"",
				),
			)
		})
	}
}
