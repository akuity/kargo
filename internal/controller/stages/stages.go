package stages

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Stage resources.
type reconciler struct {
	kargoClient    client.Client
	argocdClient   client.Client
	rolloutsClient client.Client

	shardName string

	// The following behaviors are overridable for testing purposes:

	// Loop guard:

	hasNonTerminalPromotionsFn func(
		ctx context.Context,
		stageNamespace string,
		stageName string,
	) (bool, error)

	listPromosFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	// Health checks:

	checkHealthFn func(
		context.Context,
		kargoapi.SimpleFreight,
		[]kargoapi.ArgoCDAppUpdate,
	) *kargoapi.Health

	getArgoCDAppFn func(
		ctx context.Context,
		client client.Client,
		namespace string,
		name string,
	) (*argocd.Application, error)

	// Freight verification:

	startVerificationFn func(
		context.Context,
		*kargoapi.Stage,
	) (*kargoapi.VerificationInfo, error)

	getVerificationInfoFn func(
		context.Context,
		*kargoapi.Stage,
	) (*kargoapi.VerificationInfo, error)

	getAnalysisTemplateFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*rollouts.AnalysisTemplate, error)

	listAnalysisRunsFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	buildAnalysisRunFn func(
		stage *kargoapi.Stage,
		templates []*rollouts.AnalysisTemplate,
	) (*rollouts.AnalysisRun, error)

	createAnalysisRunFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	getAnalysisRunFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*rollouts.AnalysisRun, error)

	getFreightFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Freight, error)

	verifyFreightInStageFn func(
		ctx context.Context,
		namespace string,
		freightName string,
		stageName string,
	) error

	patchFreightStatusFn func(
		ctx context.Context,
		freight *kargoapi.Freight,
		newStatus kargoapi.FreightStatus,
	) error

	// Auto-promotion:

	isAutoPromotionPermittedFn func(
		ctx context.Context,
		namespace string,
		stageName string,
	) (bool, error)

	listPromoPoliciesFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	createPromotionFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	// Discovering latest Freight:

	getLatestAvailableFreightFn func(
		ctx context.Context,
		namespace string,
		stage *kargoapi.Stage,
	) (*kargoapi.Freight, error)

	getLatestFreightFromWarehouseFn func(
		ctx context.Context,
		namespace string,
		warehouse string,
	) (*kargoapi.Freight, error)

	getAllVerifiedFreightFn func(
		ctx context.Context,
		namespace string,
		stageSubs []kargoapi.StageSubscription,
	) ([]kargoapi.Freight, error)

	getLatestVerifiedFreightFn func(
		ctx context.Context,
		namespace string,
		stageSubs []kargoapi.StageSubscription,
	) (*kargoapi.Freight, error)

	getLatestApprovedFreightFn func(
		ctx context.Context,
		namespace string,
		name string,
	) (*kargoapi.Freight, error)

	listFreightFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error
}

