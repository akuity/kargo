package acr

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
	"io"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/url"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	accessTokenUsername  = "00000000-0000-0000-0000-000000000000"
	metadataURL          = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"
	viewerRoleName       = "kargo-viewer"
	AzureDefaultAudience = "api://AzureADTokenExchange"
	AnnotationClientID   = "azure.workload.identity/client-id"
	AnnotationTenantID   = "azure.workload.identity/tenant-id"
)

type workloadIdentityCredentialHelper struct {
	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAccessTokenFn func(context.Context, string) (string, error)
	tenantID         string
	clientID         string
	kargoClient      client.Client
}

// NewWorkloadIdentityCredentialHelper returns an implementation
// credentials.Helper that utilizes a cache to avoid unnecessary calls to Azure.
func NewWorkloadIdentityCredentialHelper(ctx context.Context, kargoClient client.Client) credentials.Helper {
	logger := logging.LoggerFromContext(ctx)
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	tokenFilePath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")

	// clientId is not set by default, so just check for tenant id and token file path
	if tenantID == "" || tokenFilePath == "" {
		logger.Info("Azure environment variables not set; Azure Workload Identity integration is disabled")
		return nil
	}
	logger.Info("Azure Workload Identity integration is enabled")

	s := &workloadIdentityCredentialHelper{
		tenantID:    tenantID,
		clientID:    clientID,
		kargoClient: kargoClient,
		tokenCache: cache.New(
			// Access tokens live for three hours. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
	}
	return s.getCredentials
}

func (w *workloadIdentityCredentialHelper) getCredentials(ctx context.Context, project string, credType credentials.Type, repoURL string, _ *corev1.Secret) (*credentials.Credentials, error) {
	logger := logging.LoggerFromContext(ctx)

	// Workload Identity isn't set up for this controller
	if (credType != credentials.TypeImage && credType != credentials.TypeHelm) || w.tenantID == "" {
		// This helper can't handle this
		return nil, nil
	}

	if credType == credentials.TypeHelm && !strings.HasPrefix(repoURL, "oci://") {
		// Only OCI Helm repos are supported in ACR
		return nil, nil
	}

	cacheKey := w.tokenCacheKey(repoURL, project)

	if entry, exists := w.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	proj, err := v1alpha1.GetProject(ctx, w.kargoClient, project)
	if err != nil {
		return nil, err
	}

	clientID, ok := proj.GetAnnotations()[AnnotationClientID]
	if !ok {
		return nil, fmt.Errorf("project is missing annotation: %s", AnnotationClientID)
	}
	tenantID, ok := proj.GetAnnotations()[AnnotationTenantID]
	if !ok {
		return nil, fmt.Errorf("project is missing annotation: %s", AnnotationTenantID)
	}

	env, err := getAzureEnvironment()
	if err != nil {
		return nil, err
	}

	repoHost, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}

	audiences := []string{AzureDefaultAudience}

	logger.Info("Retrieving service account from namespace", "project", project)

	// Use the TokenRequest API to get a temporary token for the `kargo-viewer` serviceaccount for the given project namespace
	saToken, err := w.fetchSAToken(ctx, project, audiences)
	if err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("Getting Entra OAuth token for %s, with tenantId %s and clientId %s", repoHost.Host, tenantID, clientID))

	authToken, err := exchangeForEntraIDToken(ctx, saToken, clientID, tenantID, env.ActiveDirectoryEndpoint, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, err
	}

	var acrToken string
	// Get a token scoped for the whole ACR registry
	acrToken, err = fetchACRRefreshToken(authToken, tenantID, repoHost.Host)
	if err != nil {
		return nil, err
	}

	repoPath := strings.Split(repoHost.Path, "/")[1]

	// This part is not required per se - but if we want to scope down access to pull only access for a specific
	// repository, as opposed to full access for the whole registry we need to fetch an access token
	acrToken, err = fetchACRAccessToken(acrToken, repoHost.Host, fmt.Sprintf("repository:%s:pull", repoPath))
	if err != nil {
		return nil, err
	}

	w.tokenCache.Set(cacheKey, acrToken, cache.DefaultExpiration)
	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: acrToken,
	}, nil
}

