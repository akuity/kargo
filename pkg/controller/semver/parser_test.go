package semver

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name       string
		tag        string
		strict     bool
		assertions func(t *testing.T, sv *semver.Version)
	}{
		{
			name:   "invalid with strict parsing",
			strict: true,
			tag:    "invalid",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.Nil(t, sv)
			},
		},
		{
			name:   "invalid without strict parsing",
			strict: false,
			tag:    "invalid",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.Nil(t, sv)
			},
		},
		{
			name:   "valid, but not strictly so, with strict parsing",
			strict: true,
			tag:    "1",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.Nil(t, sv)
			},
		},
		{
			name:   "valid, but not strictly so, without strict parsing",
			strict: false,
			tag:    "1",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.NotNil(t, sv)
				require.Equal(t, "1.0.0", sv.String())
				require.Equal(t, "1", sv.Original())
			},
		},
		{
			name:   "strictly valid, with strict parsing",
			strict: true,
			tag:    "1.0.0",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.NotNil(t, sv)
				require.Equal(t, "1.0.0", sv.String())
				require.Equal(t, "1.0.0", sv.Original())
			},
		},
		{
			name:   "strictly valid, without strict parsing",
			strict: false,
			tag:    "1.0.0",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.NotNil(t, sv)
				require.Equal(t, "1.0.0", sv.String())
				require.Equal(t, "1.0.0", sv.Original())
			},
		},
		{
			name:   "strictly valid, with leading v and strict parsing",
			strict: true,
			tag:    "v1.0.0",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.NotNil(t, sv)
				require.Equal(t, "1.0.0", sv.String())
				require.Equal(t, "v1.0.0", sv.Original())
			},
		},
		{
			name:   "strictly valid, with leading v, but without strict parsing",
			strict: false,
			tag:    "v1.0.0",
			assertions: func(t *testing.T, sv *semver.Version) {
				require.NotNil(t, sv)
				require.Equal(t, "1.0.0", sv.String())
				require.Equal(t, "v1.0.0", sv.Original())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, Parse(testCase.tag, testCase.strict))
		})
	}
}
