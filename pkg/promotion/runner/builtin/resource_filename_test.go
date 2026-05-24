package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_validateGeneratedResourceFilename(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		assert   func(*testing.T, error)
	}{
		{
			name:     "empty",
			fileName: "",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "generated resource filename is empty")
			},
		},
		{
			name:     "plain filename",
			fileName: "configmap-test-ns-test-configmap.yaml",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:     "absolute path",
			fileName: "/etc/passwd",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unsafe generated resource filename")
			},
		},
		{
			name:     "forward slash separator",
			fileName: "subdir/file.yaml",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unsafe generated resource filename")
			},
		},
		{
			name:     "backslash separator",
			fileName: `subdir\file.yaml`,
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unsafe generated resource filename")
			},
		},
		{
			name:     "current segment alone",
			fileName: ".",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unsafe generated resource filename")
			},
		},
		{
			name:     "parent segment",
			fileName: "..",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unsafe generated resource filename")
			},
		},
		{
			name:     "parent traversal",
			fileName: "../owned-marker.yaml",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unsafe generated resource filename")
			},
		},
		{
			name:     "current-dir prefix",
			fileName: "./file.yaml",
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, "unsafe generated resource filename")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.assert(t, validateGeneratedResourceFilename(tt.fileName))
		})
	}
}
