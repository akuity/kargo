package warehouses

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	getLatestFreightFromReposFn func(
		context.Context,
		*kargoapi.Warehouse,
	) (*kargoapi.Freight, error)

	selectCommitsFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.RepoSubscription,
		LastFreight *kargoapi.FreightReference,
	) ([]kargoapi.GitCommit, error)

	getLastCommitIDFn func(repo git.Repo) (string, error)

	getDiffPathsSinceCommitIDFn func(repo git.Repo, commitId string) ([]string, error)

	listTagsFn func(repo git.Repo) ([]string, error)

	checkoutTagFn func(repo git.Repo, tag string) error

	selectImagesFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.RepoSubscription,
	) ([]kargoapi.Image, error)

	getImageRefsFn func(
		context.Context,
		kargoapi.ImageSubscription,
		*image.Credentials,
	) (string, string, error)

	selectChartsFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.RepoSubscription,
	) ([]kargoapi.Chart, error)

	selectChartVersionFn func(
		ctx context.Context,
		repoURL string,
		chart string,
		semverConstraint string,
		creds *helm.Credentials,
	) (string, error)

	selectCommitMetaFn func(
		context.Context,
		kargoapi.GitSubscription,
		*git.RepoCredentials,
		string,
	) (*gitMeta, error)

	createFreightFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error
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
		client:        kubeClient,
		credentialsDB: credentialsDB,
		imageSourceURLFnsByBaseURL: map[string]func(string, string) string{
			githubURLPrefix: getGithubImageSourceURL,
		},
	}
	r.getLatestFreightFromReposFn = r.getLatestFreightFromRepos
	r.selectCommitsFn = r.selectCommits
	r.getLastCommitIDFn = r.getLastCommitID
	r.getDiffPathsSinceCommitIDFn = r.getDiffPathsSinceCommitID
	r.listTagsFn = r.listTags
	r.checkoutTagFn = r.checkoutTag
	r.selectImagesFn = r.selectImages
	r.getImageRefsFn = getImageRefs
	r.selectChartsFn = r.selectCharts
	r.selectChartVersionFn = helm.SelectChartVersion
	r.selectCommitMetaFn = r.selectCommitMeta
	r.createFreightFn = kubeClient.Create
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx)

	logger = logger.WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"warehouse": req.NamespacedName.Name,
	})
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
		logger.Errorf("error syncing Warehouse: %s", err)
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
		logger.Errorf("error updating Warehouse status: %s", updateErr)
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
	//
	// TODO: Make this configurable
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
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

	freight, err := r.getLatestFreightFromReposFn(ctx, warehouse)
	if err != nil {
		return status, fmt.Errorf("error getting latest Freight from repositories: %w", err)
	}
	if freight == nil {
		logger.Debug("found no Freight from repositories")
		return status, nil
	}
	logger.Debug("got latest Freight from repositories")

	if err = r.createFreightFn(ctx, freight); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Debugf(
				"Freight %q in namespace %q already exists",
				freight.Name,
				freight.Namespace,
			)
			return status, nil
		}
		return status, fmt.Errorf(
			"error creating Freight %q in namespace %q: %w",
			freight.Name,
			freight.Namespace,
			err,
		)
	}
	log.Debugf(
		"created Freight %q in namespace %q",
		freight.Name,
		freight.Namespace,
	)
	status.LastFreight = &kargoapi.FreightReference{
		Name:    freight.Name,
		Commits: freight.Commits,
		Images:  freight.Images,
		Charts:  freight.Charts,
	}

	return status, nil
}

func (r *reconciler) getLatestFreightFromRepos(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (*kargoapi.Freight, error) {
	logger := logging.LoggerFromContext(ctx)

	selectedCommits, err := r.selectCommitsFn(
		ctx,
		warehouse.Namespace,
		warehouse.Spec.Subscriptions,
		warehouse.Status.LastFreight,
	)
	if err != nil {
		return nil, fmt.Errorf("error syncing git repo subscriptions: %w", err)
	}
	logger.Debug("synced git repo subscriptions")

	selectedImages, err := r.selectImagesFn(
		ctx,
		warehouse.Namespace,
		warehouse.Spec.Subscriptions,
	)
	if err != nil {
		return nil, fmt.Errorf("error syncing image repo subscriptions: %w", err)
	}
	logger.Debug("synced image repo subscriptions")

	selectedCharts, err := r.selectChartsFn(
		ctx,
		warehouse.Namespace,
		warehouse.Spec.Subscriptions,
	)
	if err != nil {
		return nil, fmt.Errorf("error syncing chart repo subscriptions: %w", err)
	}
	logger.Debug("synced chart repo subscriptions")

	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: warehouse.Namespace,
		},
		Warehouse: warehouse.Name,
		Commits:   selectedCommits,
		Images:    selectedImages,
		Charts:    selectedCharts,
	}
	freight.Name = freight.GenerateID()
	return freight, nil
}
