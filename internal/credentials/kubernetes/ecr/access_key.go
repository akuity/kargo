package ecr

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/patrickmn/go-cache"

	"github.com/akuity/kargo/internal/credentials"
)

const (
	regionKey = "awsRegion"
	idKey     = "awsAccessKeyID"
	secretKey = "awsSecretAccessKey"
)

type AccessKeyProvider struct {
	tokenCache *cache.Cache

	getAuthTokenFn func(ctx context.Context, region, accessKeyID, secretAccessKey string) (string, error)
}

func NewAccessKeyProvider() credentials.Provider {
	p := &AccessKeyProvider{
		tokenCache: cache.New(
			// Tokens live for 12 hours. We'll hang on to them for 10.
			10*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
	}
	p.getAuthTokenFn = p.getAuthToken
	return p
}

func (p *AccessKeyProvider) Supports(credType credentials.Type, repoURL string, data map[string][]byte) bool {
	if (credType != credentials.TypeImage && credType != credentials.TypeHelm) || len(data) == 0 {
		return false
	}

	if credType == credentials.TypeHelm && !strings.HasPrefix(repoURL, "oci://") {
		return false
	}

	if matches := ecrURLRegex.FindStringSubmatch(repoURL); len(matches) != 2 {
		return false
	}

	return data[regionKey] != nil && data[idKey] != nil && data[secretKey] != nil
}

func (p *AccessKeyProvider) GetCredentials(
	ctx context.Context,
	_ string,
	credType credentials.Type,
	repoURL string,
	data map[string][]byte,
) (*credentials.Credentials, error) {
	if !p.Supports(credType, repoURL, data) {
		return nil, nil
	}

	var (
		region, accessKeyID, secretAccessKey = string(data[regionKey]), string(data[idKey]), string(data[secretKey])
		cacheKey                             = tokenCacheKey(region, accessKeyID, secretAccessKey)
	)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return decodeAuthToken(entry.(string)) // nolint: forcetypeassert
	}

	// Cache miss, get a new token
	encodedToken, err := p.getAuthTokenFn(ctx, region, accessKeyID, secretAccessKey)
	if err != nil || encodedToken == "" {
		if err != nil {
			err = fmt.Errorf("error getting ECR auth token: %w", err)
		}
		return nil, err
	}

	// Cache the encoded token
	p.tokenCache.Set(cacheKey, encodedToken, cache.DefaultExpiration)

	return decodeAuthToken(encodedToken)
}

// getAuthToken gets an ECR authorization token using the provided access key ID
// and secret access key. It returns the encoded token, which is a base64 string
// containing a username and password separated by a colon.
func (p *AccessKeyProvider) getAuthToken(
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

	if output == nil || len(output.AuthorizationData) == 0 {
		return "", fmt.Errorf("no authorization data returned")
	}

	if token := output.AuthorizationData[0].AuthorizationToken; token != nil {
		return *token, nil
	}

	return "", fmt.Errorf("no authorization token returned")
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
