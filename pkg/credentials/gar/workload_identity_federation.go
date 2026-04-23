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
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	initMaxAttempts   = 5
	initRetryInterval = time.Second
)

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if p := NewWorkloadIdentityFederationProvider(ctx); p != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: p.Supports,
				Value:     p,
			},
		)
	}
}

// NewWorkloadIdentityFederationProvider returns a fully initialized
// WorkloadIdentityFederationProvider, or nil if the GCP metadata server is
// unreachable after initMaxAttempts attempts. Callers should not register a
// nil provider.
func NewWorkloadIdentityFederationProvider(ctx context.Context) *WorkloadIdentityFederationProvider {
	logger := logging.LoggerFromContext(ctx)
	if !metadata.OnGCE() {
		logger.Info("not running on GCP; skipping initialization of GCP Workload Identity Federation provider")
		return nil
	}

	var projectID string
	var tokenSource oauth2.TokenSource
	if err := retry.OnError(
		wait.Backoff{
			Steps:    initMaxAttempts,
			Duration: initRetryInterval,
		},
		func(error) bool {
			return ctx.Err() == nil
		},
		func() error {
			var err error
			projectID, err = metadata.ProjectIDWithContext(ctx)
			if err != nil {
				return fmt.Errorf("error getting GCP project ID: %w", err)
			}
			tokenSource, err = google.DefaultTokenSource(ctx, iamcredentials.CloudPlatformScope)
			if err != nil {
				return fmt.Errorf("error getting GCP default token source: %w", err)
			}
			return nil
		},
	); err != nil {
		logger.Info("GCP Workload Identity Federation provider could not be initialized", "err", err)
		return nil
	}

	logger.Debug("initialized GCP Workload Identity Federation provider", "project", projectID)

	p := &WorkloadIdentityFederationProvider{
		projectID:   projectID,
		tokenSource: tokenSource,
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40
			// minutes by default. When the actual token expiry is available, it
			// is used (minus a safety margin) instead of this default.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		tokenSourceCache: cache.New(
			// Token sources are long-lived. We could hang on to them indefinitely,
			// but we'll cap it at 12 hours to prevent memory leaks.
			12*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
	}
	p.getAccessTokenFn = p.getAccessToken
	return p
}

type WorkloadIdentityFederationProvider struct {
	tokenCache       *cache.Cache // For short-lived Project-specific tokens
	tokenSourceCache *cache.Cache // For long-lived token sources

	projectID   string
	tokenSource oauth2.TokenSource

	getAccessTokenFn func(
		ctx context.Context,
		project string,
	) (string, time.Time, error)
}

func (p *WorkloadIdentityFederationProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	if req.Type != credentials.TypeImage && req.Type != credentials.TypeHelm {
		return false, nil
	}
	return garURLRegex.MatchString(req.RepoURL) || gcrURLRegex.MatchString(req.RepoURL), nil
}

func (p *WorkloadIdentityFederationProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	cacheKey := tokenCacheKey(req.Project)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "garWorkloadIdentityFederation",
		"repoURL", req.RepoURL,
	)

	// Check the token cache for a Project-specific token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		logger.Debug("access token cache hit")
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	// Check the token source cache for a long-lived token source
	if entry, exists := p.tokenSourceCache.Get(cacheKey); exists {
		logger.Debug("token source cache hit")
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
	logger.Debug("access token cache miss")

	// We had a miss in both caches, so we'll try to get a new Project-specific
	// token.
	accessToken, expiry, err := p.getAccessTokenFn(ctx, req.Project)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}
	if accessToken != "" {
		logger.Debug("obtained new access token")
		ttl := credentials.CalculateCacheTTL(expiry, tokenCacheExpiryMargin)
		logger.Debug(
			"caching access token",
			"expiry", expiry,
			"ttl", ttl,
		)
		p.tokenCache.Set(cacheKey, accessToken, ttl)
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: accessToken,
		}, nil
	}

	// If we get to here, we found no Project-specific token and we'll cache the
	// token source instead.
	logger.Debug("no project-specific token found; caching default token source")
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

// getAccessToken attempts to get a GCP access token scoped to the given Kargo
// project by impersonating the corresponding GCP service account in the
// controller's GCP project via the IAM Credentials API. Returns an empty string
// if no such service account exists, signaling the caller to fall back to the
// controller's own identity.
func (p *WorkloadIdentityFederationProvider) getAccessToken(
	ctx context.Context,
	kargoProject string,
) (string, time.Time, error) {
	logger := logging.LoggerFromContext(ctx)

	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		logger.Error(err, "error creating IAM Credentials service client")
		return "", time.Time{}, nil
	}

	logger = logger.WithValues("gcpProjectID", p.projectID, "kargoProject", kargoProject)

	resp, err := iamSvc.Projects.ServiceAccounts.GenerateAccessToken(
		fmt.Sprintf(
			"projects/-/serviceAccounts/kargo-project-%s@%s.iam.gserviceaccount.com",
			kargoProject, p.projectID,
		),
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
			return "", time.Time{}, nil
		}
		logger.Error(err, "error generating access token")
		return "", time.Time{}, nil
	}

	var expiry time.Time
	if resp.ExpireTime != "" {
		if expiry, err = time.Parse(time.RFC3339, resp.ExpireTime); err != nil {
			logger.Error(err, "error parsing token expiry time; will use default cache TTL")
			expiry = time.Time{}
		}
	}

	logger.Debug("generated Artifact Registry access token")
	return resp.AccessToken, expiry, nil
}
