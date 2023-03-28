package environments

import (
	"context"
	"fmt"
	"time"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	api "github.com/akuityio/kargo/api/v1alpha1"
	libArgoCD "github.com/akuityio/kargo/internal/argocd"
	"github.com/akuityio/kargo/internal/credentials"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/helm"
	"github.com/akuityio/kargo/internal/images"
	"github.com/akuityio/kargo/internal/logging"
)

const (
	envsByAppIndexField = "applications"
)

// reconciler reconciles Environment resources.
type reconciler struct {
	client        client.Client
	credentialsDB credentials.Database

	// The following behaviors are overridable for testing purposes:

	// Health checks:
	getArgoCDAppFn func(
		ctx context.Context,
		client client.Client,
		namespace string,
		name string,
	) (*argocd.Application, error)

	checkHealthFn func(
		context.Context,
		api.EnvironmentState,
		*api.HealthChecks,
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
		ctx context.Context,
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
		creds *git.Credentials,
	) (string, error)
}

// SetupReconcilerWithManager initializes a reconciler for Environment resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	mgr manager.Manager,
	credentialsDB credentials.Database,
) error {
	// Index Environments by Argo CD Applications
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&api.Environment{},
		envsByAppIndexField,
		func(obj client.Object) []string {
			env := obj.(*api.Environment) // nolint: forcetypeassert
			apps := make([]string, len(env.Spec.HealthChecks.ArgoCDAppChecks))
			for i, appCheck := range env.Spec.HealthChecks.ArgoCDAppChecks {
				apps[i] =
					fmt.Sprintf("%s:%s", appCheck.AppNamespace, appCheck.AppName)
			}
			return apps
		},
	); err != nil {
		return errors.Wrap(
			err,
			"error indexing Environments by Argo CD Applications",
		)
	}

	e, err := newReconciler(
		mgr.GetClient(),
		credentialsDB,
	)
	if err != nil {
		return errors.Wrap(err, "error initializing Environment reconciler")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Environment{}).
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(event.DeleteEvent) bool {
				// We're not interested in any deletes
				return false
			},
		}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(
			&source.Kind{Type: &argocd.Application{}},
			handler.EnqueueRequestsFromMapFunc(
				func(obj client.Object) []reconcile.Request {
					return e.findEnvsForApp(ctx, obj)
				},
			),
		).
		Complete(e)
}

func newReconciler(
	client client.Client,
	credentialsDB credentials.Database,
) (*reconciler, error) {
	r := &reconciler{
		client:        client,
		credentialsDB: credentialsDB,
	}

	// The following default behaviors are overridable for testing purposes:

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
	r.getLatestCommitIDFn = git.GetLatestCommitID

	return r, nil
}

// findEnvsForApp dynamically returns reconciliation requests for all
// Environments related to a given Argo CD Application. This is used to
// propagate reconciliation requests to Environments whose state should be
// affected by changes to related Application resources.
func (r *reconciler) findEnvsForApp(
	ctx context.Context,
	app client.Object,
) []reconcile.Request {
	envs := &api.EnvironmentList{}
	if err := r.client.List(
		ctx,
		envs,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				envsByAppIndexField,
				fmt.Sprintf("%s:%s", app.GetNamespace(), app.GetName()),
			),
		},
	); err != nil {
		logging.LoggerFromContext(ctx).WithFields(log.Fields{
			"namespace":   app.GetNamespace(),
			"application": app.GetName(),
		}).Error("error listing Environments associated with Application")
		return nil
	}
	reqs := make([]reconcile.Request, len(envs.Items))
	for i, env := range envs.Items {
		reqs[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      env.GetName(),
				Namespace: env.GetNamespace(),
			},
		}
	}
	return reqs
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
	env, err := api.GetEnv(ctx, r.client, req.NamespacedName)
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

	updateErr := r.client.Status().Update(ctx, env)
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

	// Only perform health checks if we have a current state to update
	if currentState, ok := status.States.Pop(); ok {
		health := r.checkHealthFn(ctx, currentState, env.Spec.HealthChecks)
		currentState.Health = &health
		status.States.Push(currentState)
		logger.WithField("health", health.Status).Debug("completed health checks")
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
			latestState.SameMaterials(&topAvailableState) {
			logger.Debug("latest state is not new")
			return status, nil
		}
		status.AvailableStates.Push(*latestState)
		logger.Debug("latest state is new; added to available states")

	} else if len(env.Spec.Subscriptions.UpstreamEnvs) > 0 {

		// This returns de-duped, healthy states only from all upstream envs. There
		// could be up to ten per upstream environment. This is more than the usual
		// quantity we permit in status.AvailableStates, but we'll allow it.
		latestStatesFromEnvs, err := r.getAvailableStatesFromUpstreamEnvsFn(
			ctx,
			env.Spec.Subscriptions.UpstreamEnvs,
		)
		if err != nil {
			return status, err
		}
		status.AvailableStates = latestStatesFromEnvs
		if len(latestStatesFromEnvs) == 0 {
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
	} else {
		// This should be impossible if validation is working, but out of an
		// abundance of caution, bail now if this happens somehow.
		return status, nil
	}

	if !env.Spec.EnableAutoPromotion {
		logger.Debug("auto-promotion is not enabled for this environment")
		return status, nil
	}

	// Note: We're careful not to make any further modifications to the state
	// stacks until we know a promotion has been successful.

	nextStateCandidate, _ := status.AvailableStates.Top()
	if currentState, ok := status.States.Top(); ok &&
		nextStateCandidate.FirstSeen.Before(currentState.FirstSeen) {
		logger.Debug(
			"newest available state is older than current state; refusing to " +
				"auto-promote",
		)
		return status, nil
	}
	nextState := nextStateCandidate

	// If we get to here, we've determined that auto-promotion is enabled and
	// safe.
	logger = logger.WithField("state", nextState.ID)
	logger.Debug("auto-promotion will proceed")

	// TODO: If we name this deterministically, we can check first if it already
	// exists -- which is a thing that could happen if, on a previous
	// reconciliation, we succeeded in creating the Promotion, but failed to
	// update the Environment status.
	if err := r.client.Create(
		ctx,
		&api.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", env.Name),
				Namespace:    env.Namespace,
			},
			Spec: &api.PromotionSpec{
				Environment: env.Name,
				State:       nextState.ID,
			},
		},
		&client.CreateOptions{},
	); err != nil {
		return status, err
	}
	logger.Debug("created Promotion resource")

	return status, nil
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
	return &api.EnvironmentState{
		ID:        uuid.NewV4().String(),
		FirstSeen: &now,
		Commits:   latestCommits,
		Images:    latestImages,
		Charts:    latestCharts,
	}, nil
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
			r.client,
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
		for _, state := range upstreamEnv.Status.States {
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