// SetupReconcilerWithManager initializes a reconciler for Stage resources and
// registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argocdMgr manager.Manager,
	rolloutsMgr manager.Manager,
	shardName string,
) error {
	// Index Promotions in non-terminal states by Stage
	if err := kubeclient.IndexNonTerminalPromotionsByStage(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index non-terminal Promotions by Stage")
	}

	// Index Promotions by Stage + Freight
	if err := kubeclient.IndexPromotionsByStageAndFreight(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index Promotions by Stage and Freight")
	}

	// Index PromotionPolicies by Stage
	if err := kubeclient.IndexPromotionPoliciesByStage(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index PromotionPolicies by Stage")
	}

	// Index Freight by Warehouse
	if err := kubeclient.IndexFreightByWarehouse(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index Freight by Warehouse")
	}

	// Index Freight by Stages in which it has been verified
	if err :=
		kubeclient.IndexFreightByVerifiedStages(ctx, kargoMgr); err != nil {
		return errors.Wrap(
			err,
			"index Freight by Stages in which it has been verified",
		)
	}

	// Index Freight by Stages for which it has been approved
	if err :=
		kubeclient.IndexFreightByApprovedStages(ctx, kargoMgr); err != nil {
		return errors.Wrap(
			err,
			"index Freight by Stages for which it has been approved",
		)
	}

	// Index Stages by upstream Stages
	if err :=
		kubeclient.IndexStagesByUpstreamStages(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index Stages by upstream Stages")
	}

	// Index Stages by Warehouse
	if err := kubeclient.IndexStagesByWarehouse(ctx, kargoMgr); err != nil {
		return errors.Wrap(err, "index Stages by Warehouse")
	}

	shardPredicate, err := controller.GetShardPredicate(shardName)
	if err != nil {
		return errors.Wrap(err, "error creating shard predicate")
	}

	c, err := ctrl.NewControllerManagedBy(kargoMgr).
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
		WithEventFilter(kargo.IgnoreClearRefreshUpdates{}).
		WithOptions(controller.CommonOptions()).
		Build(
			newReconciler(
				kargoMgr.GetClient(),
				argocdMgr.GetClient(),
				rolloutsMgr.GetClient(),
				shardName,
			),
		)
	if err != nil {
		return errors.Wrap(err, "error building Stage reconciler")
	}

	logger := logging.LoggerFromContext(ctx)
	// Watch Promotions that completed and enqueue owning Stage key
	promoOwnerHandler := handler.EnqueueRequestForOwner(
		kargoMgr.GetScheme(),
		kargoMgr.GetRESTMapper(),
		&kargoapi.Stage{},
		handler.OnlyControllerOwner(),
	)
	promoWentTerminal := kargo.NewPromoWentTerminalPredicate(logger)
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Promotion{},
		),
		promoOwnerHandler,
		promoWentTerminal,
	); err != nil {
		return errors.Wrap(err, "unable to watch Promotions")
	}

	// Watch Freight that has been marked as verified in a Stage and enqueue
	// downstream Stages
	verifiedFreightHandler := &verifiedFreightEventHandler{
		kargoClient: kargoMgr.GetClient(),
	}
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
		),
		verifiedFreightHandler,
	); err != nil {
		return errors.Wrap(err, "unable to watch Freight")
	}

	approveFreightHandler := &approvedFreightEventHandler{
		kargoClient: kargoMgr.GetClient(),
	}
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
		),
		approveFreightHandler,
	); err != nil {
		return errors.Wrap(err, "unable to watch Freight")
	}

	createdFreightEventHandler := &createdFreightEventHandler{
		kargoClient: kargoMgr.GetClient(),
	}
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
		),
		createdFreightEventHandler,
	); err != nil {
		return errors.Wrap(err, "unable to watch Freight")
	}

	return nil
}

