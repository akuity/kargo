package warehouses

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
	intpredicate "github.com/akuity/kargo/internal/predicate"
)

type ReconcilerConfig struct {
	ShardName               string `envconfig:"SHARD_NAME"`
	MaxConcurrentReconciles int    `envconfig:"MAX_CONCURRENT_WAREHOUSE_RECONCILES" default:"4"`
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler reconciles Warehouse resources.
type reconciler struct {
	client                     client.Client
	credentialsDB              credentials.Database
	imageSourceURLFnsByBaseURL map[string]func(string, string) string

	// The following behaviors are overridable for testing purposes:

	discoverArtifactsFn func(context.Context, *kargoapi.Warehouse) (*kargoapi.DiscoveredArtifacts, error)

	discoverCommitsFn func(context.Context, string, []kargoapi.RepoSubscription) ([]kargoapi.GitDiscoveryResult, error)

	discoverImagesFn func(context.Context, string, []kargoapi.RepoSubscription) ([]kargoapi.ImageDiscoveryResult, error)

	discoverImageRefsFn func(context.Context, kargoapi.ImageSubscription, *image.Credentials) ([]image.Image, error)

	discoverChartsFn func(context.Context, string, []kargoapi.RepoSubscription) ([]kargoapi.ChartDiscoveryResult, error)

	discoverChartVersionsFn func(context.Context, string, string, string, *helm.Credentials) ([]string, error)

	buildFreightFromLatestArtifactsFn func(string, *kargoapi.DiscoveredArtifacts) (*kargoapi.Freight, error)

	gitCloneFn func(string, *git.ClientOptions, *git.CloneOptions) (git.Repo, error)

	listCommitsFn func(repo git.Repo, limit, skip uint) ([]git.CommitMetadata, error)

	listTagsFn func(repo git.Repo) ([]git.TagMetadata, error)

	discoverBranchHistoryFn func(repo git.Repo, sub kargoapi.GitSubscription) ([]git.CommitMetadata, error)

	discoverTagsFn func(repo git.Repo, sub kargoapi.GitSubscription) ([]git.TagMetadata, error)

	getDiffPathsForCommitIDFn func(repo git.Repo, commitID string) ([]string, error)

	createFreightFn func(context.Context, client.Object, ...client.CreateOption) error

	patchStatusFn func(context.Context, *kargoapi.Warehouse, func(*kargoapi.WarehouseStatus)) error
}

// SetupReconcilerWithManager initializes a reconciler for Warehouse resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	mgr manager.Manager,
	credentialsDB credentials.Database,
	cfg ReconcilerConfig,
) error {
	shardPredicate, err := controller.GetShardPredicate(cfg.ShardName)
	if err != nil {
		return fmt.Errorf("error creating shard selector predicate: %w", err)
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Warehouse{}).
		WithEventFilter(intpredicate.IgnoreDelete[client.Object]{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				kargo.RefreshRequested{},
			),
		).
		WithEventFilter(shardPredicate).
		WithOptions(controller.CommonOptions(cfg.MaxConcurrentReconciles)).
		Complete(newReconciler(mgr.GetClient(), credentialsDB)); err != nil {
		return fmt.Errorf("error building Warehouse reconciler: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized Warehouse reconciler",
		"maxConcurrentReconciles", cfg.MaxConcurrentReconciles,
	)

	return nil
}

func newReconciler(
	kubeClient client.Client,
	credentialsDB credentials.Database,
) *reconciler {
	r := &reconciler{
		client:                  kubeClient,
		credentialsDB:           credentialsDB,
		gitCloneFn:              git.Clone,
		discoverChartVersionsFn: helm.DiscoverChartVersions,
		imageSourceURLFnsByBaseURL: map[string]func(string, string) string{
			githubURLPrefix: getGithubImageSourceURL,
		},
		createFreightFn: kubeClient.Create,
	}

	r.discoverArtifactsFn = r.discoverArtifacts
	r.discoverCommitsFn = r.discoverCommits
	r.discoverImagesFn = r.discoverImages
	r.discoverImageRefsFn = r.discoverImageRefs
	r.discoverChartsFn = r.discoverCharts
	r.buildFreightFromLatestArtifactsFn = r.buildFreightFromLatestArtifacts
	r.listCommitsFn = r.listCommits
	r.listTagsFn = r.listTags
	r.discoverBranchHistoryFn = r.discoverBranchHistory
	r.discoverTagsFn = r.discoverTags
	r.getDiffPathsForCommitIDFn = r.getDiffPathsForCommitID
	r.patchStatusFn = r.patchStatus
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx)

	logger = logger.WithValues(
		"namespace", req.NamespacedName.Namespace,
		"warehouse", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Warehouse")

	// Find the Warehouse
	warehouse, err := api.GetWarehouse(ctx, r.client, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if warehouse == nil {
		// Ignore if not found. This can happen if the Warehouse was deleted after
		// the current reconciliation request was issued.
		return ctrl.Result{}, nil
	}

	newStatus, err := r.syncWarehouse(ctx, warehouse)
	if err != nil {
		logger.Error(err, "error syncing Warehouse")
	}

	updateErr := r.patchStatusFn(
		ctx,
		warehouse,
		func(status *kargoapi.WarehouseStatus) {
			*status = newStatus
		},
	)
	if updateErr != nil {
		logger.Error(updateErr, "error updating Warehouse status")
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}
	logger.Debug("done reconciling Warehouse")

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	if err != nil {
		return ctrl.Result{}, err
	}

	// Everything succeeded, look for new changes on the defined interval.
	return ctrl.Result{RequeueAfter: getRequeueInterval(warehouse)}, nil
}

func (r *reconciler) syncWarehouse(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (kargoapi.WarehouseStatus, error) {
	logger := logging.LoggerFromContext(ctx)

	status := *warehouse.Status.DeepCopy()

	// Record the current refresh token as having been handled.
	if token, ok := api.RefreshAnnotationValue(warehouse.GetAnnotations()); ok {
		status.LastHandledRefresh = token
	}

	// Discover the latest artifacts.
	if shouldDiscoverArtifacts(warehouse, status.LastHandledRefresh) {
		// As this is a long-running operation, we need to update the status
		// conditions to reflect that we are currently reconciling.
		conditions.Set(
			&status,
			&metav1.Condition{
				Type:   kargoapi.ConditionTypeReconciling,
				Status: metav1.ConditionTrue,
				Reason: "ScheduledDiscovery",
				Message: fmt.Sprintf(
					"Discovering artifacts for %d subscriptions",
					len(warehouse.Spec.Subscriptions),
				),
				ObservedGeneration: warehouse.GetGeneration(),
			},
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             "DiscoveryInProgress",
				Message:            "Waiting for discovery to complete",
				ObservedGeneration: warehouse.GetGeneration(),
			},
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionUnknown,
				Reason:             "Pending",
				Message:            "Health status cannot be determined until artifact discovery is finished",
				ObservedGeneration: warehouse.GetGeneration(),
			},
		)
		if err := r.patchStatusFn(ctx, warehouse, func(s *kargoapi.WarehouseStatus) {
			s.SetConditions(status.GetConditions())
		}); err != nil {
			logger.Error(err, "error updating Warehouse status")
		}

		// Discover the latest artifacts.
		discoveredArtifacts, err := r.discoverArtifactsFn(ctx, warehouse)
		if err != nil {
			// Mark the Warehouse as unhealthy and not ready if we failed to
			// discover artifacts.
			conditions.Set(
				&status,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "DiscoveryFailed",
					Message:            fmt.Sprintf("Unable to discover artifacts: %s", err.Error()),
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "DiscoveryFailure",
					Message:            fmt.Sprintf("Artifact discovery failed: %s", err.Error()),
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return status, fmt.Errorf("error discovering artifacts: %w", err)
		}
		logger.Debug("discovered latest artifacts")

		// Update the status with the discovered artifacts, and mark the
		// Warehouse as healthy.
		status.DiscoveredArtifacts = discoveredArtifacts
	}

	// At this point, we have successfully discovered the latest artifacts
	// for the Warehouse using the current subscriptions.
	status.ObservedGeneration = warehouse.GetGeneration()

	// Validate the discovered artifacts.
	if !validateDiscoveredArtifacts(warehouse, &status) {
		// Remove the reconciling condition and return early if the validation
		// failed. We do not return an error here, to prevent a requeue loop
		// which would cause unnecessary pressure on the upstream sources.
		conditions.Delete(&status, kargoapi.ConditionTypeReconciling)
		return status, nil
	}

	// Automatically create a Freight from the latest discovered artifacts
	// if the Warehouse is configured to do so.
	if pol := warehouse.Spec.FreightCreationPolicy; pol == kargoapi.FreightCreationPolicyAutomatic || pol == "" {
		// Mark the Warehouse as reconciling while we create the Freight.
		//
		// As this should be a quick operation, we do not issue an immediate
		// patch to the Warehouse status. However, we do update the status
		// to reflect that we are currently reconciling to ensure it
		// becomes visible when we run into an error.
		conditions.Set(
			&status,
			&metav1.Condition{
				Type:    kargoapi.ConditionTypeReconciling,
				Status:  metav1.ConditionTrue,
				Reason:  "FreightCreationInProgress",
				Message: "Creating Freight from latest artifacts",
			},
			&metav1.Condition{
				Type:    kargoapi.ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  "AwaitingFreightCreation",
				Message: "Freight creation from latest artifacts is in progress",
			},
		)

		// Build a Freight from the latest discovered artifacts.
		freight, err := r.buildFreightFromLatestArtifactsFn(warehouse.Namespace, status.DiscoveredArtifacts)
		if err != nil {
			// Make the error visible in the status and mark the Warehouse as
			// not ready.
			conditions.Set(
				&status,
				&metav1.Condition{
					Type:   kargoapi.ConditionTypeReady,
					Status: metav1.ConditionFalse,
					Reason: "FreightBuildFailure",
					Message: fmt.Sprintf(
						"Error building Freight from latest artifacts: %s",
						err.Error(),
					),
				},
			)

			return status, fmt.Errorf("failed to build Freight from latest artifacts: %w", err)
		}
		freight.Origin = kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: warehouse.Name,
		}

		// Attempt to create the Freight.
		if err = r.createFreightFn(ctx, freight); client.IgnoreAlreadyExists(err) != nil {
			// Make the error visible in the status and mark the Warehouse as
			// not ready.
			conditions.Set(
				&status,
				&metav1.Condition{
					Type:   kargoapi.ConditionTypeReady,
					Status: metav1.ConditionFalse,
					Reason: "FreightCreationFailure",
					Message: fmt.Sprintf(
						"Error creating Freight %q in namespace %q: %s",
						freight.Name,
						freight.Namespace,
						err.Error(),
					),
				},
			)

			return status, fmt.Errorf(
				"error creating Freight %q in namespace %q: %w",
				freight.Name,
				freight.Namespace,
				err,
			)
		} else if err == nil {
			logger.Debug(
				"created Freight",
				"freight", freight.Name,
				"namespace", freight.Namespace,
			)
		}

		status.LastFreightID = freight.Name
	}

	// Remove the reconciling condition and mark the Warehouse as ready.
	conditions.Delete(&status, kargoapi.ConditionTypeReconciling)
	conditions.Set(
		&status,
		&metav1.Condition{
			Type:    kargoapi.ConditionTypeReady,
			Status:  metav1.ConditionTrue,
			Reason:  "ArtifactsDiscovered",
			Message: conditions.Get(&status, kargoapi.ConditionTypeHealthy).Message,
		},
	)

	return status, nil
}

func (r *reconciler) discoverArtifacts(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (*kargoapi.DiscoveredArtifacts, error) {
	commits, err := r.discoverCommitsFn(ctx, warehouse.Namespace, warehouse.Spec.Subscriptions)
	if err != nil {
		return nil, fmt.Errorf("error discovering commits: %w", err)
	}

	images, err := r.discoverImagesFn(ctx, warehouse.Namespace, warehouse.Spec.Subscriptions)
	if err != nil {
		return nil, fmt.Errorf("error discovering images: %w", err)
	}

	charts, err := r.discoverChartsFn(ctx, warehouse.Namespace, warehouse.Spec.Subscriptions)
	if err != nil {
		return nil, fmt.Errorf("error discovering charts: %w", err)
	}

	return &kargoapi.DiscoveredArtifacts{
		DiscoveredAt: metav1.Now(),
		Git:          commits,
		Images:       images,
		Charts:       charts,
	}, nil
}

func (r *reconciler) buildFreightFromLatestArtifacts(
	namespace string,
	artifacts *kargoapi.DiscoveredArtifacts,
) (*kargoapi.Freight, error) {
	if artifacts == nil {
		return nil, fmt.Errorf("no artifacts discovered")
	}

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
	}

	for _, result := range artifacts.Git {
		if len(result.Commits) == 0 {
			return nil, fmt.Errorf("no commits discovered for repository %q", result.RepoURL)
		}
		latestCommit := result.Commits[0]
		freight.Commits = append(freight.Commits, kargoapi.GitCommit{
			RepoURL:   result.RepoURL,
			ID:        latestCommit.ID,
			Branch:    latestCommit.Branch,
			Tag:       latestCommit.Tag,
			Message:   latestCommit.Subject,
			Author:    latestCommit.Author,
			Committer: latestCommit.Committer,
		})
	}

	for _, result := range artifacts.Images {
		if len(result.References) == 0 {
			return nil, fmt.Errorf("no images discovered for repository %q", result.RepoURL)
		}
		latestImage := result.References[0]
		freight.Images = append(freight.Images, kargoapi.Image{
			RepoURL:    result.RepoURL,
			GitRepoURL: latestImage.GitRepoURL,
			Tag:        latestImage.Tag,
			Digest:     latestImage.Digest,
		})
	}

	for _, result := range artifacts.Charts {
		if len(result.Versions) == 0 {
			return nil, fmt.Errorf(
				"no versions discovered for chart %q from repository %q",
				result.RepoURL,
				result.Name,
			)
		}
		latestChart := result.Versions[0]
		freight.Charts = append(freight.Charts, kargoapi.Chart{
			RepoURL: result.RepoURL,
			Name:    result.Name,
			Version: latestChart,
		})
	}

	// Generate a unique ID for the Freight based on its contents.
	freight.Name = api.GenerateFreightID(freight)

	return freight, nil
}

func (r *reconciler) patchStatus(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
	update func(*kargoapi.WarehouseStatus),
) error {
	return kubeclient.PatchStatus(ctx, r.client, warehouse, update)
}

// validateDiscoveredArtifacts validates the discovered artifacts and updates
// the Warehouse status with the results. Returns true if the artifacts are
// valid, false otherwise.
func validateDiscoveredArtifacts(
	warehouse *kargoapi.Warehouse,
	newStatus *kargoapi.WarehouseStatus,
) bool {
	artifacts := newStatus.DiscoveredArtifacts

	if artifacts == nil || len(artifacts.Git)+len(artifacts.Images)+len(artifacts.Charts) == 0 {
		message := "No artifacts discovered"
		conditions.Set(
			newStatus,
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeHealthy,
				Status:             metav1.ConditionFalse,
				Reason:             "MissingArtifacts",
				Message:            message,
				ObservedGeneration: warehouse.GetGeneration(),
			},
			&metav1.Condition{
				Type:               kargoapi.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             "MissingArtifacts",
				Message:            message,
				ObservedGeneration: warehouse.GetGeneration(),
			},
		)
		return false
	}

	var subscriptions int
	var commits int
	for _, artifact := range artifacts.Git {
		count := len(artifact.Commits)

		if count == 0 {
			message := fmt.Sprintf("No commits discovered for Git repository %q", artifact.RepoURL)
			conditions.Set(
				newStatus,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "NoCommitsDiscovered",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "MissingCommits",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return false
		}

		subscriptions++
		commits += count
	}

	var images int
	for _, artifact := range artifacts.Images {
		count := len(artifact.References)

		if count == 0 {
			message := fmt.Sprintf("No references discovered for image repository %q", artifact.RepoURL)
			conditions.Set(
				newStatus,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "NoImageReferencesDiscovered",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "MissingImageReferences",
					Message:            message,
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return false
		}

		subscriptions++
		images += count
	}

	var charts int
	for _, artifact := range artifacts.Charts {
		count := len(artifact.Versions)
		if count == 0 {
			var sb strings.Builder
			_, _ = sb.WriteString("No versions discovered for chart ")
			if artifact.Name != "" {
				_, _ = sb.WriteString(fmt.Sprintf("%q", artifact.Name))
			}
			_, _ = sb.WriteString(" from repository ")
			_, _ = sb.WriteString(fmt.Sprintf("%q", artifact.RepoURL))

			conditions.Set(
				newStatus,
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionFalse,
					Reason:             "NoChartVersionsDiscovered",
					Message:            sb.String(),
					ObservedGeneration: warehouse.GetGeneration(),
				},
				&metav1.Condition{
					Type:               kargoapi.ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					Reason:             "MissingChartVersions",
					Message:            sb.String(),
					ObservedGeneration: warehouse.GetGeneration(),
				},
			)
			return false
		}
		subscriptions++
		charts += count
	}

	var parts []string
	if commits > 0 {
		parts = append(parts, fmt.Sprintf("%d commits", commits))
	}
	if images > 0 {
		parts = append(parts, fmt.Sprintf("%d images", images))
	}
	if charts > 0 {
		parts = append(parts, fmt.Sprintf("%d charts", charts))
	}

	var message string
	if len(parts) == 1 {
		message = parts[0]
	} else if len(parts) == 2 {
		message = parts[0] + " and " + parts[1]
	} else if len(parts) > 2 {
		message = strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}

	conditions.Set(
		newStatus,
		&metav1.Condition{
			Type:   kargoapi.ConditionTypeHealthy,
			Status: metav1.ConditionTrue,
			Reason: "ArtifactsDiscovered",
			Message: fmt.Sprintf(
				"Successfully discovered %s from %d subscriptions",
				message,
				subscriptions,
			),
			ObservedGeneration: warehouse.GetGeneration(),
		},
	)
	return true
}

// shouldDiscoverArtifacts returns true if the Warehouse should attempt to
// discover new artifacts. This is determined by the following conditions:
//
//   - The Warehouse has not yet discovered any artifacts.
//   - The Warehouse has been updated since the last time we discovered artifacts.
//   - The interval has passed since the last time we discovered artifacts.
//   - A manual refresh was requested.
func shouldDiscoverArtifacts(
	warehouse *kargoapi.Warehouse,
	refreshToken string,
) bool {
	switch {
	// We have not yet discovered any artifacts.
	case warehouse.Status.DiscoveredArtifacts == nil:
		return true
	// We have discovered artifacts, but before we started tracking the
	// last time we did so.
	case warehouse.Status.DiscoveredArtifacts.DiscoveredAt.IsZero():
		return true
	// The Warehouse has been updated since the last time we discovered
	// artifacts.
	case warehouse.Generation > warehouse.Status.ObservedGeneration:
		return true
	// A manual refresh was requested.
	case warehouse.Status.LastHandledRefresh != refreshToken:
		return true
	// We have discovered artifacts, but it's been longer than the interval
	// since we last did so.
	case warehouse.Status.DiscoveredArtifacts.DiscoveredAt.Add(
		warehouse.Spec.Interval.Duration,
	).Before(metav1.Now().Time):
		return true
	default:
		return false
	}
}

// getRequeueInterval calculates and returns the time interval remaining until
// the next requeue should occur. If the interval has already passed, it returns
// a zero duration.
func getRequeueInterval(warehouse *kargoapi.Warehouse) time.Duration {
	if warehouse.Status.DiscoveredArtifacts == nil ||
		warehouse.Status.DiscoveredArtifacts.DiscoveredAt.IsZero() {
		return warehouse.Spec.Interval.Duration
	}
	interval := warehouse.Status.DiscoveredArtifacts.DiscoveredAt.
		Add(warehouse.Spec.Interval.Duration).
		Sub(metav1.Now().Time)
	if interval < 0 {
		return 0
	}
	return interval
}
