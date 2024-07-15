package ecr

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/credentials"
)

func TestPodIdentityCredentialHelper(t *testing.T) {
	const (
		testAWSAccountID = "123456789012"
		testRegion       = "fake-region"
		testProject      = "fake-project"
		testUsername     = "fake-username"
		testPassword     = "fake-password"
	)
	testRepoURL := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/fake/repo/", testAWSAccountID, testRegion)
	testToken := fmt.Sprintf("%s:%s", testUsername, testPassword)
	testEncodedToken := base64.StdEncoding.EncodeToString([]byte(testToken))

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(
		(&managedIdentityCredentialHelper{}).tokenCacheKey(testRegion, testProject),
		testEncodedToken,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		credType   credentials.Type
		repoURL    string
		helper     *managedIdentityCredentialHelper
		assertions func(*testing.T, *credentials.Credentials, *cache.Cache, error)
	}{
		{
			name:     "cred type is not image",
			credType: credentials.TypeGit,
			helper: &managedIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "EKS Pod Identity not in use",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			helper:   &managedIdentityCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "repo URL does not match ECR URL regex",
			credType: credentials.TypeImage,
			repoURL:  "ghcr.io/fake-org/fake-repo",
			helper: &managedIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "helm repo URL does not match ECR URL regex",
			credType: credentials.TypeHelm,
			repoURL:  testRepoURL,
			helper: &managedIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
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
			helper: &managedIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
				tokenCache:   warmTokenCache,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, testUsername, creds.Username)
				require.Equal(t, testPassword, creds.Password)
			},
		},
		{
			name:     "cache miss; error getting auth token",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			helper: &managedIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
				tokenCache:   cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string) (string, error) {
					return "", fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting ECR auth token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:     "cache miss; success",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			helper: &managedIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
				tokenCache:   cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string) (string, error) {
					return testEncodedToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, testUsername, creds.Username)
				require.Equal(t, testPassword, creds.Password)
				_, found := c.Get(
					(&managedIdentityCredentialHelper{}).tokenCacheKey(testRegion, testProject),
				)
				require.True(t, found)
			},
		},
		{
			name:     "cache miss; success (helm)",
			credType: credentials.TypeHelm,
			repoURL:  fmt.Sprintf("oci://%s", testRepoURL),
			helper: &managedIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
				tokenCache:   cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string) (string, error) {
					return testEncodedToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, testUsername, creds.Username)
				require.Equal(t, testPassword, creds.Password)
				_, found := c.Get(
					(&managedIdentityCredentialHelper{}).tokenCacheKey(testRegion, testProject),
				)
				require.True(t, found)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := testCase.helper.getCredentials(
				context.Background(),
				testProject,
				testCase.credType,
				testCase.repoURL,
				nil, // Secret not used by this helper
			)
			testCase.assertions(t, creds, testCase.helper.tokenCache, err)
		})
	}
}
