package gar

import (
	"context"
	"errors"
	"fmt"
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
	tokenCache *cache.Cache

	projectID string

	getAccessTokenFn func(ctx context.Context, project string) (string, bool, error)

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
		logger.Error(err, "error getting GCP project ID")
		return nil
	}
	logger.Debug("got GCP project ID", "project", projectID)
	// Configure DefaultTokenSource as a fallback when project specific Service Account cannot be impersonated
	tS, err := google.DefaultTokenSource(ctx, iamcredentials.CloudPlatformScope)
	if err != nil {
		logger.Info("Fallback to Controller Identity Default Token Source cannot be obtained")
	}

	p := &WorkloadIdentityFederationProvider{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		projectID:   projectID,
		tokenSource: tS,
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

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	// Cache miss, get a new token
	accessToken, cacheToken, err := p.getAccessTokenFn(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return nil, nil
	}

	if cacheToken {
		// Cache the token
		p.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)
	}

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: accessToken,
	}, nil
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func (p *WorkloadIdentityFederationProvider) getAccessToken(
	ctx context.Context,
	kargoProject string,
) (string, bool, error) {
	logger := logging.LoggerFromContext(ctx)

	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		logger.Error(err, "error creating IAM Credentials service client")
		return "", false, nil
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
		if errors.As(err, &googleErr) {
			switch googleErr.Code {
			// fallback to controller identity only if GCP api return 404 code
			// project-specific	service account do not exists
			case 404:
				logger.Debug("falling back to Application Default Credentials (ADC)")
				token, err := p.tokenSource.Token()
				if err != nil {
					logger.Error(err, "Error generating access token from Application Default Credentials")
					return "", false, nil
				}
				logger.Debug("Generated access token using Application Default Credentials")
				return token.AccessToken, false, nil
			default:
				logger.Error(err, "error generating access token")
				return "", false, nil
			}
		}
		logger.Error(err, "error generating access token")
		return "", false, nil
	}

	logger.Debug("generated Artifact Registry access token")
	return resp.AccessToken, true, nil
}
