package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	const testPrefix = "fake-prefix"
	r := newRegistry(testPrefix)
	require.NotNil(t, r)
	require.Equal(t, testPrefix, r.name)
	require.NotEmpty(t, testPrefix, r.imagePrefix)
	require.Empty(t, r.defaultNamespace)
}

func TestGetRegistry(t *testing.T) {
	testCases := []struct {
		name        string
		imagePrefix string
		assertions  func(*testing.T, *registry)
	}{
		{
			name:        "hit",
			imagePrefix: "", // Docker Hub
			assertions: func(t *testing.T, reg *registry) {
				require.NotNil(t, reg)
				require.Equal(t, "Docker Hub", reg.name)
			},
		},
		{
			name:        "miss",
			imagePrefix: "fake-prefix",
			assertions: func(t *testing.T, reg *registry) {
				require.NotNil(t, reg)
				require.Equal(t, "fake-prefix", reg.name)
				// Check that it was added to the registries map
				_, ok := registries[reg.imagePrefix]
				require.True(t, ok)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, getRegistry(testCase.imagePrefix))
		})
	}
}
