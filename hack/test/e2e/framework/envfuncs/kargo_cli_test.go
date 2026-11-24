package envfuncs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessLoginConfig(t *testing.T) {
	const tempDir = "/tmp/kargo-e2e-test"
	testCases := []struct {
		name        string
		loginConfig any
		assert      func(*testing.T, context.Context, func(), error)
	}{
		{
			name:        "use_tmp_config_home absent",
			loginConfig: map[string]any{},
			assert: func(t *testing.T, ctx context.Context, cleanup func(), err error) {
				require.NoError(t, err)
				require.Nil(t, cleanup)
				require.Nil(t, ctx.Value(KargoConfigKey))
			},
		},
		{
			name:        "use_tmp_config_home is not a boolean",
			loginConfig: map[string]any{"use_tmp_config_home": "true"},
			assert: func(t *testing.T, _ context.Context, cleanup func(), err error) {
				require.ErrorContains(t, err, "must be a boolean")
				require.Nil(t, cleanup)
			},
		},
		{
			name:        "use_tmp_config_home is false",
			loginConfig: map[string]any{"use_tmp_config_home": false},
			assert: func(t *testing.T, ctx context.Context, cleanup func(), err error) {
				require.NoError(t, err)
				require.Nil(t, cleanup)
				require.Nil(t, ctx.Value(KargoConfigKey))
			},
		},
		{
			name:        "use_tmp_config_home is true",
			loginConfig: map[string]any{"use_tmp_config_home": true},
			assert: func(t *testing.T, ctx context.Context, cleanup func(), err error) {
				require.NoError(t, err)
				require.NotNil(t, cleanup)
				require.Equal(
					t,
					filepath.Join(tempDir, "config", "kargo", "config"),
					ctx.Value(KargoConfigKey),
				)
				require.Equal(
					t,
					filepath.Join(tempDir, "config"),
					os.Getenv(ConfigHomeVar),
				)
				cleanup()
				require.Equal(t, "original", os.Getenv(ConfigHomeVar))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(ConfigHomeVar, "original")
			ctx := context.WithValue(t.Context(), TmpDirKey, tempDir)
			ctx, cleanup, err := processLoginConfig(ctx, testCase.loginConfig)
			testCase.assert(t, ctx, cleanup, err)
		})
	}
}
