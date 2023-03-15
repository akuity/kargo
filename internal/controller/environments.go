package controller

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

	"github.com/akuityio/bookkeeper"
	api "github.com/akuityio/kargo/api/v1alpha1"
	libArgoCD "github.com/akuityio/kargo/internal/argocd"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/helm"
	"github.com/akuityio/kargo/internal/images"
	"github.com/akuityio/kargo/internal/kustomize"
	"github.com/akuityio/kargo/internal/logging"
	"github.com/akuityio/kargo/internal/yaml"
)

const (
	envsByAppIndexField = "applications"
)

// environmentReconciler reconciles Environment resources.
type environmentReconciler struct {
	config            config.ControllerConfig
	client            client.Client
	credentialsDB     credentialsDB
	bookkeeperService bookkeeper.Service
	logger            *log.Logger

	// The following behaviors are overridable for testing purposes:

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
		api.HealthChecks,
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

	// Promotions (general):
	promoteFn func(
		ctx context.Context,
		namespace string,
		promoMechanisms api.PromotionMechanisms,
		newState api.EnvironmentState,
	) (api.EnvironmentState, error)

	// Promotions via Git:
	gitApplyUpdateFn func(
		repoURL string,
		branch string,
		creds *git.Credentials,
		updateFn func(homeDir, workingDir string) (string, error),
	) (string, error)

	// Promotions via Git + Kustomize:
	kustomizeSetImageFn func(dir, repo, tag string) error

	// Promotions via Git + Helm:
	buildChartDependencyChangesFn func(
		repoDir string,
		charts []api.Chart,
		chartUpdates []api.HelmChartDependencyUpdate,
	) (map[string]map[string]string, error)

	updateChartDependenciesFn func(homePath, chartPath string) error

	setStringsInYAMLFileFn func(
		file string,
		changes map[string]string,
	) error

	// Promotions via Argo CD:
	applyArgoCDSourceUpdateFn func(
		argocd.ApplicationSource,
		api.EnvironmentState,
		api.ArgoCDSourceUpdate,
	) (argocd.ApplicationSource, error)

	patchFn func(
		ctx context.Context,
		obj client.Object,
		patch client.Patch,
		opts ...client.PatchOption,
	) error
}

// SetupEnvironmentReconcilerWithManager initializes a reconciler for
// Environment resources and registers it with the provided Manager.
func SetupEnvironmentReconcilerWithManager(
	ctx context.Context,
	config config.ControllerConfig,
	mgr manager.Manager,
	bookkeeperService bookkeeper.Service,
) error {
	logger := log.New()
	logger.SetLevel(config.LogLevel)

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

	e, err := newEnvironmentReconciler(
		ctx,
		config,
		mgr,
		bookkeeperService,
	)
	if err != nil {
		return errors.Wrap(err, "error initializing Environment reconciler")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Environment{}).WithEventFilter(predicate.Funcs{
		DeleteFunc: func(event.DeleteEvent) bool {
			// We're not interested in any deletes
			return false
		},
	}).Watches(
		&source.Kind{Type: &argocd.Application{}},
		handler.EnqueueRequestsFromMapFunc(e.findEnvsForApp),
	).Complete(e)
}

