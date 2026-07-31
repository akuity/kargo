package ecr

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"

	"github.com/akuity/kargo/pkg/cache/coalescing"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	regionKey = "awsRegion"
	idKey     = "awsAccessKeyID"
	secretKey = "awsSecretAccessKey"
)

func init() {
	if provider := NewAccessKeyProvider(); provider != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: provider.Supports,
				Value:     provider,
			},
		)
	}
}

// accessKeyInput identifies the token a load should obtain. The cache key is a
// hash of these values, so they cannot be recovered from it.
type accessKeyInput struct {
	region          string
	accessKeyID     string
	secretAccessKey string
}

type AccessKeyProvider struct {
	// tokenCache holds authorization tokens keyed by a hash of the credentials
	// they were obtained with. It fills its own misses, coalescing concurrent
	// loads for any given key.
	tokenCache coalescing.Cache[accessKeyInput, string]

	getAuthTokenFn func(
		ctx context.Context,
		region string,
		accessKeyID string,
		secretAccessKey string,
	) (string, time.Time, error)
}

func NewAccessKeyProvider() credentials.Provider {
	p := &AccessKeyProvider{}
	p.getAuthTokenFn = p.getAuthToken
	tokenCache, err := coalescing.NewCache(
		p.loadAuthToken,
		&coalescing.CacheOptions{
			LoadTimeout: new(tokenAcquisitionTimeout),
			// Tokens live for 12 hours. We'll hang on to them for 10 by default.
			// When the actual token expiry is available, it is used (minus a
			// safety margin) instead of this default.
			DefaultTTL:      new(10 * time.Hour),
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

func (p *AccessKeyProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	if (req.Type != credentials.TypeImage && req.Type != credentials.TypeHelm) ||
		len(req.Data) == 0 {
		return false, nil
	}
	if matches := ecrURLRegex.FindStringSubmatch(req.RepoURL); len(matches) != 2 {
		return false, nil
	}
	return req.Data[regionKey] != nil &&
		req.Data[idKey] != nil &&
		req.Data[secretKey] != nil, nil
}

func (p *AccessKeyProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	region := string(req.Data[regionKey])
	accessKeyID := string(req.Data[idKey])
	secretAccessKey := string(req.Data[secretKey])
	cacheKey := tokenCacheKey(region, accessKeyID, secretAccessKey)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "ecrAccessKey",
		"repoURL", req.RepoURL,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	encodedToken, err := p.tokenCache.Get(
		ctx,
		cacheKey,
		accessKeyInput{
			region:          region,
			accessKeyID:     accessKeyID,
			secretAccessKey: secretAccessKey,
		},
	)
	if err != nil {
		return nil, err
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if encodedToken == "" {
		return nil, nil
	}

	return decodeAuthToken(encodedToken)
}

// loadAuthToken obtains a new ECR authorization token using the given
// credentials. It is the Loader for this provider's token cache.
func (p *AccessKeyProvider) loadAuthToken(
	ctx context.Context,
	input accessKeyInput,
) (string, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	encodedToken, expiry, err := p.getAuthTokenFn(
		ctx,
		input.region,
		input.accessKeyID,
		input.secretAccessKey,
	)
	if err != nil {
		return "", nil, fmt.Errorf("error getting ECR auth token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if encodedToken == "" {
		return "", nil, nil
	}
	logger.Debug("obtained new auth token")

	ttl := credentials.CalculateCacheTTL(expiry, tokenCacheExpiryMargin)
	if ttl == nil {
		logger.Debug("token expires too soon to be worth caching", "expiry", expiry)
		return encodedToken, nil, nil
	}
	logger.Debug(
		"caching auth token",
		"expiry", expiry,
		"ttl", *ttl,
	)
	return encodedToken, ttl, nil
}

// getAuthToken gets an ECR authorization token using the provided access key ID
// and secret access key. It returns the encoded token, which is a base64 string
// containing a username and password separated by a colon.
func (p *AccessKeyProvider) getAuthToken(
	ctx context.Context, region, accessKeyID, secretAccessKey string,
) (string, time.Time, error) {
	svc := ecr.NewFromConfig(aws.Config{
		Region:      region,
		Credentials: awscreds.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
	})

	output, err := svc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("error getting ECR authorization token: %w", err)
	}

	if output == nil || len(output.AuthorizationData) == 0 {
		return "", time.Time{}, errors.New("no authorization data returned")
	}

	var expiry time.Time
	if output.AuthorizationData[0].ExpiresAt != nil {
		expiry = *output.AuthorizationData[0].ExpiresAt
	}

	if token := output.AuthorizationData[0].AuthorizationToken; token != nil {
		return *token, expiry, nil
	}

	return "", time.Time{}, errors.New("no authorization token returned")
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
		return nil, errors.New("invalid token format")
	}
	return &credentials.Credentials{
		Username: tokenParts[0],
		Password: tokenParts[1],
	}, nil
}
