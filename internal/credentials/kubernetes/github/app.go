package github

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/jferrl/go-githubauth"
	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
)

const (
	appIDKey          = "githubAppID"
	installationIDKey = "githubAppInstallationID"
	privateKeyKey     = "githubAppPrivateKey"

	accessTokenUsername = "kargo"
)

var base64Regex = regexp.MustCompile(`^[a-zA-Z0-9+/]*={0,2}$`)

type appCredentialHelper struct {
	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAccessTokenFn func(
		appID int64,
		installationID int64,
		encodedKey string,
	) (string, error)
}

// NewAppCredentialHelper returns an implementation of credentials.Helper that
// utilizes a cache to avoid unnecessary calls to GitHub.
func NewAppCredentialHelper() credentials.Helper {
	a := &appCredentialHelper{
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
	}
	a.getAccessTokenFn = a.getAccessToken
	return a.getCredentials
}

func (a *appCredentialHelper) getCredentials(
	_ context.Context,
	_ string,
	credType credentials.Type,
	_ string,
	secret *corev1.Secret,
) (*credentials.Credentials, error) {
	if credType != credentials.TypeGit || secret == nil {
		// This helper can't handle this
		return nil, nil
	}

	appIDStr := string(secret.Data[appIDKey])
	installationIDStr := string(secret.Data[installationIDKey])
	encodedPrivateKey := string(secret.Data[privateKeyKey])
	if appIDStr == "" && installationIDStr == "" && encodedPrivateKey == "" {
		// None of these fields are set, so there's nothing to do here.
		return nil, nil
	}
	// If we get to here, at least one of the fields is set. Now if they aren't
	// all set, we should return an error.
	if appIDStr == "" || installationIDStr == "" || encodedPrivateKey == "" {
		return nil, fmt.Errorf(
			"%s, %s, and %s must all be set or all be unset",
			appIDKey, installationIDKey, privateKeyKey,
		)
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing app ID: %w", err)
	}
	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing installation ID: %w", err)
	}
	return a.getUsernameAndPassword(appID, installationID, encodedPrivateKey)
}

func (a *appCredentialHelper) getUsernameAndPassword(
	appID int64,
	installationID int64,
	encodedPrivateKey string,
) (*credentials.Credentials, error) {
	cacheKey := a.tokenCacheKey(appID, installationID, encodedPrivateKey)

	if entry, exists := a.tokenCache.Get(cacheKey); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	accessToken, err := a.getAccessTokenFn(appID, installationID, encodedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error getting installation access token: %w", err)
	}

	// Cache the access token
	a.tokenCache.Set(cacheKey, accessToken, cache.DefaultExpiration)

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: accessToken,
	}, nil
}

// tokenCacheKey returns a cache key for an installation access token. The key is
// a hash of the app ID, installation ID, and encoded private key. Using a
// hash ensures that a decodable key is not stored in the cache.
func (a *appCredentialHelper) tokenCacheKey(
	appID int64,
	installationID int64,
	encodedPrivateKey string,
) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf("%d:%d:%s", appID, installationID, encodedPrivateKey),
		)),
	)
}

func (a *appCredentialHelper) getAccessToken(
	appID int64,
	installationID int64,
	encodedPrivateKey string,
) (string, error) {
	decodedKey, err := decodeKey(encodedPrivateKey)
	if err != nil {
		return "", err
	}
	appTokenSource, err := githubauth.NewApplicationTokenSource(appID, decodedKey)
	if err != nil {
		return "", fmt.Errorf("error creating application token source: %w", err)
	}
	installationTokenSource := githubauth.NewInstallationTokenSource(installationID, appTokenSource)
	token, err := installationTokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("error getting installation access token: %w", err)
	}
	return token.AccessToken, nil
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
