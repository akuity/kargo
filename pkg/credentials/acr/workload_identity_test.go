package acr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
)

func TestACRURLRegex(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected bool
		registry string
	}{
		{
			name:     "ACR URL",
			url:      "myregistry.azurecr.io/repo",
			expected: true,
			registry: "myregistry",
		},
		{
			name:     "Docker Hub URL",
			url:      "docker.io/library/nginx",
			expected: false,
		},
		{
			name:     "ECR URL",
			url:      "123456789012.dkr.ecr.us-west-2.amazonaws.com/repo",
			expected: false,
		},
		{
			name:     "Google Artifact Registry URL",
			url:      "us-central1-docker.pkg.dev/project/repo",
			expected: false,
		},
		{
			name:     "ACR URL with complex registry name",
			url:      "my-registry-123.azurecr.io/namespace/repo",
			expected: true,
			registry: "my-registry-123",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			matches := acrURLRegex.FindStringSubmatch(testCase.url)
			if testCase.expected {
				assert.Len(t, matches, 2)
				assert.Equal(t, testCase.registry, matches[1])
			} else {
				assert.Nil(t, matches, "Expected regex not to match")
			}
		})
	}
}

func TestNewWorkloadIdentityProvider(t *testing.T) {
	const azFederatedTokenFile = "AZURE_FEDERATED_TOKEN_FILE"
	const azClientID = "AZURE_CLIENT_ID"
	const azTenantID = "AZURE_TENANT_ID"
	t.Run("workload identity not available", func(t *testing.T) {
		// Make it look unavailable by ensuring key env vars are unset
		t.Setenv(azFederatedTokenFile, "") // Ensures cleanup
		os.Unsetenv(azFederatedTokenFile)  // Actually unsets
		t.Setenv(azClientID, "")           // Ensures cleanup
		os.Unsetenv(azClientID)            // Actually unsets
		t.Setenv(azTenantID, "")           // Ensures cleanup
		os.Unsetenv(azTenantID)            // Actually unsets
		require.Nil(t, NewWorkloadIdentityProvider(t.Context()))
	})
	t.Run("workload identity available", func(t *testing.T) {
		// Make it look available by ensuring key env vars are set, albeit with
		// nonsense values.
		const nonsense = "nonsense"
		t.Setenv(azFederatedTokenFile, nonsense)
		t.Setenv(azClientID, nonsense)
		t.Setenv(azTenantID, nonsense)
		require.NotNil(t, NewWorkloadIdentityProvider(t.Context()))
	})
}

// testResourceID is the xms_mirid claim as Entra actually issues it, observed
// against a real tenant. Note that ARM writes "resourcegroups" in lower case.
const testResourceID = "/subscriptions/544972a9-f7be-4152-a73d-7d4b31416a4a" +
	"/resourcegroups/kargo-identity-spike" +
	"/providers/Microsoft.ManagedIdentity/userAssignedIdentities/kargo-project-demo"

func TestParseIdentityResourceID(t *testing.T) {
	const testSubscriptionID = "544972a9-f7be-4152-a73d-7d4b31416a4a"
	const testResourceGroup = "kargo-identity-spike"

	testCases := []struct {
		name string
		// claims, when set, are marshaled into a JWT payload. rawToken is used
		// instead for inputs that are not JWTs at all.
		claims   map[string]any
		rawToken string
		assert   func(t *testing.T, subscriptionID, resourceGroup string, err error)
	}{
		{
			name:   "claim as issued",
			claims: map[string]any{"xms_mirid": testResourceID},
			assert: func(t *testing.T, subscriptionID, resourceGroup string, err error) {
				require.NoError(t, err)
				require.Equal(t, testSubscriptionID, subscriptionID)
				require.Equal(t, testResourceGroup, resourceGroup)
			},
		},
		{
			// ARM is not consistent about the case of type segments.
			name: "mixed case segments",
			claims: map[string]any{
				"xms_mirid": strings.Replace(
					testResourceID, "/resourcegroups/", "/resourceGroups/", 1,
				),
			},
			assert: func(t *testing.T, subscriptionID, resourceGroup string, err error) {
				require.NoError(t, err)
				require.Equal(t, testSubscriptionID, subscriptionID)
				require.Equal(t, testResourceGroup, resourceGroup)
			},
		},
		{
			// A system-assigned identity or a plain app registration carries no
			// such claim, and neither can be located this way.
			name:   "no xms_mirid claim",
			claims: map[string]any{"aud": "https://management.azure.com"},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "carries no xms_mirid claim")
			},
		},
		{
			name: "resource ID names no resource group",
			claims: map[string]any{
				"xms_mirid": "/subscriptions/" + testSubscriptionID,
			},
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "could not read a subscription and resource group")
			},
		},
		{
			name:     "not a JWT",
			rawToken: "nonsense",
			assert: func(t *testing.T, _, _ string, err error) {
				require.ErrorContains(t, err, "not a JWT")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			token := testCase.rawToken
			if testCase.claims != nil {
				token = testToken(t, testCase.claims)
			}
			subscriptionID, resourceGroup, err := parseIdentityResourceID(token)
			testCase.assert(t, subscriptionID, resourceGroup, err)
		})
	}
}

