package ecr

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
)

const (
	regionKey = "awsRegion"
	idKey     = "awsAccessKeyID"
	secretKey = "awsSecretAccessKey"
)

type accessKeyCredentialHelper struct {
	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAuthTokenFn func(
		ctx context.Context,
		region string,
		accessKeyID string,
		secretAccessKey string,
	) (string, error)
}

// NewAccessKeyCredentialHelper returns an implementation of credentials.Helper
// that utilizes a cache to avoid unnecessary calls to AWS.
func NewAccessKeyCredentialHelper() credentials.Helper {
	a := &accessKeyCredentialHelper{
		tokenCache: cache.New(
			// Tokens live for 12 hours. We'll hang on to them for 10.
			10*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
	}
	a.getAuthTokenFn = a.getAuthToken
	return a.getCredentials
}

func (a *accessKeyCredentialHelper) getCredentials(
	ctx context.Context,
	_ string,
	credType credentials.Type,
	repoURL string,
	secret *corev1.Secret,
) (*credentials.Credentials, error) {
	if credType == credentials.TypeGit || secret == nil {
		// This helper can't handle this
		return nil, nil
	}

	if credType == credentials.TypeHelm && !strings.HasPrefix(repoURL, "oci://") {
		// Only OCI Helm repos are supported in ECR
		return nil, nil
	}

	matches := ecrURLRegex.FindStringSubmatch(repoURL)
	if len(matches) != 2 { // This doesn't look like an ECR URL
		return nil, nil
	}

	region := string(secret.Data[regionKey])
	accessKeyID := string(secret.Data[idKey])
	secretAccessKey := string(secret.Data[secretKey])
	if region == "" && accessKeyID == "" && secretAccessKey == "" {
		// None of these fields are set, so there's nothing to do here.
		return nil, nil
	}
	// If we get to here, at least one of the fields is set. Now if they aren't
	// all set, we should return an error.
	if region == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf(
			"%s, %s, and %s must all be set or all be unset",
			regionKey, idKey, secretKey,
		)
	}

	cacheKey := a.tokenCacheKey(region, accessKeyID, secretAccessKey)

	if entry, exists := a.tokenCache.Get(cacheKey); exists {
		return decodeAuthToken(entry.(string)) // nolint: forcetypeassert
	}

	encodedToken, err := a.getAuthTokenFn(ctx, region, accessKeyID, secretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("error getting ECR auth token: %w", err)
	}

	if encodedToken == "" {
		return nil, nil
	}

	// Cache the encoded token
	a.tokenCache.Set(cacheKey, encodedToken, cache.DefaultExpiration)

	return decodeAuthToken(encodedToken)
}

// tokenCacheKey returns a cache key for an ECR authorization token. The key is
// a hash of the region, access key ID, and secret access key. Using a hash
// ensures that the secret access key is not stored in plaintext in the cache.
func (a *accessKeyCredentialHelper) tokenCacheKey(region, accessKeyID, secretAccessKey string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf("%s:%s:%s", region, accessKeyID, secretAccessKey),
		)),
	)
}

// getAuthToken returns an ECR authorization token by calling out to AWS with
// the provided credentials.
func (a *accessKeyCredentialHelper) getAuthToken(
	ctx context.Context, region, accessKeyID, secretAccessKey string,
) (string, error) {
	svc := ecr.NewFromConfig(aws.Config{
		Region:      region,
		Credentials: awscreds.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
	})
	output, err := svc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", fmt.Errorf("error getting ECR authorization token: %w", err)
	}
	return *output.AuthorizationData[0].AuthorizationToken, nil
}

// decodeAuthToken decodes an ECR authorization token by base64 decoding it and
// splitting it into a username and password.
func decodeAuthToken(token string) (*credentials.Credentials, error) {
	decodedToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("error decoding token: %w", err)
	}
	tokenParts := strings.SplitN(string(decodedToken), ":", 2)
	if len(tokenParts) != 2 {
		// This shouldn't ever happen
		return nil, fmt.Errorf("invalid token format")
	}
	return &credentials.Credentials{
		Username: tokenParts[0],
		Password: tokenParts[1],
	}, nil
}
