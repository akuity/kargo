package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	version = "v0.1.0"
	commit = "1234567"
	require.Equal(t, version, Version())
	require.Equal(t, commit, Commit())
}
