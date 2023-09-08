package stages

import (
	"context"
	"time"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/akuity/bookkeeper/pkg/git"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libArgoCD "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/images"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Stage resources.
type reconciler struct {
	kargoClient                client.Client
	argoClient                 client.Client
	credentialsDB              credentials.Database
	imageSourceURLFnsByBaseURL map[string]func(string, string) string

	// The following behaviors are overridable for testing purposes:

	// Loop guard
	hasNonTerminalPromotionsFn func(
		ctx context.Context,
		stageNamespace string,
		stageName string,
	) (bool, error)

	// Common:
	getArgoCDAppFn func(
		ctx context.Context,
		client client.Client,
		namespace string,
		name string,
	) (*argocd.Application, error)

	// Health checks:
	checkHealthFn func(
		context.Context,
		kargoapi.Freight,
		[]kargoapi.ArgoCDAppUpdate,
	) kargoapi.Health

	// Syncing:
	getLatestFreightFromReposFn func(
		ctx context.Context,
		namespace string,
		subs kargoapi.RepoSubscriptions,
	) (*kargoapi.Freight, error)

	getAvailableFreightFromUpstreamStagesFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.StageSubscription,
	) ([]kargoapi.Freight, error)

	getLatestCommitsFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.GitSubscription,
	) ([]kargoapi.GitCommit, error)

	getLatestImagesFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.ImageSubscription,
	) ([]kargoapi.Image, error)

	getLatestTagFn func(
		repoURL string,
		updateStrategy images.ImageUpdateStrategy,
		semverConstraint string,
		allowTags string,
		ignoreTags []string,
		platform string,
		creds *images.Credentials,
	) (string, error)

	getLatestChartsFn func(
		ctx context.Context,
		namespace string,
		subs []kargoapi.ChartSubscription,
	) ([]kargoapi.Chart, error)

	getLatestChartVersionFn func(
		ctx context.Context,
		registryURL string,
		chart string,
		semverConstraint string,
		creds *helm.Credentials,
	) (string, error)

	getLatestCommitMetaFn func(
		ctx context.Context,
		repoURL string,
		branch string,
		creds *git.RepoCredentials,
	) (*gitMeta, error)
}

type gitMeta struct {
	Commit  string
	Message string
	Author  string
}

// SetupReconcilerWithManager initializes a reconciler for Stage resources and
// registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argoMgr manager.Manager,
	credentialsDB credentials.Database,
	shardName string,
) error {
	// Index Promotions in non-terminal states by Stage
	if err := kubeclient.IndexNonTerminalPromotionsByStage(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index non-terminal Promotions by Stage")
	}

	// Index PromotionPolicies by Stage
	if err := kubeclient.IndexPromotionPoliciesByStage(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index PromotionPolicies by Stage")
	}

	shardPredicate, err := controller.GetShardPredicate(shardName)
	if err != nil {
		return errors.Wrap(err, "error creating shard predicate")
	}

	return errors.Wrap(
		ctrl.NewControllerManagedBy(kargoMgr).
			For(&kargoapi.Stage{}).
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
			WithOptions(controller.CommonOptions()).
			Complete(
				newReconciler(
					kargoMgr.GetClient(),
					argoMgr.GetClient(),
					credentialsDB,
				),
			),
		"error registering Stage reconciler",
	)
}

