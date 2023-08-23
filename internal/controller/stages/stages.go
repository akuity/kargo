package stages

import (
	"context"
	"fmt"
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
	api "github.com/akuity/kargo/api/v1alpha1"
	libArgoCD "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/images"
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
	hasOutstandingPromotionsFn func(
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
		api.StageState,
		[]api.ArgoCDAppUpdate,
	) api.Health

	// Syncing:
	getLatestStateFromReposFn func(
		ctx context.Context,
		namespace string,
		subs api.RepoSubscriptions,
	) (*api.StageState, error)

	getAvailableStatesFromUpstreamStagesFn func(
		ctx context.Context,
		namespace string,
		subs []api.StageSubscription,
	) ([]api.StageState, error)

	getLatestCommitsFn func(
		ctx context.Context,
		namespace string,
		subs []api.GitSubscription,
	) ([]api.GitCommit, error)

	getLatestImagesFn func(
		ctx context.Context,
		namespace string,
		subs []api.ImageSubscription,
	) ([]api.Image, error)

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
		subs []api.ChartSubscription,
	) ([]api.Chart, error)

	getLatestChartVersionFn func(
		ctx context.Context,
		registryURL string,
		chart string,
		semverConstraint string,
		creds *helm.Credentials,
	) (string, error)

	getLatestCommitIDFn func(
		repoURL string,
		branch string,
		creds *git.RepoCredentials,
	) (string, error)
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
	if err := kubeclient.IndexOutstandingPromotionsByStage(ctx, kargoMgr); err != nil {
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
			For(&api.Stage{}).
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
	r.hasOutstandingPromotionsFn = r.hasOutstandingPromotions

	// Common:
	r.getArgoCDAppFn = libArgoCD.GetApplication

	// Health checks:
	r.checkHealthFn = r.checkHealth

	// Syncing:
	r.getLatestStateFromReposFn = r.getLatestStateFromRepos
	r.getAvailableStatesFromUpstreamStagesFn = r.getAvailableStatesFromUpstreamStages
	r.getLatestCommitsFn = r.getLatestCommits
	r.getLatestImagesFn = r.getLatestImages
	r.getLatestTagFn = images.GetLatestTag
	r.getLatestChartsFn = r.getLatestCharts
	r.getLatestChartVersionFn = helm.GetLatestChartVersion
	r.getLatestCommitIDFn = getLatestCommitID

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
	stage, err := api.GetStage(ctx, r.kargoClient, req.NamespacedName)
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

	var newStatus api.StageStatus
	newStatus, err = r.syncStage(ctx, stage)
	if err != nil {
		newStatus.Error = err.Error()
		logger.Errorf("error syncing Stage: %s", stage.Status.Error)
	} else {
		// Be sure to blank this out in case there's an error in this field from
		// the previous reconciliation
		newStatus.Error = ""
	}

	updateErr := kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *api.StageStatus) {
		*status = newStatus
	})
	if updateErr != nil {
		logger.Errorf("error updating Stage status: %s", updateErr)
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}

	logger.Debug("done reconciling Stage")

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return result, err
}

