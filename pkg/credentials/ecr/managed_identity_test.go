package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cache/coalescing"
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
		// base64 of "AWS:password"
		fakeToken = "QVdTOnBhc3N3b3Jk" // nolint:gosec
	)

	testCases := []struct {
		name       string
		provider   *ManagedIdentityProvider
		project    string
		credType   credentials.Type
		repoURL    string
		assertions func(t *testing.T, creds *credentials.Credentials, err error)
	}{
		{
			name: "not supported",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			project:  fakeProject,
			credType: credentials.TypeGit,
			repoURL:  "git://repo",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "non-ECR URL",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  "not-an-ecr-url",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
		{
			name: "token obtained",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
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
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "AWS", creds.Username)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name: "token obtained, but too near expiry to cache",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
				getAuthTokenFn: func(
					context.Context,
					string,
					string,
					string,
				) (string, time.Time, error) {
					return fakeToken, time.Now().Add(-time.Hour), nil
				},
			},
			project:  fakeProject,
			credType: credentials.TypeImage,
			repoURL:  fakeRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, "password", creds.Password)
			},
		},
		{
			name: "error in getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
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
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ECR auth token")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAuthToken",
			provider: &ManagedIdentityProvider{
				accountID: fakeAccountID,
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
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.Nil(t, creds)
				assert.NoError(t, err)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// A cache that caches nothing leaves loading as the only way a caller
			// can obtain a token, so every case exercises the load. Whether a hit is
			// served from the cache instead is the cache's concern, not this
			// provider's.
			tokenCache, err := coalescing.NewCache(
				testCase.provider.loadAuthToken,
				&coalescing.CacheOptions{
					LoadTimeout:  new(tokenAcquisitionTimeout),
					CacheNothing: true,
				},
			)
			require.NoError(t, err)
			testCase.provider.tokenCache = tokenCache

			creds, err := testCase.provider.GetCredentials(
				t.Context(),
				credentials.Request{
					Type:    testCase.credType,
					Project: testCase.project,
					RepoURL: testCase.repoURL,
				},
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestManagedIdentityProvider_GetCredentials_perProject(t *testing.T) {
	t.Parallel()

	// Coalescing, cancellation, and the load deadline all belong to the cache this
	// provider delegates to, and are covered by that package's tests. What remains
	// this provider's own responsibility is that the key it caches under and the
	// input it loads from describe the same region, AWS account, and Kargo
	// Project, so that no Project is ever served another's token, and no registry
	// another account's.

	const (
		fakeAccountID      = "123456789012"
		fakeOtherAccountID = "210987654321"
	)

	provider := &ManagedIdentityProvider{
		accountID: fakeAccountID,
		getAuthTokenFn: func(
			_ context.Context,
			region string,
			accountID string,
			project string,
		) (string, time.Time, error) {
			// The token identifies everything it was obtained for.
			return base64.StdEncoding.EncodeToString(
				[]byte("AWS:" + region + "/" + accountID + "/" + project),
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

	// Each dimension of the key varies in turn, and the first request repeats at
	// the end to confirm it is still served its own token.
	for _, request := range []struct {
		region    string
		accountID string
		project   string
	}{
		{"us-west-2", fakeAccountID, "project-a"},
		{"us-west-2", fakeAccountID, "project-b"},
		{"eu-west-1", fakeAccountID, "project-a"},
		{"us-west-2", fakeOtherAccountID, "project-a"},
		{"us-west-2", fakeAccountID, "project-a"},
	} {
		creds, err := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Type:    credentials.TypeImage,
				Project: request.project,
				RepoURL: request.accountID + ".dkr.ecr." + request.region +
					".amazonaws.com/repo",
			},
		)
		require.NoError(t, err)
		require.NotNil(t, creds)
		require.Equal(t, "AWS", creds.Username)
		require.Equal(
			t,
			request.region+"/"+request.accountID+"/"+request.project,
			creds.Password,
		)
	}
}

func TestRoleSessionNameFor(t *testing.T) {
	testCases := []struct {
		name           string
		controllerName string
		expected       string
	}{
		{
			name:     "controller has no name",
			expected: "kargo-controller",
		},
		{
			name:           "controller has a name",
			controllerName: "shard-1",
			expected:       "kargo-controller-shard-1",
		},
		{
			// A name this long pushes the session name past the 64 characters AWS
			// permits, so the excess is truncated away.
			name:           "controller name too long",
			controllerName: strings.Repeat("a", 40) + strings.Repeat("b", 20),
			expected:       "kargo-controller-" + strings.Repeat("a", 40) + strings.Repeat("b", 7),
		},
		{
			// One character shorter than the case above, so it just fits.
			name:           "longest controller name that still fits",
			controllerName: strings.Repeat("a", 47),
			expected:       "kargo-controller-" + strings.Repeat("a", 47),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			actual := roleSessionNameFor(testCase.controllerName)
			assert.Equal(t, testCase.expected, actual)
			assert.LessOrEqual(t, len(actual), roleSessionNameMaxLength)
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