func newReconciler(
	kargoClient client.Client,
	argoClient client.Client,
	credentialsDB credentials.Database,
) *reconciler {
	r := &reconciler{
		kargoClient:   kargoClient,
		argoClient:    argoClient,
		credentialsDB: credentialsDB,
		imageSourceURLFnsByBaseURL: map[string]func(string, string) string{
			githubURLPrefix: getGithubImageSourceURL,
		},
	}

	// The following default behaviors are overridable for testing purposes:

	// Loop guard:
	r.hasNonTerminalPromotionsFn = r.hasNonTerminalPromotions

	// Common:
	r.getArgoCDAppFn = libArgoCD.GetApplication

	// Health checks:
	r.checkHealthFn = r.checkHealth

	// Syncing:
	r.getLatestFreightFromReposFn = r.getLatestFreightFromRepos
	r.getAvailableFreightFromUpstreamStagesFn = r.getAvailableFreightFromUpstreamStages
	r.getLatestCommitsFn = r.getLatestCommits
	r.getLatestImagesFn = r.getLatestImages
	r.getLatestTagFn = images.GetLatestTag
	r.getLatestChartsFn = r.getLatestCharts
	r.getLatestChartVersionFn = helm.GetLatestChartVersion
	r.getLatestCommitMetaFn = getLatestCommitMeta

	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{
		// TODO: Make this configurable
		// Note: If there is a failure, controller runtime ignores this and uses
		// progressive backoff instead. So this value only affects when we will
		// reconcile next if THIS reconciliation succeeds.
		RequeueAfter: 5 * time.Minute,
	}

	logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"stage":     req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Stage")

	// Find the Stage
	stage, err := kargoapi.GetStage(ctx, r.kargoClient, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if stage == nil {
		// Ignore if not found. This can happen if the Stage was deleted after the
		// current reconciliation request was issued.
		result.RequeueAfter = 0 // Do not requeue
		return result, nil
	}
	logger.Debug("found Stage")

	var newStatus kargoapi.StageStatus
	newStatus, err = r.syncStage(ctx, stage)
	if err != nil {
		newStatus.Error = err.Error()
		logger.Errorf("error syncing Stage: %s", stage.Status.Error)
	} else {
		// Be sure to blank this out in case there's an error in this field from
		// the previous reconciliation
		newStatus.Error = ""
	}

	updateErr := kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
		*status = newStatus
	})
	if updateErr != nil {
		logger.Errorf("error updating Stage status: %s", updateErr)
	}
	clearRefreshErr := kargoapi.ClearStageRefresh(ctx, r.kargoClient, stage)
	if clearRefreshErr != nil {
		logger.Errorf("error clearing Stage refresh annotation: %s", clearRefreshErr)
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}
	if err == nil {
		err = clearRefreshErr
	}
	logger.Debug("done reconciling Stage")

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return result, err
}