func (r *reconciler) syncStage(
	ctx context.Context,
	stage *api.Stage,
) (api.StageStatus, error) {
	status := *stage.Status.DeepCopy()

	logger := logging.LoggerFromContext(ctx)

	// Skip the entire reconciliation loop if there are Promotions associate with
	// this Stage in a non-terminal state. The promotion process and this
	// reconciliation loop BOTH update Stage status, so this check helps us
	// to avoid race conditions that may otherwise arise.
	hasOutstandingPromos, err :=
		r.hasOutstandingPromotionsFn(ctx, stage.Namespace, stage.Name)
	if err != nil {
		return status, err
	}
	if hasOutstandingPromos {
		logger.Debug(
			"Stage has outstanding Promotions; skipping this reconciliation loop",
		)
		return status, nil
	}

	// Only perform health checks if we have a current state
	if status.CurrentState != nil && stage.Spec.PromotionMechanisms != nil {
		health := r.checkHealthFn(
			ctx,
			*status.CurrentState,
			stage.Spec.PromotionMechanisms.ArgoCDAppUpdates,
		)
		status.CurrentState.Health = &health
		status.History.Pop()
		status.History.Push(*status.CurrentState)
	} else {
		logger.Debug("Stage has no current state; skipping health checks")
	}

	if stage.Spec.Subscriptions.Repos != nil {

		latestState, err := r.getLatestStateFromReposFn(
			ctx,
			stage.Namespace,
			*stage.Spec.Subscriptions.Repos,
		)
		if err != nil {
			return status, err
		}
		if latestState == nil {
			logger.Debug("found no state from upstream repositories")
			return status, nil
		}
		logger.Debug("got latest state from upstream repositories")

		// latestState from upstream repos will always have a shiny new ID. To
		// determine if this is actually new and needs to be pushed onto the
		// status.AvailableStates stack, either that stack needs to be empty or
		// latestState's MATERIALS must differ from what is at the top of the
		// status.AvailableStates stack.
		if topAvailableState, ok := status.AvailableStates.Top(); ok &&
			latestState.ID == topAvailableState.ID {
			logger.Debug("latest state is not new")
			return status, nil
		}
		status.AvailableStates.Push(*latestState)
		logger.Debug("latest state is new; added to available states")

	} else if len(stage.Spec.Subscriptions.UpstreamStages) > 0 {

		// Grab the latest known state before we overwrite status.AvailableStates
		var latestKnownState *api.StageState
		if lks, ok := status.AvailableStates.Top(); ok {
			latestKnownState = &lks
		}

		// This returns de-duped, healthy states only from all upstream Stages.
		// There could be up to ten per upstream Stage. This is more than the usual
		// quantity we permit in status.AvailableStates, but we'll allow it.
		var err error
		if status.AvailableStates, err = r.getAvailableStatesFromUpstreamStagesFn(
			ctx,
			stage.Namespace,
			stage.Spec.Subscriptions.UpstreamStages,
		); err != nil {
			return status, err
		}

		if status.AvailableStates.Empty() {
			logger.Debug("got no available states from upstream Stages")
			return status, nil
		}
		logger.Debug("got available states from upstream Stages")

		if len(stage.Spec.Subscriptions.UpstreamStages) > 1 {
			logger.Debug(
				"auto-promotion cannot proceed due to multiple upstream Stages",
			)
			return status, nil
		}

		if latestKnownState != nil {
			// We already know this stack isn't empty
			latestAvailableState, _ := status.AvailableStates.Top()
			if latestKnownState.ID == latestAvailableState.ID {
				logger.Debug("latest state is not new")
				return status, nil
			}
		}
	} else {
		// This should be impossible if validation is working, but out of an
		// abundance of caution, bail now if this happens somehow.
		return status, nil
	}

	nextStateCandidate, _ := status.AvailableStates.Top()
	if status.CurrentState != nil &&
		nextStateCandidate.FirstSeen.Before(status.CurrentState.FirstSeen) {
		logger.Debug(
			"newest available state is older than current state; refusing to " +
				"auto-promote",
		)
		return status, nil
	}
	nextState := nextStateCandidate

	// If we get to here, we've determined that auto-promotion is a possibility.
	// See if it's actually allowed...
	policies := api.PromotionPolicyList{}
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

	logger = logger.WithField("state", nextState.ID)
	logger.Debug("auto-promotion will proceed")

	promo := &api.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-to-%s", stage.Name, nextState.ID),
			Namespace: stage.Namespace,
		},
		Spec: &api.PromotionSpec{
			Stage: stage.Name,
			State: nextState.ID,
		},
	}

	if stage.Labels != nil && stage.Labels[controller.ShardLabelKey] != "" {
		promo.ObjectMeta.Labels = map[string]string{
			controller.ShardLabelKey: stage.Labels[controller.ShardLabelKey],
		}
	}

	if err :=
		r.kargoClient.Create(ctx, promo, &client.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Debug("Promotion resource already exists")
			return status, nil
		}
		return status, errors.Wrapf(
			err,
			"error creating Promotion of Stage %q in namespace %q to state %q",
			stage.Name,
			stage.Namespace,
			nextState.ID,
		)
	}
	logger.Debug("created Promotion resource")

	return status, nil
}

func (r *reconciler) hasOutstandingPromotions(
	ctx context.Context,
	stageNamespace string,
	stageName string,
) (bool, error) {
	promos := api.PromotionList{}
	if err := r.kargoClient.List(
		ctx,
		&promos,
		&client.ListOptions{
			Namespace: stageNamespace,
			FieldSelector: fields.Set(map[string]string{
				kubeclient.OutstandingPromotionsByStageIndexField: stageName,
			}).AsSelector(),
		},
	); err != nil {
		return false, errors.Wrapf(
			err,
			"error listing outstanding Promotions for Stage %q in "+
				"namespace %q",
			stageNamespace,
			stageName,
		)
	}
	return len(promos.Items) > 0, nil
}

func (r *reconciler) getLatestStateFromRepos(
	ctx context.Context,
	namespace string,
	repoSubs api.RepoSubscriptions,
) (*api.StageState, error) {
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
	state := &api.StageState{
		FirstSeen: &now,
		Commits:   latestCommits,
		Images:    latestImages,
		Charts:    latestCharts,
	}
	state.UpdateStateID()
	return state, nil
}

// TODO: Test this
func (r *reconciler) getAvailableStatesFromUpstreamStages(
	ctx context.Context,
	namespace string,
	subs []api.StageSubscription,
) ([]api.StageState, error) {
	if len(subs) == 0 {
		return nil, nil
	}

	availableStates := []api.StageState{}
	stateSet := map[string]struct{}{} // We'll use this to de-dupe
	for _, sub := range subs {
		upstreamStage, err := api.GetStage(
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
		for _, state := range upstreamStage.Status.History {
			if _, ok := stateSet[state.ID]; !ok &&
				state.Health != nil && state.Health.Status == api.HealthStateHealthy {
				state.Provenance = upstreamStage.Name
				for i := range state.Commits {
					state.Commits[i].HealthCheckCommit = ""
				}
				state.Health = nil
				availableStates = append(availableStates, state)
				stateSet[state.ID] = struct{}{}
			}
		}
	}

	return availableStates, nil
}
