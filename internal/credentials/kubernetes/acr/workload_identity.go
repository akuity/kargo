package acr

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/patrickmn/go-cache"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

// WorkloadIdentityCredentialHelperConfig represents configuration for the
// workload identity credential helper.
type WorkloadIdentityCredentialHelperConfig struct {
	// TenantID is the Azure tenant ID where the managed identities federated to
	// Project ServiceAccounts are located. Assuming the
	// WorkloadIdentityCredentialHelperConfig instance was populated using the
	// WorkloadIdentityCredentialHelperConfigFromEnv() function, this field will
	// take on the value of the AZURE_TENANT_ID environment variable IF that has
	// been injected into the controller's pod by AKS due to the the controller's
	// ServiceAccount having been federated with a managed identity. In the
	// absence of that environment variable (perhaps the controller's
	// ServiceAccount does is NOT federated to a managed identity), this field
	// will take on the value of the KARGO_AZURE_TENANT_ID environment variable
	// instead, which may have been set by a user at install time. If no non-empty
	// value can be found for this field, the a workload identity credential
	// helper will effectively be disabled.
	TenantID string
	// ActiveDirectoryEndpoint is the Azure Active Directory endpoint for the
	// environment in which the controller is running.
	ActiveDirectoryEndpoint string
	// AzureServiceManagementEndpoint is the Azure Service Management endpoint for
	// the environment in which the controller is running.
	AzureServiceManagementEndpoint string
}

// WorkloadIdentityCredentialHelperConfigFromEnv returns a
// WorkloadIdentityCredentialHelperConfig constructed from environment details
// and a call to the Azure Instance Metadata Service (IMDS).
func WorkloadIdentityCredentialHelperConfigFromEnv() WorkloadIdentityCredentialHelperConfig {
	cfg := WorkloadIdentityCredentialHelperConfig{
		TenantID: os.Getenv("AZURE_TENANT_ID"),
	}
	if cfg.TenantID == "" {
		// TODO: This isn't configurable via the chart yet
		cfg.TenantID = os.Getenv("KARGO_AZURE_TENANT_ID")
	}
	if azureEnv, err := getAzureEnvironment(); err != nil {
		logging.LoggerFromContext(context.Background()).Error(
			err, "failed to get Azure environment details",
		)
	} else {
		cfg.ActiveDirectoryEndpoint = azureEnv.ActiveDirectoryEndpoint
		cfg.AzureServiceManagementEndpoint = azureEnv.ServiceManagementEndpoint
	}
	return cfg
}

type workloadIdentityCredentialHelper struct {
	kargoClient client.Client
	cfg         WorkloadIdentityCredentialHelperConfig
	tokenCache  *cache.Cache

	// The following behaviors are overridable for testing purposes:
	getProjectFn    func(ctx context.Context, project string) (*v1alpha1.Project, error)
	fetchSAFn       func(ctx context.Context, project string) (string, error)
	exchangeTokenFn func(ctx context.Context, token, clientID, scope string) (string, error)
	fetchACRTokenFn func(endpoint, tokenType string, data url.Values) (string, error)
}

// NewWorkloadIdentityCredentialHelper returns an implementation of
// credentials.Helper that utilizes a cache to avoid unnecessary calls to Azure.
func NewWorkloadIdentityCredentialHelper(
	ctx context.Context,
	kargoClient client.Client,
	cfg WorkloadIdentityCredentialHelperConfig,
) credentials.Helper {
	logger := logging.LoggerFromContext(ctx)

	if cfg.TenantID == "" {
		logger.Info(
			"Azure tenant ID could not be determined; Azure Workload Identity " +
				"integration will be disabled",
		)
	}

	if cfg.ActiveDirectoryEndpoint == "" || cfg.AzureServiceManagementEndpoint == "" {
		logger.Info(
			"Azure environment details could not be determined; Azure Workload " +
				"Identity integration will be disabled",
		)
		return nil
	}
	logger.Info("Azure Workload Identity integration is enabled")

	w := &workloadIdentityCredentialHelper{
		kargoClient: kargoClient,
		tokenCache: cache.New(
			// Access tokens live for three hours. We'll hang on to them for 2.5 hours.
			150*time.Minute, // Default ttl for each entry
			time.Hour,       // Cleanup interval
		),
		cfg: cfg,
	}
	w.getProjectFn = w.getProject
	w.fetchSAFn = w.fetchSAToken
	w.exchangeTokenFn = w.exchangeForEntraIDToken
	w.fetchACRTokenFn = fetchACRToken

	return w.getCredentials
}

