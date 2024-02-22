package warehouses

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/technosophos/moniker"
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
	freightAliasGenerator      moniker.Namer

	// The following behaviors are overridable for testing purposes:

	getLatestFreightFromReposFn func(
		context.Context,
		*kargoapi.Warehouse,
	) (*kargoapi.Freight, error)

	selectCommitsFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.RepoSubscription,
	) ([]kargoapi.GitCommit, error)

	getLastCommitIDFn func(repo git.Repo) (string, error)

	listTagsFn func(repo git.Repo) ([]string, error)

	checkoutTagFn func(repo git.Repo, tag string) error

	selectImagesFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.RepoSubscription,
	) ([]kargoapi.Image, error)

	getImageRefsFn func(
		ctx context.Context,
		repoURL string,
		imageSelectionStrategy kargoapi.ImageSelectionStrategy,
		semverConstraint string,
		allowTagsRegex string,
		ignoreTags []string,
		platform string,
		creds *image.Credentials,
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
	) (*gitMeta, error)

	getAvailableFreightAliasFn func(context.Context) (string, error)

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
		return errors.Wrap(err, "error creating shard selector predicate")
	}

	return errors.Wrap(
		ctrl.NewControllerManagedBy(mgr).
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
					predicate.AnnotationChangedPredicate{},
				),
			).
			WithEventFilter(shardPredicate).
			WithEventFilter(kargo.IgnoreClearRefreshUpdates{}).
			WithOptions(controller.CommonOptions()).
			Complete(newReconciler(mgr.GetClient(), credentialsDB)),
		"error building Warehouse reconciler",
	)
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
		freightAliasGenerator: moniker.New(),
	}
	r.getLatestFreightFromReposFn = r.getLatestFreightFromRepos
	r.selectCommitsFn = r.selectCommits
	r.getLastCommitIDFn = r.getLastCommitID
	r.listTagsFn = r.listTags
	r.checkoutTagFn = r.checkoutTag
	r.selectImagesFn = r.selectImages
	r.getImageRefsFn = getImageRefs
	r.selectChartsFn = r.selectCharts
	r.selectChartVersionFn = helm.SelectChartVersion
	r.selectCommitMetaFn = r.selectCommitMeta
	r.getAvailableFreightAliasFn = r.getAvailableFreightAlias
	r.createFreightFn = kubeClient.Create
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{
		// Note: If there is a failure, controller runtime ignores this and uses
		// progressive backoff instead. So this value only affects when we will
		// reconcile next if THIS reconciliation succeeds.
		//
		// TODO: Make this configurable
		RequeueAfter: 5 * time.Minute,
	}

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
		return result, err
	}
	if warehouse == nil {
		// Ignore if not found. This can happen if the Warehouse was deleted after
		// the current reconciliation request was issued.
		return result, nil
	}

	newStatus, err := r.syncWarehouse(ctx, warehouse)
	if err != nil {
		newStatus.Error = err.Error()
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
	if clearRefreshErr := kargoapi.ClearWarehouseRefresh(ctx, r.client, warehouse); clearRefreshErr != nil {
		logger.Errorf("error clearing Warehouse refresh annotation: %s", clearRefreshErr)
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
	return result, err
}

func (r *reconciler) syncWarehouse(
	ctx context.Context,
	warehouse *kargoapi.Warehouse,
) (kargoapi.WarehouseStatus, error) {
	status := *warehouse.Status.DeepCopy()
	status.ObservedGeneration = warehouse.Generation
	status.Error = "" // Clear any previous error

	logger := logging.LoggerFromContext(ctx)

	freight, err := r.getLatestFreightFromReposFn(ctx, warehouse)
	if err != nil {
		return status,
			errors.Wrap(err, "error getting latest Freight from repositories")
	}
	if freight == nil {
		logger.Debug("found no Freight from repositories")
		return status, nil
	}
	logger.Debug("got latest Freight from repositories")

	freight.Labels = map[string]string{}
	if freight.Labels[kargoapi.AliasLabelKey], err =
		r.getAvailableFreightAliasFn(ctx); err != nil {
		return status, errors.Wrap(err, "error getting available Freight alias")
	}

	if err = r.createFreightFn(ctx, freight); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Debugf(
				"Freight %q in namespace %q already exists",
				freight.Name,
				freight.Namespace,
			)
			return status, nil
		}
		return status, errors.Wrapf(
			err,
			"error creating Freight %q in namespace %q",
			freight.Name,
			freight.Namespace,
		)
	}
	log.Debugf(
		"created Freight %q in namespace %q",
		freight.Name,
		freight.Namespace,
	)

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
	)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing git repo subscriptions")
	}
	logger.Debug("synced git repo subscriptions")

	selectedImages, err := r.selectImagesFn(
		ctx,
		warehouse.Namespace,
		warehouse.Spec.Subscriptions,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing image repo subscriptions")
	}
	logger.Debug("synced image repo subscriptions")

	selectedCharts, err := r.selectChartsFn(
		ctx,
		warehouse.Namespace,
		warehouse.Spec.Subscriptions,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing chart repo subscriptions")
	}
	logger.Debug("synced chart repo subscriptions")

	ownerRef := metav1.NewControllerRef(
		warehouse,
		kargoapi.GroupVersion.WithKind("Warehouse"),
	)
	freight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       warehouse.Namespace,
			OwnerReferences: []metav1.OwnerReference{*ownerRef},
		},
		Commits: selectedCommits,
		Images:  selectedImages,
		Charts:  selectedCharts,
	}
	freight.UpdateID()
	freight.ObjectMeta.Name = freight.ID
	return freight, nil
}
