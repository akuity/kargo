package ecr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/credentials"
)

func TestManagedIdentityProvider_Supports(t *testing.T) {
	const (
		fakeAccountID = "123456789012"
		fakeRepoURL   = "123456789012.dkr.ecr.us-west-2.amazonaws.com/my-repo"
	)

	testCases := []struct {
		name     string
		provider *ManagedIdentityProvider
		credType credentials.Type
		repoURL  string
		expected bool
	}{
		{
			name: "no account ID configured",
			provider: &ManagedIdentityProvider{
				accountID: "",
			},
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			expected: false,
		},
		{
			name: "image credentials supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			expected: true,
		},
		{
			name: "helm credentials supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeHelm,
			repoURL:  fakeRepoURL,
			expected: true,
		},
		{
			name: "git credentials not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeGit,
			repoURL:  fakeRepoURL,
			expected: false,
		},
		{
			name: "non-ECR image URL not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeImage,
			repoURL:  "us-docker.pkg.dev/project/repo/image",
			expected: false,
		},
		{
			name: "non-ECR helm URL not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			credType: credentials.TypeHelm,
			repoURL:  "us-docker.pkg.dev/project/repo/chart",
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			supports, err := testCase.provider.Supports(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					RepoURL: testCase.repoURL,
				},
			)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, supports)
		})
	}
}

func TestManagedIdentityProvider_GetCredentials(t *testing.T) {
	const (
		fakeAccountID = "123456789012"
		fakeProject   = "fake-project"
		fakeRepoURL   = "123456789012.dkr.ecr.us-west-2.amazonaws.com/repo"
		fakeRegion    = "us-west-2"
		// base64 of "AWS:password"
		fakeToken = "QVdTOnBhc3N3b3Jk" // nolint:gosec
	)

	testCases := []struct {
		name       string
		provider   *ManagedIdentityProvider
		project    string
		credType   credentials.Type
		repoURL    string
		setupCache func(cache *cache.Cache)
		assertions func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error)
	}{
		{
			name: "not supported",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeGit,
			repoURL:  "git://repo",
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "non-ECR URL",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  "not-an-ecr-url",
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "cache hit",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			setupCache: func(c *cache.Cache) {
				cacheKey := tokenCacheKey(fakeRegion, fakeAccountID, fakeProject)
				c.Set(cacheKey, fakeToken, cache.DefaultExpiration)
			},
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name: "cache miss, successful token fetch",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
					string,
				) (string, time.Time, error) {
					return fakeToken, time.Now().Add(12 * time.Hour), nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, c *cache.Cache, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)

				// Verify the token was cached with a TTL based on the
				// token's actual expiry
				items := c.Items()
				item, found := items[tokenCacheKey(fakeRegion, fakeAccountID, fakeProject)]
				assert.True(t, found)
				expectedTTL := 12*time.Hour - 5*time.Minute // 12h expiry - 5m margin
				actualTTL := time.Until(time.Unix(0, item.Expiration))
				assert.InDelta(t, expectedTTL.Seconds(), actualTTL.Seconds(), 5)
			},
		},
		{
			name: "error in getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
					string,
				) (string, time.Time, error) {
					return "", time.Time{}, errors.New("auth token error")
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ECR auth token")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID:  fakeAccountID,
				tokenCache: cache.New(10*time.Hour, time.Hour),
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
					string,
				) (string, time.Time, error) {
					return "", time.Time{}, nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, _ *cache.Cache, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.setupCache != nil {
				testCase.setupCache(testCase.provider.tokenCache)
			}
			creds, err := testCase.provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					Project: testCase.project,
					RepoURL: testCase.repoURL,
				},
			)
			testCase.assertions(t, testCase.provider.tokenCache, creds, err)
		})
	}
}

