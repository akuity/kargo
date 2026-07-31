package gar

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iamcredentials/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	initMaxAttempts   = 5
	initRetryInterval = time.Second
)

func init() {
	if p := NewWorkloadIdentityFederationProvider(context.Background()); p != nil {
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

	// The token source built below refreshes itself by issuing an HTTP request,
	// and depending on which credentials Application Default Credentials
	// resolves to, nothing may bound that request: a service account key routes
	// through oauth2's JWT source, which takes its client from this context but
	// issues the request without it. Since a refresh can occur inside a
	// singleflight group, an unanswered request would occupy that group's key for
	// the life of the process. Worse, a token source serializes refreshes behind
	// a mutex, so every caller sharing this one would block, not merely those
	// waiting on the key. A client timeout is the only bound available here.
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: cleanhttp.DefaultTransport(),
		Timeout:   tokenRequestTimeout,
	})

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
		tokenSourceCache: cache.New(
			// Token sources are long-lived. We could hang on to them indefinitely,
			// but we'll cap it at 12 hours to prevent memory leaks.
			12*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
	}
	p.getAccessTokenFn = p.getAccessToken
	tokenCache, err := coalescing.NewCache(
		p.loadAccessToken,
		&coalescing.CacheOptions{
			LoadTimeout: new(tokenAcquisitionTimeout),
			// Access tokens live for one hour. We'll hang on to them for 40
			// minutes by default. When the actual token expiry is available, it
			// is used (minus a safety margin) instead of this default.
			DefaultTTL:      new(40 * time.Minute),
			CleanupInterval: new(time.Hour),
		},
	)
	if err != nil {
		logger.Error(
			err,
			"error creating token cache; GCP Workload Identity Federation "+
				"provider could not be initialized",
		)
		return nil
	}
	p.tokenCache = tokenCache
	return p
}

type WorkloadIdentityFederationProvider struct {
	// tokenCache holds short-lived Project-specific tokens and fills its own
	// misses, coalescing concurrent loads for any given Project.
	tokenCache coalescing.Cache[string, string]
	// tokenSourceCache holds long-lived token sources for Projects that have no
	// Project-specific token to be had.
	tokenSourceCache *cache.Cache

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
	ctx = logging.ContextWithLogger(ctx, logger)

	// Check the token source cache for a long-lived token source. At most one of
	// this provider's two caches holds an entry for any given key, since a load
	// populates whichever of them applies and never both, so the order in which
	// they are consulted does not matter.
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

	accessToken, err := p.tokenCache.Get(ctx, cacheKey, req.Project)
	if err != nil {
		return nil, err
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if accessToken == "" {
		return nil, nil
	}

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: accessToken,
	}, nil
}

// loadAccessToken obtains a GCP access token scoped to the given Kargo Project.
// If no Project-specific token is available, it caches the controller's own
// token source and returns a token from that instead. It is the Loader for this
// provider's token cache.
func (p *WorkloadIdentityFederationProvider) loadAccessToken(
	ctx context.Context,
	project string,
) (string, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	accessToken, expiry, err := p.getAccessTokenFn(ctx, project)
	if err != nil {
		return "", nil, fmt.Errorf("error getting GCP access token: %w", err)
	}
	if accessToken != "" {
		logger.Debug("obtained new access token")
		ttl := credentials.CalculateCacheTTL(expiry, tokenCacheExpiryMargin)
		if ttl == nil {
			logger.Debug("token expires too soon to be worth caching", "expiry", expiry)
			return accessToken, nil, nil
		}
		logger.Debug(
			"caching access token",
			"expiry", expiry,
			"ttl", *ttl,
		)
		return accessToken, ttl, nil
	}

	// If we get to here, we found no Project-specific token and we'll cache the
	// token source instead. The token it yields is deliberately left uncached:
	// the source refreshes itself, so caching the source is what spares future
	// callers this work.
	logger.Debug("no project-specific token found; caching default token source")
	p.tokenSourceCache.Set(tokenCacheKey(project), p.tokenSource, cache.DefaultExpiration)
	token, err := p.tokenSource.Token()
	if err != nil {
		return "", nil, fmt.Errorf("error getting GCP access token: %w", err)
	}
	return token.AccessToken, nil, nil
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
	).Context(ctx).Do()
	if err != nil {
		var googleErr *googleapi.Error
		if errors.As(err, &googleErr) && googleErr.Code == http.StatusNotFound {
			logger.Debug("no Project-specific service account found; will fall back to default token source")
			return "", time.Time{}, nil
		}
		logger.Error(err, "error generating access token")
		return "", time.Time{}, nil
	}

	// A nil response alongside a nil error would be a surprise, but it is
	// reachable: the generated client decodes into a pointer to the response
	// pointer, so a success carrying a JSON null body nils the response and
	// reports no error. Dereferencing it regardless is not worth the
	// consequence: the cache this runs under would report the panic to every
	// caller waiting on this key as an error saying nothing about what actually
	// went wrong.
	if resp == nil {
		logger.Error(nil, "no response generating access token")
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