func (w *workloadIdentityCredentialHelper) getCredentials(
	ctx context.Context,
	project string,
	credType credentials.Type,
	repo string,
	_ *corev1.Secret,
) (*credentials.Credentials, error) {
	const accessTokenUsername = "00000000-0000-0000-0000-000000000000"
	logger := logging.LoggerFromContext(ctx)

	repoURL, err := url.Parse(repo)
	if err != nil {
		return nil, err
	}

	// Workload Identity isn't set up for this controller
	if credType != credentials.TypeImage && credType != credentials.TypeHelm {
		// This helper can't handle this
		return nil, nil
	}

	if credType == credentials.TypeHelm && repoURL.Scheme != "oci" {
		// Only OCI Helm repos are supported in ACR
		return nil, nil
	}

	// TODO: add regex to verify that the URL is an Azure CR URL.

	cacheKey := w.tokenCacheKey(repo, project)

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

	const annotationClientID = "azure.workload.identity/client-id"
	clientID, ok := proj.GetAnnotations()[annotationClientID]
	if !ok {
		return nil, fmt.Errorf("project is missing annotation %q", annotationClientID)
	}

	logger.Debug("Retrieving Project ServiceAccount", "namespace", project)

	// Use the TokenRequest API to get a temporary token for the given project namespace
	saToken, err := w.fetchSAFn(ctx, project)
	if err != nil {
		return nil, err
	}

	repoHost := repoURL.Host
	repoPath := strings.Split(repoURL.Path, "/")[1]

	logger.Debug("Getting Entra OAuth token",
		"repoHost", repoHost, "tenantID", w.cfg.TenantID, "clientID", clientID)

	scope := w.cfg.AzureServiceManagementEndpoint
	// .default needs to be added to the scope
	if !strings.Contains(scope, ".default") {
		scope = fmt.Sprintf("%s/.default", scope)
	}

	authToken, err := w.exchangeTokenFn(
		ctx,
		saToken,
		clientID,
		scope,
	)
	if err != nil {
		return nil, err
	}

	var acrToken string
	// Get a token scoped for the whole ACR registry
	acrToken, err = w.fetchACRTokenFn(
		fmt.Sprintf("https://%s/oauth2/exchange", repoHost),
		"refresh_token",
		url.Values{
			"grant_type":   {"access_token"},
			"service":      {repoHost},
			"tenant":       {w.cfg.TenantID},
			"access_token": {authToken},
		})
	if err != nil {
		return nil, err
	}

	// This part is not required per se - but if we want to scope down access to pull only access for a specific
	// repository, as opposed to full access for the whole registry we need to fetch an access token
	acrToken, err = w.fetchACRTokenFn(
		fmt.Sprintf("https://%s/oauth2/token", repoHost),
		"access_token",
		url.Values{
			"grant_type":    {"refresh_token"},
			"service":       {repoHost},
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

// exchangeForEntraIDToken exchanges a Kubernetes ServiceAccount token for an
// Azure AccessToken.
func (w *workloadIdentityCredentialHelper) exchangeForEntraIDToken(
	ctx context.Context,
	token string,
	clientID string,
	scope string,
) (string, error) {
	cred := confidential.NewCredFromAssertionCallback(
		func(_ context.Context, _ confidential.AssertionRequestOptions) (string, error) {
			return token, nil
		},
	)
	// TODO: The azidentity package has similar functionality. If it works here,
	// it might prove to be more ergonomic.
	cClient, err := confidential.New(
		// TODO: We can probably build this URL just once during initialization
		fmt.Sprintf("%s%s/oauth2/token", w.cfg.ActiveDirectoryEndpoint, w.cfg.TenantID),
		clientID,
		cred,
	)
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

func (w *workloadIdentityCredentialHelper) fetchSAToken(
	ctx context.Context,
	project string,
) (string, error) {
	sa := corev1.ServiceAccount{}
	err := w.kargoClient.Get(ctx, types.NamespacedName{
		Name:      "kargo-viewer", // TODO: This should use its own dedicated ServiceAccount
		Namespace: project,
	}, &sa)
	if err != nil {
		return "", err
	}

	resource := w.kargoClient.SubResource("token")

	tokenReq := &v1.TokenRequest{
		Spec: v1.TokenRequestSpec{
			Audiences: []string{"api://AzureADTokenExchange"},
		},
	}
	err = resource.Create(ctx, &sa, tokenReq)
	if err != nil {
		return "", err
	}

	return tokenReq.Status.Token, nil
}

func fetchACRToken(endpoint, tokenType string, data url.Values) (string, error) {
	res, err := http.PostForm(endpoint, data) //nolint
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
		return "", fmt.Errorf("unable to get token of type %s", tokenType)
	}
	return accessToken, nil
}

// getAzureEnvironment obtains Azure environment details from the Azure Instance
// Metadata Service.
func getAzureEnvironment() (*azure.Environment, error) {
	req, err := http.NewRequest(
		"GET",
		// This is a well-known URL that returns metadata about the Azure
		// environment
		"http://169.254.169.254/metadata/instance?api-version=2021-02-01",
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Metadata", "true")

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get metadata, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var metadata struct {
		Compute struct {
			Environment string `json:"azEnvironment"`
		} `json:"compute"`
	}
	if err = json.Unmarshal(body, &metadata); err != nil {
		return nil, err
	}

	env, err := azure.EnvironmentFromName(metadata.Compute.Environment)
	if err != nil {
		return nil, err
	}

	return &env, nil
}
