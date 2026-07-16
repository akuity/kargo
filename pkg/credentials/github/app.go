package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/jferrl/go-githubauth"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	ghutil "github.com/akuity/kargo/pkg/github"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	clientIDKey       = "githubAppClientID"
	appIDKey          = "githubAppID"
	installationIDKey = "githubAppInstallationID"
	privateKeyKey     = "githubAppPrivateKey"

	githubBaseURL = "https://github.com"

	accessTokenUsername = "kargo"

	tokenCacheExpiryMargin = 5 * time.Minute

	tokenValidationRequestTimeout = 10 * time.Second

	// retryKeySuffix is appended to a token's singleflight key when retrying a
	// mint whose winner's context was canceled, so that all callers recovering
	// from that shared failure coalesce again instead of racing.
	retryKeySuffix = ":retry"
)

var base64Regex = regexp.MustCompile(`^[a-zA-Z0-9+/]*={0,2}$`)

func init() {
	if provider := NewAppCredentialProvider(); provider != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: provider.Supports,
				Value:     provider,
			},
		)
	}
}

type AppCredentialProvider struct {
	tokenCache *cache.Cache

	// mintGroup coalesces concurrent mints of the token for any given cache
	// key into a single mint whose result is shared by all callers.
	mintGroup singleflight.Group

	// validationBackoff is the schedule of waits between successive attempts to validate a newly
	// minted installation access token. This is set as a field so that it can be overridden in
	// tests to avoid long waits.
	validationBackoff []time.Duration

	getAccessTokenFn func(
		appOrClientID string,
		installationID int64,
		encodedPrivateKey string,
		repoURL string,
	) (*oauth2.Token, error)

	validateAccessTokenFn func(
		ctx context.Context,
		accessToken string,
		repoURL string,
	) (bool, error)
}

// NewAppCredentialProvider returns an implementation of credentials.Provider.
func NewAppCredentialProvider() credentials.Provider {
	p := &AppCredentialProvider{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40
			// minutes by default. When the actual token expiry is available, it
			// is used (minus a safety margin) instead of this default.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		// GitHub's own recommendation for retrying use of a newly minted token that has not yet
		// replicated to all of their edge caches. See the Github support response in
		// https://github.com/aws-amplify/amplify-hosting/issues/4080
		validationBackoff: []time.Duration{
			3 * time.Second,
			10 * time.Second,
			30 * time.Second,
		},
	}
	p.getAccessTokenFn = p.getAccessToken
	p.validateAccessTokenFn = p.validateAccessToken
	return p
}

func (p *AppCredentialProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	if req.Type != credentials.TypeGit || len(req.Data) == 0 {
		return false, nil
	}
	return (strings.HasPrefix(req.RepoURL, "http://") || strings.HasPrefix(req.RepoURL, "https://")) &&
		(string(req.Data[clientIDKey]) != "" || string(req.Data[appIDKey]) != "") &&
		string(req.Data[installationIDKey]) != "" &&
		string(req.Data[privateKeyKey]) != "", nil
}

