package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/akuityio/bookkeeper"
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
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
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/helm"
	"github.com/akuityio/kargo/internal/images"
	"github.com/akuityio/kargo/internal/kustomize"
	"github.com/akuityio/kargo/internal/yaml"
)

const (
	envsByAppIndexField = "applications"
)

// environmentReconciler reconciles Environment resources.
type environmentReconciler struct {
	config            config.Config
	client            client.Client
	kubeClient        kubernetes.Interface
	argoDB            db.ArgoDB
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

	gitRepoCredentialsFn func(
		ctx context.Context,
		argoDB libArgoCD.DB,
		repoURL string,
	) (*git.RepoCredentials, error)

	// Health checks:
	checkHealthFn func(
		context.Context,
		api.EnvironmentState,
		api.HealthChecks,
	) api.Health

	// Syncing:
	getLatestStateFromReposFn func(
		context.Context,
		*api.Environment,
	) (*api.EnvironmentState, error)

	getAvailableStatesFromUpstreamEnvsFn func(
		context.Context,
		*api.Environment,
	) ([]api.EnvironmentState, error)

	getLatestCommitsFn func(
		context.Context,
		[]api.GitSubscription,
	) ([]api.GitCommit, error)

	getLatestImagesFn func(
		context.Context,
		[]api.ImageSubscription,
	) ([]api.Image, error)

	getLatestTagFn func(
		ctx context.Context,
		kubeClient kubernetes.Interface,
		repoURL string,
		updateStrategy images.ImageUpdateStrategy,
		semverConstraint string,
		allowTags string,
		ignoreTags []string,
		platform string,
		pullSecret string,
	) (string, error)

	chartRegistryCredentialsFn func(
		ctx context.Context,
		argoDB libArgoCD.DB,
		registryURL string,
	) (*helm.RegistryCredentials, error)

	getLatestChartsFn func(
		context.Context,
		[]api.ChartSubscription,
	) ([]api.Chart, error)

	getLatestChartVersionFn func(
		ctx context.Context,
		registryURL string,
		chart string,
		semverConstraint string,
		creds *helm.RegistryCredentials,
	) (string, error)

	getLatestCommitIDFn func(
		repoURL string,
		branch string,
		creds *git.RepoCredentials,
	) (string, error)

	// Promotions (general):
	promoteFn func(
		context.Context,
		*api.Environment,
		api.EnvironmentState,
	) (api.EnvironmentState, error)

	// Promotions via Git:
	gitApplyUpdateFn func(
		repoURL string,
		branch string,
		creds *git.RepoCredentials,
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
	config config.Config,
	mgr manager.Manager,
	kubeClient kubernetes.Interface,
	argoDB db.ArgoDB,
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

	e := newEnvironmentReconciler(
		config,
		mgr.GetClient(),
		kubeClient,
		argoDB,
		bookkeeperService,
	)

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
	config config.Config,
	client client.Client,
	kubeClient kubernetes.Interface,
	argoDB db.ArgoDB,
	bookkeeperService bookkeeper.Service,
) *environmentReconciler {
	logger := log.New()
	logger.SetLevel(config.LogLevel)
	e := &environmentReconciler{
		config:            config,
		client:            client,
		kubeClient:        kubeClient,
		argoDB:            argoDB,
		bookkeeperService: bookkeeperService,
		logger:            logger,
	}

	// The following default behaviors are overridable for testing purposes:

	// Common:
	e.getArgoCDAppFn = libArgoCD.GetApplication
	e.gitRepoCredentialsFn = libArgoCD.GetGitRepoCredentials

	// Health checks:
	e.checkHealthFn = e.checkHealth

	// Syncing:
	e.getLatestStateFromReposFn = e.getLatestStateFromRepos
	e.getAvailableStatesFromUpstreamEnvsFn = e.getAvailableStatesFromUpstreamEnvs
	e.getLatestCommitsFn = e.getLatestCommits
	e.getLatestImagesFn = e.getLatestImages
	e.getLatestTagFn = images.GetLatestTag
	e.chartRegistryCredentialsFn = libArgoCD.GetChartRegistryCredentials
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
	if client != nil { // This can be nil during testing
		e.patchFn = client.Patch
	}

	return e
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
		"namespace": req.NamespacedName.Namespace,
		"name":      req.NamespacedName.Name,
	})

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

	env.Status = e.sync(ctx, env)
	if env.Status.Error != "" {
		logger.Error(env.Status.Error)
	}
	e.updateStatus(ctx, env)

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

	// Only perform health checks if we have a current state to update
	var currentState api.EnvironmentState
	var ok bool
	if status.States, currentState, ok = status.States.Pop(); ok {
		health := e.checkHealthFn(ctx, currentState, env.Spec.HealthChecks)
		currentState.Health = &health
		status.States = status.States.Push(currentState)
	}

	if env.Spec.Subscriptions == nil {
		return status // Nothing further to do
	}

	var autoPromote bool

	if env.Spec.Subscriptions.Repos != nil {

		latestState, err := e.getLatestStateFromReposFn(ctx, env)
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
			if _, topAvailableState, ok := status.AvailableStates.Pop(); !ok ||
				!latestState.SameMaterials(&topAvailableState) {
				status.AvailableStates = status.AvailableStates.Push(*latestState)
			}
		}

		autoPromote = true

	} else if len(env.Spec.Subscriptions.UpstreamEnvs) > 0 {

		// This returns de-duped, healthy states only from all upstream envs. There
		// could be up to ten per upstream environment. This is more than the usual
		// quantity we permit in status.AvailableStates, but we'll allow it.
		latestStatesFromEnvs, err :=
			e.getAvailableStatesFromUpstreamEnvsFn(ctx, env)
		if err != nil {
			status.Error = err.Error()
			return status
		}
		status.AvailableStates = latestStatesFromEnvs

		// If we're subscribed to more than one upstream environment, then it's
		// ambiguous which of the status.AvailableStates we should use, so
		// auto-promotion is off the table.
		autoPromote = len(env.Spec.Subscriptions.UpstreamEnvs) == 1

	}

	if !autoPromote || status.AvailableStates.Empty() {
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
		nextState, err := e.promoteFn(ctx, env, nextStateCandidate)
		if err != nil {
			status.Error = err.Error()
			return status
		}
		status.States = status.States.Push(nextState)

		// Promotion is successful that this point. Replace the top available state
		// because the promotion process may have updated some commit IDs.
		var topAvailableState api.EnvironmentState
		status.AvailableStates, topAvailableState, _ = status.AvailableStates.Pop()
		for i := range topAvailableState.Commits {
			topAvailableState.Commits[i].ID = nextState.Commits[i].ID
		}
		status.AvailableStates = status.AvailableStates.Push(topAvailableState)
	}

	return status
}