func newReconciler(
	kargoClient client.Client,
	argocdClient client.Client,
	rolloutsClient client.Client,
	shardName string,
) *reconciler {
	r := &reconciler{
		kargoClient:    kargoClient,
		argocdClient:   argocdClient,
		rolloutsClient: rolloutsClient,
		shardName:      shardName,
	}
	// The following default behaviors are overridable for testing purposes:
	// Loop guard:
	r.hasNonTerminalPromotionsFn = r.hasNonTerminalPromotions
	r.listPromosFn = r.kargoClient.List
	// Health checks:
	r.checkHealthFn = r.checkHealth
	r.getArgoCDAppFn = argocd.GetApplication
	// Freight verification:
	r.startVerificationFn = r.startVerification
	r.getVerificationInfoFn = r.getVerificationInfo
	r.getAnalysisTemplateFn = rollouts.GetAnalysisTemplate
	r.listAnalysisRunsFn = r.kargoClient.List
	r.buildAnalysisRunFn = r.buildAnalysisRun
	r.createAnalysisRunFn = r.rolloutsClient.Create
	r.getAnalysisRunFn = rollouts.GetAnalysisRun
	r.getFreightFn = kargoapi.GetFreight
	r.verifyFreightInStageFn = r.verifyFreightInStage
	r.patchFreightStatusFn = r.patchFreightStatus
	// Auto-promotion:
	r.isAutoPromotionPermittedFn = r.isAutoPromotionPermitted
	r.listPromoPoliciesFn = r.kargoClient.List
	r.createPromotionFn = kargoClient.Create
	// Discovering latest Freight:
	r.getLatestAvailableFreightFn = r.getLatestAvailableFreight
	r.getLatestFreightFromWarehouseFn = r.getLatestFreightFromWarehouse
	r.getAllVerifiedFreightFn = r.getAllVerifiedFreight
	r.getLatestVerifiedFreightFn = r.getLatestVerifiedFreight
	r.getLatestApprovedFreightFn = r.getLatestApprovedFreight
	r.listFreightFn = r.kargoClient.List
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
	if stage.Spec.PromotionMechanisms == nil {
		newStatus, err = r.syncControlFlowStage(ctx, stage)
	} else {
		newStatus, err = r.syncNormalStage(ctx, stage)
	}
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

func (r *reconciler) syncControlFlowStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	status := *stage.Status.DeepCopy()
	status.ObservedGeneration = stage.Generation
	status.Health = nil // Reset health
	status.Phase = kargoapi.StagePhaseNotApplicable
	status.CurrentPromotion = nil

	// A Stage without promotion mechanisms shouldn't have a currentFreight. Make
	// sure this is empty to avoid confusion. A reason this could be non-empty to
	// begin with is that the Stage USED TO have promotion mechanisms, but they
	// were removed, thus becoming a control flow Stage.
	status.CurrentFreight = nil

	// For now all Freight verified in any upstream Stage(s) should automatically
	// and immediately be verified in this Stage, making it available downstream.
	// In the future, we may have more options before marking Freight as verified
	// in a control flow Stage (e.g. require that it was verified in ALL upstreams
	// Stages)
	var availableFreight []kargoapi.Freight
	if stage.Spec.Subscriptions.Warehouse != "" {
		var freight kargoapi.FreightList
		if err := r.listFreightFn(
			ctx,
			&freight,
			&client.ListOptions{
				Namespace: stage.Namespace,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.FreightByWarehouseIndexField,
					stage.Spec.Subscriptions.Warehouse,
				),
			},
		); err != nil {
			return status, errors.Wrapf(
				err,
				"error listing Freight from Warehouse %q in namespace %q",
				stage.Spec.Subscriptions.Warehouse,
				stage.Namespace,
			)
		}
		availableFreight = freight.Items
	} else {
		// Get all Freight verified in upstream Stages. Merely being approved for an
		// upstream Stage is not enough. If Freight is only approved for a Stage,
		// that is because someone manually did that. This does not speak to its
		// suitability for promotion downstream. Expect a nil if the specified
		// Freight is not found or doesn't meet these conditions. Errors are
		// indicative only of internal problems.
		var err error
		if availableFreight, err = r.getAllVerifiedFreightFn(
			ctx,
			stage.Namespace,
			stage.Spec.Subscriptions.UpstreamStages,
		); err != nil {
			return status, errors.Wrapf(
				err,
				"error getting all Freight verified in Stages upstream from Stage "+
					"%q in namespace %q",
				stage.Name,
				stage.Namespace,
			)
		}
	}
	for _, available := range availableFreight {
		af := available // Avoid implicit memory aliasing
		// Only bother to mark as verified in this Stage if not already the case.
		if _, verified := af.Status.VerifiedIn[stage.Name]; !verified {
			newStatus := *af.Status.DeepCopy()
			if newStatus.VerifiedIn == nil {
				newStatus.VerifiedIn = map[string]kargoapi.VerifiedStage{}
			}
			newStatus.VerifiedIn[stage.Name] = kargoapi.VerifiedStage{}
			if err := r.patchFreightStatusFn(ctx, &af, newStatus); err != nil {
				return status, errors.Wrapf(
					err,
					"error marking Freight %q in namespace %q as verified in Stage %q",
					af.ID,
					stage.Namespace,
					stage.Name,
				)
			}
		}
	}
	return status, nil
}

