package promotions

import (
	"context"
	"sync"
	"time"

	argocd "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/akuity/bookkeeper"
	"github.com/akuity/bookkeeper/pkg/git"
	api "github.com/akuity/kargo/api/v1alpha1"
	libArgoCD "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/helm"
	"github.com/akuity/kargo/internal/kustomize"
	"github.com/akuity/kargo/internal/logging"
	"github.com/akuity/kargo/internal/yaml"
)

// reconciler reconciles Promotion resources.
type reconciler struct {
	kargoClient       client.Client
	argoClient        client.Client
	credentialsDB     credentials.Database
	bookkeeperService bookkeeper.Service

	promoQueuesByStage   map[types.NamespacedName]runtime.PriorityQueue
	promoQueuesByStageMu sync.Mutex
	initializeOnce       sync.Once

	// The following behaviors are overridable for testing purposes:

	// Promotions (general):
	promoteFn func(
		ctx context.Context,
		stageName string,
		stageNamespace string,
		stateID string,
	) error

	applyPromotionMechanismsFn func(
		ctx context.Context,
		stageMeta metav1.ObjectMeta,
		promoMechanisms api.PromotionMechanisms,
		newState api.StageState,
	) (api.StageState, error)

	// Promotions via Git:
	gitApplyUpdateFn func(
		repoURL string,
		readRef string,
		writeBranch string,
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
	) (map[string]map[string]string, []string, error)

	updateChartDependenciesFn func(homePath, chartPath string) error

	setStringsInYAMLFileFn func(
		file string,
		changes map[string]string,
	) error

	// Promotions via Argo CD:
	getArgoCDAppFn func(
		ctx context.Context,
		client client.Client,
		namespace string,
		name string,
	) (*argocd.Application, error)

	applyArgoCDSourceUpdateFn func(
		argocd.ApplicationSource,
		api.StageState,
		api.ArgoCDSourceUpdate,
	) (argocd.ApplicationSource, error)

	argoCDAppPatchFn func(
		ctx context.Context,
		obj client.Object,
		patch client.Patch,
		opts ...client.PatchOption,
	) error
}

// SetupReconcilerWithManager initializes a reconciler for Promotion resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	kargoMgr manager.Manager,
	argoMgr manager.Manager,
	credentialsDB credentials.Database,
	bookkeeperService bookkeeper.Service,
	shardName string,
) error {

	shardPredicate, err := controller.GetShardPredicate(shardName)
	if err != nil {
		return errors.Wrap(err, "error creating shard selector predicate")
	}

	return errors.Wrap(
		ctrl.NewControllerManagedBy(kargoMgr).
			For(&api.Promotion{}).
			WithEventFilter(predicate.GenerationChangedPredicate{}).
			WithEventFilter(shardPredicate).
			Complete(
				newReconciler(
					kargoMgr.GetClient(),
					argoMgr.GetClient(),
					credentialsDB,
					bookkeeperService,
				),
			),
		"error registering Promotion reconciler",
	)
}

func newReconciler(
	kargoClient client.Client,
	argoClient client.Client,
	credentialsDB credentials.Database,
	bookkeeperService bookkeeper.Service,
) *reconciler {
	r := &reconciler{
		kargoClient:        kargoClient,
		argoClient:         argoClient,
		credentialsDB:      credentialsDB,
		bookkeeperService:  bookkeeperService,
		promoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}

	// Promotions (general):
	r.promoteFn = r.promote
	r.applyPromotionMechanismsFn = r.applyPromotionMechanisms
	// Promotions via Git:
	r.gitApplyUpdateFn = gitApplyUpdate
	// Promotions via Git + Kustomize:
	r.kustomizeSetImageFn = kustomize.SetImage
	// Promotions via Git + Helm:
	r.buildChartDependencyChangesFn = buildChartDependencyChanges
	r.updateChartDependenciesFn = helm.UpdateChartDependencies
	r.setStringsInYAMLFileFn = yaml.SetStringsInFile
	// Promotions via Argo CD:
	r.getArgoCDAppFn = libArgoCD.GetApplication
	r.applyArgoCDSourceUpdateFn = r.applyArgoCDSourceUpdate
	r.argoCDAppPatchFn = argoClient.Patch

	return r
}

