package gar

import (
	"context"
	"fmt"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/credentials"
)

func TestWorkloadIdentityFederationCredentialHelper(t *testing.T) {
	const (
		testGCPProjectID = "fake-project-123456"
		testRegion       = "fake-region"
		testKargoProject = "fake-project"
		testToken        = "fake-token"
	)
	testRepoURL := fmt.Sprintf("%s-docker.pkg.dev/%s/debian/debian", testRegion, testGCPProjectID)

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(testKargoProject, testToken, cache.DefaultExpiration)

	testCases := []struct {
		name       string
		credType   credentials.Type
		repoURL    string
		helper     *workloadIdentityFederationCredentialHelper
		assertions func(*testing.T, *credentials.Credentials, *cache.Cache, error)
	}{
		{
			name:     "cred type is not image",
			credType: credentials.TypeGit,
			helper:   &workloadIdentityFederationCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "GCP Workload Identity Federation not in use",
			credType: credentials.TypeImage,
			helper:   &workloadIdentityFederationCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "repo URL does not match Artifact Registry URL regex",
			credType: credentials.TypeImage,
			repoURL:  "ghcr.io/fake-org/fake-repo",
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "cache hit",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
				tokenCache:   warmTokenCache,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, accessTokenUsername, creds.Username)
				require.Equal(t, testToken, creds.Password)
			},
		},
		{
			name:     "cache miss; error getting access token",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
				tokenCache:   cache.New(0, 0),
				getAccessTokenFn: func(context.Context, string) (string, error) {
					return "", fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting GCP access token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:     "cache miss; success (artifact registry)",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
				tokenCache:   cache.New(0, 0),
				getAccessTokenFn: func(context.Context, string) (string, error) {
					return testToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, accessTokenUsername, creds.Username)
				require.Equal(t, testToken, creds.Password)
				_, found := c.Get(testKargoProject)
				require.True(t, found)
			},
		},
		{
			name:     "cache miss; success (container registry; legacy)",
			credType: credentials.TypeImage,
			repoURL:  "us.gcr.io/fake-project/fake-image:fake-tag",
			helper: &workloadIdentityFederationCredentialHelper{
				gcpProjectID: testGCPProjectID,
				tokenCache:   cache.New(0, 0),
				getAccessTokenFn: func(context.Context, string) (string, error) {
					return testToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, accessTokenUsername, creds.Username)
				require.Equal(t, testToken, creds.Password)
				_, found := c.Get(testKargoProject)
				require.True(t, found)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := testCase.helper.getCredentials(
				context.Background(),
				testKargoProject,
				testCase.credType,
				testCase.repoURL,
				nil, // Secret not used by this helper
			)
			testCase.assertions(t, creds, testCase.helper.tokenCache, err)
		})
	}
}
