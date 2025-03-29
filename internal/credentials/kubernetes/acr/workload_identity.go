package acr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
	"github.com/patrickmn/go-cache"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var acrURLRegex = regexp.MustCompile("^(?:oci://)?[a-z0-9]+\\.azurecr.io\\/(.*)")

var errNotConfiguredForProjectIdentities = errors.New("missing ARM_RESOURCE_GROUP or ARM_SUBSCRIPTION_ID, cannot use project identities")
var errProjectIdentityNotFound = errors.New("project identity not found")
var tenantId = os.Getenv("AZURE_TENANT_ID")

const (
	acrSuffix = ".azurecr.io"
	// TODO: Make this configurable to deal with sovereign clouds
	tokenExchangeAudience = "api://AzureADTokenExchange"
)

type WorkloadIdentityProvider struct {
	cache                *cache.Cache
	controllerCredential azcore.TokenCredential
}

func NewWorkloadIdentityProvider(ctx context.Context) credentials.Provider {
	logger := logging.LoggerFromContext(ctx)
	switch {
	case os.Getenv("AZURE_FEDERATED_TOKEN_FILE") != "":
		logger.Info("AKS Workload Identity appears to be in use")
		break
	default:
		logger.Info("AZURE_FEDERATED_TOKEN_FILE is not set; assuming AKS Workload Identity is not in use")
		return nil
	}
	tokenCache := azidentity.Cache{}
	controllerCredential, err := azidentity.NewWorkloadIdentityCredential(&azidentity.WorkloadIdentityCredentialOptions{
		Cache: tokenCache,
	})
	if err != nil {
		logger.Error(err, "unable to initialize AKS Workload Identity")
	}
	return &WorkloadIdentityProvider{
		cache: cache.New(
			60*time.Minute,
			time.Hour,
		),
		controllerCredential: controllerCredential,
	}
}

func (w *WorkloadIdentityProvider) Supports(credType credentials.Type, repoURL string, data map[string][]byte) bool {
	if credType != credentials.TypeImage && credType != credentials.TypeHelm {
		return false
	}
	if credType == credentials.TypeHelm && !strings.HasPrefix(repoURL, "oci://") {
		return false
	}
	if !acrURLRegex.MatchString(repoURL) {
		return false
	}

	return true
}

func newChainedTokenCredential(creds ...azcore.TokenCredential) (*azidentity.ChainedTokenCredential, error) {
	return azidentity.NewChainedTokenCredential(creds, &azidentity.ChainedTokenCredentialOptions{
		RetrySources: false,
	})
}

func (w *WorkloadIdentityProvider) GetCredentials(ctx context.Context, project string, credType credentials.Type, repoURL string, data map[string][]byte) (*credentials.Credentials, error) {
	logger := logging.LoggerFromContext(ctx)

	registryUrl, err := url.Parse(repoURL)
	if err != nil {
		logger.Error(err, "failed to parse ACR host")
		return nil, err
	}

	tokenRequestOptions := policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	}

	credential := w.controllerCredential
	if projectCredential, err := w.getFederatedCredentialForProject(ctx, project); err == nil {
		logger.Info("attempting to use project credential for ACR access")
		chainedCredential, err := newChainedTokenCredential(projectCredential, credential)
		if err != nil {
			logger.Error(err, "failed to create chained credential, falling back to controller credential")
		} else {
			credential = chainedCredential
		}
	}

	accessToken, err := credential.GetToken(ctx, tokenRequestOptions)
	if err != nil {
		logger.Error(err, "failed to get token for ACR")
		return nil, err
	}

	repoHost := registryUrl.Host
	exchangeUrl := fmt.Sprint("https://%s/oauth2/exchange", repoHost)
	resp, err := http.PostForm(exchangeUrl, url.Values{
		"access_token": {accessToken.Token},
		"grant_type":   {"access_token"},
		"service":      {repoHost},
		"tenant":       {tenantId},
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to exchange token: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return nil, err
	}
	repoToken := payload["access_token"]
	return &credentials.Credentials{
		Username: "00000000-0000-0000-0000-000000000000",
		Password: repoToken,
	}, nil
}

func (w *WorkloadIdentityProvider) getFederatedCredentialForProject(ctx context.Context, project string) (azcore.TokenCredential, error) {
	tenantId := os.Getenv("AZURE_TENANT_ID")
	identitySubscriptionId := os.Getenv("ARM_SUBSCRIPTION_ID")
	identityResourceGroup := os.Getenv("ARM_RESOURCE_GROUP")

	if identitySubscriptionId == "" || identityResourceGroup == "" {
		return nil, errNotConfiguredForProjectIdentities
	}

	managedIdentityClient, err := armmsi.NewUserAssignedIdentitiesClient(identitySubscriptionId, w.controllerCredential, nil)
	if err != nil {
		return nil, err
	}

	identityName := fmt.Sprintf("kargo-%s", project)

	identity, err := managedIdentityClient.Get(ctx, identityResourceGroup, identityName, nil)
	if err != nil {
		var respError *azcore.ResponseError
		if errors.As(err, &respError) && respError.StatusCode == 404 {
			return nil, errProjectIdentityNotFound
		}
		return nil, err
	}

	projectClientId := *identity.Properties.ClientID

	return azidentity.NewClientAssertionCredential(tenantId, projectClientId, func(ctx context.Context) (string, error) {
		accessToken, err := w.controllerCredential.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{tokenExchangeAudience},
		})
		if err != nil {
			return "", err
		}

		return accessToken.Token, nil
	}, nil)
}
