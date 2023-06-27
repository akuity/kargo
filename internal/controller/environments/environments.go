package environments

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
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/images"
	"github.com/akuity/kargo/internal/logging"
)

const (
	envsByAppIndexField              = "applications"
	outstandingPromosByEnvIndexField = "environment"
	promoPoliciesByEnvIndexField     = "environment"
)

// reconciler reconciles Environment resources.
type reconciler struct {
	kargoClient   client.Client
	argoClient    client.Client
	credentialsDB credentials.Database

	// The following behaviors are overridable for testing purposes:

	// Loop guard
	hasOutstandingPromotionsFn func(
		ctx context.Context,
		envNamespace string,
		envName string,
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
		api.EnvironmentState,
		[]api.ArgoCDAppUpdate,
	) api.Health

	// Syncing:
	getLatestStateFromReposFn func(
		ctx context.Context,
		namespace string,
		subs api.RepoSubscriptions,
	) (*api.EnvironmentState, error)

	getAvailableStatesFromUpstreamEnvsFn func(
		context.Context,
		[]api.EnvironmentSubscription,
	) ([]api.EnvironmentState, error)

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

// SetupReconcilerWithManager initializes a reconciler for Environment resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argoMgr manager.Manager,
	credentialsDB credentials.Database,
) error {
	// Index Environments by Argo CD Applications
	if err := kargoMgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Environment{},
		envsByAppIndexField,
		indexEnvsByApp,
	); err != nil {
		return errors.Wrap(
			err,
			"error indexing Environments by Argo CD Applications",
		)
	}

	// Index Promotions in non-terminal states by Environment
	if err := kargoMgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Promotion{},
		outstandingPromosByEnvIndexField,
		indexOutstandingPromotionsByEnvironment,
	); err != nil {
		return errors.Wrap(
			err,
			"error indexing non-terminal Promotions by Environment",
		)
	}

	// Index PromotionPolicies by Environment
	if err := kargoMgr.GetFieldIndexer().IndexField(
		ctx,
		&api.PromotionPolicy{},
		promoPoliciesByEnvIndexField,
		func(obj client.Object) []string {
			policy := obj.(*api.PromotionPolicy) // nolint: forcetypeassert
			return []string{policy.Environment}
		},
	); err != nil {
		return errors.Wrap(err, "error indexing PromotionPolicies by Environment")
	}

	return ctrl.NewControllerManagedBy(kargoMgr).
		For(&api.Environment{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(event.DeleteEvent) bool {
				// We're not interested in any deletes
				return false
			},
		}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Complete(newReconciler(
			kargoMgr.GetClient(),
			argoMgr.GetClient(),
			credentialsDB,
		))
}

func indexEnvsByApp(obj client.Object) []string {
	env := obj.(*api.Environment) // nolint: forcetypeassert
	if env.Spec.PromotionMechanisms == nil ||
		len(env.Spec.PromotionMechanisms.ArgoCDAppUpdates) == 0 {
		return nil
	}
	apps := make([]string, len(env.Spec.PromotionMechanisms.ArgoCDAppUpdates))
	for i, appCheck := range env.Spec.PromotionMechanisms.ArgoCDAppUpdates {
		apps[i] =
			fmt.Sprintf("%s:%s", appCheck.AppNamespace, appCheck.AppName)
	}
	return apps
}