func newPromotionsQueue() runtime.PriorityQueue {
	// We can safely ignore errors here because the only error that can happen
	// involves initializing the queue with a nil priority function, which we
	// know we aren't doing.
	pq, _ := runtime.NewPriorityQueue(func(left, right client.Object) bool {
		return left.GetCreationTimestamp().Time.
			Before(right.GetCreationTimestamp().Time)
	})
	return pq
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	// We count all of Reconcile() as a critical section of code to ensure we
	// don't start reconciling a second Promotion before lazy initialization
	// completes upon reconciliation of the FIRST promotion.
	r.promoQueuesByStageMu.Lock()
	defer r.promoQueuesByStageMu.Unlock()

	result := ctrl.Result{
		// Note: If there is a failure, controller runtime ignores this and uses
		// progressive backoff instead. So this value only prevents requeueing
		// a Promotion if THIS reconciliation succeeds.
		RequeueAfter: 0,
	}

	logger := logging.LoggerFromContext(ctx)

	// Note that initialization occurs here because we basically know that the
	// controller runtime client's cache is ready at this point. We cannot attempt
	// to list Promotions prior to that point.
	var err error
	r.initializeOnce.Do(func() {
		if err = r.initializeQueues(ctx); err == nil {
			logger.Debug(
				"initialized Stage-specific Promotion queues from list of " +
					"existing Promotions",
			)
		}
		// TODO: Do not hardcode this interval
		go r.serializedSync(ctx, 10*time.Second)
	})
	if err != nil {
		return result, errors.Wrap(err, "error initializing Promotion queues")
	}

	logger = logger.WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"promotion": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Promotion")

	// Find the Promotion
	promo, err := r.getPromo(ctx, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if promo == nil {
		// Ignore if not found. This can happen if the Promotion was deleted after
		// the current reconciliation request was issued.
		return result, nil
	}

	promo.Status = r.syncPromo(ctx, promo)

	updateErr := r.kargoClient.Status().Update(ctx, promo)
	if updateErr != nil {
		logger.Errorf("error updating Promotion status: %s", updateErr)
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return result, err
}

// initializeQueues lists all Promotions and adds them to relevant priority
// queues. This is intended to be invoked ONCE and the caller MUST ensure that.
// It is also assumed that the caller has already obtained a lock on
// promoQueuesByStageMu.
func (r *reconciler) initializeQueues(ctx context.Context) error {
	promos := api.PromotionList{}
	if err := r.kargoClient.List(ctx, &promos); err != nil {
		return errors.Wrap(err, "error listing promotions")
	}
	logger := logging.LoggerFromContext(ctx)
	for _, p := range promos.Items {
		promo := p // This is to sidestep implicit memory aliasing in this for loop
		switch promo.Status.Phase {
		case api.PromotionPhaseComplete, api.PromotionPhaseFailed:
			continue
		case "":
			promo.Status.Phase = api.PromotionPhasePending
			if err := r.kargoClient.Status().Update(ctx, &promo); err != nil {
				return errors.Wrapf(
					err,
					"error updating status of Promotion %q in namespace %q",
					promo.Name,
					promo.Namespace,
				)
			}
		}
		stage := types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		}
		pq, ok := r.promoQueuesByStage[stage]
		if !ok {
			pq = newPromotionsQueue()
			r.promoQueuesByStage[stage] = pq
		}
		// The only error that can occur here happens when you push a nil and we
		// know we're not doing that.
		pq.Push(&promo) // nolint: errcheck
		logger.WithFields(log.Fields{
			"promotion": promo.Name,
			"namespace": promo.Namespace,
			"stage":     promo.Spec.Stage,
			"phase":     promo.Status.Phase,
		}).Debug("pushed Promotion onto Stage-specific Promotion queue")
	}
	if logger.Logger.IsLevelEnabled(log.DebugLevel) {
		for stage, pq := range r.promoQueuesByStage {
			logger.WithFields(log.Fields{
				"stage":     stage.Name,
				"namespace": stage.Namespace,
				"depth":     pq.Depth(),
			}).Debug("Stage-specific Promotion queue initialized")
		}
	}
	return nil
}

// syncPromo enqueues Promotion requests to a Stage-specific priority queue. This
// functions assumes the caller has obtained a lock on promoQueuesByStageMu.
func (r *reconciler) syncPromo(
	ctx context.Context,
	promo *api.Promotion,
) api.PromotionStatus {
	status := *promo.Status.DeepCopy()

	// Only deal with brand new Promotions
	if promo.Status.Phase != "" {
		return status
	}

	promo.Status.Phase = api.PromotionPhasePending

	stage := types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Stage,
	}

	pq, ok := r.promoQueuesByStage[stage]
	if !ok {
		pq = newPromotionsQueue()
		r.promoQueuesByStage[stage] = pq
	}

	status.Phase = api.PromotionPhasePending

	// Ignore any errors from this operation. Errors can only occur when you
	// try to push a nil onto the queue and we know we're not doing that.
	pq.Push(promo) // nolint: errcheck

	logging.LoggerFromContext(ctx).WithField("depth", pq.Depth()).
		Infof("pushed Promotion %q to Queue for Stage %q in namespace %q ",
			promo.Name,
			promo.Spec.Stage,
			promo.Namespace,
		)

	return status
}

