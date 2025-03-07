package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/jferrl/go-githubauth"
	"github.com/patrickmn/go-cache"

	"github.com/akuity/kargo/internal/credentials"
)

const (
	appIDKey          = "githubAppID"
	installationIDKey = "githubAppInstallationID"
	privateKeyKey     = "githubAppPrivateKey"

	githubHost = "github.com"

	accessTokenUsername = "kargo"
)

var base64Regex = regexp.MustCompile(`^[a-zA-Z0-9+/]*={0,2}$`)

type AppCredentialProvider struct {
	tokenCache *cache.Cache

	getAccessTokenFn func(appID, installationID int64, encodedPrivateKey, baseURL string) (string, error)
}

// NewAppCredentialProvider returns an implementation of credentials.Provider.
func NewAppCredentialProvider() *AppCredentialProvider {
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

func (p *AppCredentialProvider) Supports(credType credentials.Type, _ string, data map[string][]byte) bool {
	if credType != credentials.TypeGit || len(data) == 0 {
		return false
	}

	return data[appIDKey] != nil && data[installationIDKey] != nil && data[privateKeyKey] != nil
}

func (p *AppCredentialProvider) GetCredentials(
	_ context.Context,
	_ string,
	credType credentials.Type,
	repoURL string,
	data map[string][]byte,
) (*credentials.Credentials, error) {
	if !p.Supports(credType, repoURL, data) {
		return nil, nil
	}

	appID, err := strconv.ParseInt(string(data[appIDKey]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing app ID: %w", err)
	}

	installID, err := strconv.ParseInt(string(data[installationIDKey]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing installation ID: %w", err)
	}

	baseURL, err := extractBaseURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("error extracting base URL from : %w", err)
	}

	return p.getUsernameAndPassword(appID, installID, string(data[privateKeyKey]), baseURL)
}

// getUsernameAndPassword gets a username and password for the given app and
// installation IDs. The private key is the PEM-encoded private key for the
// GitHub App. The base URL is the scheme and host of the repository URL, which
// is used to determine whether the repository is hosted on GitHub Enterprise.
func (p *AppCredentialProvider) getUsernameAndPassword(
	appID int64,
	installationID int64,
	encodedPrivateKey, baseURL string,
) (*credentials.Credentials, error) {
	cacheKey := tokenCacheKey(baseURL, appID, installationID, encodedPrivateKey)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	// Cache miss, get a new token
	accessToken, err := p.getAccessTokenFn(
		appID, installationID,
		encodedPrivateKey, baseURL,
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

// getAccessToken gets an installation access token for the given app and
// installation IDs. The private key is the PEM-encoded private key for the
// GitHub App. The base URL is the scheme and host of the repository URL, which
// is used to determine whether the repository is hosted on GitHub Enterprise.
func (p *AppCredentialProvider) getAccessToken(
	appID, installationID int64,
	encodedPrivateKey, baseURL string,
) (string, error) {
	decodedKey, err := decodeKey(encodedPrivateKey)
	if err != nil {
		return "", err
	}

	appTokenSource, err := githubauth.NewApplicationTokenSource(appID, decodedKey)
	if err != nil {
		return "", fmt.Errorf("error creating application token source: %w", err)
	}

	installationOpts := []githubauth.InstallationTokenSourceOpt{
		githubauth.WithHTTPClient(cleanhttp.DefaultClient()),
	}
	if baseURL != "" {
		if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
			return "", fmt.Errorf("can only request access tokens for HTTP or HTTPS URLs")
		}
		if !strings.HasSuffix(baseURL, "://"+githubHost) {
			installationOpts = append(installationOpts, githubauth.WithEnterpriseURLs(baseURL, baseURL))
		}
	}
	installationTokenSource := githubauth.NewInstallationTokenSource(installationID, appTokenSource, installationOpts...)

	token, err := installationTokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("error getting installation access token: %w", err)
	}
	return token.AccessToken, nil
}

// tokenCacheKey returns a cache key for an installation access token. The key
// is a hash of the hostname, app ID, installation ID, and encoded private key.
// Using a hash ensures that a decodable key is not stored in the cache.
func tokenCacheKey(baseURL string, appID, installationID int64, encodedPrivateKey string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf(
				"%s:%d:%d:%s",
				baseURL, appID, installationID, encodedPrivateKey,
			),
		)),
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
func decodeKey(key string) ([]byte, error) {
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

// extractBaseURL extracts the base URL from a full repository URL. The base
// URL is the scheme and host of the repository URL.
func extractBaseURL(fullURL string) (string, error) {
	u, err := url.Parse(fullURL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %w", err)
	}
	return u.Scheme + "://" + u.Host, nil
}