func indexOutstandingPromotionsByEnvironment(obj client.Object) []string {
	promo := obj.(*api.Promotion) // nolint: forcetypeassert
	switch promo.Status.Phase {
	case api.PromotionPhaseComplete, api.PromotionPhaseFailed:
		return nil
	}
	return []string{promo.Spec.Environment}
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
	r.getAvailableStatesFromUpstreamEnvsFn = r.getAvailableStatesFromUpstreamEnvs
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
		"namespace":   req.NamespacedName.Namespace,
		"environment": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Environment")

	// Find the Environment
	env, err := api.GetEnv(ctx, r.kargoClient, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if env == nil {
		// Ignore if not found. This can happen if the Environment was deleted after
		// the current reconciliation request was issued.
		result.RequeueAfter = 0 // Do not requeue
		return result, nil
	}
	logger.Debug("found Environment")

	env.Status, err = r.sync(ctx, env)
	if err != nil {
		env.Status.Error = err.Error()
		logger.Errorf("error syncing Environment: %s", env.Status.Error)
	} else {
		// Be sure to blank this out in case there's an error in this field from
		// the previous reconciliation
		env.Status.Error = ""
	}

	updateErr := r.kargoClient.Status().Update(ctx, env)
	if updateErr != nil {
		logger.Errorf("error updating Environment status: %s", updateErr)
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}

	logger.Debug("done reconciling Environment")

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return result, err
}

func (r *reconciler) sync(
	ctx context.Context,
	env *api.Environment,
) (api.EnvironmentStatus, error) {
	status := *env.Status.DeepCopy()

	logger := logging.LoggerFromContext(ctx)

	// Skip the entire reconciliation loop if there are Promotions associate with
	// this Environment in a non-terminal state. The promotion process and this
	// reconciliation loop BOTH update Environment status, so this check helps us
	// to avoid race conditions that may otherwise arise.
	hasOutstandingPromos, err :=
		r.hasOutstandingPromotionsFn(ctx, env.Namespace, env.Name)
	if err != nil {
		return status, err
	}
	if hasOutstandingPromos {
		logger.Debug(
			"Environment has outstanding Promotions; skipping this reconciliation " +
				"loop",
		)
		return status, nil
	}

	// Only perform health checks if we have a current state
	if status.CurrentState != nil && env.Spec.PromotionMechanisms != nil {
		health := r.checkHealthFn(
			ctx,
			*status.CurrentState,
			env.Spec.PromotionMechanisms.ArgoCDAppUpdates,
		)
		status.CurrentState.Health = &health
		status.History.Pop()
		status.History.Push(*status.CurrentState)
	} else {
		logger.Debug("Environment has no current state; skipping health checks")
	}

	if env.Spec.Subscriptions.Repos != nil {

		latestState, err := r.getLatestStateFromReposFn(
			ctx,
			env.Namespace,
			*env.Spec.Subscriptions.Repos,
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

	} else if len(env.Spec.Subscriptions.UpstreamEnvs) > 0 {

		// Grab the latest known state before we overwrite status.AvailableStates
		var latestKnownState *api.EnvironmentState
		if lks, ok := status.AvailableStates.Top(); ok {
			latestKnownState = &lks
		}

		// This returns de-duped, healthy states only from all upstream envs. There
		// could be up to ten per upstream environment. This is more than the usual
		// quantity we permit in status.AvailableStates, but we'll allow it.
		var err error
		if status.AvailableStates, err = r.getAvailableStatesFromUpstreamEnvsFn(
			ctx,
			env.Spec.Subscriptions.UpstreamEnvs,
		); err != nil {
			return status, err
		}

		if status.AvailableStates.Empty() {
			logger.Debug("got no available states from upstream Environments")
			return status, nil
		}
		logger.Debug("got available states from upstream Environments")

		if len(env.Spec.Subscriptions.UpstreamEnvs) > 1 {
			logger.Debug(
				"auto-promotion cannot proceed due to multiple upstream Environments",
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
			Namespace: env.Namespace,
			FieldSelector: fields.Set(map[string]string{
				promoPoliciesByEnvIndexField: env.Name,
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
			"Environment; auto-promotion will not proceed",
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

	if err := r.kargoClient.Create(
		ctx,
		&api.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-to-%s", env.Name, nextState.ID),
				Namespace: env.Namespace,
			},
			Spec: &api.PromotionSpec{
				Environment: env.Name,
				State:       nextState.ID,
			},
		},
		&client.CreateOptions{},
	); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Debug("Promotion resource already exists")
			return status, nil
		}
		return status, errors.Wrapf(
			err,
			"error creating Promotion of Environment %q in namespace %q to state %q",
			env.Name,
			env.Namespace,
			nextState.ID,
		)
	}
	logger.Debug("created Promotion resource")

	return status, nil
}

func (r *reconciler) hasOutstandingPromotions(
	ctx context.Context,
	envNamespace string,
	envName string,
) (bool, error) {
	promos := api.PromotionList{}
	if err := r.kargoClient.List(
		ctx,
		&promos,
		&client.ListOptions{
			Namespace: envNamespace,
			FieldSelector: fields.Set(map[string]string{
				outstandingPromosByEnvIndexField: envName,
			}).AsSelector(),
		},
	); err != nil {
		return false, errors.Wrapf(
			err,
			"error listing outstanding Promotions for Environment %q in "+
				"namespace %q",
			envNamespace,
			envName,
		)
	}
	return len(promos.Items) > 0, nil
}

func (r *reconciler) getLatestStateFromRepos(
	ctx context.Context,
	namespace string,
	repoSubs api.RepoSubscriptions,
) (*api.EnvironmentState, error) {
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
	state := &api.EnvironmentState{
		FirstSeen: &now,
		Commits:   latestCommits,
		Images:    latestImages,
		Charts:    latestCharts,
	}
	state.UpdateStateID()
	return state, nil
}

// TODO: Test this
func (r *reconciler) getAvailableStatesFromUpstreamEnvs(
	ctx context.Context,
	subs []api.EnvironmentSubscription,
) ([]api.EnvironmentState, error) {
	if len(subs) == 0 {
		return nil, nil
	}

	availableStates := []api.EnvironmentState{}
	stateSet := map[string]struct{}{} // We'll use this to de-dupe
	for _, sub := range subs {
		upstreamEnv, err := api.GetEnv(
			ctx,
			r.kargoClient,
			types.NamespacedName{
				Namespace: sub.Namespace,
				Name:      sub.Name,
			},
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error finding upstream environment %q in namespace %q",
				sub.Name,
				sub.Namespace,
			)
		}
		if upstreamEnv == nil {
			return nil, errors.Errorf(
				"found no upstream environment %q in namespace %q",
				sub.Name,
				sub.Namespace,
			)
		}
		for _, state := range upstreamEnv.Status.History {
			if _, ok := stateSet[state.ID]; !ok &&
				state.Health != nil && state.Health.Status == api.HealthStateHealthy {
				state.Provenance = upstreamEnv.Name
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
