package ecr

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/patrickmn/go-cache"

	"github.com/akuity/kargo/internal/logging"
)

var ecrURLRegex = regexp.MustCompile(`^([0-9]{12})\.dkr\.ecr\.(.+)\.amazonaws\.com`)

// PodIdentityCredentialHelper is an interface for components that can obtain a
// username and password for ECR using EKS Pod Identity.
type PodIdentityCredentialHelper interface {
	GetUsernameAndPassword(
		ctx context.Context,
		repoURL string,
		project string,
	) (string, string, error)
}

type podIdentityCredentialHelper struct {
	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAuthTokenFn func(
		ctx context.Context,
		accountID string,
		region string,
		project string,
	) (string, error)
}

// NewPodIdentityCredentialHelper returns an implementation of the
// PodIdentityCredentialHelper interface that utilizes a cache to avoid
// unnecessary calls to AWS.
func NewPodIdentityCredentialHelper() PodIdentityCredentialHelper {

	p := &podIdentityCredentialHelper{
		tokenCache: cache.New(
			// Tokens live for 12 hours. We'll hang on to them for 10.
			10*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
	}
	p.getAuthTokenFn = p.getAuthToken
	return p
}

// GetUsernameAndPassword implements the PodIdentityCredentialHelper interface.
func (p *podIdentityCredentialHelper) GetUsernameAndPassword(
	ctx context.Context,
	repoURL string,
	project string,
) (string, string, error) {
	if os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") == "" {
		// Don't even try if it looks like EKS Pod Identity isn't set up for this
		// controller.
		return "", "", nil
	}

	matches := ecrURLRegex.FindStringSubmatch(repoURL)
	if len(matches) != 3 { // This doesn't look like an ECR URL
		return "", "", nil
	}
	// TODO: We actually might not want to get the account ID from the repoURL
	// because the account ID in the repoURL may be for a different account from
	// the one containing the Kargo controller's IAM role and the Project-specific
	// IAM roles it assumes. (Access across accounts IS possible. It is just not
	// clear to me yet where else I can get the correct account ID from without
	// requiring it to be explicitly configured at install-time.)
	accountID := matches[1]
	region := matches[2]

	cacheKey := p.tokenCacheKey(region, project)

	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return decodeAuthToken(entry.(string)) // nolint: forcetypeassert
	}

	encodedToken, err := p.getAuthTokenFn(ctx, accountID, region, project)
	if err != nil {
		// This might mean the controller's IAM role isn't authorized to assume the
		// project-specific IAM role, or that the project-specific IAM role doesn't
		// have the necessary permissions to get an ECR auth token. We're making
		// a choice to consider this the will of the AWS admins and not a controller
		// error. We'll just log it and move on as if we found no credentials.
		return "", "", fmt.Errorf("error getting ECR auth token: %w", err)
	}

	// Cache the encoded token
	p.tokenCache.Set(project, encodedToken, cache.DefaultExpiration)

	return decodeAuthToken(encodedToken)
}

func (p *podIdentityCredentialHelper) tokenCacheKey(region, project string) string {
	return fmt.Sprintf(
		"%x",
		sha256.Sum256([]byte(
			fmt.Sprintf("%s:%s", region, project),
		)),
	)
}

// getAuthToken returns an ECR authorization token obtained by assuming a
// project-specific IAM role and using that to obtain a short-lived ECR access
// token.
func (p *podIdentityCredentialHelper) getAuthToken(
	ctx context.Context,
	accountID string,
	region string,
	project string,
) (string, error) {
	logger := logging.LoggerFromContext(ctx)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("error loading AWS config: %w", err)
		return "", nil
	}
	ecrSvc := ecr.NewFromConfig(aws.Config{
		Region: region,
		Credentials: stscreds.NewAssumeRoleProvider(
			sts.NewFromConfig(cfg),
			fmt.Sprintf("arn:aws:iam::%s:role/kargo-project-%s", accountID, project),
		),
	})
	output, err := ecrSvc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		logger.Error("error getting ECR authorization token: %w", err)
		return "", nil
	}
	return *output.AuthorizationData[0].AuthorizationToken, nil
}