func (r *reconciler) syncNormalStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	status := *stage.Status.DeepCopy()

	logger := logging.LoggerFromContext(ctx)

	// Skip the entire reconciliation loop if there are Promotions associate with
	// this Stage in a non-terminal state. The promotion process and this
	// reconciliation loop BOTH update Stage status, so this check helps us
	// to avoid race conditions that may otherwise arise.
	if hasNonTerminalPromos, err := r.hasNonTerminalPromotionsFn(
		ctx,
		stage.Namespace,
		stage.Name,
	); err != nil {
		return status, err
	} else if hasNonTerminalPromos {
		logger.Debug(
			"Stage has one or more Promotions in a non-terminal phase; skipping " +
				"this reconciliation loop",
		)
		return status, nil
	}

	status.ObservedGeneration = stage.Generation
	status.Health = nil // Reset health
	status.CurrentPromotion = nil

	if status.CurrentFreight == nil {
		status.Phase = kargoapi.StagePhaseNotApplicable
		logger.Debug(
			"Stage has no current Freight; no health checks or verification to perform",
		)
	} else {
		freightLogger := logger.WithField("freight", status.CurrentFreight.ID)

		// Check health
		status.Health = r.checkHealthFn(
			ctx,
			*status.CurrentFreight,
			stage.Spec.PromotionMechanisms.ArgoCDAppUpdates,
		)
		if status.Health != nil {
			freightLogger.WithField("health", status.Health.Status).
				Debug("Stage health assessed")
		} else {
			freightLogger.Debug("Stage health deemed not applicable")
		}

		// If the Stage is healthy and no verification process is defined, then the
		// Stage should transition to the Steady phase.
		if (status.Health == nil || status.Health.Status == kargoapi.HealthStateHealthy) &&
			stage.Spec.Verification == nil && status.Phase == kargoapi.StagePhaseVerifying {
			status.Phase = kargoapi.StagePhaseSteady
		}

		// Initiate or follow-up on verification if required
		if status.Phase == kargoapi.StagePhaseVerifying && stage.Spec.Verification != nil {
			if status.CurrentFreight.VerificationInfo == nil {
				if status.Health == nil || status.Health.Status == kargoapi.HealthStateHealthy {
					// Start verification
					verInfo, err := r.startVerificationFn(ctx, stage)
					if err != nil {
						return status, errors.Wrapf(
							err,
							"error starting verification process for Stage %q and Freight %q in namespace %q",
							stage.Name,
							status.CurrentFreight.ID,
							stage.Namespace,
						)
					}
					status.CurrentFreight.VerificationInfo = verInfo
				}
			} else {
				log.Debug("checking verification results")
				verInfo, err := r.getVerificationInfoFn(ctx, stage)
				if err != nil {
					return status, errors.Wrapf(
						err,
						"error getting verification result for Stage %q and Freight %q in namespace %q",
						stage.Name,
						status.CurrentFreight.ID,
						stage.Namespace,
					)
				}
				status.CurrentFreight.VerificationInfo = verInfo
				switch rollouts.AnalysisPhase(status.CurrentFreight.VerificationInfo.AnalysisRun.Phase) {
				case rollouts.AnalysisPhasePending:
					log.Debug("verification is pending")
				case rollouts.AnalysisPhaseRunning:
					log.Debug("verification is running")
				case rollouts.AnalysisPhaseSuccessful,
					rollouts.AnalysisPhaseFailed,
					rollouts.AnalysisPhaseError,
					rollouts.AnalysisPhaseInconclusive:
					// Verification is complete
					status.Phase = kargoapi.StagePhaseSteady
					log.Debug("verification is complete")
				}
			}
		}

		// If health is not applicable or healthy
		// AND
		// Verification is not applicable or successful
		// THEN
		// Mark the Freight as verified in this Stage
		if (status.Health == nil || status.Health.Status == kargoapi.HealthStateHealthy) &&
			(stage.Spec.Verification == nil ||
				(status.CurrentFreight.VerificationInfo != nil &&
					status.CurrentFreight.VerificationInfo.AnalysisRun.Phase == string(rollouts.AnalysisPhaseSuccessful))) {
			if err := r.verifyFreightInStageFn(
				ctx,
				stage.Namespace,
				status.CurrentFreight.ID,
				stage.Name,
			); err != nil {
				return status, errors.Wrapf(
					err,
					"error marking Freight %q in namespace %q as verified in Stage %q",
					status.CurrentFreight.ID,
					stage.Namespace,
					stage.Name,
				)
			}
		}
	}

	// Stop here if we have no chance of finding any Freight to promote.
	if stage.Spec.Subscriptions == nil ||
		(stage.Spec.Subscriptions.Warehouse == "" && len(stage.Spec.Subscriptions.UpstreamStages) == 0) {
		logger.Warn(
			"Stage has no subscriptions. This may indicate an issue with resource" +
				"validation logic.",
		)
		return status, nil
	}

	logger.Debug("checking if auto-promotion is permitted...")
	if permitted, err :=
		r.isAutoPromotionPermittedFn(ctx, stage.Namespace, stage.Name); err != nil {
		return status, errors.Wrapf(
			err,
			"error checking if auto-promotion is permitted for Stage %q in "+
				"namespace %q",
			stage.Name,
			stage.Namespace,
		)
	} else if !permitted {
		logger.Debug("auto-promotion is not permitted for the Stage")
		return status, nil
	}

	// If we get to here, auto-promotion is permitted. Time to go looking for new
	// Freight...

	latestFreight, err :=
		r.getLatestAvailableFreightFn(ctx, stage.Namespace, stage)
	if err != nil {
		return status, errors.Wrapf(
			err,
			"error finding latest Freight for Stage %q in namespace %q",
			stage.Name,
			stage.Namespace,
		)
	}

	if latestFreight == nil {
		logger.Debug("no Freight found")
		return status, nil
	}

	logger = logger.WithField("freight", latestFreight.Name)

	// Only proceed if nextFreight isn't the one we already have
	if stage.Status.CurrentFreight != nil &&
		stage.Status.CurrentFreight.ID == latestFreight.Name {
		logger.Debug("Stage already has latest available Freight")
		return status, nil
	}

	// If a promotion already exists for this Stage + Freight, then we're
	// disqualified from auto-promotion.
	promos := kargoapi.PromotionList{}
	if err := r.listPromosFn(
		ctx,
		&promos,
		&client.ListOptions{
			Namespace: stage.Namespace,
			FieldSelector: fields.Set(
				map[string]string{
					kubeclient.PromotionsByStageAndFreightIndexField: kubeclient.
						StageAndFreightKey(stage.Name, latestFreight.Name),
				},
			).AsSelector(),
		},
	); err != nil {
		return status, errors.Wrapf(
			err,
			"error listing existing Promotions for Freight %q in namespace %q",
			latestFreight.Name,
			stage.Namespace,
		)
	}

	if len(promos.Items) > 0 {
		logger.Debug("Promotion already exists for Freight")
		return status, nil
	}

	logger.Debug("auto-promotion will proceed")

	promo := kargo.NewPromotion(*stage, latestFreight.ID)
	if err :=
		r.createPromotionFn(ctx, &promo, &client.CreateOptions{}); err != nil {
		return status, errors.Wrapf(
			err,
			"error creating Promotion of Stage %q in namespace %q to Freight %q",
			stage.Name,
			stage.Namespace,
			latestFreight.Name,
		)
	}
	logger.WithField("promotion", promo.Name).Debug("created Promotion resource")

	return status, nil
}