func TestManagedIdentityProvider_authIdentities(t *testing.T) {
	const (
		fakeControllerAccountID = "123456789012"
		fakeRegistryAccountID   = "210987654321"
		fakeProject             = "fake-project"
		fakeExternalID          = "kargo-project-fake-project"
	)

	testCases := []struct {
		name      string
		accountID string
		expected  []authIdentity
	}{
		{
			name:      "registry in the controller's own account",
			accountID: fakeControllerAccountID,
			expected: []authIdentity{
				{
					description: "Project-specific role in the registry's AWS account",
					roleARN:     "arn:aws:iam::123456789012:role/kargo-project-fake-project",
					externalID:  fakeExternalID,
				},
				{
					// The controller's own role is used directly, so there is no role
					// assumption to present an external ID to.
					description: "controller's own role",
				},
			},
		},
		{
			name:      "registry in a different account",
			accountID: fakeRegistryAccountID,
			expected: []authIdentity{
				{
					description: "Project-specific role in the registry's AWS account",
					roleARN:     "arn:aws:iam::210987654321:role/kargo-project-fake-project",
					externalID:  fakeExternalID,
				},
				{
					description: "Project-specific role in the controller's AWS account",
					roleARN:     "arn:aws:iam::123456789012:role/kargo-project-fake-project",
					externalID:  fakeExternalID,
				},
				{
					description: "controller's own role",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			p := &ManagedIdentityProvider{accountID: fakeControllerAccountID}
			assert.Equal(
				t,
				testCase.expected,
				p.authIdentities(testCase.accountID, fakeProject),
			)
		})
	}
}

func TestManagedIdentityProvider_getAuthToken(t *testing.T) {
	const (
		fakeControllerAccountID = "123456789012"
		fakeRegistryAccountID   = "210987654321"
		fakeProject             = "fake-project"
		fakeRegion              = "us-west-2"
		fakeToken               = "fake-token"
	)
	fakeExpiry := time.Now().Add(12 * time.Hour)

	testCases := []struct {
		name string
		// accountID of the registry. Defaults to the controller's own.
		accountID string
		// outcomes are returned by successive calls to getAuthTokenAsFn.
		outcomes   []error
		assertions func(
			t *testing.T,
			usedIdentities []authIdentity,
			token string,
			expiry time.Time,
			err error,
		)
	}{
		{
			name:     "first identity is authorized",
			outcomes: []error{nil},
			assertions: func(
				t *testing.T,
				usedIdentities []authIdentity,
				token string,
				expiry time.Time,
				err error,
			) {
				require.NoError(t, err)
				assert.Equal(t, fakeToken, token)
				assert.Equal(t, fakeExpiry, expiry)
				assert.Len(t, usedIdentities, 1)
			},
		},
		{
			name:      "denied identities are skipped",
			accountID: fakeRegistryAccountID,
			outcomes:  []error{forbiddenErr(), nil},
			assertions: func(
				t *testing.T,
				usedIdentities []authIdentity,
				token string,
				_ time.Time,
				err error,
			) {
				require.NoError(t, err)
				assert.Equal(t, fakeToken, token)
				require.Len(t, usedIdentities, 2)
				// The Project-specific role in the controller's own account is
				// tried before falling back to the controller's own role.
				assert.Equal(
					t,
					fmt.Sprintf(roleARNFormat, fakeControllerAccountID, fakeProject),
					usedIdentities[1].roleARN,
				)
			},
		},
		{
			name:      "all identities denied",
			accountID: fakeRegistryAccountID,
			outcomes:  []error{forbiddenErr(), forbiddenErr(), forbiddenErr()},
			assertions: func(
				t *testing.T,
				usedIdentities []authIdentity,
				token string,
				_ time.Time,
				err error,
			) {
				// Treated as no credentials found rather than an error.
				require.NoError(t, err)
				assert.Empty(t, token)
				assert.Len(t, usedIdentities, 3)
			},
		},
		{
			name:      "error that is not a denial halts the chain",
			accountID: fakeRegistryAccountID,
			outcomes:  []error{errors.New("something went wrong"), nil},
			assertions: func(
				t *testing.T,
				usedIdentities []authIdentity,
				token string,
				_ time.Time,
				err error,
			) {
				require.ErrorContains(t, err, "something went wrong")
				assert.Empty(t, token)
				// No weaker identity is substituted for one whose authorization
				// was never actually established.
				assert.Len(t, usedIdentities, 1)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			accountID := testCase.accountID
			if accountID == "" {
				accountID = fakeControllerAccountID
			}
			var usedIdentities []authIdentity
			p := &ManagedIdentityProvider{accountID: fakeControllerAccountID}
			p.getAuthTokenAsFn = func(
				_ context.Context,
				region string,
				identity authIdentity,
			) (string, time.Time, error) {
				assert.Equal(t, fakeRegion, region)
				usedIdentities = append(usedIdentities, identity)
				require.LessOrEqual(
					t,
					len(usedIdentities),
					len(testCase.outcomes),
					"tried more identities than this test case anticipated",
				)
				if err := testCase.outcomes[len(usedIdentities)-1]; err != nil {
					return "", time.Time{}, err
				}
				return fakeToken, fakeExpiry, nil
			}
			token, expiry, err := p.getAuthToken(
				t.Context(),
				fakeRegion,
				accountID,
				fakeProject,
			)
			testCase.assertions(t, usedIdentities, token, expiry, err)
		})
	}
}

// forbiddenErr returns an error indistinguishable, to getAuthToken, from AWS
// denying a request.
func forbiddenErr() error {
	return &awshttp.ResponseError{
		ResponseError: &smithyhttp.ResponseError{
			Response: &smithyhttp.Response{
				Response: &http.Response{StatusCode: http.StatusForbidden},
			},
			Err: errors.New("access denied"),
		},
	}
}
