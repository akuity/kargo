package ecr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/patrickmn/go-cache"

	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
)

const roleARNFormat = "arn:aws:iam::%s:role/kargo-project-%s"

func init() {
	if provider := NewManagedIdentityProvider(context.Background()); provider != nil {
		credentials.DefaultProviderRegistry.MustRegister(
			credentials.ProviderRegistration{
				Predicate: provider.Supports,
				Value:     provider,
			},
		)
	}
}

type ManagedIdentityProvider struct {
	tokenCache *cache.Cache

	accountID string

	getAuthTokenFn func(
		ctx context.Context,
		region string,
		project string,
	) (string, time.Time, error)
}

func NewManagedIdentityProvider(ctx context.Context) credentials.Provider {
	logger := logging.LoggerFromContext(ctx)

	switch {
	case os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "":
		logger.Info("EKS Pod Identity appears to be in use")
	case os.Getenv("AWS_ROLE_ARN") != "" && os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "":
		logger.Info("AWS_WEB_IDENTITY_TOKEN_FILE and AWS_ROLE_ARN set; assuming IRSA is being used")
	default:
		logger.Info("Neither AWS_CONTAINER_CREDENTIALS_FULL_URI nor AWS_WEB_IDENTITY_TOKEN_FILE " +
			"and AWS_ROLE_ARN are set; assuming neither EKS Pod Identity nor IRSA are in use")
		return nil
	}

	var awsAccountID string
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(
			err, "error loading AWS config; AWS credentials integration will be disabled",
		)
		return nil
	}
	cfg.HTTPClient = cleanhttp.DefaultClient()

	stsSvc := sts.NewFromConfig(cfg)
	res, err := stsSvc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		logger.Error(
			err, "error getting caller identity; AWS credentials integration will be disabled",
		)
		return nil
	}

	logger.Debug("got AWS account ID", "account", *res.Account)
	awsAccountID = *res.Account

	p := &ManagedIdentityProvider{
		tokenCache: cache.New(
			// Tokens live for 12 hours. We'll hang on to them for 10 by default.
			// When the actual token expiry is available, it is used (minus a
			// safety margin) instead of this default.
			10*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
		accountID: awsAccountID,
	}
	p.getAuthTokenFn = p.getAuthToken
	return p
}

func (p *ManagedIdentityProvider) Supports(
	_ context.Context,
	req credentials.Request,
) (bool, error) {
	if p.accountID == "" {
		return false, nil
	}
	if req.Type != credentials.TypeImage && req.Type != credentials.TypeHelm {
		return false, nil
	}
	return true, nil
}

func (p *ManagedIdentityProvider) GetCredentials(
	ctx context.Context,
	req credentials.Request,
) (*credentials.Credentials, error) {
	// Extract the region from the ECR URL
	matches := ecrURLRegex.FindStringSubmatch(req.RepoURL)
	if len(matches) != 2 { // This doesn't look like an ECR URL
		return nil, nil
	}

	region := matches[1]
	cacheKey := tokenCacheKey(region, req.Project)

	logger := logging.LoggerFromContext(ctx).WithValues(
		"provider", "ecrManagedIdentity",
		"repoURL", req.RepoURL,
	)

	// Check the cache for the token
	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		logger.Debug("auth token cache hit")
		return decodeAuthToken(entry.(string)) // nolint: forcetypeassert
	}
	logger.Debug("auth token cache miss")

	// Cache miss, get a new token
	encodedToken, expiry, err := p.getAuthTokenFn(ctx, region, req.Project)
	if err != nil {
		// This might mean the controller's IAM role isn't authorized to assume the
		// project-specific IAM role, or that the project-specific IAM role doesn't
		// have the necessary permissions to get an ECR auth token. We're making
		// a choice to consider this the will of the AWS admins and not a controller
		// error. We'll just log it and move on as if we found no credentials.
		return nil, fmt.Errorf("error getting ECR auth token: %w", err)
	}

	// If we didn't get a token, we'll treat this as no credentials found
	if encodedToken == "" {
		return nil, nil
	}
	logger.Debug("obtained new auth token")

	// Cache the encoded token, preferring a TTL derived from the actual token
	// expiry when available.
	ttl := cache.DefaultExpiration
	if !expiry.IsZero() {
		if remaining := time.Until(expiry) - tokenCacheExpiryMargin; remaining > 0 {
			ttl = remaining
		}
	}
	logger.Debug(
		"caching auth token",
		"expiry", expiry,
		"ttl", ttl,
	)
	p.tokenCache.Set(cacheKey, encodedToken, ttl)

	return decodeAuthToken(encodedToken)
}

// getAuthToken returns an ECR authorization token obtained by assuming a
// project-specific IAM role and using that to obtain a short-lived ECR access
// token.
func (p *ManagedIdentityProvider) getAuthToken(
	ctx context.Context,
	region string,
	project string,
) (string, time.Time, error) {
	logger := logging.LoggerFromContext(ctx)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(err, "error loading AWS config")
		return "", time.Time{}, nil
	}
	cfg.HTTPClient = cleanhttp.DefaultClient()

	ecrSvc := ecr.NewFromConfig(aws.Config{
		HTTPClient: cleanhttp.DefaultClient(),
		Region:     region,
		Credentials: stscreds.NewAssumeRoleProvider(
			sts.NewFromConfig(cfg),
			fmt.Sprintf(roleARNFormat, p.accountID, project),
		),
	})

	logger = logger.WithValues(
		"awsAccountID", p.accountID,
		"awsRegion", region,
		"project", project,
	)

	output, err := ecrSvc.GetAuthorizationToken(
		ctx,
		&ecr.GetAuthorizationTokenInput{},
	)
	if err != nil {
		var re *awshttp.ResponseError
		if !errors.As(err, &re) || re.HTTPStatusCode() != http.StatusForbidden {
			return "", time.Time{}, err
		}
		logger.Debug(
			"Controller IAM role is not authorized to assume project-specific role " +
				"or project-specific role is not authorized to obtain an ECR auth token. " +
				"Falling back to using controller's IAM role directly.",
		)

		cfg.Region = region
		ecrSvc = ecr.NewFromConfig(cfg)
		output, err = ecrSvc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			if !errors.As(err, &re) || re.HTTPStatusCode() != http.StatusForbidden {
				return "", time.Time{}, err
			}
			logger.Debug(
				"Controller's IAM role is not authorized to obtain an ECR auth token. " +
					"Treating this as no credentials found.",
			)
			return "", time.Time{}, nil
		}
	}

	var expiry time.Time
	if output.AuthorizationData[0].ExpiresAt != nil {
		expiry = *output.AuthorizationData[0].ExpiresAt
	}

	logger.Debug("got ECR authorization token")
	return *output.AuthorizationData[0].AuthorizationToken, expiry, nil
}
