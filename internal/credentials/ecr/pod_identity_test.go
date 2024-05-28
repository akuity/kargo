package ecr

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

func TestNewPodIdentityCredentialHelper(t *testing.T) {
	// Without env var AWS_CONTAINER_CREDENTIALS_FULL_URI set, we expect not to
	// be making any calls out to AWS.
	h := NewPodIdentityCredentialHelper(context.Background())
	require.NotNil(t, h)
	p, ok := h.(*podIdentityCredentialHelper)
	require.True(t, ok)
	require.Empty(t, p.awsAccountID)
	require.NotNil(t, p.tokenCache)
	require.NotNil(t, p.getAuthTokenFn)
}

func TestPodIdentityCredentialHelper(t *testing.T) {
	const (
		testAWSAccountID = "123456789012"
		testRegion       = "fake-region"
		testProject      = "fake-project"
		testUsername     = "fake-username"
		testPassword     = "fake-password"
	)
	testRepoURL := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", testAWSAccountID, testRegion)
	testToken := fmt.Sprintf("%s:%s", testUsername, testPassword)
	testEncodedToken := base64.StdEncoding.EncodeToString([]byte(testToken))

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(
		(&podIdentityCredentialHelper{}).tokenCacheKey(testRegion, testProject),
		testEncodedToken,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		repoURL    string
		helper     PodIdentityCredentialHelper
		assertions func(t *testing.T, username, password string, c *cache.Cache, err error)
	}{
		{
			name:    "EKS Pod Identity not in use",
			repoURL: testRepoURL,
			helper:  &podIdentityCredentialHelper{},
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Empty(t, username)
				require.Empty(t, password)
			},
		},
		{
			name:    "repo URL does not match ECR URL regex",
			repoURL: "ghcr.io/fake-org/fake-repo",
			helper: &podIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
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
			helper: &podIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
				tokenCache:   warmTokenCache,
			},
			assertions: func(t *testing.T, username, password string, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Equal(t, testUsername, username)
				require.Equal(t, testPassword, password)
			},
		},
		{
			name:    "cache miss; error getting auth token",
			repoURL: testRepoURL,
			helper: &podIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
				tokenCache:   cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string) (string, error) {
					return "", fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _, _ string, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting ECR auth token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:    "cache miss; success",
			repoURL: testRepoURL,
			helper: &podIdentityCredentialHelper{
				awsAccountID: testAWSAccountID,
				tokenCache:   cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string) (string, error) {
					return testEncodedToken, nil
				},
			},
			assertions: func(t *testing.T, username, password string, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.Equal(t, testUsername, username)
				require.Equal(t, testPassword, password)
				_, found := c.Get(
					(&podIdentityCredentialHelper{}).tokenCacheKey(testRegion, testProject),
				)
				require.True(t, found)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			username, password, err := testCase.helper.GetUsernameAndPassword(
				context.Background(),
				testCase.repoURL,
				testProject,
			)
			cache := testCase.helper.(*podIdentityCredentialHelper).tokenCache // nolint: forcetypeassert
			testCase.assertions(t, username, password, cache, err)
		})
	}
}