func TestWorkloadIdentityProvider_Supports(t *testing.T) {
	const testOCIRepoURL = "myregistry.azurecr.io/my-repo"
	const testHTTPSRepoURL = "https://myregistry.azurecr.io/my-repo"

	testCases := []struct {
		name     string
		credType credentials.Type
		repoURL  string
		expected bool
	}{
		{
			name:     "image credential type supported",
			credType: credentials.TypeImage,
			repoURL:  testOCIRepoURL,
			expected: true,
		},
		{
			name:     "helm credential type supported",
			credType: credentials.TypeHelm,
			repoURL:  testOCIRepoURL,
			expected: true,
		},
		{
			name:     "helm HTTP/S repo URLs not supported",
			credType: credentials.TypeHelm,
			repoURL:  testHTTPSRepoURL,
			expected: false,
		},
		{
			name:     "git credential type not supported",
			credType: credentials.TypeGit,
			repoURL:  testOCIRepoURL,
			expected: false,
		},
		{
			name: "non-ACR repo URL not supported",

			credType: credentials.TypeImage,
			repoURL:  "docker.io/library/nginx",
			expected: false,
		},
	}

	p := &WorkloadIdentityProvider{credential: &mockCredential{}}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			supports, err := p.Supports(
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

func TestWorkloadIdentityProvider_GetCredentials(t *testing.T) {
	const testRepoURL = "myregistry.azurecr.io/repo"
	const testToken = "fake-access-token"

	testCases := []struct {
		name       string
		provider   *WorkloadIdentityProvider
		credType   credentials.Type
		repoURL    string
		assertions func(*testing.T, *credentials.Credentials, error)
	}{
		{
			name:     "not supported",
			provider: &WorkloadIdentityProvider{},
			credType: credentials.TypeGit,
			repoURL:  "git://repo",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name:     "non-ACR URL",
			provider: &WorkloadIdentityProvider{},
			credType: credentials.TypeImage,
			repoURL:  "not-an-acr-url",
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
		{
			name: "token obtained",
			provider: &WorkloadIdentityProvider{
				getAccessTokenFn: func(_ context.Context, _ tokenRequest) (string, error) {
					return testToken, nil
				},
			},
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, creds)
				assert.Equal(t, acrTokenUsername, creds.Username)
				assert.Equal(t, testToken, creds.Password)
			},
		},
		{
			name: "error in getAccessToken",
			provider: &WorkloadIdentityProvider{
				getAccessTokenFn: func(_ context.Context, _ tokenRequest) (string, error) {
					return "", errors.New("access token error")
				},
			},
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.ErrorContains(t, err, "error getting ACR access token")
				assert.Nil(t, creds)
			},
		},
		{
			name: "empty token from getAccessToken",
			provider: &WorkloadIdentityProvider{
				getAccessTokenFn: func(_ context.Context, _ tokenRequest) (string, error) {
					return "", nil
				},
			},
			credType: credentials.TypeImage,
			repoURL:  testRepoURL,
			assertions: func(t *testing.T, creds *credentials.Credentials, err error) {
				assert.NoError(t, err)
				assert.Nil(t, creds)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.provider.credential = &mockCredential{}
			// A cache that caches nothing leaves loading as the only way a caller
			// can obtain a token, so every case exercises the load. Whether a hit is
			// served from the cache instead is the cache's concern, not this
			// provider's.
			tokenCache, err := coalescing.NewCache(
				testCase.provider.loadAccessToken,
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
					RepoURL: testCase.repoURL,
				},
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestWorkloadIdentityProvider_GetCredentials_perRegistry(t *testing.T) {
	t.Parallel()

	// Coalescing, cancellation, and the load deadline all belong to the cache this
	// provider delegates to, and are covered by that package's tests. What remains
	// this provider's own responsibility is that the key it caches under and the
	// input it loads from describe the same registry, so that no registry is ever
	// served another's token.

	provider := &WorkloadIdentityProvider{
		credential: &mockCredential{},
		getAccessTokenFn: func(
			_ context.Context,
			req tokenRequest,
		) (string, error) {
			// The token identifies the registry it was obtained for.
			return "token-for-" + req.registryName, nil
		},
	}
	tokenCache, err := coalescing.NewCache(
		provider.loadAccessToken,
		&coalescing.CacheOptions{
			LoadTimeout: new(tokenAcquisitionTimeout),
			DefaultTTL:  new(time.Hour),
		},
	)
	require.NoError(t, err)
	provider.tokenCache = tokenCache

	for _, registry := range []string{"registry-a", "registry-b", "registry-a"} {
		creds, err := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Type:    credentials.TypeImage,
				RepoURL: registry + ".azurecr.io/repo",
			},
		)
		require.NoError(t, err)
		require.NotNil(t, creds)
		require.Equal(t, acrTokenUsername, creds.Username)
		require.Equal(t, "token-for-"+registry, creds.Password)
	}
}

func TestWorkloadIdentityProvider_GetCredentials_perProject(t *testing.T) {
	t.Parallel()

	// A single controller acts for many Projects against a single registry. What
	// this provider is responsible for is that the key it caches under describes
	// the Project as well as the registry, so that no Project is ever served
	// another's token.

	provider := &WorkloadIdentityProvider{
		credential: &mockCredential{},
		getAccessTokenFn: func(
			_ context.Context,
			req tokenRequest,
		) (string, error) {
			return "token-for-" + req.project + "/" + req.registryName, nil
		},
	}
	tokenCache, err := coalescing.NewCache(
		provider.loadAccessToken,
		&coalescing.CacheOptions{
			LoadTimeout: new(tokenAcquisitionTimeout),
			DefaultTTL:  new(time.Hour),
		},
	)
	require.NoError(t, err)
	provider.tokenCache = tokenCache

	for _, project := range []string{"project-a", "project-b", "project-a"} {
		creds, err := provider.GetCredentials(
			t.Context(),
			credentials.Request{
				Project: project,
				Type:    credentials.TypeImage,
				RepoURL: "shared.azurecr.io/repo",
			},
		)
		require.NoError(t, err)
		require.NotNil(t, creds)
		require.Equal(
			t, "token-for-"+project+"/shared", creds.Password,
		)
	}
}

func TestWorkloadIdentityProvider_getAccessToken(t *testing.T) {
	const testProject = "demo"
	const testClientID = "11111111-2222-3333-4444-555555555555"

	// Building a Project-specific credential reads the tenant and the token file
	// from the environment Azure's workload identity webhook establishes. Nothing
	// here reads the file itself; the exchange is faked below. This is why these
	// subtests do not run in parallel.
	t.Setenv("AZURE_TENANT_ID", "test-tenant")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/var/run/secrets/azure/tokens/token")

	testCases := []struct {
		name string
		// getter, when set, equips the provider to act as Project-specific
		// identities. Without it, the controller's own identity is all it has.
		getter      *fakeIdentityGetter
		exchangeErr error
		assert      func(t *testing.T, token string, err error)
	}{
		{
			name: "Project-specific identities not configured",
			assert: func(t *testing.T, token string, err error) {
				require.NoError(t, err)
				require.Equal(t, "controller-token", token)
			},
		},
		{
			name: "no Project-specific identity exists",
			getter: &fakeIdentityGetter{
				err: &azcore.ResponseError{StatusCode: http.StatusNotFound},
			},
			assert: func(t *testing.T, token string, err error) {
				require.NoError(t, err)
				require.Equal(t, "controller-token", token)
			},
		},
		{
			name:   "identity resolution fails",
			getter: &fakeIdentityGetter{err: errors.New("ARM is unhappy")},
			assert: func(t *testing.T, token string, err error) {
				require.NoError(t, err)
				require.Equal(t, "controller-token", token)
			},
		},
		{
			// An identity that exists but cannot reach the registry must not be a
			// hard failure. Falling back preserves the behavior operators have
			// today while they roll per-Project identities out.
			name:        "Project-specific identity cannot reach the registry",
			getter:      &fakeIdentityGetter{clientID: testClientID},
			exchangeErr: errors.New("AcrPull not granted"),
			assert: func(t *testing.T, token string, err error) {
				require.NoError(t, err)
				require.Equal(t, "controller-token", token)
			},
		},
		{
			name:   "Project-specific identity used",
			getter: &fakeIdentityGetter{clientID: testClientID},
			assert: func(t *testing.T, token string, err error) {
				require.NoError(t, err)
				require.Equal(t, "project-token", token)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			controllerCred := &mockCredential{}
			p := &WorkloadIdentityProvider{credential: controllerCred}
			if testCase.getter != nil {
				p.resourceGroup = "test-rg"
				p.identities = testCase.getter
			}

			p.exchangeFn = func(
				_ context.Context,
				cred azcore.TokenCredential,
				_ string,
			) (string, error) {
				if cred == azcore.TokenCredential(controllerCred) {
					return "controller-token", nil
				}
				if testCase.exchangeErr != nil {
					return "", testCase.exchangeErr
				}
				return "project-token", nil
			}

			token, err := p.getAccessToken(
				t.Context(),
				tokenRequest{project: testProject, registryName: "myregistry"},
			)
			testCase.assert(t, token, err)
		})
	}
}

// testToken returns an unsigned JWT carrying the given claims. Only the payload
// is read by the code under test.
func testToken(t *testing.T, claims map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	require.NoError(t, err)
	return "header." +
		base64.RawURLEncoding.EncodeToString(payload) +
		".signature"
}

// mockCredential is a mock implementation of azcore.TokenCredential for testing
type mockCredential struct{}

func (m *mockCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// Return a mock token for testing
	return azcore.AccessToken{
		Token:     "mock-access-token",
		ExpiresOn: time.Now().Add(time.Hour),
	}, nil
}