func newEnvironmentReconciler(
	ctx context.Context,
	config config.ControllerConfig,
	mgr manager.Manager,
	bookkeeperService bookkeeper.Service,
) (*environmentReconciler, error) {
	logger := log.New()
	logger.SetLevel(config.LogLevel)

	var credentialsDB credentialsDB
	if mgr != nil { // This can be nil during tests
		// TODO: Do not hardcode the Argo CD namespace
		var err error
		if credentialsDB, err =
			newKubernetesCredentialsDB(ctx, "argo-cd", mgr); err != nil {
			return nil, errors.Wrap(err, "error initializing credentials DB")
		}
	}

	e := &environmentReconciler{
		config:            config,
		credentialsDB:     credentialsDB,
		bookkeeperService: bookkeeperService,
		logger:            logger,
	}
	if mgr != nil { // This can be nil during tests
		e.client = mgr.GetClient()
	}

	// The following default behaviors are overridable for testing purposes:

	// Common:
	e.getArgoCDAppFn = libArgoCD.GetApplication

	// Health checks:
	e.checkHealthFn = e.checkHealth

	// Syncing:
	e.getLatestStateFromReposFn = e.getLatestStateFromRepos
	e.getAvailableStatesFromUpstreamEnvsFn = e.getAvailableStatesFromUpstreamEnvs
	e.getLatestCommitsFn = e.getLatestCommits
	e.getLatestImagesFn = e.getLatestImages
	e.getLatestTagFn = images.GetLatestTag
	e.getLatestChartsFn = e.getLatestCharts
	e.getLatestChartVersionFn = helm.GetLatestChartVersion
	e.getLatestCommitIDFn = git.GetLatestCommitID

	// Promotions (general):
	e.promoteFn = e.promote
	// Promotions via Git:
	e.gitApplyUpdateFn = git.ApplyUpdate
	// Promotions via Git + Kustomize:
	e.kustomizeSetImageFn = kustomize.SetImage
	// Promotions via Git + Helm:
	e.buildChartDependencyChangesFn = buildChartDependencyChanges
	e.updateChartDependenciesFn = helm.UpdateChartDependencies
	e.setStringsInYAMLFileFn = yaml.SetStringsInFile
	// Promotions via Argo CD:
	e.applyArgoCDSourceUpdateFn = e.applyArgoCDSourceUpdate
	if mgr != nil { // This can be nil during testing
		e.patchFn = mgr.GetClient().Patch
	}

	return e, nil
}

