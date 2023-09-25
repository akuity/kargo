package images

import (
	"testing"

	"github.com/Masterminds/semver"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestGetLatestTag(t *testing.T) {
	testCases := []struct {
		name             string
		repoURL          string
		platform         string
		semverConstraint string
		assertions       func(string, error)
	}{
		{
			name:     "error parsing platform",
			repoURL:  "nginx",
			platform: "bogus",
			assertions: func(s string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error parsing platform")
			},
		},

		{
			name: "error getting tags",
			// This will force a failure because this repo doesn't exist
			repoURL: "bogus",
			assertions: func(s string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error fetching tags for image")
			},
		},

		{
			name:             "no suitable version found",
			repoURL:          "nginx",
			semverConstraint: "^15.0.0", // Doesn't exist
			assertions: func(tag string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found no suitable version of image")
			},
		},

		{
			name:             "success",
			repoURL:          "nginx",
			platform:         "linux/amd64",
			semverConstraint: "^1.0.0",
			assertions: func(tag string, err error) {
				require.NoError(t, err)
				ver, err := semver.NewVersion(tag)
				require.NoError(t, err)
				require.Equal(t, int64(1), ver.Major())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				GetLatestTag(
					testCase.repoURL,
					kargoapi.ImageUpdateStrategySemVer,
					testCase.semverConstraint,
					"",
					nil,
					testCase.platform,
					&Credentials{},
				),
			)
		})
	}
}
