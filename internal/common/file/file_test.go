package file

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExists(t *testing.T) {
	file := "file_test.go"
	exists, err := Exists(file)
	require.NoError(t, err)
	require.True(t, exists)

	file = "bogus.go"
	exists, err = Exists(file)
	require.NoError(t, err)
	require.False(t, exists)
}
