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
		accountID string,
		project string,
	) (string, time.Time, error)

	getAuthTokenAsFn func(
		ctx context.Context,
		cfg aws.Config,
		region string,
		identity authIdentity,
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
	p.getAuthTokenAsFn = getAuthTokenAs
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
	// Extract the account ID and region from the ECR URL
	matches := ecrURLRegex.FindStringSubmatch(req.RepoURL)
	if len(matches) != 3 { // This doesn't look like an ECR URL
		return nil, nil
	}

	accountID := matches[1]
	region := matches[2]
	cacheKey := tokenCacheKey(region, accountID, req.Project)

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
	encodedToken, expiry, err := p.getAuthTokenFn(ctx, region, accountID, req.Project)
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

	ttl := credentials.CalculateCacheTTL(expiry, tokenCacheExpiryMargin)
	logger.Debug(
		"caching auth token",
		"expiry", expiry,
		"ttl", ttl,
	)
	p.tokenCache.Set(cacheKey, encodedToken, ttl)

	return decodeAuthToken(encodedToken)
}

// getAuthToken loads the controller's AWS configuration and delegates to
// getAuthTokenWithConfig(). An empty token and nil error are returned if the
// configuration cannot be loaded.
func (p *ManagedIdentityProvider) getAuthToken(
	ctx context.Context,
	region string,
	accountID string,
	project string,
) (string, time.Time, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"registryAWSAccountID", accountID,
		"controllerAWSAccountID", p.accountID,
		"awsRegion", region,
		"project", project,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(err, "error loading AWS config")
		return "", time.Time{}, nil
	}
	cfg.HTTPClient = cleanhttp.DefaultClient()

	return p.getAuthTokenWithConfig(ctx, cfg, region, accountID, project)
}

// getAuthTokenWithConfig returns a short-lived ECR authorization token,
// obtained using the given AWS configuration and the first of the identities
// returned by authIdentities() that both exists and is authorized to obtain
// that token. An empty token and nil error are returned if none of them are.
func (p *ManagedIdentityProvider) getAuthTokenWithConfig(
	ctx context.Context,
	cfg aws.Config,
	region string,
	accountID string,
	project string,
) (string, time.Time, error) {
	logger := logging.LoggerFromContext(ctx)

	for _, identity := range p.authIdentities(accountID, project) {
		token, expiry, err := p.getAuthTokenAsFn(ctx, cfg, region, identity)
		if err == nil {
			logger.Debug(
				"got ECR authorization token",
				"identity", identity.description,
			)
			return token, expiry, nil
		}
		// Only a denial is grounds for trying the next identity. Anything else
		// is a genuine failure and must not be mistaken for one.
		var re *awshttp.ResponseError
		if !errors.As(err, &re) || re.HTTPStatusCode() != http.StatusForbidden {
			return "", time.Time{}, err
		}
		logger.Debug(
			"not authorized to obtain an ECR authorization token",
			"identity", identity.description,
		)
	}

	// Every identity was denied. We're making a choice to consider this the will
	// of the AWS admins and not a controller error, so we treat it as no
	// credentials found.
	logger.Debug("no identity is authorized to obtain an ECR authorization token")
	return "", time.Time{}, nil
}

// authIdentity describes an AWS identity the controller MAY be able to use to
// obtain an ECR authorization token. An empty roleARN denotes the controller's
// own identity, which requires no role assumption.
type authIdentity struct {
	description string
	roleARN     string
}

// authIdentities returns the AWS identities the controller MAY be able to use
// to obtain an ECR authorization token on behalf of the specified Project for a
// registry in the specified account. There is no guarantee that any of these
// exist or that they are authorized to obtain the token. They are simply the
// identities that the controller will try, in order, until one succeeds or all
// fail.
func (p *ManagedIdentityProvider) authIdentities(
	accountID string,
	project string,
) []authIdentity {
	// A Project-specific role in the registry's own account.
	identities := []authIdentity{{
		description: "Project-specific role in the registry's AWS account",
		roleARN:     fmt.Sprintf(roleARNFormat, accountID, project),
	}}

	// A Project-specific role in the controller's own account IF the registry and
	// controller are in different accounts.
	if accountID != p.accountID {
		identities = append(identities, authIdentity{
			description: "Project-specific role in the controller's AWS account",
			roleARN:     fmt.Sprintf(roleARNFormat, p.accountID, project),
		})
	}

	// The controller's own role. This forgoes Project-level isolation, so it is
	// the last resort.
	return append(identities, authIdentity{
		description: "controller's own role",
	})
}

// getAuthTokenAs attempts to obtain an ECR authorization token for the given
// region as the given identity. Errors are returned unexamined; interpreting
// them is the caller's responsibility.
func getAuthTokenAs(
	ctx context.Context,
	cfg aws.Config,
	region string,
	identity authIdentity,
) (string, time.Time, error) {
	if identity.roleARN != "" {
		// The STS client is deliberately built before cfg is retargeted at the
		// registry's region, so that role assumption continues to use whichever
		// region the controller's own configuration resolved to.
		cfg.Credentials = stscreds.NewAssumeRoleProvider(
			sts.NewFromConfig(cfg),
			identity.roleARN,
		)
	}
	cfg.Region = region

	output, err := ecr.NewFromConfig(cfg).GetAuthorizationToken(
		ctx,
		&ecr.GetAuthorizationTokenInput{},
	)
	if err != nil {
		return "", time.Time{}, err
	}

	var expiry time.Time
	if output.AuthorizationData[0].ExpiresAt != nil {
		expiry = *output.AuthorizationData[0].ExpiresAt
	}
	return *output.AuthorizationData[0].AuthorizationToken, expiry, nil
}
