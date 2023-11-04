package image

import (
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestNewTag(t *testing.T) {
	testDigest := digest.Digest("fake-digest")
	testDate := time.Now().UTC()
	testCases := []struct {
		name       string
		tagName    string
		assertions func(Tag)
	}{
		{
			name:    "tag name is not semver",
			tagName: "fake-tag",
			assertions: func(tag Tag) {
				require.Equal(t, "fake-tag", tag.Name)
				require.Nil(t, tag.semVer)
				require.NotNil(t, tag.CreatedAt)
				require.Equal(t, testDate, *tag.CreatedAt)
				require.Equal(t, testDigest, tag.Digest)
			},
		},
		{
			name:    "tag name is semver",
			tagName: "v1.2.3",
			assertions: func(tag Tag) {
				require.Equal(t, "v1.2.3", tag.Name)
				require.NotNil(t, tag.semVer)
				require.Equal(t, "1.2.3", tag.semVer.String())
				require.NotNil(t, tag.CreatedAt)
				require.Equal(t, testDate, *tag.CreatedAt)
				require.Equal(t, testDigest, tag.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(newTag(testCase.tagName, &testDate, testDigest))
		})
	}
}
