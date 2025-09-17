package image

import (
	"os"
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

func TestCheckForCustomRateLimit(t *testing.T) {
	testCases := []struct {
		name              string
		imagePrefix       string
		customHostnames   string
		customRateLimit   string
		expectedRateLimit int
	}{
		{
			name:              "no custom hostnames configured",
			imagePrefix:       "docker.io",
			customHostnames:   "",
			expectedRateLimit: 20,
		},
		{
			name:              "custom hostname matches imagePrefix",
			imagePrefix:       "registry.example.com",
			customHostnames:   "registry.example.com,other.registry.com",
			customRateLimit:   "50",
			expectedRateLimit: 50,
		},
		{
			name:              "custom hostname does not match imagePrefix",
			imagePrefix:       "docker.io",
			customHostnames:   "registry.example.com,other.registry.com",
			customRateLimit:   "50",
			expectedRateLimit: 20,
		},
		{
			name:              "multiple hostnames, first matches",
			imagePrefix:       "registry.example.com",
			customHostnames:   "registry.example.com,other.registry.com",
			customRateLimit:   "100",
			expectedRateLimit: 100,
		},
		{
			name:              "multiple hostnames, second matches",
			imagePrefix:       "other.registry.com",
			customHostnames:   "registry.example.com,other.registry.com",
			customRateLimit:   "75",
			expectedRateLimit: 75,
		},
		{
			name:              "hostname with spaces",
			imagePrefix:       "spaced.registry.com",
			customHostnames:   " spaced.registry.com , other.registry.com ",
			customRateLimit:   "30",
			expectedRateLimit: 30,
		},
		{
			name:              "default rate limit when custom not set",
			imagePrefix:       "registry.example.com",
			customHostnames:   "registry.example.com",
			customRateLimit:   "",
			expectedRateLimit: 20,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment variables
			if tc.customHostnames != "" {
				os.Setenv("CUSTOM_IMAGE_REGISTRY_HOSTNAMES", tc.customHostnames)
			}
			if tc.customRateLimit != "" {
				os.Setenv("CUSTOM_IMAGE_REGISTRY_RATE_LIMIT", tc.customRateLimit)
			} else {
				os.Unsetenv("CUSTOM_IMAGE_REGISTRY_RATE_LIMIT")
			}
			result := checkForCustomRateLimit(tc.imagePrefix)
			require.Equal(t, tc.expectedRateLimit, result)
		})
	}
}
