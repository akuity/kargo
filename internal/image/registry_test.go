package image

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	const testPrefix = "fake-prefix"
	r := newRegistry(testPrefix)
	require.NotNil(t, r)
	require.Equal(t, testPrefix, r.name)
	require.NotEmpty(t, testPrefix, r.imagePrefix)
	require.NotEmpty(
		t,
		fmt.Sprintf("https://%s", testPrefix),
		r.apiAddress,
	)
	require.Empty(t, r.defaultNamespace)
	require.NotNil(t, r.imageCache)
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

func TestNormalizeImageName(t *testing.T) {
	testCases := []struct {
		name       string
		imageName  string
		registry   *registry
		assertions func(*testing.T, string)
	}{
		{
			name:      "registry has no default namespace",
			imageName: "fake-image",
			registry:  &registry{},
			assertions: func(t *testing.T, normalizedName string) {
				require.Equal(t, "fake-image", normalizedName)
			},
		},
		{
			name:      "image name does not need default namespace added",
			imageName: "fake-namespace/fake-image",
			registry: &registry{
				defaultNamespace: "library",
			},
			assertions: func(t *testing.T, normalizedName string) {
				require.Equal(t, "fake-namespace/fake-image", normalizedName)
			},
		},
		{
			name:      "image name does needs default namespace added",
			imageName: "fake-image",
			registry: &registry{
				defaultNamespace: "library",
			},
			assertions: func(t *testing.T, normalizedName string) {
				require.Equal(t, "library/fake-image", normalizedName)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				t,
				testCase.registry.normalizeImageName(testCase.imageName),
			)
		})
	}
}