func (r *reconciler) syncStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	status := *stage.Status.DeepCopy()

	logger := logging.LoggerFromContext(ctx)

	// Skip the entire reconciliation loop if there are Promotions associate with
	// this Stage in a non-terminal state. The promotion process and this
	// reconciliation loop BOTH update Stage status, so this check helps us
	// to avoid race conditions that may otherwise arise.
	hasNonTerminalPromos, err :=
		r.hasNonTerminalPromotionsFn(ctx, stage.Namespace, stage.Name)
	if err != nil {
		return status, err
	}
	if hasNonTerminalPromos {
		logger.Debug(
			"Stage has one or more Promotions in a non-terminal phase; skipping " +
				"this reconciliation loop",
		)
		return status, nil
	}

	status.ObservedGeneration = stage.Generation
	// Only perform health checks if we have a current Freight
	if status.CurrentFreight != nil {
		var health kargoapi.Health
		if stage.Spec.PromotionMechanisms != nil {
			health = r.checkHealthFn(
				ctx,
				*status.CurrentFreight,
				stage.Spec.PromotionMechanisms.ArgoCDAppUpdates,
			)
		} else {
			// Healthy by default if there are no promotion mechanisms.
			health = kargoapi.Health{
				Status: kargoapi.HealthStateHealthy,
			}
		}
		status.Health = &health
		if health.Status == kargoapi.HealthStateHealthy {
			status.CurrentFreight.Qualified = true
		}
		status.History.Pop()
		status.History.Push(*status.CurrentFreight)
	} else {
		logger.Debug("Stage has no current Freight; skipping health checks")
	}

	if stage.Spec.Subscriptions.Repos != nil {

		latestFreight, err := r.getLatestFreightFromReposFn(
			ctx,
			stage.Namespace,
			*stage.Spec.Subscriptions.Repos,
		)
		if err != nil {
			return status, err
		}
		if latestFreight == nil {
			logger.Debug("found no Freight from upstream repositories")
			return status, nil
		}
		logger.Debug("got latest Freight from upstream repositories")

		// latestFreight from upstream repos will always have a shiny new ID. To
		// determine if this is actually new and needs to be pushed onto the
		// status.AvailableFreight stack, either that stack needs to be empty or
		// latestFreight's MATERIALS must differ from what is at the top of the
		// status.AvailableFreight stack.
		if topAvailableFreight, ok := status.AvailableFreight.Top(); ok &&
			latestFreight.ID == topAvailableFreight.ID {
			logger.Debug("latest Freight is not new")
			return status, nil
		}
		status.AvailableFreight.Push(*latestFreight)
		logger.Debug("latest Freight is new; added to available Freight")

	} else if len(stage.Spec.Subscriptions.UpstreamStages) > 0 {

		// Grab the latest known Freight before we overwrite status.AvailableFreight
		var latestKnownFreight *kargoapi.Freight
		if lks, ok := status.AvailableFreight.Top(); ok {
			latestKnownFreight = &lks
		}

		// This returns de-duped, healthy Freight only from all upstream Stages.
		// There could be up to ten per upstream Stage. This is more than the usual
		// quantity we permit in status.AvailableFreight, but we'll allow it.
		var err error
		if status.AvailableFreight, err = r.getAvailableFreightFromUpstreamStagesFn(
			ctx,
			stage.Namespace,
			stage.Spec.Subscriptions.UpstreamStages,
		); err != nil {
			return status, err
		}

		if status.AvailableFreight.Empty() {
			logger.Debug("got no available Freight from upstream Stages")
			return status, nil
		}
		logger.Debug("got available Freight from upstream Stages")

		if len(stage.Spec.Subscriptions.UpstreamStages) > 1 {
			logger.Debug(
				"auto-promotion cannot proceed due to multiple upstream Stages",
			)
			return status, nil
		}

		if latestKnownFreight != nil {
			// We already know this stack isn't empty
			latestAvailableFreight, _ := status.AvailableFreight.Top()
			if latestKnownFreight.ID == latestAvailableFreight.ID {
				logger.Debug("latest Freight is not new")
				return status, nil
			}
		}
	} else {
		// This should be impossible if validation is working, but out of an
		// abundance of caution, bail now if this happens somehow.
		return status, nil
	}

	nextFreightCandidate, _ := status.AvailableFreight.Top()
	if status.CurrentFreight != nil &&
		nextFreightCandidate.FirstSeen.Before(status.CurrentFreight.FirstSeen) {
		logger.Debug(
			"newest available Freight is older than current Freight; refusing to " +
				"auto-promote",
		)
		return status, nil
	}
	nextFreight := nextFreightCandidate

	// If we get to here, we've determined that auto-promotion is a possibility.
	// See if it's actually allowed...
	policies := kargoapi.PromotionPolicyList{}
	if err := r.kargoClient.List(
		ctx,
		&policies,
		&client.ListOptions{
			Namespace: stage.Namespace,
			FieldSelector: fields.Set(map[string]string{
				kubeclient.PromotionPoliciesByStageIndexField: stage.Name,
			}).AsSelector(),
		},
	); err != nil {
		return status, err
	}
	if len(policies.Items) == 0 {
		logger.Debug(
			"no PromotionPolicy exists to enable auto-promotion; auto-promotion " +
				"will not proceed",
		)
		return status, nil
	}
	if len(policies.Items) > 1 {
		logger.Debug("found multiple PromotionPolicies associated with " +
			"Stage; auto-promotion will not proceed",
		)
		return status, nil
	}
	if !policies.Items[0].EnableAutoPromotion {
		logger.Debug(
			"PromotionPolicy does not enable auto-promotion; auto-promotion " +
				"will not proceed",
		)
		return status, nil
	}

	logger = logger.WithField("freight", nextFreight.ID)
	logger.Debug("auto-promotion will proceed")

	promo := kargo.NewPromotion(*stage, nextFreight.ID)

	if err :=
		r.kargoClient.Create(ctx, &promo, &client.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Debug("Promotion resource already exists")
			return status, nil
		}
		return status, errors.Wrapf(
			err,
			"error creating Promotion of Stage %q in namespace %q to Freight %q",
			stage.Name,
			stage.Namespace,
			nextFreight.ID,
		)
	}
	logger.Debug("created Promotion resource")

	return status, nil
}

