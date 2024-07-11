package ecr

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

type podIdentityCredentialHelper struct {
	awsAccountID string

	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAuthTokenFn func(
		ctx context.Context,
		region string,
		project string,
	) (string, error)
}

// NewPodIdentityCredentialHelper returns an implementation of
// credentials.Helper that utilizes a cache to avoid unnecessary calls to AWS.
func NewPodIdentityCredentialHelper(ctx context.Context) credentials.Helper {
	logger := logging.LoggerFromContext(ctx)
	var awsAccountID string
	if os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") == "" {
		logger.Info("AWS_CONTAINER_CREDENTIALS_FULL_URI not set; assuming EKS Pod Identity is not in use")
		return nil
	}
	logger.Info("EKS Pod Identity appears to be in use")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(
			err, "error loading AWS config; EKS Pod Identity integration will be disabled",
		)
	} else {
		stsSvc := sts.NewFromConfig(cfg)
		res, err := stsSvc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			logger.Error(
				err, "error getting caller identity; EKS Pod Identity integration will be disabled",
			)
		} else {
			logger.Debug(
				"got AWS account ID",
				"account", *res.Account,
			)
			awsAccountID = *res.Account
		}
	}
	p := &podIdentityCredentialHelper{
		awsAccountID: awsAccountID,
		tokenCache: cache.New(
			// Tokens live for 12 hours. We'll hang on to them for 10.
			10*time.Hour, // Default ttl for each entry
			time.Hour,    // Cleanup interval
		),
	}
	p.getAuthTokenFn = p.getAuthToken
	return p.getCredentials
}

func (p *podIdentityCredentialHelper) getCredentials(
	ctx context.Context,
	project string,
	credType credentials.Type,
	repoURL string,
	_ *corev1.Secret,
) (*credentials.Credentials, error) {
	if (credType != credentials.TypeImage && credType != credentials.TypeHelm) ||
		p.awsAccountID == "" { // Pod Identity isn't set up for this controller
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
	region := matches[1]

	cacheKey := p.tokenCacheKey(region, project)

	if entry, exists := p.tokenCache.Get(cacheKey); exists {
		return decodeAuthToken(entry.(string)) // nolint: forcetypeassert
	}

	encodedToken, err := p.getAuthTokenFn(ctx, region, project)
	if err != nil {
		// This might mean the controller's IAM role isn't authorized to assume the
		// project-specific IAM role, or that the project-specific IAM role doesn't
		// have the necessary permissions to get an ECR auth token. We're making
		// a choice to consider this the will of the AWS admins and not a controller
		// error. We'll just log it and move on as if we found no credentials.
		return nil, fmt.Errorf("error getting ECR auth token: %w", err)
	}

	if encodedToken == "" {
		return nil, nil
	}

	// Cache the encoded token
	p.tokenCache.Set(cacheKey, encodedToken, cache.DefaultExpiration)

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
	region string,
	project string,
) (string, error) {
	logger := logging.LoggerFromContext(ctx)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(err, "error loading AWS config")
		return "", nil
	}
	ecrSvc := ecr.NewFromConfig(aws.Config{
		Region: region,
		Credentials: stscreds.NewAssumeRoleProvider(
			sts.NewFromConfig(cfg),
			fmt.Sprintf("arn:aws:iam::%s:role/kargo-project-%s", p.awsAccountID, project),
		),
	})
	logger = logger.WithValues(
		"awsAccountID", p.awsAccountID,
		"awsRegion", region,
		"project", project,
	)
	output, err := ecrSvc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		var re *awshttp.ResponseError
		if !errors.As(err, &re) || re.HTTPStatusCode() != http.StatusForbidden {
			return "", err
		}
		logger.Debug(
			"controller IAM role is not authorized to assume project-specific role. falling back to default config",
		)
		ecrSvc = ecr.NewFromConfig(cfg)
		output, err = ecrSvc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			logger.Error(err, "error getting ECR authorization token")
			return "", err
		}
	}
	logger.Debug("got ECR authorization token")
	return *output.AuthorizationData[0].AuthorizationToken, nil
}
