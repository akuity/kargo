package ecr

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

func TestAccessKeyCredentialHelper(t *testing.T) {
	const (
		testAWSAccountID    = "123456789012"
		testRegion          = "fake-region"
		testAccessKeyID     = "fake-id"
		testSecretAccessKey = "fake-secret"
		testUsername        = "fake-username"
		testPassword        = "fake-password"
	)
	testRepoURL := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/fake/repo/", testAWSAccountID, testRegion)
	testToken := fmt.Sprintf("%s:%s", testUsername, testPassword)
	testEncodedToken := base64.StdEncoding.EncodeToString([]byte(testToken))

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(
		(&accessKeyCredentialHelper{}).tokenCacheKey(testRegion, testAccessKeyID, testSecretAccessKey),
		testEncodedToken,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		credType   credentials.Type
		repoURL    string
		secret     *corev1.Secret
		helper     *accessKeyCredentialHelper
		assertions func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error)
	}{
		{
			name:     "cred type is not image",
			credType: credentials.TypeGit,
			secret:   &corev1.Secret{},
			helper:   &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "cred type is not oci helm",
			credType: credentials.TypeHelm,
			repoURL:  testRepoURL,
			secret:   &corev1.Secret{},
			helper:   &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "secret is nil",
			credType: credentials.TypeImage,
			helper:   &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "not an ecr url",
			credType: credentials.TypeImage,
			repoURL:  "not-an-ecr-url",
			secret:   &corev1.Secret{},
			helper:   &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "no aws details provided",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret:   &corev1.Secret{},
			helper:   &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "region missing",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					idKey:     []byte(testAccessKeyID),
					secretKey: []byte(testSecretAccessKey),
				},
			},
			helper: &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "access key id missing",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					regionKey: []byte(testRegion),
					secretKey: []byte(testSecretAccessKey),
				},
			},
			helper: &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "secret access key missing",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					regionKey: []byte(testRegion),
					idKey:     []byte(testAccessKeyID),
				},
			},
			helper: &accessKeyCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "cache hit",
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					regionKey: []byte(testRegion),
					idKey:     []byte(testAccessKeyID),
					secretKey: []byte(testSecretAccessKey),
				},
			},
			helper: &accessKeyCredentialHelper{
				tokenCache: warmTokenCache,
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
			secret: &corev1.Secret{
				Data: map[string][]byte{
					regionKey: []byte(testRegion),
					idKey:     []byte(testAccessKeyID),
					secretKey: []byte(testSecretAccessKey),
				},
			},
			helper: &accessKeyCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string, string) (string, error) {
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
			secret: &corev1.Secret{
				Data: map[string][]byte{
					regionKey: []byte(testRegion),
					idKey:     []byte(testAccessKeyID),
					secretKey: []byte(testSecretAccessKey),
				},
			},
			helper: &accessKeyCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string, string) (string, error) {
					return testEncodedToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, testUsername, creds.Username)
				require.Equal(t, testPassword, creds.Password)
				_, found := c.Get(
					(&accessKeyCredentialHelper{}).tokenCacheKey(testRegion, testAccessKeyID, testSecretAccessKey),
				)
				require.True(t, found)
			},
		},
		{
			name:     "cache miss; success (helm)",
			credType: credentials.TypeHelm,
			repoURL:  fmt.Sprintf("oci://%s", testRepoURL),
			secret: &corev1.Secret{
				Data: map[string][]byte{
					regionKey: []byte(testRegion),
					idKey:     []byte(testAccessKeyID),
					secretKey: []byte(testSecretAccessKey),
				},
			},
			helper: &accessKeyCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAuthTokenFn: func(context.Context, string, string, string) (string, error) {
					return testEncodedToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, c *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, testUsername, creds.Username)
				require.Equal(t, testPassword, creds.Password)
				_, found := c.Get(
					(&accessKeyCredentialHelper{}).tokenCacheKey(testRegion, testAccessKeyID, testSecretAccessKey),
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
