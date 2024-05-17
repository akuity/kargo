package ecr

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
)

const (
	regionKey = "awsRegion"
	idKey     = "awsAccessKeyID"
	secretKey = "awsSecretAccessKey"
)

// CredentialHelper is an interface for components that can extract a username
// and password from a Secret containing an AWS region, access key id, and
// secret access key.
type CredentialHelper interface {
	// GetUsernameAndPassword extracts username and password (a token that lives
	// for 12 hours) from a Secret IF the Secret contains an AWS region, access
	// key id, and secret access key. If the Secret does not contain ANY of these
	// fields, this function will return empty strings and a nil error. If the
	// Secret contains some but not all of these fields, this function will return
	// an error. Implementations may cache the token for efficiency.
	GetUsernameAndPassword(*corev1.Secret) (string, string, error)
}

type credentialHelper struct {
	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAuthTokenFn func(string, string, string) (string, error)
}

// NewCredentialHelper returns an implementation of the CredentialHelper
// interface that utilizes a cache to avoid unnecessary calls to AWS.
func NewCredentialHelper() CredentialHelper {
	return &credentialHelper{
		tokenCache: cache.New(
			// Tokens live for 12 hours. We'll hang on to them for 10.
			10*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
		getAuthTokenFn: getAuthToken,
	}
}

// GetUsernameAndPassword implements the CredentialHelper interface.
func (c *credentialHelper) GetUsernameAndPassword(
	secret *corev1.Secret,
) (string, string, error) {
	region := string(secret.Data[regionKey])
	accessKeyID := string(secret.Data[idKey])
	secretAccessKey := string(secret.Data[secretKey])
	if region == "" && accessKeyID == "" && secretAccessKey == "" {
		// None of these fields are set, so there's nothing to do here.
		return "", "", nil
	}
	// If we get to here, at least one of the fields is set. Now if they aren't
	// all set, we should return an error.
	if region == "" || accessKeyID == "" || secretAccessKey == "" {
		return "", "", fmt.Errorf(
			"%s, %s, and %s must all be set or all be unset",
			regionKey, idKey, secretKey,
		)
	}
	return c.getUsernameAndPassword(region, accessKeyID, secretAccessKey)
}

func (c *credentialHelper) getUsernameAndPassword(
	region, accessKeyID, secretAccessKey string,
) (string, string, error) {
	cacheKey := tokenCacheKey(region, accessKeyID, secretAccessKey)

	if entry, exists := c.tokenCache.Get(cacheKey); exists {
		return decodeAuthToken(entry.(string)) // nolint: forcetypeassert
	}

	encodedToken, err := c.getAuthTokenFn(region, accessKeyID, secretAccessKey)
	if err != nil {
		return "", "", fmt.Errorf("error getting ECR auth token: %w", err)
	}

	// Cache the encoded token
	c.tokenCache.Set(cacheKey, encodedToken, cache.DefaultExpiration)

	return decodeAuthToken(encodedToken)
}

// tokenCacheKey returns a cache key for an ECR authorization token. The key is
// a hash of the region, access key ID, and secret access key. Using a hash
// ensures that the secret access key is not stored in plaintext in the cache.
func tokenCacheKey(region, accessKeyID, secretAccessKey string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf("%s:%s:%s", region, accessKeyID, secretAccessKey),
		)),
	)
}

// getAuthToken returns an ECR authorization token by calling out to AWS with
// the provided credentials.
func getAuthToken(
	region, accessKeyID, secretAccessKey string,
) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return "", fmt.Errorf("error creating AWS session: %w", err)
	}
	svc := ecr.New(sess)
	output, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", fmt.Errorf("error getting ECR authorization token: %w", err)
	}
	return *output.AuthorizationData[0].AuthorizationToken, nil
}

// decodeAuthToken decodes an ECR authorization token by base64 decoding it and
// splitting it into a username and password.
func decodeAuthToken(token string) (string, string, error) {
	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", "", fmt.Errorf("error decoding token: %w", err)
	}
	tokenParts := strings.SplitN(string(decodedToken), ":", 2)
	if len(tokenParts) != 2 {
		// This shouldn't ever happen
		return "", "", fmt.Errorf("invalid token format")
	}
	return tokenParts[0], tokenParts[1], nil
}
