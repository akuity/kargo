package warehouses

import (
	"context"
	"fmt"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

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

	listFreightFn func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error

	createFreightFn func(context.Context, client.Object, ...client.CreateOption) error
}

// SetupReconcilerWithManager initializes a reconciler for Warehouse resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	mgr manager.Manager,
	credentialsDB credentials.Database,
	shardName string,
) error {

	shardPredicate, err := controller.GetShardPredicate(shardName)
	if err != nil {
		return fmt.Errorf("error creating shard selector predicate: %w", err)
	}

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Warehouse{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					// We're not interested in any deletes
					return false
				},
			},
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				kargo.RefreshRequested{},
			),
		).
		WithEventFilter(shardPredicate).
		WithOptions(controller.CommonOptions()).
		Complete(newReconciler(mgr.GetClient(), credentialsDB)); err != nil {
		return fmt.Errorf("error building Warehouse reconciler: %w", err)
	}
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
		listFreightFn:   kubeClient.List,
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
	warehouse, err := kargoapi.GetWarehouse(ctx, r.client, req.NamespacedName)
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
		newStatus.Message = err.Error()
		logger.Error(err, "error syncing Warehouse")
	}

	updateErr := kubeclient.PatchStatus(
		ctx,
		r.client,
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
	return ctrl.Result{RequeueAfter: warehouse.Spec.Interval.Duration}, nil
}

func (r *reconciler) syncWarehouse(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (kargoapi.WarehouseStatus, error) {
	status := *warehouse.Status.DeepCopy()
	status.ObservedGeneration = warehouse.Generation
	status.Message = "" // Clear any previous error

	// Record the current refresh token as having been handled.
	if token, ok := kargoapi.RefreshAnnotationValue(warehouse.GetAnnotations()); ok {
		status.LastHandledRefresh = token
	}

	logger := logging.LoggerFromContext(ctx)

	// Discover the latest artifacts.
	discoveredArtifacts, err := r.discoverArtifactsFn(ctx, warehouse)
	if err != nil {
		return status, fmt.Errorf("error discovering artifacts: %w", err)
	}
	logger.Debug("discovered latest artifacts")
	status.DiscoveredArtifacts = discoveredArtifacts

	// Automatically create a Freight from the latest discovered artifacts
	// if the Warehouse is configured to do so.
	if pol := warehouse.Spec.FreightCreationPolicy; pol == kargoapi.FreightCreationPolicyAutomatic || pol == "" {
		freight, err := r.buildFreightFromLatestArtifactsFn(warehouse.Namespace, discoveredArtifacts)
		if err != nil {
			return status, fmt.Errorf("failed to build Freight from latest artifacts: %w", err)
		}
		freight.Origin = kargoapi.FreightOrigin{
			Kind: kargoapi.FreightOriginKindWarehouse,
			Name: warehouse.Name,
		}

		var existingFreight kargoapi.FreightList
		if err = r.listFreightFn(
			ctx,
			&existingFreight,
			&client.ListOptions{
				Namespace: warehouse.Namespace,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.FreightByWarehouseIndexField,
					warehouse.Name,
				),
			},
		); err != nil {
			return status,
				fmt.Errorf("error listing existing Freight from Warehouse: %w", err)
		}
		slices.SortFunc(existingFreight.Items, func(lhs, rhs kargoapi.Freight) int {
			return rhs.CreationTimestamp.Time.Compare(lhs.CreationTimestamp.Time)
		})
		if len(existingFreight.Items) == 0 || !compareFreight(&existingFreight.Items[0], freight) {
			if err = r.createFreightFn(ctx, freight); client.IgnoreAlreadyExists(err) != nil {
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
		}

		status.LastFreightID = freight.Name
	}
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
		Git:    commits,
		Images: images,
		Charts: charts,
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
	freight.Name = freight.GenerateID()

	return freight, nil
}

func compareFreight(old, new *kargoapi.Freight) bool {
	if len(old.Commits) != len(new.Commits) {
		return false
	}
	for i, commit := range old.Commits {
		if !commit.DeepEquals(&new.Commits[i]) {
			return false
		}
	}

	if len(old.Images) != len(new.Images) {
		return false
	}
	for i, img := range old.Images {
		if !img.DeepEquals(&new.Images[i]) {
			return false
		}
	}

	if len(old.Charts) != len(new.Charts) {
		return false
	}
	for i, chart := range old.Charts {
		if !chart.DeepEquals(&new.Charts[i]) {
			return false
		}
	}

	return true
}
