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

const (
	// roleARNFormat produces the ARN of a Project-specific role in a given AWS
	// account.
	roleARNFormat = "arn:aws:iam::%s:role/kargo-project-%s"

	// externalIDFormat produces the external ID presented when assuming a
	// Project-specific role. It repeats the "kargo-project-" prefix of
	// roleARNFormat instead of sharing it, and deliberately so: the external ID
	// must go on being derived from the Project alone even if the role name ever
	// ceases to be. Collapsing the two into one expression would couple them back
	// together.
	externalIDFormat = "kargo-project-%s"

	// roleSessionNameBase identifies Kargo as the assumer of a Project-specific
	// role. It appears in CloudTrail logs and in the ARN of the resulting
	// session.
	roleSessionNameBase = "kargo-controller"
	// roleSessionNameMaxLength is the longest role session name AWS will accept.
	roleSessionNameMaxLength = 64
)

func init() {
	if !credentials.ProvidersEnabled() {
		return
	}
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
	region    string
	accountID string
	project   string
}

type ManagedIdentityProvider struct {
	// tokenCache holds authorization tokens keyed by a hash of the region and
	// Kargo Project they were obtained for. It fills its own misses, coalescing
	// concurrent loads for any given key.
	tokenCache coalescing.Cache[managedIdentityInput, string]

	accountID string

	// cfg is the controller's own AWS configuration. Retaining and reusing it is
	// safe because the credentials it carries are wrapped in a cache that
	// refreshes them as they approach expiry.
	cfg aws.Config

	// roleSessionName is presented when assuming a Project-specific role. See
	// roleSessionNameFor().
	roleSessionName string

	getAuthTokenFn func(
		ctx context.Context,
		region string,
		accountID string,
		project string,
	) (string, time.Time, error)

	getAuthTokenAsFn func(
		ctx context.Context,
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
	// This configuration lives as long as the provider, which is what makes
	// pooled connections safe to hold onto. They also earn their keep: role
	// assumption on behalf of every Project targets the STS endpoint in the
	// controller's own region, so given enough Projects, requests reach it
	// frequently enough to reuse connections instead of repeatedly negotiating
	// TLS. That is most pronounced at startup, when no authorization tokens are
	// cached yet and every Project needs one.
	cfg.HTTPClient = cleanhttp.DefaultPooledClient()

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

	p := &ManagedIdentityProvider{
		accountID: awsAccountID,
		cfg:       cfg,
		// SHARD_NAME carries the controller's own name, when it has one.
		roleSessionName: roleSessionNameFor(os.Getenv("SHARD_NAME")),
	}
	p.getAuthTokenFn = p.getAuthToken
	p.getAuthTokenAsFn = p.getAuthTokenAs
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
	ctx = logging.ContextWithLogger(ctx, logger)

	encodedToken, err := p.tokenCache.Get(
		ctx,
		cacheKey,
		managedIdentityInput{
			region:    region,
			accountID: accountID,
			project:   req.Project,
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

// loadAuthToken obtains a new ECR authorization token for the given region and
// Kargo Project. It is the Loader for this provider's token cache.
func (p *ManagedIdentityProvider) loadAuthToken(
	ctx context.Context,
	input managedIdentityInput,
) (string, *time.Duration, error) {
	logger := logging.LoggerFromContext(ctx)

	encodedToken, expiry, err := p.getAuthTokenFn(
		ctx,
		input.region,
		input.accountID,
		input.project,
	)
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

// getAuthToken returns a short-lived ECR authorization token, obtained as the
// first of the identities returned by authIdentities() that both exists and is
// authorized to obtain that token. An empty token and nil error are returned if
// none of them are.
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

	for _, identity := range p.authIdentities(accountID, project) {
		token, expiry, err := p.getAuthTokenAsFn(ctx, region, identity)
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
	// externalID is presented when assuming roleARN and is always of the form
	// kargo-project-<project name>. It is set whenever roleARN is. It exists to
	// head off the confused deputy problem described at
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/confused-deputy.html.
	// Kargo is not susceptible to it today: a single controller identity acts on
	// behalf of many Projects, but the role it assumes is computed as a pure
	// function of the name of the Project on whose behalf it is acting. No
	// Project can ask Kargo to assume a role of its own choosing. The scenario
	// the AWS docs describe does not manifest.
	//
	// Sending an external ID regardless is proactive. Were Kargo ever to, in the
	// future, accept a role ARN supplied by a Project, the situation would then
	// match the scenario AWS describes exactly. At that point an external ID that
	// Kargo derives from a Project, without user influence, would be the only
	// thing mitigating the confused deputy scenario, and forward-thinking trust
	// policies written today would already require it.
	externalID string
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
	externalID := fmt.Sprintf(externalIDFormat, project)

	// A Project-specific role in the registry's own account.
	identities := []authIdentity{{
		description: "Project-specific role in the registry's AWS account",
		roleARN:     fmt.Sprintf(roleARNFormat, accountID, project),
		externalID:  externalID,
	}}

	// A Project-specific role in the controller's own account IF the registry and
	// controller are in different accounts.
	if accountID != p.accountID {
		identities = append(identities, authIdentity{
			description: "Project-specific role in the controller's AWS account",
			roleARN:     fmt.Sprintf(roleARNFormat, p.accountID, project),
			externalID:  externalID,
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
func (p *ManagedIdentityProvider) getAuthTokenAs(
	ctx context.Context,
	region string,
	identity authIdentity,
) (string, time.Time, error) {
	// The region override applies only to the ECR client. Role assumption uses
	// whichever region the controller's own configuration resolved to.
	stsSvc := sts.NewFromConfig(p.cfg)
	ecrSvc := ecr.NewFromConfig(p.cfg, func(o *ecr.Options) {
		o.Region = region
		if identity.roleARN != "" {
			o.Credentials = stscreds.NewAssumeRoleProvider(
				stsSvc,
				identity.roleARN,
				func(aro *stscreds.AssumeRoleOptions) {
					aro.RoleSessionName = p.roleSessionName
					// IMPORTANT: See the comment on authIdentity.externalID for why this
					// is done.
					aro.ExternalID = aws.String(identity.externalID)
				},
			)
		}
	})

	output, err := ecrSvc.GetAuthorizationToken(
		ctx,
		&ecr.GetAuthorizationTokenInput{},
	)
	if err != nil {
		return "", time.Time{}, err
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

	return *output.AuthorizationData[0].AuthorizationToken, expiry, nil
}

// roleSessionNameFor returns the session name to present when assuming a
// Project-specific role. The controller's own name, if it has one, is appended,
// so that installations running more than one controller can tell them apart. A
// name long enough to breach AWS's limit is truncated.
func roleSessionNameFor(controllerName string) string {
	if controllerName == "" {
		return roleSessionNameBase
	}
	name := fmt.Sprintf("%s-%s", roleSessionNameBase, controllerName)
	if len(name) > roleSessionNameMaxLength {
		return name[:roleSessionNameMaxLength]
	}
	return name
}