func (r *reconciler) hasNonTerminalPromotions(
	ctx context.Context,
	stageNamespace string,
	stageName string,
) (bool, error) {
	promos := kargoapi.PromotionList{}
	if err := r.listPromosFn(
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

func (r *reconciler) verifyFreightInStage(
	ctx context.Context,
	namespace string,
	freightName string,
	stageName string,
) error {
	logger := logging.LoggerFromContext(ctx).WithField("freight", freightName)

	// Find the Freight
	freight, err := r.getFreightFn(
		ctx,
		r.kargoClient,
		types.NamespacedName{
			Namespace: namespace,
			Name:      freightName,
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Freight %q in namespace %q",
			freightName,
			namespace,
		)
	}
	if freight == nil {
		return errors.Errorf(
			"found no Freight %q in namespace %q",
			freightName,
			namespace,
		)
	}

	newStatus := *freight.Status.DeepCopy()
	if newStatus.VerifiedIn == nil {
		newStatus.VerifiedIn = map[string]kargoapi.VerifiedStage{}
	}

	// Only try to mark as verified in this Stage if not already the case.
	if _, ok := newStatus.VerifiedIn[stageName]; ok {
		logger.Debug("Freight already marked as verified in Stage")
		return nil
	}

	newStatus.VerifiedIn[stageName] = kargoapi.VerifiedStage{}
	if err = r.patchFreightStatusFn(ctx, freight, newStatus); err != nil {
		return err
	}

	logger.Debug("marked Freight as verified in Stage")
	return nil
}

func (r *reconciler) patchFreightStatus(
	ctx context.Context,
	freight *kargoapi.Freight,
	newStatus kargoapi.FreightStatus,
) error {
	err := kubeclient.PatchStatus(
		ctx,
		r.kargoClient,
		freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	)
	return errors.Wrapf(
		err,
		"error patching Freight %q status in namespace %q",
		freight.Name,
		freight.Namespace,
	)
}

func (r *reconciler) isAutoPromotionPermitted(
	ctx context.Context,
	namespace string,
	stageName string,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)
	policies := kargoapi.PromotionPolicyList{}
	if err := r.listPromoPoliciesFn(
		ctx,
		&policies,
		&client.ListOptions{
			Namespace: namespace,
			FieldSelector: fields.Set(map[string]string{
				kubeclient.PromotionPoliciesByStageIndexField: stageName,
			}).AsSelector(),
		},
	); err != nil {
		return false, errors.Wrapf(
			err,
			"error listing PromotionPolicies for Stage %q in namespace %q",
			stageName,
			namespace,
		)
	}
	if len(policies.Items) == 0 {
		logger.Debug("no PromotionPolicy is associated with the Stage")
		return false, nil
	}
	if len(policies.Items) > 1 {
		logger.Debug("multiple PromotionPolicies are associated with the Stage")
		return false, nil
	}
	if !policies.Items[0].EnableAutoPromotion {
		logger.Debug(
			"PromotionPolicy does not enable auto-promotion for the Stage",
		)
		return false, nil
	}
	return true, nil
}

func (r *reconciler) getLatestAvailableFreight(
	ctx context.Context,
	namespace string,
	stage *kargoapi.Stage,
) (*kargoapi.Freight, error) {
	logger := logging.LoggerFromContext(ctx)

	if stage.Spec.Subscriptions.Warehouse != "" {
		latestFreight, err := r.getLatestFreightFromWarehouseFn(
			ctx,
			namespace,
			stage.Spec.Subscriptions.Warehouse,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"error checking Warehouse %q in namespace %q for Freight",
				stage.Spec.Subscriptions.Warehouse,
				namespace,
			)
		}
		if latestFreight == nil {
			logger.WithField("warehouse", stage.Spec.Subscriptions.Warehouse).
				Debug("no Freight found from Warehouse")
		}
		return latestFreight, nil
	}

	latestVerifiedFreight, err := r.getLatestVerifiedFreightFn(
		ctx,
		namespace,
		stage.Spec.Subscriptions.UpstreamStages,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error finding latest Freight verified in Stages upstream from "+
				"Stage %q in namespace %q",
			stage.Name,
			namespace,
		)
	}
	if latestVerifiedFreight == nil {
		logger.Debug("no verified Freight found upstream from Stage")
	}

	latestApprovedFreight, err := r.getLatestApprovedFreightFn(
		ctx,
		namespace,
		stage.Name,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"error finding latest Freight approved for Stage %q in namespace %q",
			stage.Name,
			namespace,
		)
	}
	if latestVerifiedFreight == nil {
		logger.Debug("no approved Freight found for Stage")
	}

	if latestVerifiedFreight == nil && latestApprovedFreight == nil {
		return nil, nil
	}
	if latestVerifiedFreight != nil && latestApprovedFreight == nil {
		return latestVerifiedFreight, nil
	}
	if latestVerifiedFreight == nil && latestApprovedFreight != nil {
		return latestApprovedFreight, nil
	}
	if latestVerifiedFreight.CreationTimestamp.
		After(latestApprovedFreight.CreationTimestamp.Time) {
		return latestVerifiedFreight, nil
	}
	return latestApprovedFreight, nil
}