// GetCredentials implements the credentials.Provider interface for GitHub Apps.
// If the provided data represents a GitHub App installation and any optional
// constraints specified by the metadata do not prevent it, this method returns
// an App installation access token that is scoped only to the repository
// specified by repoURL.
func (p *AppCredentialProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	repoName := p.extractRepoName(req.RepoURL)
	if repoName == "" {
		// Doesn't look like a URL we can do anything with.
		return nil, nil
	}

	// If there's a scope map in the metadata, take it into consideration...
	if scopeMapStr := req.Metadata[kargoapi.AnnotationKeyGitHubTokenScope]; scopeMapStr != "" {
		var scopeMap map[string][]string
		if err := json.Unmarshal([]byte(scopeMapStr), &scopeMap); err != nil {
			return nil, fmt.Errorf("error unmarshaling scope map: %w", err)
		}
		if !slices.Contains(scopeMap[req.Project], repoName) {
			// repoName is NOT one of the scopes the Project is allowed to use.
			return nil, nil
		}
	}

	// Client ID is the newer unique identifier for GitHub Apps. GitHub recommends
	// using this when possible. If no client ID is found in the data map, we will
	// fall back on the old/deprecated unique identifier, App ID.
	appOrClientID := string(req.Data[clientIDKey])
	if appOrClientID == "" {
		appOrClientID = string(req.Data[appIDKey])
	}

	installID, err := strconv.ParseInt(string(req.Data[installationIDKey]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing installation ID: %w", err)
	}

	return p.getUsernameAndPassword(
		ctx,
		appOrClientID,
		installID,
		string(req.Data[privateKeyKey]),
		req.RepoURL,
	)
}

// getUsernameAndPassword gets a username (kargo) and password (installation
// access token) for the given app/client ID, installation ID, PEM-encoded
// GitHub App private key, and repo URL.
func (p *AppCredentialProvider) getUsernameAndPassword(
	ctx context.Context,
	appOrClientID string,
	installationID int64,
	encodedPrivateKey string,
	repoURL string,
) (*credentials.Credentials, error) {
	cacheKey := p.tokenCacheKey(
		appOrClientID,
		installationID,
		encodedPrivateKey,
		repoURL,
	)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "githubApp",
		"repoURL", repoURL,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		logger.Debug("installation access token cache hit")
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}
	logger.Debug("installation access token cache miss")

	// Cache miss, get a new token. All consumers of any given repository share one cache entry, so
	// they miss the cache in unison when that entry expires and, without coordination, would each
	// mint their own token. Coalescing concurrent mints for the same cache key avoids that
	// thundering herd.
	accessToken, err, _ := p.mintGroup.Do(cacheKey, func() (any, error) {
		return p.mintAndCacheToken(
			ctx,
			appOrClientID,
			installationID,
			encodedPrivateKey,
			repoURL,
			cacheKey,
		)
	})
	// NOTE(thomastaylor312): This is to handle the edge case where the context of the first minting
	// operation to fire is canceled or timed out, but this context is not. In that case, we immediately
	// re-enqueue a request to mint a new token. Otherwise, if a whole bunch of running promotions
	// use the same credentials and the first one to fire is canceled, all of the others will be
	// canceled as well, which is not what we want. This is best effort to avoid the issue, but it
	// isn't perfect. It only handles _one_ canceled operation, but is better than nothing for now.
	//
	// The retry deliberately runs under a suffixed key rather than Forget + reuse of the primary
	// key. Forget detaches whatever call is currently registered for the key, so with several
	// waiters recovering from the same canceled flight, each Forget could orphan a replacement
	// flight another waiter had already started.
	//
	// Right now if there is a single error that isn't a context error, it will fail all of the
	// other requests as well. This _could_ end up biting us if we have a transient error. If that
	// ever becomes the case, we can add an OR to this that uses the `shared` return value from the
	// singleflight `Do` call and then re-enqueue here as well, but for now we'll let real errors
	// fall through to everyone.
	if err != nil &&
		(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) &&
		ctx.Err() == nil {
		accessToken, err, _ = p.mintGroup.Do(cacheKey+retryKeySuffix, func() (any, error) {
			return p.mintAndCacheToken(
				ctx,
				appOrClientID,
				installationID,
				encodedPrivateKey,
				repoURL,
				cacheKey,
			)
		})
	}
	// At this point if we're checking an error, we've now already retried
	if err != nil {
		return nil, err
	}

	return &credentials.Credentials{
		Username: accessTokenUsername,
		// We know the type is string so we're not checking the assertion here
		Password: accessToken.(string), // nolint: forcetypeassert
	}, nil
}

