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

	"github.com/akuity/kargo/pkg/cache/coalescing"
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

// managedIdentityInput identifies the token a load should obtain. The cache key
// is a hash of these values, so they cannot be recovered from it.
type managedIdentityInput struct {
	region  string
	project string
}

type ManagedIdentityProvider struct {
	// tokenCache holds authorization tokens keyed by a hash of the region and
	// Kargo Project they were obtained for. It fills its own misses, coalescing
	// concurrent loads for any given key.
	tokenCache coalescing.Cache[managedIdentityInput, string]

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

	// A response carrying no account would be a surprise, but this runs at
	// startup, where a panic takes the whole process with it.
	if res.Account == nil {
		logger.Error(
			nil, "no account ID returned; AWS credentials integration will be disabled",
		)
		return nil
	}
	logger.Debug("got AWS account ID", "account", *res.Account)
	awsAccountID = *res.Account

	p := &ManagedIdentityProvider{accountID: awsAccountID}
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
		logger.Error(
			err, "error creating token cache; AWS credentials integration will be disabled",
		)
		return nil
	}
	p.tokenCache = tokenCache
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
	return ecrURLRegex.MatchString(req.RepoURL), nil
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
	ctx = logging.ContextWithLogger(ctx, logger)

	encodedToken, err := p.tokenCache.Get(
		ctx,
		cacheKey,
		managedIdentityInput{region: region, project: req.Project},
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

// loadAuthToken obtains a new ECR authorization token for the given region and
// Kargo Project. It is the Loader for this provider's token cache.
func (p *ManagedIdentityProvider) loadAuthToken(
	ctx context.Context,
	input managedIdentityInput,
) (string, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	encodedToken, expiry, err := p.getAuthTokenFn(ctx, input.region, input.project)
	if err != nil {
		// An IAM role not authorized to assume the project-specific role, or a
		// project-specific role not authorized to obtain an ECR auth token, is
		// taken to be the will of the AWS admins rather than an error, and reaches
		// here as an empty token rather than as this error. See getAuthToken.
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

	// A response carrying no authorization data would be a surprise, but indexing
	// into it regardless is not worth the consequence: the cache this runs under
	// would report the panic to every caller waiting on this key as an error
	// saying nothing about what actually went wrong.
	if output == nil || len(output.AuthorizationData) == 0 {
		return "", time.Time{}, errors.New("no authorization data returned")
	}

	var expiry time.Time
	if output.AuthorizationData[0].ExpiresAt != nil {
		expiry = *output.AuthorizationData[0].ExpiresAt
	}

	if output.AuthorizationData[0].AuthorizationToken == nil {
		return "", time.Time{}, errors.New("no authorization token returned")
	}

	logger.Debug("got ECR authorization token")
	return *output.AuthorizationData[0].AuthorizationToken, expiry, nil
}
