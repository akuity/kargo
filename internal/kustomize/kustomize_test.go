package kustomize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSetImageCmd(t *testing.T) {
	const testDir = "/some-dir"
	const testImageRef = "some-image:some-tag"
	cmd := buildSetImageCmd(testDir, testImageRef)
	require.NotNil(t, cmd)
	require.True(t, strings.HasSuffix(cmd.Path, "kustomize"))
	require.Equal(
		t,
		[]string{
			"kustomize",
			"edit",
			"set",
			"image",
			testImageRef,
		},
		cmd.Args,
	)
	require.Equal(t, testDir, cmd.Dir)
}