// findEnvsForApp dynamically returns reconciliation requests for all
// Environments related to a given Argo CD Application. This is used to
// propagate reconciliation requests to Environments whose state should be
// affected by changes to related Application resources.
func (e *environmentReconciler) findEnvsForApp(
	app client.Object,
) []reconcile.Request {
	envs := &api.EnvironmentList{}
	if err := e.client.List(
		context.Background(),
		envs,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				envsByAppIndexField,
				fmt.Sprintf("%s:%s", app.GetNamespace(), app.GetName()),
			),
		},
	); err != nil {
		e.logger.WithFields(log.Fields{
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
func (e *environmentReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{}

	logger := e.logger.WithFields(log.Fields{
		"namespace":   req.NamespacedName.Namespace,
		"environment": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Environment")

	// Find the environment
	env, err := e.getEnv(ctx, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if env == nil {
		// Ignore if not found. This can happen if the Environment was deleted after
		// the current reconciliation request was issued.
		return result, nil
	}
	logger.Debug("found Environment")

	env.Status = e.sync(ctx, env)
	if env.Status.Error != "" {
		logger.Errorf("error syncing Environment: %s", env.Status.Error)
	}
	e.updateStatus(ctx, env)

	logger.Debug("done reconciling Environment")

	// TODO: Make RequeueAfter configurable (via API, probably)
	// TODO: Or consider using a progressive backoff here when there has been an
	// error.
	return ctrl.Result{RequeueAfter: time.Minute}, err
}

func (e *environmentReconciler) sync(
	ctx context.Context,
	env *api.Environment,
) api.EnvironmentStatus {
	statusPtr := env.Status.DeepCopy()
	status := *statusPtr
	status.Error = ""

	logger := logging.LoggerFromContext(ctx)

	// Only perform health checks if we have a current state to update
	var currentState api.EnvironmentState
	var ok bool
	if status.States, currentState, ok = status.States.Pop(); ok {
		health := e.checkHealthFn(ctx, currentState, *env.Spec.HealthChecks)
		currentState.Health = &health
		status.States = status.States.Push(currentState)
		logger.WithField("health", health.Status).Debug("completed health checks")
	} else {
		logger.Debug("Environment has no current state; skipping health checks")
	}

	var autoPromote bool

	if env.Spec.Subscriptions.Repos != nil {

		latestState, err := e.getLatestStateFromReposFn(
			ctx,
			env.Namespace,
			*env.Spec.Subscriptions.Repos,
		)
		if err != nil {
			status.Error = err.Error()
			return status
		}

		// If not nil, latestState from upstream repos will always have a shiny new
		// ID. To determine if this is actually new and needs to be pushed onto the
		// status.AvailableStates stack, either that stack needs to be empty or
		// latestState's MATERIALS must differ from what is at the top of the
		// status.AvailableStates stack.
		if latestState != nil {
			logger.Debug("got latest state from upstream repositories")
			if _, topAvailableState, ok := status.AvailableStates.Pop(); !ok ||
				!latestState.SameMaterials(&topAvailableState) {
				status.AvailableStates = status.AvailableStates.Push(*latestState)
				logger.Debug("latest state is new; added to available states")
			} else {
				logger.Debug("latest state is not new")
			}
		} else {
			logger.Debug("found no state from upstream repositories")
		}

		autoPromote = true

	} else if len(env.Spec.Subscriptions.UpstreamEnvs) > 0 {

		// This returns de-duped, healthy states only from all upstream envs. There
		// could be up to ten per upstream environment. This is more than the usual
		// quantity we permit in status.AvailableStates, but we'll allow it.
		latestStatesFromEnvs, err := e.getAvailableStatesFromUpstreamEnvsFn(
			ctx,
			env.Spec.Subscriptions.UpstreamEnvs,
		)
		if err != nil {
			status.Error = err.Error()
			return status
		}
		status.AvailableStates = latestStatesFromEnvs
		if len(latestStatesFromEnvs) == 0 {
			logger.Debug("got no available states from upstream Environments")
		} else {
			logger.Debug("got available states from upstream Environments")
		}

		// If we're subscribed to more than one upstream environment, then it's
		// ambiguous which of the status.AvailableStates we should use, so
		// auto-promotion is off the table.
		autoPromote = len(env.Spec.Subscriptions.UpstreamEnvs) == 1

	}

	if !autoPromote || status.AvailableStates.Empty() {
		logger.Debug("auto-promotion cannot proceed")
		return status // Nothing further to do
	}

	// Note: We're careful not to make any further modifications to the state
	// stacks until we know a promotion has been successful.
	_, nextStateCandidate, _ := status.AvailableStates.Pop()
	// Proceed with promotion if there is no currentState OR the
	// nextStateCandidate is different and NEWER than the currentState
	if _, currentState, ok := status.States.Pop(); !ok ||
		(nextStateCandidate.ID != currentState.ID &&
			nextStateCandidate.FirstSeen.After(currentState.FirstSeen.Time)) {
		logger = logger.WithField("state", nextStateCandidate.ID)
		logger.Debug("auto-promotion will proceed")
		ctx = logging.ContextWithLogger(ctx, logger)
		nextState, err := e.promoteFn(
			ctx,
			env.Namespace,
			*env.Spec.PromotionMechanisms,
			nextStateCandidate,
		)
		if err != nil {
			status.Error = err.Error()
			return status
		}
		status.States = status.States.Push(nextState)
		logger.Debug("promoted Environment to new state")

		// Promotion is successful at this point. Replace the top available state
		// because the promotion process may have updated some commit IDs.
		var topAvailableState api.EnvironmentState
		status.AvailableStates, topAvailableState, _ = status.AvailableStates.Pop()
		for i := range topAvailableState.Commits {
			topAvailableState.Commits[i].ID = nextState.Commits[i].ID
		}
		status.AvailableStates = status.AvailableStates.Push(topAvailableState)
	} else {
		logger.Debug("found nothing to promote")
	}

	return status
}

func (e *environmentReconciler) getLatestStateFromRepos(
	ctx context.Context,
	namespace string,
	repoSubs api.RepoSubscriptions,
) (*api.EnvironmentState, error) {
	logger := logging.LoggerFromContext(ctx)

	latestCommits, err := e.getLatestCommitsFn(ctx, namespace, repoSubs.Git)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing git repo subscriptions")
	}
	if len(repoSubs.Git) > 0 {
		logger.Debug("synced git repo subscriptions")
	}

	latestImages, err := e.getLatestImagesFn(ctx, namespace, repoSubs.Images)
	if err != nil {
		return nil, errors.Wrap(err, "error syncing image repo subscriptions")
	}
	if len(repoSubs.Images) > 0 {
		logger.Debug("synced image repo subscriptions")
	}

	latestCharts, err := e.getLatestChartsFn(ctx, namespace, repoSubs.Charts)
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
func (e *environmentReconciler) getAvailableStatesFromUpstreamEnvs(
	ctx context.Context,
	subs []api.EnvironmentSubscription,
) ([]api.EnvironmentState, error) {
	if len(subs) == 0 {
		return nil, nil
	}

	availableStates := []api.EnvironmentState{}
	stateSet := map[string]struct{}{} // We'll use this to de-dupe
	for _, sub := range subs {
		upstreamEnv, err := e.getEnv(
			ctx,
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

// TODO: This function could use some tests
func (e *environmentReconciler) promote(
	ctx context.Context,
	namespace string,
	promoMechanisms api.PromotionMechanisms,
	newState api.EnvironmentState,
) (api.EnvironmentState, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.WithField("state", newState.ID)
	logger.Debug("executing promotion to new state")
	var err error
	for _, gitRepoUpdate := range promoMechanisms.GitRepoUpdates {
		if gitRepoUpdate.Bookkeeper != nil {
			if newState, err = e.applyBookkeeperUpdate(
				ctx,
				namespace,
				newState,
				gitRepoUpdate,
			); err != nil {
				return newState, errors.Wrap(err, "error promoting via Git")
			}
		} else {
			if newState, err = e.applyGitRepoUpdate(
				ctx,
				namespace,
				newState,
				gitRepoUpdate,
			); err != nil {
				return newState, errors.Wrap(err, "error promoting via Git")
			}
		}
	}
	if len(promoMechanisms.GitRepoUpdates) > 0 {
		logger.Debug("completed git-based promotion steps")
	}

	for _, argoCDAppUpdate := range promoMechanisms.ArgoCDAppUpdates {
		if err =
			e.applyArgoCDAppUpdate(ctx, newState, argoCDAppUpdate); err != nil {
			return newState, errors.Wrap(err, "error promoting via Argo CD")
		}
	}
	if len(promoMechanisms.ArgoCDAppUpdates) > 0 {
		logger.Debug("completed Argo CD-based promotion steps")
	}

	newState.Health = &api.Health{
		Status:       api.HealthStateUnknown,
		StatusReason: "Health has not yet been assessed",
	}

	logger.Debug("completed promotion")

	return newState, nil
}

// getEnv returns a pointer to the Environment resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func (e *environmentReconciler) getEnv(
	ctx context.Context,
	namespacedName types.NamespacedName,
) (*api.Environment, error) {
	env := api.Environment{}
	if err := e.client.Get(ctx, namespacedName, &env); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			logging.LoggerFromContext(ctx).WithFields(log.Fields{
				"namespace":   namespacedName.Namespace,
				"environment": namespacedName.Name,
			}).Warn("Environment not found")
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Environment %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &env, nil
}

// updateStatus updates the status subresource of the provided Environment.
func (e *environmentReconciler) updateStatus(
	ctx context.Context,
	env *api.Environment,
) {
	if err := e.client.Status().Update(ctx, env); err != nil {
		logging.LoggerFromContext(ctx).WithFields(log.Fields{
			"namespace":   env.Namespace,
			"environment": env.Name,
		}).Errorf("error updating Environment status: %s", err)
	}
}
