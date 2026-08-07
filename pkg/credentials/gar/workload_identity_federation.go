package gar

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/hashicorp/go-cleanhttp"
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

	// noProjectTokenTTL is how long the absence of a Project-specific service
	// account is remembered. Nothing about that determination expires, so it is
	// held for a good while: a shorter TTL would buy only a sooner discovery of a
	// service account created after the fact, at the cost of repeating the lookup
	// for every Project that will never have one. It is bounded at all only so
	// that Projects that have come and gone do not accumulate.
	noProjectTokenTTL = 12 * time.Hour
)

func init() {
	if !credentials.ProvidersEnabled() {
		return
	}
	if p := NewWorkloadIdentityFederationProvider(context.Background()); p != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: p.Supports,
				Value:     p,
			},
		)
	}
}

// projectToken is what this provider's token cache holds: either an access
// token scoped to a Kargo Project, or a signal that the Project has no service
// account of its own and the controller's identity is to be used instead.
type projectToken struct {
	accessToken string
	useDefault  bool
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
	// tokenCache holds what is known about each Kargo Project's access to
	// Artifact Registry. It fills its own misses, coalescing concurrent loads for
	// any given Project.
	tokenCache coalescing.Cache[string, projectToken]

	projectID string

	// tokenSource is the controller's own identity, used on behalf of Projects
	// having no service account of their own. It is a cache in its own right,
	// holding the token it last obtained and refreshing it as expiry nears, so
	// what it yields is never cached here. What is worth remembering about such a
	// Project is only that it has nothing better than this.
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
	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "garWorkloadIdentityFederation",
		"repoURL", req.RepoURL,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	cachedToken, err := p.tokenCache.Get(
		ctx,
		tokenCacheKey(req.Project),
		req.Project,
	)
	if err != nil {
		return nil, err
	}

	accessToken := cachedToken.accessToken
	if cachedToken.useDefault {
		// Obtained anew on every call, for the reason given where tokenSource is
		// declared.
		var defaultToken *oauth2.Token
		if defaultToken, err = p.tokenSource.Token(); err != nil {
			return nil, fmt.Errorf("error getting GCP access token: %w", err)
		}
		accessToken = defaultToken.AccessToken
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
// If the Project has no service account of its own, what is returned says so
// instead of carrying a token. It is the Loader for this provider's token
// cache.
func (p *WorkloadIdentityFederationProvider) loadAccessToken(
	ctx context.Context,
	project string,
) (projectToken, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	accessToken, expiry, err := p.getAccessTokenFn(ctx, project)
	if err != nil {
		return projectToken{}, nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	if accessToken == "" {
		logger.Debug("no Project-specific token found; will use default token source")
		return projectToken{useDefault: true}, new(noProjectTokenTTL), nil
	}
	logger.Debug("obtained new access token")

	ttl := credentials.CalculateCacheTTL(expiry, tokenCacheExpiryMargin)
	if ttl == nil {
		logger.Debug("token expires too soon to be worth caching", "expiry", expiry)
		return projectToken{accessToken: accessToken}, nil, nil
	}
	logger.Debug(
		"caching access token",
		"expiry", expiry,
		"ttl", *ttl,
	)
	return projectToken{accessToken: accessToken}, ttl, nil
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
