package promotions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/directives"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	libEvent "github.com/akuity/kargo/internal/kubernetes/event"
	"github.com/akuity/kargo/internal/logging"
)

// ReconcilerConfig represents configuration for the promotion reconciler.
type ReconcilerConfig struct {
	ShardName string `envconfig:"SHARD_NAME"`
}

func (c ReconcilerConfig) Name() string {
	name := "promotion-controller"
	if c.ShardName != "" {
		return name + "-" + c.ShardName
	}
	return name
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	var cfg ReconcilerConfig
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler reconciles Promotion resources.
type reconciler struct {
	kargoClient      client.Client
	directivesEngine directives.Engine

	cfg ReconcilerConfig

	recorder record.EventRecorder

	pqs            *promoQueues
	initializeOnce sync.Once

	// The following behaviors are overridable for testing purposes:

	getStageFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Stage, error)

	promoteFn func(
		context.Context,
		kargoapi.Promotion,
		*kargoapi.Stage,
		*kargoapi.Freight,
	) (*kargoapi.PromotionStatus, error)
}

// SetupReconcilerWithManager initializes a reconciler for Promotion resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argocdMgr manager.Manager,
	directivesEngine directives.Engine,
	cfg ReconcilerConfig,
) error {
	// Index running Promotions by Argo CD Applications
	if err := indexer.IndexRunningPromotionsByArgoCDApplications(ctx, kargoMgr, cfg.ShardName); err != nil {
		return fmt.Errorf("index running Promotions by Argo CD Applications: %w", err)
	}

	shardPredicate, err := controller.GetShardPredicate(cfg.ShardName)
	if err != nil {
		return fmt.Errorf("error creating shard selector predicate: %w", err)
	}
	shardRequirement, err := controller.GetShardRequirement(cfg.ShardName)
	if err != nil {
		return fmt.Errorf("error creating shard requirement: %w", err)
	}
	shardSelector := labels.NewSelector().Add(*shardRequirement)

	reconciler := newReconciler(
		kargoMgr.GetClient(),
		libEvent.NewRecorder(ctx, kargoMgr.GetScheme(), kargoMgr.GetClient(), cfg.Name()),
		directivesEngine,
		cfg,
	)

	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Promotion{}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			kargo.RefreshRequested{},
		)).
		WithEventFilter(shardPredicate).
		WithOptions(controller.CommonOptions()).
		Build(reconciler)
	if err != nil {
		return fmt.Errorf("error building Promotion controller: %w", err)
	}

	logger := logging.LoggerFromContext(ctx)

	// If Argo CD integration is disabled, this manager will be nil and we won't
	// care about this watch anyway.
	if argocdMgr != nil {
		if err := c.Watch(
			source.Kind(
				argocdMgr.GetCache(),
				&argocd.Application{},
				&UpdatedArgoCDAppHandler[*argocd.Application]{
					kargoClient:   kargoMgr.GetClient(),
					shardSelector: shardSelector,
				},
				ArgoCDAppOperationCompleted[*argocd.Application]{
					logger: logger,
				},
			),
		); err != nil {
			return fmt.Errorf("unable to watch Applications: %w", err)
		}
	}

	// Watch Promotions that complete and enqueue the next highest promotion key
	priorityQueueHandler := &EnqueueHighestPriorityPromotionHandler[*kargoapi.Promotion]{
		ctx:         ctx,
		logger:      logger,
		kargoClient: reconciler.kargoClient,
		pqs:         reconciler.pqs,
	}
	promoWentTerminal := kargo.NewPromoWentTerminalPredicate(logger)
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Promotion{},
			priorityQueueHandler,
			promoWentTerminal,
		),
	); err != nil {
		return fmt.Errorf("unable to watch Promotions: %w", err)
	}

	return nil
}

