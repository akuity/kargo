package image

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewImage(t *testing.T) {
	const testDigest = "fake-digest"
	testDate := time.Now().UTC()
	testCases := []struct {
		name       string
		tag        string
		assertions func(*testing.T, image)
	}{
		{
			name: "tag is not a semver",
			tag:  "fake-tag",
			assertions: func(t *testing.T, image image) {
				require.Equal(t, "fake-tag", image.Tag)
				require.Nil(t, image.semVer)
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testDate, *image.CreatedAt)
				require.Equal(t, testDigest, image.Digest)
			},
		},
		{
			name: "tag is a semver",
			tag:  "v1.2.3",
			assertions: func(t *testing.T, image image) {
				require.Equal(t, "v1.2.3", image.Tag)
				require.NotNil(t, image.semVer)
				require.Equal(t, "1.2.3", image.semVer.String())
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testDate, *image.CreatedAt)
				require.Equal(t, testDigest, image.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, newImage(testCase.tag, testDigest, &testDate))
		})
	}
}
