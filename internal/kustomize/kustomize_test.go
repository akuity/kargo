package kustomize

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSetImageCmd(t *testing.T) {
	const testDir = "/some-dir"
	const testImage = "some-image"
	const testTag = "some-tag"
	cmd := buildSetImageCmd(testDir, testImage, testTag)
	require.NotNil(t, cmd)
	require.True(t, strings.HasSuffix(cmd.Path, "/kustomize"))
	require.Equal(
		t,
		[]string{
			"kustomize",
			"edit",
			"set",
			"image",
			fmt.Sprintf("%s=%s:%s", testImage, testImage, testTag),
		},
		cmd.Args,
	)
	require.Equal(t, testDir, cmd.Dir)
}