func (r *reconciler) hasNonTerminalPromotions(
	ctx context.Context,
	stageNamespace string,
	stageName string,
) (bool, error) {
	promos := kargoapi.PromotionList{}
	if err := r.kargoClient.List(
		ctx,
		&promos,
		&client.ListOptions{
			Namespace: stageNamespace,
			FieldSelector: fields.Set(map[string]string{
				kubeclient.NonTerminalPromotionsByStageIndexField: stageName,
			}).AsSelector(),
		},
	); err != nil {
		return false, errors.Wrapf(
			err,
			"error listing Promotions in non-terminal phases for Stage %q in "+
				"namespace %q",
			stageNamespace,
			stageName,
		)
	}
	return len(promos.Items) > 0, nil
}

func (r *reconciler) getLatestFreightFromRepos(
	ctx context.Context,
	namespace string,
	repoSubs kargoapi.RepoSubscriptions,
) (*kargoapi.Freight, error) {
	logger := logging.LoggerFromContext(ctx)

	latestCommits, err := r.getLatestCommitsFn(ctx, namespace, repoSubs.Git)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing git repo subscriptions")
	}
	if len(repoSubs.Git) > 0 {
		logger.Debug("synced git repo subscriptions")
	}

	latestImages, err := r.getLatestImagesFn(ctx, namespace, repoSubs.Images)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing image repo subscriptions")
	}
	if len(repoSubs.Images) > 0 {
		logger.Debug("synced image repo subscriptions")
	}

	latestCharts, err := r.getLatestChartsFn(ctx, namespace, repoSubs.Charts)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing chart repo subscriptions")
	}
	if len(repoSubs.Charts) > 0 {
		logger.Debug("synced chart repo subscriptions")
	}

	now := metav1.Now()
	freight := &kargoapi.Freight{
		FirstSeen: &now,
		Commits:   latestCommits,
		Images:    latestImages,
		Charts:    latestCharts,
	}
	freight.UpdateFreightID()
	return freight, nil
}

// TODO: Test this
func (r *reconciler) getAvailableFreightFromUpstreamStages(
	ctx context.Context,
	namespace string,
	subs []kargoapi.StageSubscription,
) ([]kargoapi.Freight, error) {
	if len(subs) == 0 {
		return nil, nil
	}

	availableFreight := []kargoapi.Freight{}
	freightSet := map[string]struct{}{} // We'll use this to de-dupe
	for _, sub := range subs {
		upstreamStage, err := kargoapi.GetStage(
			ctx,
			r.kargoClient,
			types.NamespacedName{
				Namespace: namespace,
				Name:      sub.Name,
			},
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error finding upstream Stage %q in namespace %q",
				sub.Name,
				namespace,
			)
		}
		if upstreamStage == nil {
			return nil, errors.Errorf(
				"found no upstream Stage %q in namespace %q",
				sub.Name,
				namespace,
			)
		}
		for _, freight := range upstreamStage.Status.History {
			if _, ok := freightSet[freight.ID]; !ok && freight.Qualified {
				freight.Provenance = upstreamStage.Name
				for i := range freight.Commits {
					freight.Commits[i].HealthCheckCommit = ""
				}
				freight.Qualified = false
				availableFreight = append(availableFreight, freight)
				freightSet[freight.ID] = struct{}{}
			}
		}
	}

	return availableFreight, nil
}
