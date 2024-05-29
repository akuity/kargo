package gar

import (
	"context"
	"fmt"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

func TestNewWorkloadIdentityFederationCredentialHelper(t *testing.T) {
	// We don't expect this to make any calls out to GCP when not running within
	// GCE/GKE.
	h := NewWorkloadIdentityFederationCredentialHelper(context.Background())
	require.NotNil(t, h)
	w, ok := h.(*workloadIdentityFederationCredentialHelper)
	require.True(t, ok)
	require.Empty(t, w.gcpProjectID)
	require.NotNil(t, w.tokenCache)
	require.NotNil(t, w.getAccessTokenFn)
}

func TestWorkloadIdentityFederationCredentialHelper(t *testing.T) {
	const (
		testGCPProjectID = "fake-project-123456"
		testRegion       = "fake-region"
		testKargoProject = "fake-project"
		testToken        = "fake-token"
	)
	testRepoURL := fmt.Sprintf("%s.pkg.dev/%s/debian/debian", testRegion, testGCPProjectID)

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(testKargoProject, testToken, cache.DefaultExpiration)

	testCases := []struct {
		name       string
		repoURL    string
		helper     WorkloadIdentityFederationCredentialHelper
		assertions func(t *testing.T, username, password string, c *cache.Cache, err error)
	}{
		{
			name:    "GCP Workload Identity Federation not in use",
			repoURL: testRepoURL,
			helper:  &workloadIdentityFederationCredentialHelper{},
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Empty(t, username)
				require.Empty(t, password)
			},
		},
		{
			name:    "repo URL does not match Artifact Registry URL regex",
			repoURL: "ghcr.io/fake-org/fake-repo",
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
			},
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Empty(t, username)
				require.Empty(t, password)
			},
		},
		{
			name:    "cache hit",
			repoURL: testRepoURL,
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
				tokenCache:   warmTokenCache,
			},
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Equal(t, accessTokenUsername, username)
				require.Equal(t, testToken, password)
			},
		},
		{
			name:    "cache miss; error getting access token",
			repoURL: testRepoURL,
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
				tokenCache:   cache.New(0, 0),
				getAccessTokenFn: func(context.Context, string) (string, error) {
					return "", fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _, _ string, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting GCP access token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:    "cache miss; success",
			repoURL: testRepoURL,
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
				tokenCache:   cache.New(0, 0),
				getAccessTokenFn: func(context.Context, string) (string, error) {
					return testToken, nil
				},
			},
			assertions: func(t *testing.T, username, password string, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.Equal(t, accessTokenUsername, username)
				require.Equal(t, testToken, password)
				_, found := c.Get(testKargoProject)
				require.True(t, found)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			username, password, err := testCase.helper.GetUsernameAndPassword(
				context.Background(),
				testCase.repoURL,
				testKargoProject,
			)
			cache := testCase.helper.(*workloadIdentityFederationCredentialHelper).tokenCache // nolint: forcetypeassert
			testCase.assertions(t, username, password, cache, err)
		})
	}
}
