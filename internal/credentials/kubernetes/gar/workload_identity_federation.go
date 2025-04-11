package gar

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iamcredentials/v1"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

const (
	gcpResourceNameFormat = "projects/-/serviceAccounts/kargo-project-%s@%s.iam.gserviceaccount.com"
)

type WorkloadIdentityFederationProvider struct {
	tokenCache       *cache.Cache // For short-lived Project-specific tokens
	tokenSourceCache *cache.Cache // For long-lived token sources

	projectID string

	getAccessTokenFn func(ctx context.Context, project string) (string, error)

	tokenSource oauth2.TokenSource
}

func NewWorkloadIdentityFederationProvider(ctx context.Context) credentials.Provider {
	logger := logging.LoggerFromContext(ctx)

	if !metadata.OnGCE() {
		logger.Info("not running within GCE; assuming GCP Workload Identity Federation is not in use")
		return nil
	}
	logger.Info("controller appears to be running within GCE")

	projectID, err := metadata.ProjectIDWithContext(ctx)
	if err != nil {
		logger.Error(
			err,
			"error getting GCP project ID; GCP Workload Identity Federation disabled",
		)
		return nil
	}
	logger.Debug("got GCP project ID", "project", projectID)

	tokenSource, err := google.DefaultTokenSource(ctx, iamcredentials.CloudPlatformScope)
	if err != nil {
		logger.Error(
			err,
			"error getting GCP default token source; GCP Workload Identity Federation disabled",
		)
		return nil
	}

	p := &WorkloadIdentityFederationProvider{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		tokenSourceCache: cache.New(
			// Token sources are long-lived. We could hang on to them indefinitely,
			// but we'll cap it at 12 hours to prevent memory leaks.
			12*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
		projectID:   projectID,
		tokenSource: tokenSource,
	}
	p.getAccessTokenFn = p.getAccessToken
	return p
}

func (p *WorkloadIdentityFederationProvider) Supports(
	credType credentials.Type,
	repoURL string,
	_ map[string][]byte,
) bool {
	if p.projectID == "" || credType != credentials.TypeImage {
		return false
	}

	if !garURLRegex.MatchString(repoURL) && !gcrURLRegex.MatchString(repoURL) {
		return false
	}

	return true
}

func (p *WorkloadIdentityFederationProvider) GetCredentials(
	ctx context.Context,
	project string,
	credType credentials.Type,
	repoURL string,
	_ map[string][]byte,
) (*credentials.Credentials, error) {
	if !p.Supports(credType, repoURL, nil) {
		return nil, nil
	}

	var cacheKey = tokenCacheKey(project)

	// Check the token cache for a Project-specific token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	// Check the token source cache for a long-lived token source
	if entry, exists := p.tokenSourceCache.Get(cacheKey); exists {
		tokenSource := entry.(oauth2.TokenSource) // nolint: forcetypeassert
		token, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("error getting GCP access token: %w", err)
		}
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: token.AccessToken,
		}, nil
	}

	// We had a miss in both caches, so we'll try to get a new Project-specific
	// token.
	accessToken, err := p.getAccessTokenFn(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}
	if accessToken != "" {
		p.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: accessToken,
		}, nil
	}

	// If we get to here, we found no Project-specific token and we'll cache the
	// token source instead.
	p.tokenSourceCache.Set(cacheKey, p.tokenSource, cache.DefaultExpiration)
	token, err := p.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}
	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: token.AccessToken,
	}, nil
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func (p *WorkloadIdentityFederationProvider) getAccessToken(
	ctx context.Context,
	kargoProject string,
) (string, error) {
	logger := logging.LoggerFromContext(ctx)

	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		logger.Error(err, "error creating IAM Credentials service client")
		return "", nil
	}

	logger = logger.WithValues("gcpProjectID", p.projectID, "kargoProject", kargoProject)

	resp, err := iamSvc.Projects.ServiceAccounts.GenerateAccessToken(
		fmt.Sprintf(gcpResourceNameFormat, kargoProject, p.projectID),
		&iamcredentials.GenerateAccessTokenRequest{
			Scope: []string{
				iamcredentials.CloudPlatformScope,
			},
		},
	).Do()
	if err != nil {
		var googleErr *googleapi.Error
		if errors.As(err, &googleErr) && googleErr.Code == http.StatusNotFound {
			logger.Debug("no Project-specific service account found; will fall back to default token source")
			return "", nil
		}
		logger.Error(err, "error generating access token")
		return "", nil
	}

	logger.Debug("generated Artifact Registry access token")
	return resp.AccessToken, nil
}