func (r *reconciler) getLatestFreightFromWarehouse(
	ctx context.Context,
	namespace string,
	warehouse string,
) (*kargoapi.Freight, error) {
	var freight kargoapi.FreightList
	if err := r.listFreightFn(
		ctx,
		&freight,
		&client.ListOptions{
			Namespace: namespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.FreightByWarehouseIndexField,
				warehouse,
			),
		},
	); err != nil {
		return nil, errors.Wrapf(
			err,
			"error listing Freight for Warehouse %q in namespace %q",
			warehouse,
			namespace,
		)
	}
	if len(freight.Items) == 0 {
		return nil, nil
	}
	// Sort by creation timestamp, descending
	sort.SliceStable(freight.Items, func(i, j int) bool {
		return freight.Items[j].CreationTimestamp.
			Before(&freight.Items[i].CreationTimestamp)
	})
	return &freight.Items[0], nil
}

func (r *reconciler) getAllVerifiedFreight(
	ctx context.Context,
	namespace string,
	stageSubs []kargoapi.StageSubscription,
) ([]kargoapi.Freight, error) {
	// Start by building a de-duped map of Freight verified in any upstream
	// Stage(s)
	verifiedFreight := map[string]kargoapi.Freight{}
	for _, stageSub := range stageSubs {
		var freight kargoapi.FreightList
		if err := r.listFreightFn(
			ctx,
			&freight,
			&client.ListOptions{
				Namespace: namespace,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.FreightByVerifiedStagesIndexField,
					stageSub.Name,
				),
			},
		); err != nil {
			return nil, errors.Wrapf(
				err,
				"error listing Freight verified in Stage %q in namespace %q",
				stageSub.Name,
				namespace,
			)
		}
		for _, freight := range freight.Items {
			verifiedFreight[freight.Name] = freight
		}
	}
	if len(verifiedFreight) == 0 {
		return nil, nil
	}
	// Turn the map to a list
	verifiedFreightList := make([]kargoapi.Freight, len(verifiedFreight))
	i := 0
	for _, freight := range verifiedFreight {
		verifiedFreightList[i] = freight
		i++
	}
	return verifiedFreightList, nil
}