func (r *reconciler) serializedSync(
	ctx context.Context,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
		for _, pq := range r.promoQueuesByStage {
			if popped := pq.Pop(); popped != nil {
				promo := popped.(*api.Promotion) // nolint: forcetypeassert

				logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
					"promotion": promo.Name,
					"namespace": promo.Namespace,
				})

				// Refresh promo instead of working with something stale
				var err error
				if promo, err = r.getPromo(
					ctx,
					types.NamespacedName{
						Namespace: promo.Namespace,
						Name:      promo.Name,
					},
				); err != nil {
					logger.Error("error finding Promotion")
					continue
				}
				if promo == nil || promo.Status.Phase != api.PromotionPhasePending {
					continue
				}

				logger = logger.WithFields(log.Fields{
					"stage": promo.Spec.Stage,
					"state": promo.Spec.State,
				})
				logger.Debug("executing Promotion")

				promoCtx := logging.ContextWithLogger(ctx, logger)

				if err = r.promoteFn(
					promoCtx,
					promo.Spec.Stage,
					promo.Namespace,
					promo.Spec.State,
				); err != nil {
					promo.Status.Phase = api.PromotionPhaseFailed
					promo.Status.Error = err.Error()
					logger.Errorf("error executing Promotion: %s", err)
				} else {
					promo.Status.Phase = api.PromotionPhaseComplete
					promo.Status.Error = ""
				}

				if err = r.kargoClient.Status().Update(ctx, promo); err != nil {
					logger.Errorf("error updating Promotion status: %s", err)
				}

				if promo.Status.Phase == api.PromotionPhaseComplete && err == nil {
					logger.Debug("completed Promotion")
				}
			}
		}
	}
}

func (r *reconciler) promote(
	ctx context.Context,
	stageName string,
	stageNamespace string,
	stateID string,
) error {
	logger := logging.LoggerFromContext(ctx)

	stage, err := api.GetStage(
		ctx,
		r.kargoClient,
		types.NamespacedName{
			Namespace: stageNamespace,
			Name:      stageName,
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Stage %q in namespace %q",
			stageName,
			stageNamespace,
		)
	}
	if stage == nil {
		return errors.Errorf(
			"could not find Stage %q in namespace %q",
			stageName,
			stageNamespace,
		)
	}
	logger.Debug("found associated Stage")

	if currentState, ok :=
		stage.Status.History.Top(); ok && currentState.ID == stateID {
		logger.Debug("Stage is already in desired state")
		return nil
	}

	var targetStateIndex int
	var targetState *api.StageState
	for i, availableState := range stage.Status.AvailableStates {
		if availableState.ID == stateID {
			targetStateIndex = i
			targetState = availableState.DeepCopy()
			break
		}
	}
	if targetState == nil {
		return errors.Errorf(
			"target state %q not found among available states of Stage %q "+
				"in namespace %q",
			stateID,
			stageName,
			stageNamespace,
		)
	}

	nextState, err := r.applyPromotionMechanismsFn(
		ctx,
		stage.ObjectMeta,
		*stage.Spec.PromotionMechanisms,
		*targetState,
	)
	if err != nil {
		return err
	}
	stage.Status.CurrentState = &nextState
	stage.Status.AvailableStates[targetStateIndex] = nextState
	stage.Status.History.Push(nextState)

	err = r.kargoClient.Status().Update(ctx, stage)
	return errors.Wrapf(
		err,
		"error updating status of Stage %q in namespace %q",
		stageName,
		stageNamespace,
	)
}

// TODO: This function could use some tests
func (r *reconciler) applyPromotionMechanisms(
	ctx context.Context,
	stageMeta metav1.ObjectMeta,
	promoMechanisms api.PromotionMechanisms,
	newState api.StageState,
) (api.StageState, error) {
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("executing promotion mechanisms")
	var err error
	for _, gitRepoUpdate := range promoMechanisms.GitRepoUpdates {
		if gitRepoUpdate.Bookkeeper != nil {
			if newState, err = r.applyBookkeeperUpdate(
				ctx,
				stageMeta.Namespace,
				newState,
				gitRepoUpdate,
			); err != nil {
				return newState, errors.Wrap(err, "error promoting via Git")
			}
		} else {
			if newState, err = r.applyGitRepoUpdate(
				ctx,
				stageMeta.Namespace,
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
		if err = r.applyArgoCDAppUpdate(
			ctx,
			stageMeta,
			newState,
			argoCDAppUpdate,
		); err != nil {
			return newState, errors.Wrap(err, "error promoting via Argo CD")
		}
	}
	if len(promoMechanisms.ArgoCDAppUpdates) > 0 {
		logger.Debug("completed Argo CD-based promotion steps")
	}

	newState.Health = &api.Health{
		Status: api.HealthStateUnknown,
		Issues: []string{"Health has not yet been assessed"},
	}

	return newState, nil
}

// getPromo returns a pointer to the Promotion resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func (r *reconciler) getPromo(
	ctx context.Context,
	namespacedName types.NamespacedName,
) (*api.Promotion, error) {
	promo := api.Promotion{}
	if err := r.kargoClient.Get(ctx, namespacedName, &promo); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			logging.LoggerFromContext(ctx).WithFields(log.Fields{
				"namespace": namespacedName.Namespace,
				"promotion": namespacedName.Name,
			}).Warn("Promotion not found")
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Promotion %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &promo, nil
}
