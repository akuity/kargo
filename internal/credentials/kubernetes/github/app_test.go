package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
)

func TestAppCredentialHelper(t *testing.T) {
	const (
		testAppIDStr          = "12345"
		testInstallationIDStr = "67890"
		testPrivateKey        = "fake-private-key"
		testAccessToken       = "fake-access-token"
	)
	testInstallationID, err := strconv.ParseInt(testInstallationIDStr, 10, 64)
	require.NoError(t, err)
	testAppID, err := strconv.ParseInt(testAppIDStr, 10, 64)
	require.NoError(t, err)

	warmTokenCache := cache.New(0, 0)
	warmTokenCache.Set(
		(&appCredentialHelper{}).tokenCacheKey(testAppID, testInstallationID, testPrivateKey),
		testAccessToken,
		cache.DefaultExpiration,
	)

	testCases := []struct {
		name       string
		credType   credentials.Type
		secret     *corev1.Secret
		helper     *appCredentialHelper
		assertions func(*testing.T, *credentials.Credentials, *cache.Cache, error)
	}{
		{
			name:     "cred type is not git",
			credType: credentials.TypeImage,
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "secret is nil",
			credType: credentials.TypeGit,
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "no github details provided",
			credType: credentials.TypeGit,
			secret:   &corev1.Secret{},
			helper:   &appCredentialHelper{},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name:     "app ID missing",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "installation ID missing",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:      []byte(testAppIDStr),
					privateKeyKey: []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "private key missing",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
				},
			},
			helper: &appCredentialHelper{},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "must all be set or all be unset")
			},
		},
		{
			name:     "cache hit",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: warmTokenCache,
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, "kargo", creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
			},
		},
		{
			name:     "cache miss; error getting access token",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAccessTokenFn: func(int64, int64, string) (string, error) {
					return "", fmt.Errorf("something went wrong")
				},
			},
			assertions: func(t *testing.T, _ *credentials.Credentials, _ *cache.Cache, err error) {
				require.ErrorContains(t, err, "error getting installation access token")
				require.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name:     "cache miss; success",
			credType: credentials.TypeGit,
			secret: &corev1.Secret{
				Data: map[string][]byte{
					appIDKey:          []byte(testAppIDStr),
					installationIDKey: []byte(testInstallationIDStr),
					privateKeyKey:     []byte(testPrivateKey),
				},
			},
			helper: &appCredentialHelper{
				tokenCache: cache.New(0, 0),
				getAccessTokenFn: func(int64, int64, string) (string, error) {
					return testAccessToken, nil
				},
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, _ *cache.Cache, err error) {
				require.NoError(t, err)
				require.NotNil(t, creds)
				require.Equal(t, "kargo", creds.Username)
				require.Equal(t, testAccessToken, creds.Password)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := testCase.helper.getCredentials(
				context.Background(),
				"", // project is irrelevant for this helper
				testCase.credType,
				"", // repoURL is irrelevant for this helper
				testCase.secret,
			)
			testCase.assertions(t, creds, testCase.helper.tokenCache, err)
		})
	}
}

func TestGetAccessToken(t *testing.T) {
	const key = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCg7I2MQea22DNj
aGKrTx2Bl3rEkKf+mqLZp8N45qtkjzw6EiCRPpvDav4+e2uXwWZkvD0CXYsQGsa4
ntperwJJY9OsqRQ+TbFzowps7u4Fhb3FnqRbE4uiZrInNYwuArnOCJwSbaxTFkxm
EjypUUs/lRbhJwpfv71BvcVjNztwc9TXOu4bgeOntfmLWUjvaatVx1XGPWqiQEdp
O0+2y7N9lCkW+euLT3awd7q+N/R/r7fNEfn4N+NZe/dwUVE2QnHYhKUk0Cr6+KtD
zbCtQysE7sDj67pNsIXbRn7fUfdzVmxZ0Mfbko+ABkUVajBhOsN4ubzL1K/JQU0w
Bmgonsk7AgMBAAECggEAO9Qmvg4kRjd0R5GgGXu1ByC32ovDgZOhVxWZxYHRF/Zu
2FIz/HwP8iP9uWIuesHDHVGkxxPbJ1YlKq+YlVowmfN817UW2yEMh+cGccCVCsWR
6/6SsW+/WtYehxhP8S0/QjwONoXC6zMfnVVLa1HXjaCS3IukvjttlBsHX19CAIjh
D9RqZ28R1Nwy5pFJB++dAfchW2TtlXVkXlOna6Mlc1lwtZby1lMj9xT5yPMKdPB3
JoKjiOZuOwI6h8b40MZhxPwJ4ePJv+A3ZQC8idQ9C8iJtWKvfUN2DXpstDziSlTs
MUOXU+P9yPZLvjKL4OZ+edGYDWk53FYoxVm/FWyBjQKBgQDQVoHmZoAzYKnMnoVY
ThfDMGMlj0xhUkNnHkEXCcompC7Kh7p/q70Bq8ZrT9o+R2EXyyQa1rLpUp1HsWKM
/7hiA9X9RIrUzk3Bbk8fg3muETEdFtcME+s+FbCski4iYE2/bBVjqODvaO/rzxu9
sSIqf5wM8BROGwHbEFqAl3vMZwKBgQDFvTW4fmH8DWL+mTQUghD99QG6H/CBipVs
lKu/84P6m2hnue9EUBlBJl35DSkISv0hElO5AdI7Ev63NJtz8ziXvbd6PbsldWks
+B4i8UN6GsWqwd8M9vmdr3a1mG4FAOL7jEpOqNkvs3D4xWL+M3uymc8wE4yniJTM
/uarQx5YDQKBgEk8f8F8esiUzFvPxdQ674N/+Pp1G0aC4orXSc5NdLCMup4bhGXo
+zIhLkj+8xs9gFYa5QBCRPZcQkm3g4tJQYnDC3BSrfMM6qx6mHndf+K+zGMLamEm
h2V1vnuLj4gqDmqiFgrIjPncC6r7TScro3UJEtRBeQHT4J0fbJETr0M1AoGAaX8y
KxVah5RIzZbFP2/JSwStgDTMJwDeCckj/MwaDNlfEYAU1Hh7kNO8bUSFMMR5Wmyh
uGHtXNEcjngFvA32kpaITjKjJzAGBhT2VyQrIPkpnpnCu/MEaAmWJvqFMCwx7Y0C
lAbnoNh2nHMLBp5HD5mZ/Ydgkn1/DgOs45Bynv0CgYEAiiPCMoyzqUoVYM7VPrE3
ib8XhxGI267P8OKWwRu2w8+wU3yQPThkXOhIknLyg5EMN/8zOxVmFUJ8FLzGGz30
N64Yi5HQL++EXu+7g8QfA3JYZvF0yTWW4HIQbT1MO8Gw+oPkueZ41z8DAqzHcUuk
gV5Uyur1krfumoTPJpjqTo0=
-----END PRIVATE KEY-----`
	testCases := []struct {
		name        string
		key         string
		expectedKey string
		expectsErr  bool
	}{
		{
			name:        "key is PEM encoded",
			key:         key,
			expectedKey: key,
		},
		{
			name:        "key is base64 encoding of PEM encoded key",
			key:         base64.StdEncoding.EncodeToString([]byte(key)),
			expectedKey: key,
		},
		{
			name:       "key is garbage",
			key:        "garbage",
			expectsErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			key, err := decodeKey(testCase.key)
			if testCase.expectsErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []byte(testCase.expectedKey), key)
		})
	}
}
