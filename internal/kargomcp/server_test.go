package kargomcp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cli/config"
)

func TestResolveProject(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		cfgProj  string
		explicit string
		assert   func(*testing.T, string, error)
	}{
		{
			name:     "explicit argument is used as-is",
			cfgProj:  "default-proj",
			explicit: "explicit-proj",
			assert: func(t *testing.T, got string, err error) {
				require.NoError(t, err)
				require.Equal(t, "explicit-proj", got)
			},
		},
		{
			name:    "falls back to configured default",
			cfgProj: "default-proj",
			assert: func(t *testing.T, got string, err error) {
				require.NoError(t, err)
				require.Equal(t, "default-proj", got)
			},
		},
		{
			name: "errors when neither explicit nor default is set",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "project is required")
				require.ErrorContains(t, err, "kargo config set-project")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{cfg: config.CLIConfig{Project: tc.cfgProj}}
			got, err := s.resolveProject(tc.explicit)
			tc.assert(t, got, err)
		})
	}
}
