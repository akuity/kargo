package gar

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestServiceAccountKeyCredentialHelper(t *testing.T) {
	const (
		testServiceAccountKey = "fake-key"
		testAccessToken       = "fake-token"
	)
	testEncodedServiceAccountKey := base64.StdEncoding.EncodeToString([]byte(testServiceAccountKey))

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(
		(&serviceAccountKeyCredentialHelper{}).tokenCacheKey(testEncodedServiceAccountKey),
		testAccessToken,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		secret     *corev1.Secret
		helper     ServiceAccountKeyCredentialHelper
		assertions func(t *testing.T, username, password string, err error)
	}{
		{
			name:   "no service account token provided",
			secret: &corev1.Secret{},
			helper: NewServiceAccountKeyCredentialHelper(),
			assertions: func(t *testing.T, username, password string, err error) {
				require.NoError(t, err)
				require.Empty(t, username)
				require.Empty(t, password)
			},
		},
		{
			name: "cache hit",
			secret: &corev1.Secret{
				Data: map[string][]byte{
					serviceAccountKeyKey: []byte(testEncodedServiceAccountKey),
				},
			},
			helper: &serviceAccountKeyCredentialHelper{
				tokenCache: warmTokenCache,
			},
			assertions: func(t *testing.T, username, password string, err error) {
				require.NoError(t, err)
				require.Equal(t, accessTokenUsername, username)
				require.Equal(t, testAccessToken, password)
			},
		},
		{
			name: "cache miss; error getting access token",
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
			assertions: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "error getting GCP access token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "cache miss; success",
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
			assertions: func(t *testing.T, username, password string, err error) {
				require.NoError(t, err)
				require.Equal(t, accessTokenUsername, username)
				require.Equal(t, testAccessToken, password)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			username, password, err := testCase.helper.GetUsernameAndPassword(
				context.Background(),
				testCase.secret,
			)
			testCase.assertions(t, username, password, err)
		})
	}
}