func (w *workloadIdentityCredentialHelper) tokenCacheKey(repoUrl, project string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf("%s:%s", repoUrl, project),
		)),
	)
}

func exchangeForEntraIDToken(ctx context.Context, token, clientID, tenantID, aadEndpoint, kvResource string) (string, error) {
	// exchange token with Azure AccessToken
	cred := confidential.NewCredFromAssertionCallback(func(ctx context.Context, aro confidential.AssertionRequestOptions) (string, error) {
		return token, nil
	})
	cClient, err := confidential.New(fmt.Sprintf("%s%s/oauth2/token", aadEndpoint, tenantID), clientID, cred)
	if err != nil {
		return "", err
	}
	scope := kvResource
	// .default needs to be added to the scope
	if !strings.Contains(kvResource, ".default") {
		scope = fmt.Sprintf("%s/.default", kvResource)
	}
	authRes, err := cClient.AcquireTokenByCredential(ctx, []string{
		scope,
	})
	if err != nil {
		return "", err
	}

	return authRes.AccessToken, nil
}

func (w *workloadIdentityCredentialHelper) fetchSAToken(ctx context.Context, project string, audiences []string) (string, error) {
	sa := corev1.ServiceAccount{}
	err := w.kargoClient.Get(ctx, types.NamespacedName{
		Name:      viewerRoleName,
		Namespace: project,
	}, &sa)
	if err != nil {
		return "", err
	}

	resource := w.kargoClient.SubResource("token")

	tokenReq := &v1.TokenRequest{
		Spec: v1.TokenRequestSpec{
			Audiences: audiences,
		},
	}
	err = resource.Create(ctx, &sa, tokenReq)
	if err != nil {
		return "", err
	}

	return tokenReq.Status.Token, nil
}

func fetchACRAccessToken(acrRefreshToken, registryURL, scope string) (string, error) {
	formData := url.Values{
		"grant_type":    {"refresh_token"},
		"service":       {registryURL},
		"scope":         {scope},
		"refresh_token": {acrRefreshToken},
	}
	res, err := http.PostForm(fmt.Sprintf("https://%s/oauth2/token", registryURL), formData)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not generate access token, unexpected status code: %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return "", err
	}
	accessToken, ok := payload["access_token"]
	if !ok {
		return "", fmt.Errorf("unable to get token")
	}
	return accessToken, nil
}

func fetchACRRefreshToken(aadAccessToken, tenantID, registryURL string) (string, error) {
	// https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#overview
	// https://docs.microsoft.com/en-us/azure/container-registry/container-registry-authentication?tabs=azure-cli
	formData := url.Values{
		"grant_type":   {"access_token"},
		"service":      {registryURL},
		"tenant":       {tenantID},
		"access_token": {aadAccessToken},
	}
	res, err := http.PostForm(fmt.Sprintf("https://%s/oauth2/exchange", registryURL), formData)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not generate refresh token, unexpected status code %d, expected %d", res.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return "", err
	}
	refreshToken, ok := payload["refresh_token"]
	if !ok {
		return "", fmt.Errorf("unable to get token")
	}
	return refreshToken, nil
}

// Use the Azure metadata API to determine the correct OAuth endpoints for the given cluster.
func getAzureEnvironment() (azure.Environment, error) {
	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return azure.Environment{}, err
	}
	req.Header.Set("Metadata", "true")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return azure.Environment{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return azure.Environment{}, fmt.Errorf("failed to get metadata, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return azure.Environment{}, err
	}

	var metadata struct {
		Compute struct {
			Environment       string `json:"azEnvironment"`
			SubscriptionID    string `json:"subscriptionId"`
			ResourceGroupName string `json:"resourceGroupName"`
		} `json:"compute"`
	}
	if err := json.Unmarshal(body, &metadata); err != nil {
		return azure.Environment{}, err
	}

	env, err := azure.EnvironmentFromName(metadata.Compute.Environment)
	if err != nil {
		return azure.Environment{}, err
	}

	return env, nil
}
