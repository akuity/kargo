package controller

import (
	"context"
	"time"

	"github.com/akuityio/bookkeeper"
	"github.com/argoproj-labs/argocd-image-updater/pkg/image"
	"github.com/argoproj-labs/argocd-image-updater/pkg/registry"
	"github.com/argoproj-labs/argocd-image-updater/pkg/tag"
	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/db"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
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
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/git"
	"github.com/akuityio/kargo/internal/helm"
)

const (
	envsByAppIndexField = "applications"
)

// environmentReconciler reconciles Environment resources.
type environmentReconciler struct {
	config     config.Config
	client     client.Client
	kubeClient kubernetes.Interface
	argoDB     db.ArgoDB
	logger     *log.Logger
	// The following behaviors are all overridable for testing purposes
	getNextAvailableStateFn func(
		context.Context,
		*api.Environment,
	) (*api.EnvironmentState, error)
	getNextStateFromUpstreamReposFn func(
		context.Context,
		*api.Environment,
	) (*api.EnvironmentState, error)
	getLatestCommitFn func(
		context.Context,
		*api.Environment,
	) (*api.GitCommit, error)
	getGitRepoCredentialsFn func(
		ctx context.Context,
		repoURL string,
	) (git.RepoCredentials, error)
	gitCloneFn func(
		ctx context.Context,
		url string,
		repoCreds git.RepoCredentials,
	) (git.Repo, error)
	checkoutBranchFn  func(repo git.Repo, branch string) error
	getLastCommitIDFn func(git.Repo) (string, error)
	getLatestImagesFn func(
		ctx context.Context,
		env *api.Environment,
	) ([]api.Image, error)
	getImageRepoCredentialsFn func(
		ctx context.Context,
		namespace string,
		sub api.ImageSubscription,
		rep *registry.RegistryEndpoint,
	) (image.Credential, error)
	getImageTagsFn func(
		*registry.RegistryEndpoint,
		*image.ContainerImage,
		registry.RegistryClient,
		*image.VersionConstraint,
	) (*tag.ImageTagList, error)
	getNewestImageTagFn func(
		*image.ContainerImage,
		*image.VersionConstraint,
		*tag.ImageTagList,
	) (*tag.ImageTag, error)
	getLatestChartsFn func(
		ctx context.Context,
		env *api.Environment,
	) ([]api.Chart, error)
	getChartRegistryCredentialsFn func(
		ctx context.Context,
		repoURL string,
	) (*helm.RepoCredentials, error)
	promoteFn func(
		ctx context.Context,
		env *api.Environment,
		newState api.EnvironmentState,
	) (api.EnvironmentState, error)
	renderManifestsWithBookkeeperFn func(
		context.Context,
		bookkeeper.RenderRequest,
	) (bookkeeper.RenderResponse, error)
	getArgoCDAppFn func(
		ctx context.Context,
		namespace string,
		name string,
	) (*argocd.Application, error)
	updateArgoCDAppFn func(
		ctx context.Context,
		env *api.Environment,
		newState api.EnvironmentState,
		appUpdate api.ArgoCDAppUpdate,
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
			if env.Spec.HealthChecks != nil {
				return env.Spec.HealthChecks.ArgoCDApps
			}
			return nil
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
		config:     config,
		client:     client,
		kubeClient: kubeClient,
		argoDB:     argoDB,
		logger:     logger,
	}
	// Defaults for overridable behaviors:
	e.getNextAvailableStateFn = e.getNextAvailableState
	e.getNextStateFromUpstreamReposFn = e.getNextStateFromUpstreamRepos
	e.getLatestCommitFn = e.getLatestCommit
	e.getGitRepoCredentialsFn = e.getGitRepoCredentials
	e.gitCloneFn = git.Clone
	e.checkoutBranchFn = checkoutBranch
	e.getLastCommitIDFn = getLastCommitID
	e.getLatestImagesFn = e.getLatestImages
	e.getImageRepoCredentialsFn = e.getImageRepoCredentials
	e.getImageTagsFn = getImageTags
	e.getNewestImageTagFn = getNewestImageTag
	e.getLatestChartsFn = e.getLatestCharts
	e.getChartRegistryCredentialsFn = e.getChartRegistryCredentials
	e.promoteFn = e.promote
	e.renderManifestsWithBookkeeperFn = bookkeeperService.RenderManifests
	e.getArgoCDAppFn = e.getArgoCDApp
	e.updateArgoCDAppFn = e.updateArgoCDApp
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
				app.GetName(),
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

	health := e.checkHealth(ctx, env)
	if len(status.States) > 0 {
		status.States[0].Health = health
	}

	nextAvailableState, err := e.getNextAvailableStateFn(ctx, env)
	if err != nil {
		status.Error = err.Error()
		return status
	}

	if nextAvailableState != nil &&
		(len(status.AvailableStates) == 0 ||
			!nextAvailableState.SameMaterials(&status.AvailableStates[0])) {
		status.AvailableStates = append(
			[]api.EnvironmentState{*nextAvailableState},
			status.AvailableStates...,
		)
		const maxAvailableStates = 10 // TODO: Make this configurable?
		if len(status.AvailableStates) > maxAvailableStates {
			status.AvailableStates = status.AvailableStates[:maxAvailableStates]
		}
	}

	if len(status.AvailableStates) == 0 {
		// Nothing further to do
		return status
	}

	nextStateCandidate := status.AvailableStates[0]
	if len(status.States) == 0 ||
		!nextStateCandidate.SameMaterials(&status.States[0]) {
		nextState := nextStateCandidate
		if nextState, err = e.promoteFn(ctx, env, nextState); err != nil {
			status.Error = err.Error()
			return status
		}
		status.States = append([]api.EnvironmentState{nextState}, status.States...)
		const maxStates = 10 // TODO: Make this configurable?
		if len(status.States) > maxStates {
			status.States = status.States[:maxStates]
		}
	}

	return status
}

func (e *environmentReconciler) getNextAvailableState(
	ctx context.Context,
	env *api.Environment,
) (*api.EnvironmentState, error) {
	if env.Spec.Subscriptions == nil || env.Spec.Subscriptions.Repos == nil {
		return nil, nil
	}

	nextState, err := e.getNextStateFromUpstreamReposFn(ctx, env)
	return nextState, errors.Wrapf(
		err,
		"error getting next state for Environment %q in namespace %q from "+
			"upstream repos",
		env.Name,
		env.Namespace,
	)

	// TODO: Handle subscription to upstream environments
}

func (e *environmentReconciler) getNextStateFromUpstreamRepos(
	ctx context.Context,
	env *api.Environment,
) (*api.EnvironmentState, error) {
	if env.Spec.Subscriptions == nil || env.Spec.Subscriptions.Repos == nil {
		return nil, nil
	}

	latestGitCommit, err := e.getLatestCommitFn(ctx, env)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error syncing git repo subscription for Environment %q in namespace %q",
			env.Name,
			env.Namespace,
		)
	}

	latestImages, err := e.getLatestImagesFn(ctx, env)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error syncing image repo subscriptions for Environment %q in "+
				"namespace %q",
			env.Name,
			env.Namespace,
		)
	}

	latestCharts, err := e.getLatestChartsFn(ctx, env)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error syncing chart repo subscriptions for Environment %q in "+
				"namespace %q",
			env.Name,
			env.Namespace,
		)
	}

	return &api.EnvironmentState{
		ID:        uuid.NewV4().String(),
		GitCommit: latestGitCommit,
		Images:    latestImages,
		Charts:    latestCharts,
	}, nil
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
		}).Error("error updating Environment status")
	}
}
