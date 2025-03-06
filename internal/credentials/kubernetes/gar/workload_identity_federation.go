package gar

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iamcredentials/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

var tokenRequestScope = []string{
	"https://www.googleapis.com/auth/cloud-platform",
}

type workloadIdentityFederationCredentialHelper struct {
	gcpProjectID string

	tokenSource oauth2.TokenSource

	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAccessTokenFn func(ctx context.Context, kargoProject string) (string, bool, error)
}

// NewWorkloadIdentityFederationCredentialHelper returns an implementation of
// credentials.Helper that utilizes a cache to avoid unnecessary calls to GCP.
func NewWorkloadIdentityFederationCredentialHelper(
	ctx context.Context,
) credentials.Helper {
	logger := logging.LoggerFromContext(ctx)
	var gcpProjectID string
	var tokenSource oauth2.TokenSource
	if !metadata.OnGCE() {
		logger.Info("not running within GCE; assuming GCP Workload Identity Federation is not in use")
		return nil
	}
	logger.Info("controller appears to be running within GCE")
	var err error
	if gcpProjectID, err = metadata.ProjectIDWithContext(ctx); err != nil {
		logger.Error(err, "error getting GCP project ID")
	} else {
		logger.Debug(
			"got GCP project ID",
			"project", gcpProjectID,
		)
	}
	// Configure token source to enable fallback mechanism
	// when project-specific credentials not available
	if tokenSource, err = google.DefaultTokenSource(ctx, tokenRequestScope...); err != nil {
		logger.Error(err, "Failed to retrieved GCP token source from Application Default Credentials")
	} else {
		logger.Debug("Succesfully retrieved GCP token source from Application Default Credentials")
	}

	w := &workloadIdentityFederationCredentialHelper{
		gcpProjectID: gcpProjectID,
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
		// Configure token source to enable fallback mechanism
		// when project-specific credentials not available
		tokenSource: tokenSource,
	}
	w.getAccessTokenFn = w.getAccessToken
	return w.getCredentials
}

func (w *workloadIdentityFederationCredentialHelper) getCredentials(
	ctx context.Context,
	kargoProject string,
	credType credentials.Type,
	repoURL string,
	_ *corev1.Secret,
) (*credentials.Credentials, error) {
	if credType != credentials.TypeImage ||
		w.gcpProjectID == "" { // Controller isn't running within GCE
		// This helper can't handle this
		return nil, nil
	}

	if !garURLRegex.MatchString(repoURL) && !gcrURLRegex.MatchString(repoURL) {
		// This doesn't look like a Google Artifact Registry URL
		return nil, nil
	}

	if entry, exists := w.tokenCache.Get(kargoProject); exists {
		return &credentials.Credentials{
			Username: accessTokenUsername,
			Password: entry.(string), // nolint: forcetypeassert
		}, nil
	}

	accessToken, cacheToken, err := w.getAccessTokenFn(ctx, kargoProject)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	if accessToken == "" {
		return nil, nil
	}

	if cacheToken {
		// Cache the access token if using service account impersonation
		w.tokenCache.Set(kargoProject, accessToken, cache.DefaultExpiration)
	}

	return &credentials.Credentials{
		Username: accessTokenUsername,
		Password: accessToken,
	}, nil
}

// getAccessToken returns a GCP access token retrieved using the provided base64
// encoded service account key. The access token is valid for one hour.
func (w *workloadIdentityFederationCredentialHelper) getAccessToken(
	ctx context.Context,
	kargoProject string,
) (string, bool, error) {
	logger := logging.LoggerFromContext(ctx)
	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		logger.Error(err, "error creating IAM Credentials service client")
		return "", false, nil
	}
	logger = logger.WithValues(
		"gcpProjectID", w.gcpProjectID,
		"kargoProject", kargoProject,
	)
	resp, err := iamSvc.Projects.ServiceAccounts.GenerateAccessToken(
		fmt.Sprintf(
			"projects/-/serviceAccounts/kargo-project-%s@%s.iam.gserviceaccount.com",
			kargoProject, w.gcpProjectID,
		),
		&iamcredentials.GenerateAccessTokenRequest{
			Scope: tokenRequestScope,
		},
	).Do()
	if err != nil {
		var googleErr *googleapi.Error
		if errors.As(err, &googleErr) {
			switch googleErr.Code {
			// fallback to controller identity only if GCP api return 404 code
			// project-specific	service account do not exists
			case 404:
				logger.Debug("falling back to Application Default Credentials (ADC)")
				token, err := w.tokenSource.Token()
				if err != nil {
					logger.Error(err, "Error generating access token from Application Default Credentials")
					return "", false, nil
				}
				logger.Debug("Generated access token using Application Default Credentials")
				return token.AccessToken, false, nil
			default:
				logger.Error(err, "error generating access token")
				return "", false, nil
			}
		}
		logger.Error(err, "error generating access token")
		return "", false, nil
	}
	logger.Debug("generated Artifact Registry access token")
	return resp.AccessToken, true, nil
}
