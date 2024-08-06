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

	tenantID    string
	clientID    string
	kargoClient client.Client
	azureEnv    *azure.Environment

	// The following behaviors are overridable for testing purposes:
	getProjectFn    func(ctx context.Context, project string) (*v1alpha1.Project, error)
	exchangeTokenFn func(ctx context.Context, token, clientID, oAuthEndpoint, scope string) (string, error)
	fetchSAFn       func(ctx context.Context, project string, audiences []string) (string, error)
	fetchACRTokenFn func(endpoint, tokenType string, data url.Values) (string, error)
}

// NewWorkloadIdentityCredentialHelper returns an implementation
// credentials.Helper that utilizes a cache to avoid unnecessary calls to Azure.
func NewWorkloadIdentityCredentialHelper(ctx context.Context, kargoClient client.Client) credentials.Helper {
	logger := logging.LoggerFromContext(ctx)
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")

	env, err := getAzureEnvironment()
	if err != nil {
		logger.Info("Azure environment not set; Azure Workload Identity integration is disabled")
		return nil
	}
	logger.Info("Azure Workload Identity integration is enabled")

	w := &workloadIdentityCredentialHelper{
		tenantID:    tenantID,
		clientID:    clientID,
		kargoClient: kargoClient,
		azureEnv:    &env,
		tokenCache: cache.New(
			// Access tokens live for three hours. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
	}

	w.fetchSAFn = w.fetchSAToken
	w.getProjectFn = w.getProject
	w.fetchACRTokenFn = fetchACRToken
	w.exchangeTokenFn = exchangeForEntraIDToken

	return w.getCredentials
}

func (w *workloadIdentityCredentialHelper) getCredentials(ctx context.Context, project string, credType credentials.Type, repoURL string, _ *corev1.Secret) (*credentials.Credentials, error) {
	logger := logging.LoggerFromContext(ctx)

	// Workload Identity isn't set up for this controller
	if (credType != credentials.TypeImage && credType != credentials.TypeHelm) || w.azureEnv == nil {
		// This helper can't handle this
		return nil, nil
	}

	if credType == credentials.TypeHelm && !strings.HasPrefix(repoURL, "oci://") {
		// Only OCI Helm repos are supported in ACR
		return nil, nil
	}

	// TODO: add regex to verify that the URL is an Azure CR URL.

	cacheKey := w.tokenCacheKey(repoURL, project)

	if entry, exists := w.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	proj, err := w.getProjectFn(ctx, project)
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

	repoHost, err := url.Parse(repoURL)
	if err != nil {
		return nil, err
	}

	audiences := []string{AzureDefaultAudience}

	logger.Info("Retrieving service account from namespace", "project", project)

	// Use the TokenRequest API to get a temporary token for the `kargo-viewer` serviceaccount for the given project namespace
	saToken, err := w.fetchSAFn(ctx, project, audiences)
	if err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("Getting Entra OAuth token for %s, with tenantId %s and clientId %s", repoHost.Host, tenantID, clientID))

	scope := w.azureEnv.ServiceManagementEndpoint
	// .default needs to be added to the scope
	if !strings.Contains(scope, ".default") {
		scope = fmt.Sprintf("%s/.default", scope)
	}

	authToken, err := w.exchangeTokenFn(ctx, saToken, clientID, fmt.Sprintf("%s%s/oauth2/token", w.azureEnv.ActiveDirectoryEndpoint, tenantID), scope)
	if err != nil {
		return nil, err
	}

	var acrToken string
	// Get a token scoped for the whole ACR registry
	acrToken, err = w.fetchACRTokenFn(fmt.Sprintf("https://%s/oauth2/exchange", repoHost.Host), "refresh_token", url.Values{
		"grant_type":   {"access_token"},
		"service":      {repoHost.Host},
		"tenant":       {tenantID},
		"access_token": {authToken},
	})
	if err != nil {
		return nil, err
	}

	repoPath := strings.Split(repoHost.Path, "/")[1]

	// This part is not required per se - but if we want to scope down access to pull only access for a specific
	// repository, as opposed to full access for the whole registry we need to fetch an access token
	acrToken, err = w.fetchACRTokenFn(fmt.Sprintf("https://%s/oauth2/token", repoHost.Host), "access_token", url.Values{
		"grant_type":    {"refresh_token"},
		"service":       {repoHost.Host},
		"scope":         {fmt.Sprintf("repository:%s:pull", repoPath)},
		"refresh_token": {acrToken},
	})
	if err != nil {
		return nil, err
	}

	w.tokenCache.Set(cacheKey, acrToken, cache.DefaultExpiration)
	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: acrToken,
	}, nil
}

func (w *workloadIdentityCredentialHelper) getProject(ctx context.Context, project string) (*v1alpha1.Project, error) {
	return v1alpha1.GetProject(ctx, w.kargoClient, project)
}

func (w *workloadIdentityCredentialHelper) tokenCacheKey(repoUrl, project string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf("%s:%s", repoUrl, project),
		)),
	)
}

func exchangeForEntraIDToken(ctx context.Context, token, clientID, oAuthEndpoint, scope string) (string, error) {
	// exchange token with Azure AccessToken
	cred := confidential.NewCredFromAssertionCallback(func(ctx context.Context, aro confidential.AssertionRequestOptions) (string, error) {
		return token, nil
	})
	cClient, err := confidential.New(oAuthEndpoint, clientID, cred)
	if err != nil {
		return "", err
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

func fetchACRToken(endpoint, tokenType string, data url.Values) (string, error) {
	res, err := http.PostForm(endpoint, data)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not generate token of type %s, unexpected status code: %d", tokenType, res.StatusCode)
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
	accessToken, ok := payload[tokenType]
	if !ok {
		return "", fmt.Errorf("unable to get token")
	}
	return accessToken, nil
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