func newReconciler(
	kargoClient client.Client,
	recorder record.EventRecorder,
	directivesEngine directives.Engine,
	cfg ReconcilerConfig,
) *reconciler {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	r := &reconciler{
		kargoClient:      kargoClient,
		directivesEngine: directivesEngine,
		recorder:         recorder,
		cfg:              cfg,
		pqs:              &pqs,
	}
	r.getStageFn = kargoapi.GetStage
	r.promoteFn = r.promote
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"namespace", req.NamespacedName.Namespace,
		"promotion", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Promotion")

	// Note that initialization occurs here because we basically know that the
	// controller runtime client's cache is ready at this point. We cannot attempt
	// to list Promotions prior to that point.
	var err error
	r.initializeOnce.Do(func() {
		promos := kargoapi.PromotionList{}
		if err = r.kargoClient.List(ctx, &promos); err != nil {
			err = fmt.Errorf("error listing promotions: %w", err)
		} else {
			r.pqs.initializeQueues(ctx, promos)
			logger.Debug(
				"initialized Stage-specific Promotion queues from list of existing Promotions",
			)
		}
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error initializing Promotion queues: %w", err)
	}

	// Find the Promotion
	promo, err := kargoapi.GetPromotion(ctx, r.kargoClient, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if promo == nil || promo.Status.Phase.IsTerminal() {
		// Ignore if not found or already finished. Promo might be nil if the
		// Promotion was deleted after the current reconciliation request was issued.
		return ctrl.Result{}, nil
	}
	// Find the Freight
	freight, err := kargoapi.GetFreight(ctx, r.kargoClient, types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Freight,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"error finding Freight %q in namespace %q: %w",
			promo.Spec.Freight,
			promo.Namespace,
			err,
		)
	}

	logger = logger.WithValues(
		"namespace", req.NamespacedName.Namespace,
		"promotion", req.NamespacedName.Name,
		"stage", promo.Spec.Stage,
		"freight", promo.Spec.Freight,
	)

	if promo.Status.Phase == kargoapi.PromotionPhaseRunning {
		// anything we've already marked Running, we allow it to continue to reconcile
		logger.Debug("continuing Promotion")
	} else {
		// promo is Pending. Try to begin it.
		if !r.pqs.tryBegin(ctx, promo) {
			// It wasn't our turn. Mark this promo as Pending (if it wasn't already)
			if promo.Status.Phase != kargoapi.PromotionPhasePending {
				err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
					status.Phase = kargoapi.PromotionPhasePending
				})
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Info("began promotion")
	}

	// Update promo status as Running to give visibility in UI. Also, a promo which
	// has already entered Running status will be allowed to continue to reconcile.
	if promo.Status.Phase != kargoapi.PromotionPhaseRunning {
		if err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
			status.Phase = kargoapi.PromotionPhaseRunning
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Retrieve the Stage associated with the Promotion.
	stage, err := r.getStageFn(
		ctx,
		r.kargoClient,
		types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		},
	)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"error finding Stage %q in namespace %q: %w",
			promo.Spec.Stage, promo.Namespace, err,
		)
	}
	if stage == nil {
		return ctrl.Result{}, fmt.Errorf(
			"could not find Stage %q in namespace %q",
			promo.Spec.Stage, promo.Namespace,
		)
	}
	logger.Debug("found associated Stage")

	// Confirm that the Stage is awaiting this Promotion.
	//
	// This is a temporary measure to ensure that the Promotion is only
	// allowed to proceed if the Stage is expecting it. This is necessary
	// to ensure we can derive Freight from the previous Promotion in the
	// Stage's status to construct the Freight collection for the current
	// Promotion.
	//
	// TODO(hidde): This adds tight coupling between the Promotion and the
	// Stage (again, but without patching the Stage this time). We should
	// explore a more loosely-coupled approach, perhaps by making the
	// Freight self-aware of the Stages it has been promoted to, or even
	// more radically, by making the Promotion self-aware of the Freight
	// collection it is promoting.
	if stage.Status.CurrentPromotion == nil || stage.Status.CurrentPromotion.Name != promo.Name {
		logger.Debug("Stage is not awaiting Promotion", "stage", stage.Name, "promotion", promo.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	promoCtx := logging.ContextWithLogger(ctx, logger)

	newStatus := promo.Status.DeepCopy()

	// Wrap the promoteFn() call in an anonymous function to recover() any panics, so
	// we can update the promo's phase with Error if it does. This breaks an infinite
	// cycle of a bad promo continuously failing to reconcile, and surfaces the error.
	func() {
		defer func() {
			if err := recover(); err != nil {
				if theErr, ok := err.(error); ok {
					logger.Error(theErr, "Promotion panic")
				} else {
					logger.Error(nil, "Promotion panic")
				}
				newStatus.Phase = kargoapi.PromotionPhaseErrored
				newStatus.Message = fmt.Sprintf("%v", err)
			}
		}()
		otherStatus, promoteErr := r.promoteFn(
			promoCtx,
			*promo,
			stage,
			freight,
		)
		if otherStatus != nil {
			newStatus = otherStatus
		}
		if promoteErr != nil {
			newStatus.Phase = kargoapi.PromotionPhaseErrored
			newStatus.Message = promoteErr.Error()
			logger.Error(promoteErr, "error executing Promotion")
		}
	}()

	if newStatus.Phase.IsTerminal() {
		newStatus.FinishedAt = &metav1.Time{Time: time.Now()}
		logger.Info("promotion", "phase", newStatus.Phase)
	}

	// Record the current refresh token as having been handled.
	if token, ok := kargoapi.RefreshAnnotationValue(promo.GetAnnotations()); ok {
		newStatus.LastHandledRefresh = token
	}

	if err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
		*status = *newStatus
	}); err != nil {
		logger.Error(err, "error updating Promotion status")

		if apierrors.IsInvalid(err) {
			// If the error is due to an invalid status update, we should mark
			// the Promotion as errored to prevent it from being requeued.
			//
			// NB: This should be a rare occurrence, and is either due to the
			// CustomResourceDefinition being out of sync with the controller
			// version, or us inventing non-backwards-compatible changes.
			err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
				status.Phase = kargoapi.PromotionPhaseErrored
				status.Message = fmt.Sprintf("error updating status: %v", err)
			})
		}
	}

	// Record event after patching status if new phase is terminal
	if newStatus.Phase.IsTerminal() {
		stage, getStageErr := r.getStageFn(
			ctx,
			r.kargoClient,
			types.NamespacedName{
				Namespace: promo.Namespace,
				Name:      promo.Spec.Stage,
			},
		)
		if getStageErr != nil {
			return ctrl.Result{}, fmt.Errorf("get stage: %w", err)
		}
		if stage == nil {
			return ctrl.Result{}, fmt.Errorf(
				"stage %q not found in namespace %q",
				promo.Spec.Stage,
				promo.Namespace,
			)
		}

		var reason string
		switch newStatus.Phase {
		case kargoapi.PromotionPhaseSucceeded:
			reason = kargoapi.EventReasonPromotionSucceeded
		case kargoapi.PromotionPhaseFailed:
			reason = kargoapi.EventReasonPromotionFailed
		case kargoapi.PromotionPhaseErrored:
			reason = kargoapi.EventReasonPromotionErrored
		}

		msg := fmt.Sprintf("Promotion %s", newStatus.Phase)
		if newStatus.Message != "" {
			msg += fmt.Sprintf(": %s", newStatus.Message)
		}

		eventAnnotations := kargoapi.NewPromotionEventAnnotations(ctx,
			kargoapi.FormatEventControllerActor(r.cfg.Name()),
			promo, freight)

		if newStatus.Phase == kargoapi.PromotionPhaseSucceeded {
			eventAnnotations[kargoapi.AnnotationKeyEventVerificationPending] =
				strconv.FormatBool(stage.Spec.Verification != nil)
		}
		r.recorder.AnnotatedEventf(promo, eventAnnotations, corev1.EventTypeNormal, reason, msg)
	}

	if err != nil {
		// Controller runtime automatically gives us a progressive backoff if err is
		// not nil
		return ctrl.Result{}, err
	}

	// If the promotion is still running, we'll need to periodically check on
	// it.
	//
	// TODO: Make this configurable
	if newStatus.Phase == kargoapi.PromotionPhaseRunning {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) promote(
	ctx context.Context,
	promo kargoapi.Promotion,
	stage *kargoapi.Stage,
	targetFreight *kargoapi.Freight,
) (*kargoapi.PromotionStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	stageName := stage.Name
	stageNamespace := promo.Namespace

	if targetFreight == nil {
		return nil, fmt.Errorf("Freight %q not found in namespace %q", promo.Spec.Freight, promo.Namespace)
	}
	var upstreams []string
	for _, req := range stage.Spec.RequestedFreight {
		upstreams = append(upstreams, req.Sources.Stages...)
	}
	// De-dupe upstreams
	slices.Sort(upstreams)
	upstreams = slices.Compact(upstreams)

	if !kargoapi.IsFreightAvailable(targetFreight, stageName, upstreams) {
		return nil, fmt.Errorf(
			"Freight %q is not available to Stage %q in namespace %q",
			promo.Spec.Freight,
			stageName,
			stageNamespace,
		)
	}

	logger = logger.WithValues("targetFreight", targetFreight.Name)

	targetFreightRef := kargoapi.FreightReference{
		Name:    targetFreight.Name,
		Commits: targetFreight.Commits,
		Images:  targetFreight.Images,
		Charts:  targetFreight.Charts,
		Origin:  targetFreight.Origin,
	}

	// Make a deep copy of the Promotion to pass to the promotion steps execution
	// engine, which may modify its status.
	workingPromo := promo.DeepCopy()
	workingPromo.Status.Freight = &targetFreightRef
	workingPromo.Status.FreightCollection = r.buildTargetFreightCollection(
		ctx,
		targetFreightRef,
		stage,
	)

	// If the Promotion has steps, execute them in sequence.
	var steps []directives.PromotionStep
	for _, step := range workingPromo.Spec.Steps {
		steps = append(steps, directives.PromotionStep{
			Kind:   step.Uses,
			Alias:  step.As,
			Config: step.GetConfig(),
		})
	}

	promoCtx := directives.PromotionContext{
		WorkDir:         filepath.Join(os.TempDir(), "promotion-"+string(workingPromo.UID)),
		Project:         stageNamespace,
		Stage:           stageName,
		FreightRequests: stage.Spec.RequestedFreight,
		Freight:         *workingPromo.Status.FreightCollection.DeepCopy(),
		StartFromStep:   promo.Status.CurrentStep,
		State:           directives.State(workingPromo.Status.GetState()),
	}
	if err := os.Mkdir(promoCtx.WorkDir, 0o700); err == nil {
		// If we're working with a fresh directory, we should start the promotion
		// process again from the beginning.
		promoCtx.StartFromStep = 0
		promoCtx.State = nil
	} else if !os.IsExist(err) {
		return nil, fmt.Errorf("error creating working directory: %w", err)
	}
	defer func() {
		if workingPromo.Status.Phase.IsTerminal() {
			if err := os.RemoveAll(promoCtx.WorkDir); err != nil {
				logger.Error(err, "could not remove working directory")
			}
		}
	}()

	res, err := r.directivesEngine.Promote(ctx, promoCtx, steps)
	workingPromo.Status.Phase = res.Status
	workingPromo.Status.Message = res.Message
	workingPromo.Status.CurrentStep = res.CurrentStep
	workingPromo.Status.State = &apiextensionsv1.JSON{Raw: res.State.ToJSON()}
	if res.Status == kargoapi.PromotionPhaseSucceeded {
		var healthChecks []kargoapi.HealthCheckStep
		for _, step := range res.HealthCheckSteps {
			healthChecks = append(healthChecks, kargoapi.HealthCheckStep{
				Uses:   step.Kind,
				Config: &apiextensionsv1.JSON{Raw: step.Config.ToJSON()},
			})
		}
		workingPromo.Status.HealthChecks = healthChecks
	}
	if err != nil {
		workingPromo.Status.Phase = kargoapi.PromotionPhaseErrored
		return &workingPromo.Status, err
	}

	logger.Debug("promotion", "phase", workingPromo.Status.Phase)

	if workingPromo.Status.Phase == kargoapi.PromotionPhaseSucceeded {
		// Trigger re-verification of the Stage if the promotion succeeded and
		// this is a re-promotion of the same Freight.
		current := stage.Status.FreightHistory.Current()
		if current != nil && current.VerificationHistory.Current() != nil {
			for _, f := range current.Freight {
				if f.Name == targetFreight.Name {
					if err := kargoapi.ReverifyStageFreight(
						ctx,
						r.kargoClient,
						types.NamespacedName{
							Namespace: stageNamespace,
							Name:      stageName,
						},
					); err != nil {
						// Log the error, but don't let failure to initiate re-verification
						// prevent the promotion from succeeding.
						logger.Error(err, "error triggering re-verification")
					}
					break
				}
			}
		}
	}

	return &workingPromo.Status, nil
}

// buildTargetFreightCollection constructs a FreightCollection that contains all
// FreightReferences from the previous Promotion (excepting those that are no
// longer requested), plus a FreightReference for the provided targetFreight.
func (r *reconciler) buildTargetFreightCollection(
	ctx context.Context,
	targetFreight kargoapi.FreightReference,
	stage *kargoapi.Stage,
) *kargoapi.FreightCollection {
	logger := logging.LoggerFromContext(ctx)
	freightCol := &kargoapi.FreightCollection{}

	// We don't simply copy the current FreightCollection because we want to
	// account for the possibility that some freight contained therein are no
	// longer requested by the Stage.
	if len(stage.Spec.RequestedFreight) > 1 {
		lastPromo := stage.Status.LastPromotion
		if lastPromo.Status != nil && lastPromo.Status.FreightCollection != nil {
			for _, req := range stage.Spec.RequestedFreight {
				if freight, ok := lastPromo.Status.FreightCollection.Freight[req.Origin.String()]; ok {
					freightCol.UpdateOrPush(freight)
				}
			}
		} else {
			logger.Debug("last promotion has no collection to inherit Freight from")
		}
	}
	freightCol.UpdateOrPush(targetFreight)
	return freightCol
}