// mintAndCacheToken mints a new installation access token, waits for it to
// become usable, caches it, and returns it. It is intended to be executed
// within the provider's singleflight group.
func (p *AppCredentialProvider) mintAndCacheToken(
	ctx context.Context,
	appOrClientID string,
	installationID int64,
	encodedPrivateKey string,
	repoURL string,
	cacheKey string,
) (string, error) {
	logger := logging.LoggerFromContext(ctx)

	// A concurrent call may have minted and cached a token between this
	// call's cache miss and its turn in the singleflight group.
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		logger.Debug("installation access token cached by concurrent call")
		return entry.(string), nil // nolint: forcetypeassert
	}

	token, err := p.getAccessTokenFn(
		appOrClientID,
		installationID,
		encodedPrivateKey,
		repoURL,
	)
	if err != nil {
		return "", fmt.Errorf("error getting installation access token: %w", err)
	}
	logger.Debug("obtained new installation access token")

	// GitHub replicates newly minted tokens to its infrastructure
	// asynchronously, so a token used immediately after being minted is
	// sometimes rejected. Since GitHub masks authorization failures on private
	// repositories as 404s, this manifested for a long time as intermittent,
	// difficult-to-diagnose "repository not found" errors. Waiting until the
	// token demonstrably works before releasing it to the caller prevents
	// this.
	if err = p.waitForTokenUsable(ctx, token.AccessToken, repoURL); err != nil {
		return "", err
	}

	ttl := credentials.CalculateCacheTTL(token.Expiry, tokenCacheExpiryMargin)
	logger.Debug(
		"caching installation access token",
		"expiry", token.Expiry,
		"ttl", ttl,
	)
	p.tokenCache.Set(cacheKey, token.AccessToken, ttl)

	return token.AccessToken, nil
}

// waitForTokenUsable checks that a newly minted installation access token is actually usable,
// retrying on a fixed backoff schedule if it is not. A token that cannot be validated after
// exhausting the schedule is presumed usable, since erring on that side leaves the caller no worse
// off than if no validation had been attempted. A non-nil error is returned only if the provided
// context is canceled. This is required due to how GitHub replicates newly minted tokens to its
// edge caches. See https://github.com/aws-amplify/amplify-hosting/issues/4080 for more information
func (p *AppCredentialProvider) waitForTokenUsable(
	ctx context.Context,
	accessToken string,
	repoURL string,
) error {
	logger := logging.LoggerFromContext(ctx)
	start := time.Now()
	// We start at 1 here because this is mostly used for logging and we want the first attempt to
	// be logged as "1" rather than "0". We subtract 1 in the single place we use it as an index
	for attempt := 1; ; attempt++ {
		valid, err := p.validateAccessTokenFn(ctx, accessToken, repoURL)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			logger.Debug(
				"error validating installation access token",
				"attempt", attempt,
				"error", err.Error(),
			)
		}
		if valid {
			logger.Debug(
				"installation access token validated",
				"attempts", attempt,
				"elapsed", time.Since(start),
			)
			return nil
		}
		if attempt > len(p.validationBackoff) {
			logger.Info(
				"proceeding with installation access token that could not be "+
					"validated; it may not have finished replicating on GitHub's "+
					"end and operations using it may fail",
				"attempts", attempt,
				"elapsed", time.Since(start),
			)
			return nil
		}
		timer := time.NewTimer(p.validationBackoff[attempt-1])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// validateAccessToken checks whether an installation access token is usable yet by making a
// lightweight, authenticated request for metadata about the repository to which the token is
// scoped. Installations implicitly have read access to the metadata of any repository they can
// access, so any authorization failure (401, 403, or GitHub's masking of such failures as 404) is
// interpreted as the token not (yet) being usable. Any other failure says nothing about the
// token's validity and is surfaced as an error.
func (p *AppCredentialProvider) validateAccessToken(
	ctx context.Context,
	accessToken string,
	repoURL string,
) (bool, error) {
	_, _, owner, repoName, err := ghutil.ParseRepoURL(repoURL)
	if err != nil {
		return false, err
	}
	client, err := ghutil.NewClient(repoURL, &ghutil.ClientOptions{Token: accessToken})
	if err != nil {
		return false, fmt.Errorf("error creating GitHub client: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, tokenValidationRequestTimeout)
	defer cancel()
	_, resp, err := client.Repositories.Get(ctx, owner, repoName)
	if err == nil {
		return true, nil
	}
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
			// These are the failures with the potential to resolve on their own
			// once the token has finished replicating on GitHub's end.
			return false, nil
		}
	}
	return false, fmt.Errorf("error validating installation access token: %w", err)
}

// getAccessToken gets an installation access token for the given app/client ID,
// installation ID, PEM-encoded GitHub App private key, and repo URL.
func (p *AppCredentialProvider) getAccessToken(
	appOrClientID string,
	installationID int64,
	encodedPrivateKey string,
	repoURL string,
) (*oauth2.Token, error) {
	decodedKey, err := p.decodeKey(encodedPrivateKey)
	if err != nil {
		return nil, err
	}

	appTokenSource, err := githubauth.NewApplicationTokenSource(appOrClientID, decodedKey)
	if err != nil {
		return nil, fmt.Errorf("error creating application token source: %w", err)
	}

	installationOpts := []githubauth.InstallationTokenSourceOpt{
		githubauth.WithHTTPClient(cleanhttp.DefaultClient()),
		// In all cases, the access token is scoped only to the repo specified by
		// repoURL.
		githubauth.WithInstallationTokenOptions(
			&githubauth.InstallationTokenOptions{
				Repositories: []string{p.extractRepoName(repoURL)},
			},
		),
	}
	baseURL, err := p.extractBaseURL(repoURL)
	if err != nil {
		return nil, err
	}
	if baseURL != githubBaseURL {
		// This looks like a GitHub Enterprise URL
		installationOpts = append(installationOpts, githubauth.WithEnterpriseURL(baseURL))
	}

	installationTokenSource := githubauth.NewInstallationTokenSource(
		installationID,
		appTokenSource,
		installationOpts...,
	)

	token, err := installationTokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting installation access token: %w", err)
	}
	return token, nil
}

