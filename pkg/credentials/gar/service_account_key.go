package gar

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	serviceAccountKeyKey = "gcpServiceAccountKey"
	scopeStorageRead     = "https://www.googleapis.com/auth/devstorage.read_only"
)

func init() {
	if !credentials.ProvidersEnabled() {
		return
	}
	if provider := NewServiceAccountKeyProvider(); provider != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: provider.Supports,
				Value:     provider,
			},
		)
	}
}

type ServiceAccountKeyProvider struct {
	// tokenCache holds access tokens keyed by a hash of the service account key
	// they were obtained with. It fills its own misses, coalescing concurrent
	// loads for any given service account key.
	tokenCache coalescing.Cache[string, string]

	getAccessTokenFn func(
		ctx context.Context,
		encodedServiceAccountKey string,
	) (*oauth2.Token, error)
}

func NewServiceAccountKeyProvider() credentials.Provider {
	p := &ServiceAccountKeyProvider{}
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
		logging.LoggerFromContext(context.Background()).Error(
			err, "error creating token cache; this provider will not be registered",
		)
		return nil
	}
	p.tokenCache = tokenCache
	return p
}

func (p *ServiceAccountKeyProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	if req.Type != credentials.TypeImage && req.Type != credentials.TypeHelm ||
		req.Data == nil ||
		req.Data[serviceAccountKeyKey] == nil {
		return false, nil
	}
	if !garURLRegex.MatchString(req.RepoURL) &&
		!gcrURLRegex.MatchString(req.RepoURL) {
		return false, nil
	}
	return true, nil
}

func (p *ServiceAccountKeyProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	encodedServiceAccountKey := string(req.Data[serviceAccountKeyKey])
	cacheKey := tokenCacheKey(encodedServiceAccountKey)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "garServiceAccountKey",
		"repoURL", req.RepoURL,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	accessToken, err := p.tokenCache.Get(ctx, cacheKey, encodedServiceAccountKey)
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

// loadAccessToken obtains a GCP access token using the given base64 encoded
// service account key. It is the Loader for this provider's token cache.
func (p *ServiceAccountKeyProvider) loadAccessToken(
	ctx context.Context,
	encodedServiceAccountKey string,
) (string, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	token, err := p.getAccessTokenFn(ctx, encodedServiceAccountKey)
	if err != nil {
		return "", nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if token == nil || token.AccessToken == "" {
		return "", nil, nil
	}
	logger.Debug("obtained new access token")

	ttl := credentials.CalculateCacheTTL(token.Expiry, tokenCacheExpiryMargin)
	if ttl == nil {
		logger.Debug(
			"token expires too soon to be worth caching", "expiry", token.Expiry,
		)
		return token.AccessToken, nil, nil
	}
	logger.Debug(
		"caching access token",
		"expiry", token.Expiry,
		"ttl", *ttl,
	)
	return token.AccessToken, ttl, nil
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func (p *ServiceAccountKeyProvider) getAccessToken(
	ctx context.Context,
	encodedServiceAccountKey string,
) (*oauth2.Token, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(encodedServiceAccountKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding service account key: %w", err)
	}

	config, err := google.JWTConfigFromJSON(decodedKey, scopeStorageRead)
	if err != nil {
		return nil, fmt.Errorf("error parsing service account key: %w", err)
	}

	// oauth2 takes its HTTP client from the context but issues the token request
	// without the context, so a client timeout is the only available bound.
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: cleanhttp.DefaultTransport(),
		Timeout:   tokenRequestTimeout,
	})

	tokenSource := config.TokenSource(ctx)
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}
	return token, nil
}
