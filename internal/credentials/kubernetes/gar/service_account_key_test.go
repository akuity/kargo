package gar

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
)

func TestServiceAccountKeyCredentialHelper(t *testing.T) {
	const (
		testGCPProjectID      = "fake-project-123456"
		testRegion            = "fake-region"
		testServiceAccountKey = "fake-key"
		testAccessToken       = "fake-token"
	)
	testRepoURL := fmt.Sprintf("%s-docker.pkg.dev/%s/debian/debian", testRegion, testGCPProjectID)
	testEncodedServiceAccountKey := base64.StdEncoding.EncodeToString([]byte(testServiceAccountKey))

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(
		(&serviceAccountKeyCredentialHelper{}).tokenCacheKey(testEncodedServiceAccountKey),
		testAccessToken,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		credType   credentials.Type
		repoURL    string
		secret     *corev1.Secret
		helper     *serviceAccountKeyCredentialHelper
		assertions func(*testing.T, *credentials.Credentials, *cache.Cache, error)
	}{
		{
			name:     "cred type is not image",
			credType: credentials.TypeGit,
			secret:   &corev1.Secret{},
			helper:   &serviceAccountKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "secret is nil",
			credType: credentials.TypeImage,
			helper:   &serviceAccountKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "not a gar url",
			credType: credentials.TypeImage,
			repoURL:  "not-a-gar-url",
			secret:   &corev1.Secret{},
			helper:   &serviceAccountKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "no service account key provided",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret:   &corev1.Secret{},
			helper:   &serviceAccountKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "cache hit",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					serviceAccountKeyKey: []byte(testEncodedServiceAccountKey),
				},
			},
			helper: &serviceAccountKeyCredentialHelper{
				tokenCache: warmTokenCache,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, accessTokenUsername, creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
			},
		},
		{
			name:     "cache miss; error getting access token",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					serviceAccountKeyKey: []byte(testEncodedServiceAccountKey),
				},
			},
			helper: &serviceAccountKeyCredentialHelper{
				tokenCache: cache.New(0, 0),
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
			name:     "cache miss; success",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					serviceAccountKeyKey: []byte(testEncodedServiceAccountKey),
				},
			},
			helper: &serviceAccountKeyCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAccessTokenFn: func(context.Context, string) (string, error) {
					return testAccessToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, accessTokenUsername, creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
				_, found := c.Get(
					(&serviceAccountKeyCredentialHelper{}).tokenCacheKey(testEncodedServiceAccountKey),
				)
				require.True(t, found)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := testCase.helper.getCredentials(
				context.Background(),
				"", // project is irrelevant for this helper
				testCase.credType,
				testCase.repoURL,
				testCase.secret,
			)
			testCase.assertions(t, creds, testCase.helper.tokenCache, err)
		})
	}
}
