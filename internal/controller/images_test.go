package controller

import (
	"context"
	"errors"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/images"
)

func TestGetLatestImages(t *testing.T) {
	testCases := []struct {
		name           string
		getLatestTagFn func(
			context.Context,
			kubernetes.Interface,
			string,
			images.ImageUpdateStrategy,
			string,
			string,
			[]string,
			string,
			string,
		) (string, error)
		assertions func([]api.Image, error)
	}{
		{
			name: "error getting latest version of an image",
			getLatestTagFn: func(
				ctx context.Context,
				kubeClient kubernetes.Interface,
				repoURL string,
				updateStrategy images.ImageUpdateStrategy,
				semverConstraint string,
				allowTags string,
				ignoreTags []string,
				platform string,
				pullSecret string,
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_ []api.Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting latest suitable tag for image",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			getLatestTagFn: func(
				ctx context.Context,
				kubeClient kubernetes.Interface,
				repoURL string,
				updateStrategy images.ImageUpdateStrategy,
				semverConstraint string,
				allowTags string,
				ignoreTags []string,
				platform string,
				pullSecret string,
			) (string, error) {
				return "fake-tag", nil
			},
			assertions: func(images []api.Image, err error) {
				require.NoError(t, err)
				require.Len(t, images, 1)
				require.Equal(
					t,
					api.Image{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
					images[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testSubs := []api.ImageSubscription{
				{
					RepoURL: "fake-url",
				},
			}
			reconciler := environmentReconciler{
				logger:         log.New(),
				getLatestTagFn: testCase.getLatestTagFn,
			}
			reconciler.logger.SetLevel(log.ErrorLevel)
			testCase.assertions(
				reconciler.getLatestImages(context.Background(), testSubs),
			)
		})
	}
}