func (e *environmentReconciler) getLatestStateFromRepos(
	ctx context.Context,
	env *api.Environment,
) (*api.EnvironmentState, error) {
	if env.Spec.Subscriptions == nil || env.Spec.Subscriptions.Repos == nil {
		return nil, nil
	}

	latestCommits, err :=
		e.getLatestCommitsFn(ctx, env.Spec.Subscriptions.Repos.Git)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error syncing git repo subscriptions for Environment %q in namespace %q",
			env.Name,
			env.Namespace,
		)
	}

	latestImages, err :=
		e.getLatestImagesFn(ctx, env.Spec.Subscriptions.Repos.Images)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error syncing image repo subscriptions for Environment %q in "+
				"namespace %q",
			env.Name,
			env.Namespace,
		)
	}

	latestCharts, err :=
		e.getLatestChartsFn(ctx, env.Spec.Subscriptions.Repos.Charts)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error syncing chart repo subscriptions for Environment %q in "+
				"namespace %q",
			env.Name,
			env.Namespace,
		)
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

// TODO: Implement this
func (e *environmentReconciler) getAvailableStatesFromUpstreamEnvs(
	ctx context.Context,
	env *api.Environment,
) ([]api.EnvironmentState, error) {
	return nil, nil
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
			e.logger.WithFields(log.Fields{
				"namespace": namespacedName.Namespace,
				"name":      namespacedName.Name,
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
		e.logger.WithFields(log.Fields{
			"namespace": env.Namespace,
			"name":      env.Name,
		}).Errorf("error updating Environment status: %s", err)
	}
}
