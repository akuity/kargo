package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/jferrl/go-githubauth"
	"github.com/patrickmn/go-cache"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
)

const (
	clientIDKey       = "githubAppClientID"
	appIDKey          = "githubAppID"
	installationIDKey = "githubAppInstallationID"
	privateKeyKey     = "githubAppPrivateKey"

	githubBaseURL = "https://github.com"

	accessTokenUsername = "kargo"
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

	getAccessTokenFn func(
		appOrClientID string,
		installationID int64,
		encodedPrivateKey string,
		repoURL string,
	) (string, error)
}

// NewAppCredentialProvider returns an implementation of credentials.Provider.
func NewAppCredentialProvider() credentials.Provider {
	p := &AppCredentialProvider{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
	}
	p.getAccessTokenFn = p.getAccessToken
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
	_ context.Context,
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

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	// Cache miss, get a new token
	accessToken, err := p.getAccessTokenFn(
		appOrClientID,
		installationID,
		encodedPrivateKey,
		repoURL,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting installation access token: %w", err)
	}

	// Cache the new token
	p.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: accessToken,
	}, nil
}

// getAccessToken gets an installation access token for the given app/client ID,
// installation ID, PEM-encoded GitHub App private key, and repo URL.
func (p *AppCredentialProvider) getAccessToken(
	appOrClientID string,
	installationID int64,
	encodedPrivateKey string,
	repoURL string,
) (string, error) {
	decodedKey, err := p.decodeKey(encodedPrivateKey)
	if err != nil {
		return "", err
	}

	appTokenSource, err := githubauth.NewApplicationTokenSource(appOrClientID, decodedKey)
	if err != nil {
		return "", fmt.Errorf("error creating application token source: %w", err)
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
		return "", err
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
		return "", fmt.Errorf("error getting installation access token: %w", err)
	}
	return token.AccessToken, nil
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
		sha256.Sum256([]byte(
			fmt.Sprintf(
				"%s:%d:%s:%s",
				appOrClientID, installationID, encodedPrivateKey, repoURL),
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
	return strings.TrimSuffix(parts[len(parts)-1], ".git")
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
