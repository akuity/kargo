package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
	"github.com/google/go-github/v73/github"
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
	clientIDKey       = "githubAppClientID"
	appIDKey          = "githubAppID"
	installationIDKey = "githubAppInstallationID"
	privateKeyKey     = "githubAppPrivateKey"

	githubHost = "github.com"

	accessTokenUsername = "kargo"
)

var base64Regex = regexp.MustCompile(`^[a-zA-Z0-9+/]*={0,2}$`)

type AppCredentialProvider struct {
	tokenCache *cache.Cache

	getAccessTokenFn func(
		clientID string,
		installationID int64,
		encodedPrivateKey string,
		baseURL string,
		allowedRepos []string,
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
	credType credentials.Type,
	_ string,
	data map[string][]byte,
) bool {
	if credType != credentials.TypeGit || len(data) == 0 {
		return false
	}

	return (strings.TrimSpace(string(data[clientIDKey])) != "" || strings.TrimSpace(string(data[appIDKey])) != "") &&
		strings.TrimSpace(string(data[installationIDKey])) != "" && strings.TrimSpace(string(data[privateKeyKey])) != ""
}

// GetCredentials implements the credentials.Provider interface for GitHub Apps.
// It returns GitHub installation access tokens scoped to a repository, with
// optional restrictions enforced from the `project-repos` annotation.
func (p *AppCredentialProvider) GetCredentials(
	ctx context.Context,
	project string,
	credType credentials.Type,
	repoURL string,
	data map[string][]byte,
) (*credentials.Credentials, error) {
	logger := logging.LoggerFromContext(ctx).WithValues()
	if !p.Supports(credType, repoURL, data) {
		return nil, nil
	}

	// Extract the pproect-repos JSON from annotation
	// in data or directly from data key
	projectReposJSON, hasProjectRepos := data[kargoapi.AnnotationProjectReposKey]

	var allowedRepos []string
	unrestricted := true

	if hasProjectRepos && len(projectReposJSON) > 0 {
		logger.Debug("1")
		var projectRepoMap map[string][]string
		if err := json.Unmarshal(projectReposJSON, &projectRepoMap); err != nil {
			// If JSON is invalid, we mark as restricted (deny by default)
			logger.Debug("error unmarshalling project-repos JSON")
			unrestricted = false
		} else {
			// Look up repos for the specific project
			repos, found := projectRepoMap[project]
			if !found || len(repos) == 0 {
				// project not allowed any repos, deny creds
				return nil, fmt.Errorf("no repositories allowed for project %q", project)
			}
			// otherwise restrict token scope to these repos
			allowedRepos = repos
			unrestricted = false
		}
	}

	// Client ID is the newer unique identifier for GitHub Apps. GitHub recommends
	// using this when possible. If no client ID is found in the data map, we will
	// fall back on the old/deprecated unique identifier, App ID.
	clientID := string(data[clientIDKey])
	if clientID == "" {
		clientID = string(data[appIDKey])
	}

	installID, err := strconv.ParseInt(string(data[installationIDKey]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing installation ID: %w", err)
	}

	baseURL, err := extractBaseURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("error extracting base URL from : %w", err)
	}

	repoName := extractRepoName(repoURL)
	if repoName == "" {
		return nil, fmt.Errorf("could not extract repository name from repo URL %q", repoURL)
	}

	if !unrestricted {
		found := false
		for _, r := range allowedRepos {
			if r == repoName {
				found = true
				break
			}
		}

		if !found {
			// repo not allowed for this project
			return nil, fmt.Errorf("repository %q is not allowed for project %q", repoName, project)
		}
	} else {
		allowedRepos = []string{repoName}
	}

	return p.getUsernameAndPassword(
		clientID,
		installID,
		string(data[privateKeyKey]),
		baseURL,
		allowedRepos,
	)
}

// getUsernameAndPassword gets a username and password for the given client ID
// and installation ID. The private key is the PEM-encoded private key for the
// GitHub App. The base URL is the scheme and host of the repository URL, which
// is used to determine whether the repository is hosted on GitHub Enterprise.
func (p *AppCredentialProvider) getUsernameAndPassword(
	clientID string,
	installationID int64,
	encodedPrivateKey, baseURL string,
	allowedRepos []string,
) (*credentials.Credentials, error) {
	cacheKey := tokenCacheKey(
		baseURL,
		clientID,
		installationID,
		encodedPrivateKey,
		allowedRepos,
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
		clientID,
		installationID,
		encodedPrivateKey,
		baseURL,
		allowedRepos,
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
	clientID string,
	installationID int64,
	encodedPrivateKey string,
	baseURL string,
	allowedRepos []string,
) (string, error) {
	decodedKey, err := decodeKey(encodedPrivateKey)
	if err != nil {
		return "", err
	}

	appTokenSource, err := newApplicationTokenSource(clientID, decodedKey)
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

	if len(allowedRepos) > 0 {
		opts := &github.InstallationTokenOptions{
			Repositories: allowedRepos,
		}
		installationOpts = append(installationOpts, githubauth.WithInstallationTokenOptions(opts))
	}
	installationTokenSource := githubauth.NewInstallationTokenSource(installationID, appTokenSource, installationOpts...)

	token, err := installationTokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("error getting installation access token: %w", err)
	}
	return token.AccessToken, nil
}

// tokenCacheKey returns a cache key for an installation access token. The key
// is a hash of the hostname, client ID, installation ID, and encoded private
// key. Using a hash ensures that a decodable key is not stored in the cache.
func tokenCacheKey(
	baseURL string,
	clientID string,
	installationID int64,
	encodedPrivateKey string,
	allowedRepos []string,
) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf(
				"%s:%s:%d:%s:%s",
				baseURL, clientID, installationID, encodedPrivateKey, strings.Join(allowedRepos, ","),
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

// extractRepoName returns the repository name from a Git repository URL.
// It trims the optional ".git" suffix and then extracts the last path segment.
func extractRepoName(repoURL string) string {
	trimmed := strings.TrimSuffix(repoURL, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