func (r *reconciler) getLatestVerifiedFreight(
	ctx context.Context,
	namespace string,
	stageSubs []kargoapi.StageSubscription,
) (*kargoapi.Freight, error) {
	verifiedFreight, err :=
		r.getAllVerifiedFreightFn(ctx, namespace, stageSubs)
	if err != nil {
		return nil, err
	}
	if len(verifiedFreight) == 0 {
		return nil, nil
	}
	// Sort the list by creation timestamp, descending
	sort.SliceStable(verifiedFreight, func(i, j int) bool {
		return verifiedFreight[j].CreationTimestamp.
			Before(&verifiedFreight[i].CreationTimestamp)
	})
	return &verifiedFreight[0], nil
}

func (r *reconciler) getLatestApprovedFreight(
	ctx context.Context,
	namespace string,
	stage string,
) (*kargoapi.Freight, error) {
	var freight kargoapi.FreightList
	if err := r.listFreightFn(
		ctx,
		&freight,
		&client.ListOptions{
			Namespace: namespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.FreightApprovedForStagesIndexField,
				stage,
			),
		},
	); err != nil {
		return nil, errors.Wrapf(
			err,
			"error listing Freight verified in Stage %q in namespace %q",
			stage,
			namespace,
		)
	}
	if len(freight.Items) == 0 {
		return nil, nil
	}
	// Sort the list by creation timestamp, descending
	sort.SliceStable(freight.Items, func(i, j int) bool {
		return freight.Items[j].CreationTimestamp.
			Before(&freight.Items[i].CreationTimestamp)
	})
	return &freight.Items[0], nil
}
