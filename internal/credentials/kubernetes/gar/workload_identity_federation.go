package gar

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/iamcredentials/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/logging"
)

type workloadIdentityFederationCredentialHelper struct {
	gcpProjectID string

	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAccessTokenFn func(ctx context.Context, kargoProject string) (string, error)
}

// NewWorkloadIdentityFederationCredentialHelper returns an implementation of
// credentials.Helper that utilizes a cache to avoid unnecessary calls to GCP.
func NewWorkloadIdentityFederationCredentialHelper(
	ctx context.Context,
) credentials.Helper {
	logger := logging.LoggerFromContext(ctx)
	var gcpProjectID string
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
	w := &workloadIdentityFederationCredentialHelper{
		gcpProjectID: gcpProjectID,
		tokenCache: cache.New(
			// Access tokens live for one hour. We'll hang on to them for 40 minutes.
			40*time.Minute, // Default ttl for each entry
			time.Hour,      // Cleanup interval
		),
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

	accessToken, err := w.getAccessTokenFn(ctx, kargoProject)
	if err != nil {
		return nil, fmt.Errorf("error getting GCP access token: %w", err)
	}

	if accessToken == "" {
		return nil, nil
	}

	// Cache the access token
	w.tokenCache.Set(kargoProject, accessToken, cache.DefaultExpiration)

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
) (string, error) {
	logger := logging.LoggerFromContext(ctx)
	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		logger.Error(err, "error creating IAM Credentials service client")
		return "", nil
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
			Scope: []string{
				"https://www.googleapis.com/auth/cloud-platform",
			},
		},
	).Do()
	if err != nil {
		logger.Error(err, "error generating access token")
		return "", nil
	}
	logger.Debug("generated Artifact Registry access token")
	return resp.AccessToken, nil
}
