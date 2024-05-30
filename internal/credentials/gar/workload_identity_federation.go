package gar

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/iamcredentials/v1"

	"github.com/akuity/kargo/internal/logging"
)

var (
	gcrURLRegex = regexp.MustCompile(`^(?:.+\.)?gcr\.io/`) // Legacy
	garURLRegex = regexp.MustCompile(`^.+-docker\.pkg\.dev/`)
)

// WorkloadIdentityFederationCredentialHelper is an interface for components
// that can obtain a username and password for Google Artifact Registry using
// GCP Workload Identity Federation.
type WorkloadIdentityFederationCredentialHelper interface {
	GetUsernameAndPassword(ctx context.Context, repoURL, kargoProject string) (string, string, error)
}

type workloadIdentityFederationCredentialHelper struct {
	gcpProjectID string

	tokenCache *cache.Cache

	// The following behaviors are overridable for testing purposes:

	getAccessTokenFn func(ctx context.Context, kargoProject string) (string, error)
}

// NewWorkloadIdentityFederationCredentialHelper returns an implementation of
// the WorkloadIdentityFederationCredentialHelper interface that utilizes a
// cache to avoid unnecessary calls to GCP.
func NewWorkloadIdentityFederationCredentialHelper(
	ctx context.Context,
) WorkloadIdentityFederationCredentialHelper {
	logger := logging.LoggerFromContext(ctx)
	var gcpProjectID string
	if !metadata.OnGCE() {
		logger.Info("not running within GCE; assuming GCP Workload Identity Federation is not in use")
	} else {
		logger.Info("controller appears to be running within GCE")
		var err error
		if gcpProjectID, err = metadata.ProjectID(); err != nil {
			logger.Errorf("error getting GCP project ID: %s", err)
		} else {
			logger.WithField("project", gcpProjectID).Debug("got GCP project ID")
		}
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
	return w
}

// GetUsernameAndPassword implements the
// WorkloadIdentityFederationCredentialHelper interface.
func (w *workloadIdentityFederationCredentialHelper) GetUsernameAndPassword(
	ctx context.Context,
	repoURL string,
	kargoProject string,
) (string, string, error) {
	if w.gcpProjectID == "" {
		// Don't even try if it looks like the controller isn't running within GCE
		return "", "", nil
	}

	if !garURLRegex.MatchString(repoURL) && !gcrURLRegex.MatchString(repoURL) {
		// This doesn't look like a Google Artifact Registry URL
		return "", "", nil
	}

	if entry, exists := w.tokenCache.Get(kargoProject); exists {
		return accessTokenUsername, entry.(string), nil // nolint: forcetypeassert
	}

	accessToken, err := w.getAccessTokenFn(ctx, kargoProject)
	if err != nil {
		return "", "", fmt.Errorf("error getting GCP access token: %w", err)
	}

	if accessToken == "" {
		return "", "", nil
	}

	// Cache the access token
	w.tokenCache.Set(kargoProject, accessToken, cache.DefaultExpiration)

	return accessTokenUsername, accessToken, nil
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
		logger.Errorf("error creating IAM Credentials service client: %s", err)
		return "", nil
	}
	logger = logger.WithFields(map[string]any{
		"gcpProjectID": w.gcpProjectID,
		"kargoProject": kargoProject,
	})
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
		logger.Errorf("error generating access token: %s", err)
		return "", nil
	}
	logger.Debug("generated Artifact Registry access token")
	return resp.AccessToken, nil
}
