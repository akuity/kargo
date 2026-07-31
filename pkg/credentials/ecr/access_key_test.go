package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
)

func TestNewAccessKeyProvider(t *testing.T) {
	provider := NewAccessKeyProvider().(*AccessKeyProvider) // nolint:forcetypeassert

	assert.NotNil(t, provider)
	assert.NotNil(t, provider.tokenCache)
	assert.NotNil(t, provider.getAuthTokenFn)
}

func TestAccessKeyProvider_Supports(t *testing.T) {
	const (
		fakeRepoURL = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"

		fakeRegion = "us-west-2"
		fakeID     = "AKIAIOSFODNN7EXAMPLE"                     // nolint:gosec
		fakeSecret = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" // nolint:gosec
	)

	testCases := []struct {
		name     string
		credType credentials.Type
		repoURL  string
		data     map[string][]byte
		expected bool
	}{
		{
			name:     "valid image credentials",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: true,
		},
		{
			name:     "valid helm oci credentials",
			credType: credentials.TypeHelm,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: true,
		},
		{
			name:     "helm but not oci",
			credType: credentials.TypeHelm,
			repoURL:  "https://123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo",
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "missing region",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "missing access key ID",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "missing secret key",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
			},
			expected: false,
		},
		{
			name:     "invalid URL format",
			credType: credentials.TypeImage,
			repoURL:  "not-an-ecr-url",
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "unsupported credential type",
			credType: credentials.TypeGit,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			expected: false,
		},
		{
			name:     "empty data",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data:     map[string][]byte{},
			expected: false,
		},
	}

	p := NewAccessKeyProvider()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			supports, err := p.Supports(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
					Data:    testCase.data,
				},
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, supports)
		})
	}
}

func TestAccessKeyProvider_GetCredentials(t *testing.T) {
	const (
		fakeRepoURL = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
		fakeRegion  = "us-west-2"
		fakeID      = "AKIAIOSFODNN7EXAMPLE"                     // nolint:gosec
		fakeSecret  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" // nolint:gosec
		// base64 of "AWS:password"
		fakeToken = "QVdTOnBhc3N3b3Jk" // nolint:gosec
	)

	testCases := []struct {
		name           string
		credType       credentials.Type
		repoURL        string
		data           map[string][]byte
		getAuthTokenFn func(
			ctx context.Context,
			region string,
			accessKeyID string,
			secretAccessKey string,
		) (string, time.Time, error)
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:     "unsupported credentials",
			credType: credentials.TypeGit,
			repoURL:  "not-an-ecr-url",
			data:     map[string][]byte{},
			getAuthTokenFn: func(
				context.Context,
				string,
				string,
				string,
			) (string, time.Time, error) {
				return "", time.Time{}, nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name:     "token obtained",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			getAuthTokenFn: func(
				context.Context,
				string,
				string,
				string,
			) (string, time.Time, error) {
				return fakeToken, time.Now().Add(12 * time.Hour), nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name:     "token obtained, but too near expiry to cache",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			getAuthTokenFn: func(
				context.Context,
				string,
				string,
				string,
			) (string, time.Time, error) {
				return fakeToken, time.Now().Add(-time.Hour), nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name:     "error in getAuthToken",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			getAuthTokenFn: func(
				context.Context,
				string,
				string,
				string,
			) (string, time.Time, error) {
				return "", time.Time{}, errors.New("auth token error")
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ECR auth token")
				assert.Nil(t, creds)
			},
		},
		{
			name:     "empty token from getAuthToken",
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			data: map[string][]byte{
				regionKey: []byte(fakeRegion),
				idKey:     []byte(fakeID),
				secretKey: []byte(fakeSecret),
			},
			getAuthTokenFn: func(
				context.Context,
				string,
				string,
				string,
			) (string, time.Time, error) {
				return "", time.Time{}, nil
			},
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			provider := NewAccessKeyProvider().(*AccessKeyProvider) // nolint:forcetypeassert
			provider.getAuthTokenFn = testCase.getAuthTokenFn

			// A cache that caches nothing leaves loading as the only way a caller
			// can obtain a token, so every case exercises the load. Whether a hit is
			// served from the cache instead is the cache's concern, not this
			// provider's.
			tokenCache, err := coalescing.NewCache(
				provider.loadAuthToken,
				&coalescing.CacheOptions{
					LoadTimeout:  new(tokenAcquisitionTimeout),
					CacheNothing: true,
				},
			)
			require.NoError(t, err)
			provider.tokenCache = tokenCache

			creds, err := provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
					Data:    testCase.data,
				},
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func Test_decodeAuthToken(t *testing.T) {
	testCases := []struct {
		name       string
		token      string
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name:  "valid token",
			token: "QVdTOnBhc3N3b3Jk", // base64 of "AWS:password"
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name:  "invalid base64",
			token: "invalid-base64",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error decoding token")
				assert.Nil(t, creds)
			},
		},
		{
			name:  "valid base64 but invalid format",
			token: "bm90LWEtdmFsaWQtdG9rZW4=", // base64 of "not-a-valid-token"
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "invalid token format")
				assert.Nil(t, creds)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			creds, err := decodeAuthToken(testCase.token)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestAccessKeyProvider_GetCredentials_perCredentials(t *testing.T) {
	t.Parallel()

	// Coalescing, cancellation, and the load deadline all belong to the cache this
	// provider delegates to, and are covered by that package's tests. What remains
	// this provider's own responsibility is that the key it caches under and the
	// input it loads from describe the same credentials, so that no set of
	// credentials is ever served another's token.

	const (
		fakeRepoURL = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
		fakeRegion  = "us-west-2"
		fakeID      = "AKIAIOSFODNN7EXAMPLE"                     // nolint:gosec
		fakeOtherID = "AKIAI44QH8DHBEXAMPLE"                     // nolint:gosec
		fakeSecret  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" // nolint:gosec
		fakeRotated = "je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY" // nolint:gosec
	)

	provider := &AccessKeyProvider{
		getAuthTokenFn: func(
			_ context.Context,
			_ string,
			accessKeyID string,
			secretAccessKey string,
		) (string, time.Time, error) {
			// The token identifies the credentials it was obtained with.
			return base64.StdEncoding.EncodeToString(
				[]byte("AWS:" + accessKeyID + "|" + secretAccessKey),
			), time.Now().Add(12 * time.Hour), nil
		},
	}
	tokenCache, err := coalescing.NewCache(
		provider.loadAuthToken,
		&coalescing.CacheOptions{
			LoadTimeout: new(tokenAcquisitionTimeout),
			DefaultTTL:  new(time.Hour),
		},
	)
	require.NoError(t, err)
	provider.tokenCache = tokenCache

	// Each dimension of the credentials varies in turn, and the first pair
	// repeats at the end to confirm it is still served its own token.
	for _, creds := range []struct {
		id     string
		secret string
	}{
		{fakeID, fakeSecret},
		{fakeOtherID, fakeSecret},
		{fakeID, fakeRotated},
		{fakeID, fakeSecret},
	} {
		got, err := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Type:    credentials.TypeImage,
				RepoURL: fakeRepoURL,
				Data: map[string][]byte{
					regionKey: []byte(fakeRegion),
					idKey:     []byte(creds.id),
					secretKey: []byte(creds.secret),
				},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "AWS", got.Username)
		require.Equal(t, creds.id+"|"+creds.secret, got.Password)
	}
}