// tokenCacheKey returns a cache key for an installation access token. The key
// is a hash of the app/client ID, installation ID, PEM-encoded GitHub App
// private key, and repo URL. Using a hash ensures that a private decodable key
// is not stored in the cache.
func (p *AppCredentialProvider) tokenCacheKey(
	appOrClientID string,
	installationID int64,
	encodedPrivateKey string,
	repoURL string,
) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256(
			[]byte(
				fmt.Sprintf(
					"%s:%d:%s:%s",
					appOrClientID, installationID, encodedPrivateKey, repoURL,
				),
			),
		),
	)
}

// decodeKey attempts to base64 decode a key. If successful, it returns the
// result. If it fails, it attempts to infer whether the input was simply NOT
// base64 encoded or whether it appears to have been base64 encoded but
// corrupted -- due perhaps to a copy/paste error. This inference determines
// whether to return the input as is or surface the decoding error. All other
// errors are surfaced as is. This function is necessary because we initially
// required the PEM-encoded key to be base64 encoded (for reasons unknown today)
// and then we later dropped that requirement.
func (p *AppCredentialProvider) decodeKey(key string) ([]byte, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		if !errors.As(err, new(base64.CorruptInputError)) {
			return nil, fmt.Errorf("error decoding private key: %w", err)
		}
		if base64Regex.MatchString(key) {
			return nil, fmt.Errorf(
				"probable corrupt base64 encoding of private key; base64 encoding "+
					"this key is no longer required and is discouraged: %w", err,
			)
		}
		return []byte(key), nil
	}
	return decodedKey, nil
}

// extractRepoName returns the repository name from a GitHub repository URL.
// This function assumes the provided repoURL has been normalized by the caller.
func (p *AppCredentialProvider) extractRepoName(repoURL string) string {
	parts := strings.Split(repoURL, "/")
	// A valid GitHub repo URL should have no fewer than five parts when split
	// on a forward slash. A GitHub Enterprise URL could theoretically have more.
	if len(parts) < 5 {
		return ""
	}
	return parts[len(parts)-1]
}

// extractBaseURL extracts the base URL from a full repository URL. The base
// URL is the scheme and host of the repository URL.
func (p *AppCredentialProvider) extractBaseURL(fullURL string) (string, error) {
	u, err := url.Parse(fullURL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %w", err)
	}
	return u.Scheme + "://" + u.Host, nil
}
