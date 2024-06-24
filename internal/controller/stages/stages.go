package stages

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/internal/argocd"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	libEvent "github.com/akuity/kargo/internal/kubernetes/event"
	"github.com/akuity/kargo/internal/logging"
)

// ReconcilerConfig represents configuration for the stage reconciler.
type ReconcilerConfig struct {
	ShardName                    string `envconfig:"SHARD_NAME"`
	RolloutsIntegrationEnabled   bool   `envconfig:"ROLLOUTS_INTEGRATION_ENABLED"`
	RolloutsControllerInstanceID string `envconfig:"ROLLOUTS_CONTROLLER_INSTANCE_ID"`
}

func (c ReconcilerConfig) Name() string {
	name := "stage-controller"
	if c.ShardName != "" {
		return name + "-" + c.ShardName
	}
	return name
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler reconciles Stage resources.
type reconciler struct {
	kargoClient  client.Client
	argocdClient client.Client

	recorder record.EventRecorder

	cfg ReconcilerConfig

	// The following behaviors are overridable for testing purposes:

	// Promotion-related:

	nowFn func() time.Time

	syncPromotionsFn func(
		context.Context,
		*kargoapi.Stage,
		kargoapi.StageStatus,
	) (kargoapi.StageStatus, error)

	getPromotionsForStageFn func(
		context.Context,
		string,
		string,
	) ([]kargoapi.Promotion, error)

	listPromosFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	// Health checks:

	appHealth libargocd.ApplicationHealthEvaluator

	// Freight verification:

	startVerificationFn func(
		context.Context,
		*kargoapi.Stage,
	) (*kargoapi.VerificationInfo, error)

	abortVerificationFn func(
		context.Context,
		*kargoapi.Stage,
	) *kargoapi.VerificationInfo

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
		freight *kargoapi.Freight,
		templates []*rollouts.AnalysisTemplate,
	) (*rollouts.AnalysisRun, error)

	createAnalysisRunFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	patchAnalysisRunFn func(
		context.Context,
		client.Object,
		client.Patch,
		...client.PatchOption,
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
	) (bool, error)

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

	getProjectFn func(
		context.Context,
		client.Client,
		string,
	) (*kargoapi.Project, error)

	createPromotionFn func(
		context.Context,
		client.Object,
		...client.CreateOption,
	) error

	// Discovering Freight:

	getAvailableFreightFn func(
		ctx context.Context,
		stage *kargoapi.Stage,
		includeApproved bool,
	) ([]kargoapi.Freight, error)

	listFreightFn func(
		context.Context,
		client.ObjectList,
		...client.ListOption,
	) error

	// Stage deletion:

	clearVerificationsFn func(context.Context, *kargoapi.Stage) error

	clearApprovalsFn func(context.Context, *kargoapi.Stage) error

	clearAnalysisRunsFn func(context.Context, *kargoapi.Stage) error

	shardRequirement *labels.Requirement
}

// SetupReconcilerWithManager initializes a reconciler for Stage resources and
// registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argocdMgr manager.Manager,
	cfg ReconcilerConfig,
) error {
	// Index Promotions by Stage
	if err := kubeclient.IndexPromotionsByStage(ctx, kargoMgr); err != nil {
		return fmt.Errorf("index non-terminal Promotions by Stage: %w", err)
	}

	// Index Promotions by Stage + Freight
	if err := kubeclient.IndexPromotionsByStageAndFreight(ctx, kargoMgr); err != nil {
		return fmt.Errorf("index Promotions by Stage and Freight: %w", err)
	}

	// Index Freight by Warehouse
	if err := kubeclient.IndexFreightByWarehouse(ctx, kargoMgr); err != nil {
		return fmt.Errorf("index Freight by Warehouse: %w", err)
	}

	// Index Freight by Stages in which it has been verified
	if err :=
		kubeclient.IndexFreightByVerifiedStages(ctx, kargoMgr); err != nil {
		return fmt.Errorf("index Freight by Stages in which it has been verified: %w", err)
	}

	// Index Freight by Stages for which it has been approved
	if err :=
		kubeclient.IndexFreightByApprovedStages(ctx, kargoMgr); err != nil {
		return fmt.Errorf("index Freight by Stages for which it has been approved: %w", err)
	}

	// Index Stages by upstream Stages
	if err :=
		kubeclient.IndexStagesByUpstreamStages(ctx, kargoMgr); err != nil {
		return fmt.Errorf("index Stages by upstream Stages: %w", err)
	}

	// Index Stages by Warehouse
	if err := kubeclient.IndexStagesByWarehouse(ctx, kargoMgr); err != nil {
		return fmt.Errorf("index Stages by Warehouse: %w", err)
	}

	// Index Stages by Argo CD Applications
	if err := kubeclient.IndexStagesByArgoCDApplications(ctx, kargoMgr, cfg.ShardName); err != nil {
		return fmt.Errorf("index Stages by Argo CD Applications: %w", err)
	}

	// Index Stages by AnalysisRun
	if err := kubeclient.IndexStagesByAnalysisRun(ctx, kargoMgr, cfg.ShardName); err != nil {
		return fmt.Errorf("index Stages by Argo Rollouts AnalysisRun: %w", err)
	}

	shardPredicate, err := controller.GetShardPredicate(cfg.ShardName)
	if err != nil {
		return fmt.Errorf("error creating shard predicate: %w", err)
	}

	shardRequirement, err := controller.GetShardRequirement(cfg.ShardName)
	if err != nil {
		return fmt.Errorf("error creating shard requirement: %w", err)
	}
	shardSelector := labels.NewSelector().Add(*shardRequirement)
	var argocdClient client.Client
	if argocdMgr != nil {
		argocdClient = argocdMgr.GetClient()
	}

	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Stage{}).
		WithEventFilter(
			predicate.Funcs{
				DeleteFunc: func(event.DeleteEvent) bool {
					// We're not interested in any ACTUAL deletes. (We do care about
					// updates where DeletionTimestamp is non-nil, but that's not a delete
					// event.)
					return false
				},
			},
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				kargo.RefreshRequested{},
				kargo.ReverifyRequested{},
				kargo.AbortRequested{},
			),
		).
		WithEventFilter(shardPredicate).
		WithOptions(controller.CommonOptions()).
		Build(
			newReconciler(
				kargoMgr.GetClient(),
				argocdClient,
				libEvent.NewRecorder(ctx, kargoMgr.GetScheme(), kargoMgr.GetClient(), cfg.Name()),
				cfg,
				shardRequirement,
			),
		)
	if err != nil {
		return fmt.Errorf("error building Stage reconciler: %w", err)
	}

	logger := logging.LoggerFromContext(ctx)
	// Watch Promotions for which the phase changed and enqueue owning Stage key
	promoOwnerHandler := handler.TypedEnqueueRequestForOwner[*kargoapi.Promotion](
		kargoMgr.GetScheme(),
		kargoMgr.GetRESTMapper(),
		&kargoapi.Stage{},
		handler.OnlyControllerOwner(),
	)
	promoPhaseChanged := kargo.NewPromoPhaseChangedPredicate(logger)
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Promotion{},
			promoOwnerHandler,
			promoPhaseChanged,
		),
	); err != nil {
		return fmt.Errorf("unable to watch Promotions: %w", err)
	}

	// Watch Freight that has been marked as verified in a Stage and enqueue
	// downstream Stages
	verifiedFreightHandler := &verifiedFreightEventHandler[*kargoapi.Freight]{
		kargoClient:   kargoMgr.GetClient(),
		shardSelector: shardSelector,
	}
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			verifiedFreightHandler,
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight: %w", err)
	}

	approveFreightHandler := &approvedFreightEventHandler[*kargoapi.Freight]{
		kargoClient: kargoMgr.GetClient(),
	}
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			approveFreightHandler,
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight: %w", err)
	}

	createdFreightEventHandler := &createdFreightEventHandler[*kargoapi.Freight]{
		kargoClient:   kargoMgr.GetClient(),
		shardSelector: shardSelector,
	}
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			createdFreightEventHandler,
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight: %w", err)
	}

	// If Argo CD integration is disabled, this manager will be nil and we won't
	// care about this watch anyway.
	if argocdMgr != nil {
		updatedArgoCDAppHandler := &updatedArgoCDAppHandler[*argocd.Application]{
			kargoClient:   kargoMgr.GetClient(),
			shardSelector: shardSelector,
		}
		if err := c.Watch(
			source.Kind(
				argocdMgr.GetCache(),
				&argocd.Application{},
				updatedArgoCDAppHandler,
			),
		); err != nil {
			return fmt.Errorf("unable to watch Applications: %w", err)
		}
	}

	// We only care about this if Rollouts integration is enabled.
	if cfg.RolloutsIntegrationEnabled {
		phaseChangedAnalysisRunHandler := &phaseChangedAnalysisRunHandler[*rollouts.AnalysisRun]{
			kargoClient:   kargoMgr.GetClient(),
			shardSelector: shardSelector,
		}
		if err := c.Watch(
			source.Kind(
				kargoMgr.GetCache(),
				&rollouts.AnalysisRun{},
				phaseChangedAnalysisRunHandler,
			),
		); err != nil {
			return fmt.Errorf("unable to watch AnalysisRuns: %w", err)
		}
	}

	return nil
}

func newReconciler(
	kargoClient client.Client,
	argocdClient client.Client,
	recorder record.EventRecorder,
	cfg ReconcilerConfig,
	shardRequirement *labels.Requirement,
) *reconciler {
	r := &reconciler{
		kargoClient:      kargoClient,
		argocdClient:     argocdClient,
		recorder:         recorder,
		cfg:              cfg,
		appHealth:        libargocd.NewApplicationHealthEvaluator(argocdClient),
		shardRequirement: shardRequirement,
	}
	// The following default behaviors are overridable for testing purposes:
	// Promotion-related:
	r.nowFn = time.Now
	r.syncPromotionsFn = r.syncPromotions
	r.listPromosFn = r.kargoClient.List
	r.getPromotionsForStageFn = r.getPromotionsForStage
	// Freight verification:
	r.startVerificationFn = r.startVerification
	r.abortVerificationFn = r.abortVerification
	r.getVerificationInfoFn = r.getVerificationInfo
	r.getAnalysisTemplateFn = rollouts.GetAnalysisTemplate
	r.listAnalysisRunsFn = r.kargoClient.List
	r.buildAnalysisRunFn = r.buildAnalysisRun
	r.createAnalysisRunFn = r.kargoClient.Create
	r.patchAnalysisRunFn = r.kargoClient.Patch
	r.getAnalysisRunFn = rollouts.GetAnalysisRun
	r.getFreightFn = kargoapi.GetFreight
	r.verifyFreightInStageFn = r.verifyFreightInStage
	r.patchFreightStatusFn = r.patchFreightStatus
	// Auto-promotion:
	r.isAutoPromotionPermittedFn = r.isAutoPromotionPermitted
	r.getProjectFn = kargoapi.GetProject
	r.createPromotionFn = kargoClient.Create
	// Discovering Freight:
	r.getAvailableFreightFn = r.getAvailableFreight
	r.listFreightFn = r.kargoClient.List
	// Stage deletion:
	r.clearVerificationsFn = r.clearVerifications
	r.clearApprovalsFn = r.clearApprovals
	r.clearAnalysisRunsFn = r.clearAnalysisRuns
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
		"stage", req.NamespacedName.Name,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Stage")

	// Find the Stage
	stage, err := kargoapi.GetStage(ctx, r.kargoClient, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if stage == nil {
		// Ignore if not found. This can happen if the Stage was deleted after the
		// current reconciliation request was issued.
		return ctrl.Result{}, nil // Do not requeue
	}

	if ok := r.shardRequirement.Matches(labels.Set(stage.Labels)); !ok {
		// Ignore if stage does not belong to given shard
		return ctrl.Result{}, err
	}
	logger.Debug("found Stage")

	var newStatus kargoapi.StageStatus
	if stage.DeletionTimestamp != nil {
		newStatus, err = r.syncStageDelete(ctx, stage)
		if err == nil && controllerutil.RemoveFinalizer(stage, kargoapi.FinalizerName) {
			if err = r.kargoClient.Update(ctx, stage); err != nil {
				err = fmt.Errorf("error removing finalizer: %w", err)
			}
		}
	} else {
		err = kargoapi.AddFinalizer(ctx, r.kargoClient, stage)
		if err != nil {
			newStatus = stage.Status
		} else {
			if stage.Spec.PromotionMechanisms == nil {
				newStatus, err = r.syncControlFlowStage(ctx, stage)
			} else {
				newStatus, err = r.syncNormalStage(ctx, stage)
			}
		}
	}
	if err != nil {
		newStatus.Message = err.Error()
		logger.Error(err, "error syncing Stage")
	} else {
		// Be sure to blank this out in case there's an error in this field from
		// the previous reconciliation
		newStatus.Message = ""
	}

	// Record the current refresh token as having been handled.
	if token, ok := kargoapi.RefreshAnnotationValue(stage.GetAnnotations()); ok {
		newStatus.LastHandledRefresh = token
	}

	updateErr := kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
		*status = newStatus
	})
	if updateErr != nil {
		logger.Error(updateErr, "error updating Stage status")
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}
	logger.Debug("done reconciling Stage")

	// If we do have an error at this point, return it so controller runtime
	// retries with a progressive backoff.
	if err != nil {
		return ctrl.Result{}, err
	}

	// Everything succeeded, look for new changes on the defined interval.
	//
	// TODO: Make this configurable
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *reconciler) syncControlFlowStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	startTime := r.nowFn()

	status := *stage.Status.DeepCopy()
	status.ObservedGeneration = stage.Generation
	status.Phase = kargoapi.StagePhaseNotApplicable

	// A Stage without promotion mechanisms shouldn't have history, health, or
	// promotions. Make sure this is empty to avoid confusion. A reason this
	// could be non-empty to begin with is that the Stage USED TO have promotion
	// mechanisms, but they were removed, thus becoming a control flow Stage.
	status.FreightHistory = nil
	status.Health = nil
	status.CurrentPromotion = nil
	status.LastPromotion = nil

	// TODO: Remove this once we remove the fields from the API.
	status.History = nil
	status.CurrentFreight = nil

	// Find all Freight available to this Stage, except for those that
	// are available on account of manual approval.
	availableFreight, err := r.getAvailableFreightFn(ctx, stage, false)
	if err != nil {
		return status, fmt.Errorf(
			"error getting available Freight for control flow Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}

	finishTime := r.nowFn()
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
				return status, fmt.Errorf(
					"error marking Freight %q in namespace %q as verified in Stage %q: %w",
					af.Name,
					stage.Namespace,
					stage.Name,
					err,
				)
			}

			r.recordFreightVerificationEvent(
				stage,
				&af,
				&kargoapi.VerificationInfo{
					StartTime:  ptr.To(metav1.NewTime(startTime)),
					FinishTime: ptr.To(metav1.NewTime(finishTime)),
					Phase:      kargoapi.VerificationPhaseSuccessful,
				},
				nil, // Explicitly pass `nil` here since there is no associated AnalysisRun
			)
		}
	}
	return status, nil
}

func (r *reconciler) syncNormalStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	startTime := r.nowFn()
	status := *stage.Status.DeepCopy()

	logger := logging.LoggerFromContext(ctx)

	// Sync Promotions and update the Stage status.
	var syncErr error
	if status, syncErr = r.syncPromotionsFn(ctx, stage, status); syncErr != nil {
		return status, syncErr
	}
	if err := kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(s *kargoapi.StageStatus) {
		*s = status
	}); err != nil {
		return status, err
	}

	// Take note of the current Generation of the Stage as being observed,
	// and reset the health status.
	status.ObservedGeneration = stage.Generation
	status.Health = nil

	if currentFreight := status.FreightHistory.Current(); currentFreight == nil || len(currentFreight.Freight) == 0 {
		status.Phase = kargoapi.StagePhaseNotApplicable
		logger.Debug(
			"Stage has no current Freight; no health checks or verification to perform",
		)
	} else {
		for _, freight := range currentFreight.Freight {
			freightLogger := logger.WithValues("freight", freight.Name)
			shouldRecordFreightVerificationEvent := false

			// Always check the health of the Argo CD Applications associated with the
			// Stage. This is regardless of the phase of the Stage, as the health of the
			// Argo CD Applications is always relevant.
			if status.Health = r.appHealth.EvaluateHealth(
				ctx,
				freight,
				stage.Spec.PromotionMechanisms.ArgoCDAppUpdates,
			); status.Health != nil {
				freightLogger.WithValues("health", status.Health.Status).Debug("Stage health assessed")
			} else {
				freightLogger.Debug("Stage health deemed not applicable")
			}

			if stage.Spec.Verification != nil {
				// Update the verification history with the current verification info.
				// NOTE: We do this regardless of the phase of the verification process
				// and before potentially creating a new AnalysisRun to ensure we add
				// any previous verification attempt which may have been recorded
				// before we started tracking history.
				if freight.VerificationInfo != nil {
					freight.VerificationHistory.UpdateOrPush(*freight.VerificationInfo)
				}

				// If the Stage is in a steady state, we should check if we need to
				// start or rerun verification.
				if status.Phase == kargoapi.StagePhaseSteady {
					info := freight.VerificationInfo
					switch {
					case freight.VerificationInfo == nil && status.CurrentPromotion == nil:
						status.Phase = kargoapi.StagePhaseVerifying
					case info.Phase.IsTerminal():
						if req, _ := kargoapi.ReverifyAnnotationValue(stage.GetAnnotations()); req.ForID(info.ID) {
							logger.Debug("rerunning verification")
							status.Phase = kargoapi.StagePhaseVerifying
							freight.VerificationInfo = nil
						}
					}
				}

				// Initiate or follow-up on verification if required.
				if status.Phase == kargoapi.StagePhaseVerifying {
					if !freight.VerificationInfo.HasAnalysisRun() {
						if status.Health == nil || status.Health.Status == kargoapi.HealthStateHealthy {
							logger.Debug("starting verification")
							var err error
							if freight.VerificationInfo, err = r.startVerificationFn(
								ctx,
								stage,
							); err != nil && !freight.VerificationInfo.HasAnalysisRun() {
								freight.VerificationHistory.UpdateOrPush(*freight.VerificationInfo)
								currentFreight.UpdateOrPush(freight)
								return status, fmt.Errorf("error starting verification: %w", err)
							}
						}
					} else {
						logger.Debug("checking verification results")
						var err error
						if freight.VerificationInfo, err = r.getVerificationInfoFn(
							ctx,
							stage,
						); err != nil && freight.VerificationInfo.HasAnalysisRun() {
							freight.VerificationHistory.UpdateOrPush(*freight.VerificationInfo)
							currentFreight.UpdateOrPush(freight)
							return status, fmt.Errorf("error getting verification info: %w", err)
						}

						// Abort the verification if it's still running and the Stage has
						// been marked to do so.
						newInfo := freight.VerificationInfo
						if !newInfo.Phase.IsTerminal() {
							if req, _ := kargoapi.AbortAnnotationValue(stage.GetAnnotations()); req.ForID(newInfo.ID) {
								logger.Debug("aborting verification")
								freight.VerificationInfo = r.abortVerificationFn(ctx, stage)
							}
						}
					}

					if freight.VerificationInfo != nil {
						logger.Debug(
							"verification", "phase",
							freight.VerificationInfo.Phase,
						)

						if freight.VerificationInfo.Phase.IsTerminal() {
							// Verification is complete
							shouldRecordFreightVerificationEvent = true
							status.Phase = kargoapi.StagePhaseSteady
							logger.Debug("verification is complete")
						}

						// Add latest verification info to history.
						freight.VerificationHistory.UpdateOrPush(*freight.VerificationInfo)
					}
				}
			} else {
				// If verification is not applicable, mark the Stage as steady.
				// This ensures that if the Stage had verification enabled previously,
				// it will not be stuck in a verification phase.
				status.Phase = kargoapi.StagePhaseSteady
			}

			// If health is not applicable or healthy
			// AND
			// Verification is not applicable or successful
			// THEN
			// Mark the Freight as verified in this Stage
			if (status.Health == nil || status.Health.Status == kargoapi.HealthStateHealthy) &&
				(stage.Spec.Verification == nil ||
					(freight.VerificationInfo != nil &&
						freight.VerificationInfo.Phase == kargoapi.VerificationPhaseSuccessful)) {
				updated, err := r.verifyFreightInStageFn(
					ctx,
					stage.Namespace,
					freight.Name,
					stage.Name,
				)
				if err != nil {
					return status, fmt.Errorf(
						"error marking Freight %q in namespace %q as verified in Stage %q: %w",
						freight.Name,
						stage.Namespace,
						stage.Name,
						err,
					)
				}

				// Always record verification event when the Freight is marked as verified
				if updated {
					shouldRecordFreightVerificationEvent = true
				}
			}

			finishTime := r.nowFn()

			// Record freight verification event only if the freight is newly verified
			if shouldRecordFreightVerificationEvent {
				vi := freight.VerificationInfo
				if stage.Spec.Verification == nil {
					vi = &kargoapi.VerificationInfo{
						StartTime:  ptr.To(metav1.NewTime(startTime)),
						FinishTime: ptr.To(metav1.NewTime(finishTime)),
						Phase:      kargoapi.VerificationPhaseSuccessful,
					}
				}

				var ar *rollouts.AnalysisRun
				if vi.HasAnalysisRun() {
					var err error
					ar, err = r.getAnalysisRunFn(
						ctx,
						r.kargoClient,
						types.NamespacedName{
							Namespace: vi.AnalysisRun.Namespace,
							Name:      vi.AnalysisRun.Name,
						},
					)
					if err != nil {
						currentFreight.UpdateOrPush(freight)
						return status, fmt.Errorf("get analysisRun: %w", err)
					}
				}

				fr, err := r.getFreightFn(
					ctx,
					r.kargoClient,
					types.NamespacedName{
						Namespace: stage.Namespace,
						Name:      freight.Name,
					},
				)
				if err != nil {
					currentFreight.UpdateOrPush(freight)
					return status, fmt.Errorf("get freight: %w", err)
				}
				if fr != nil {
					r.recordFreightVerificationEvent(stage, fr, vi, ar)
				}
			}

			currentFreight.UpdateOrPush(freight)
		}
	}

	// Stop here if we have no chance of finding any Freight to promote.
	if len(stage.Spec.RequestedFreight) == 0 {
		logger.Info(
			"Stage requests no Freight. This may indicate an issue with resource" +
				"validation logic.",
		)
		return status, nil
	}

	// TODO: Remove this when we have added support for multiple Freight below.
	if len(stage.Spec.RequestedFreight) > 1 {
		logger.Info(
			"Stage requests multiple Freight. Auto-promotion support has however not been implemented for this (yet).",
		)
		return status, nil
	}

	logger.Debug("checking if auto-promotion is permitted...")
	if permitted, err :=
		r.isAutoPromotionPermittedFn(ctx, stage.Namespace, stage.Name); err != nil {
		return status, fmt.Errorf(
			"error checking if auto-promotion is permitted for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	} else if !permitted {
		logger.Debug("auto-promotion is not permitted for the Stage")
		return status, nil
	}

	// If we get to here, auto-promotion is permitted. Time to go looking for new
	// Freight...

	availableFreight, err := r.getAvailableFreightFn(ctx, stage, true)
	if err != nil {
		return status, fmt.Errorf(
			"error finding latest Freight for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}

	if len(availableFreight) == 0 {
		logger.Debug("no available Freight found")
		return status, nil
	}

	// Find the latest Freight by sorting the available Freight by creation time
	// in descending order.
	slices.SortFunc(availableFreight, func(lhs, rhs kargoapi.Freight) int {
		return rhs.CreationTimestamp.Time.Compare(lhs.CreationTimestamp.Time)
	})
	latestFreight := availableFreight[0]

	logger = logger.WithValues("freight", latestFreight.Name)

	// Only proceed if latest Freight isn't the one we already have
	// TODO(hidde): This is a naive check, and should be replaced with proper
	// logic that works with multiple Freight.
	if currentFreight := status.FreightHistory.Current(); currentFreight != nil && len(currentFreight.Freight) > 0 {
		for _, requested := range stage.Spec.RequestedFreight {
			if _, ok := currentFreight.Freight[requested.Origin]; ok {
				if currentFreight.Freight[requested.Origin].Name == latestFreight.Name {
					logger.Debug("Stage already has latest available Freight")
					return status, nil
				}
			}
		}
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
		return status, fmt.Errorf(
			"error listing existing Promotions for Freight %q in namespace %q: %w",
			latestFreight.Name,
			stage.Namespace,
			err,
		)
	}

	if len(promos.Items) > 0 {
		logger.Debug("Promotion already exists for Freight")
		return status, nil
	}

	logger.Debug("auto-promotion will proceed")

	promo := kargo.NewPromotion(ctx, *stage, latestFreight.Name)
	if err :=
		r.createPromotionFn(ctx, &promo); err != nil {
		return status, fmt.Errorf(
			"error creating Promotion of Stage %q in namespace %q to Freight %q: %w",
			stage.Name,
			stage.Namespace,
			latestFreight.Name,
			err,
		)
	}

	r.recorder.AnnotatedEventf(
		&promo,
		kargoapi.NewPromotionEventAnnotations(
			ctx,
			kargoapi.FormatEventControllerActor(r.cfg.Name()),
			&promo,
			&latestFreight,
		),
		corev1.EventTypeNormal,
		kargoapi.EventReasonPromotionCreated,
		"Automatically promoted Freight for Stage %q",
		promo.Spec.Stage,
	)

	logger.Debug(
		"created Promotion resource",
		"promotion", promo.Name,
	)

	return status, nil
}

// syncPromotions determines the current state of the Stage and its Freight by
// examining the Promotions that have been created for the Stage. It returns the
// updated Stage status.
//
// The Stage is considered to be promoting if the latest Promotion is in a
// running phase. In this case, the Stage is marked as promoting, and the
// current Promotion is recorded in the Stage status. If the latest Promotion
// is not in a running phase, the Stage is considered to be steady.
//
// New Promotions that have terminated since the last reconciliation are
// discovered by comparing a list of terminated Promotions to the last known
// Promotion. Any newer Promotions found are recorded in the Stage status, and
// the Freight that was successfully promoted is recorded in the Freight
// history.
func (r *reconciler) syncPromotions(
	ctx context.Context,
	stage *kargoapi.Stage,
	status kargoapi.StageStatus,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)

	promotions, err := r.getPromotionsForStageFn(ctx, stage.Namespace, stage.Name)
	if err != nil || len(promotions) == 0 {
		return status, err
	}

	// Sort the Promotions by phase and creation time so that we can determine the
	// current state of the Stage.
	slices.SortFunc(promotions, kargoapi.ComparePromotionByPhaseAndCreationTime)

	// If the latest Promotion is in a running phase, the Stage is promoting.
	if latestPromo := promotions[0]; !latestPromo.Status.Phase.IsTerminal() {
		logger.WithValues("promotion", latestPromo.Name).Debug("Stage has a running Promotion")
		status.Phase = kargoapi.StagePhasePromoting
		status.CurrentPromotion = &kargoapi.PromotionInfo{
			Name: latestPromo.Name,
		}
		if latestPromo.Status.Freight != nil {
			status.CurrentPromotion.Freight = *latestPromo.Status.Freight.DeepCopy()
		}
	} else {
		status.CurrentPromotion = nil
	}

	// Determine if there are any new Promotions that have been completed since
	// the last reconciliation.
	logger.Debug("checking for new terminated Promotions")
	var newPromotions []kargoapi.PromotionInfo
	for _, promo := range promotions {
		if status.LastPromotion != nil {
			// We can break here since we know that all subsequent Promotions
			// will be older than the last Promotion we saw.
			// NB: This makes use of the fact that Promotion names are
			// generated, and contain a timestamp component which will ensure
			// that they can be sorted in a consistent order.
			if strings.Compare(promo.Name, status.LastPromotion.Name) <= 0 {
				break
			}
		}

		if promo.Status.Phase.IsTerminal() {
			logger.WithValues("promotion", promo.Name).Debug("found new terminated Promotion")
			info := kargoapi.PromotionInfo{
				Name:   promo.Name,
				Status: promo.Status.DeepCopy(),
			}
			if promo.Status.Freight != nil {
				info.Freight = *promo.Status.Freight.DeepCopy()
			}
			newPromotions = append(newPromotions, info)
		}
	}

	// As we will be appending to the Freight history, we need to ensure that
	// we order the Promotions from oldest to newest. This is because the
	// Freight history is garbage collected based on the number of entries,
	// and we want to ensure that the oldest entries are removed first.
	slices.SortFunc(newPromotions, func(a, b kargoapi.PromotionInfo) int {
		return strings.Compare(a.Name, b.Name)
	})

	// Update the Stage status with the information about the newly terminated
	// Promotions, and any new Freight that was successfully promoted.
	for _, p := range newPromotions {
		promo := p
		status.LastPromotion = &promo
		if promo.Status.Phase == kargoapi.PromotionPhaseSucceeded {
			// TODO(hidde): This should ensure that it properly handles
			// multiple Freight when implemented on the Promotion side.
			status.FreightHistory.Record(&kargoapi.FreightHistoryEntry{
				Freight: map[string]kargoapi.FreightReference{
					status.LastPromotion.Freight.Warehouse: *status.LastPromotion.Freight.DeepCopy(),
				},
			})
			if status.CurrentPromotion == nil {
				status.Phase = kargoapi.StagePhaseSteady
			}
		}
	}

	return status, nil
}

func (r *reconciler) syncStageDelete(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	status := *stage.Status.DeepCopy()
	status.ObservedGeneration = stage.Generation
	if !controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName) {
		return status, nil
	}
	if err := r.clearVerificationsFn(ctx, stage); err != nil {
		return status, fmt.Errorf(
			"error clearing verifications for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}
	if err := r.clearApprovalsFn(ctx, stage); err != nil {
		return status, fmt.Errorf(
			"error clearing approvals for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}
	if err := r.clearAnalysisRunsFn(ctx, stage); err != nil {
		return status, fmt.Errorf(
			"error clearing AnalysisRuns for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}
	return status, nil
}

func (r *reconciler) clearVerifications(
	ctx context.Context,
	stage *kargoapi.Stage,
) error {
	verified := kargoapi.FreightList{}
	if err := r.listFreightFn(
		ctx,
		&verified,
		&client.ListOptions{
			Namespace: stage.Namespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.FreightByVerifiedStagesIndexField,
				stage.Name,
			),
		},
	); err != nil {
		return fmt.Errorf(
			"error listing Freight verified in Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}
	for _, f := range verified.Items {
		freight := f // Avoid implicit memory aliasing
		newStatus := *freight.Status.DeepCopy()
		if newStatus.VerifiedIn == nil {
			continue
		}
		delete(newStatus.VerifiedIn, stage.Name)
		if err := r.patchFreightStatusFn(ctx, &freight, newStatus); err != nil {
			return fmt.Errorf(
				"error patching status of Freight %q in namespace %q: %w",
				freight.Name,
				freight.Namespace,
				err,
			)
		}
	}
	return nil
}

func (r *reconciler) clearApprovals(
	ctx context.Context,
	stage *kargoapi.Stage,
) error {
	approved := kargoapi.FreightList{}
	if err := r.listFreightFn(
		ctx,
		&approved,
		&client.ListOptions{
			Namespace: stage.Namespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.FreightApprovedForStagesIndexField,
				stage.Name,
			),
		},
	); err != nil {
		return fmt.Errorf(
			"error listing Freight approved for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}
	for _, f := range approved.Items {
		freight := f // Avoid implicit memory aliasing
		newStatus := *freight.Status.DeepCopy()
		if newStatus.ApprovedFor == nil {
			continue
		}
		delete(newStatus.ApprovedFor, stage.Name)
		if err := r.patchFreightStatusFn(ctx, &freight, newStatus); err != nil {
			return fmt.Errorf(
				"error patching status of Freight %q in namespace %q: %w",
				freight.Name,
				freight.Namespace,
				err,
			)
		}
	}
	return nil
}

func (r *reconciler) clearAnalysisRuns(
	ctx context.Context,
	stage *kargoapi.Stage,
) error {
	if !r.cfg.RolloutsIntegrationEnabled {
		return nil
	}
	if err := r.kargoClient.DeleteAllOf(
		ctx,
		&rollouts.AnalysisRun{},
		client.InNamespace(stage.Namespace),
		client.MatchingLabels(map[string]string{
			kargoapi.StageLabelKey: stage.Name,
		}),
	); err != nil {
		return fmt.Errorf(
			"error deleting AnalysisRuns for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}
	return nil
}

// verifyFreightInStage marks the given Freight as verified in the given Stage.
// It returns true if succeeded to mark Freight as verified in the Stage,
// or false if it was already marked as verified in the Stage.
func (r *reconciler) verifyFreightInStage(
	ctx context.Context,
	namespace string,
	freightName string,
	stageName string,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx).WithValues("freight", freightName)

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
		return false, fmt.Errorf(
			"error finding Freight %q in namespace %q: %w",
			freightName,
			namespace,
			err,
		)
	}
	if freight == nil {
		return false, fmt.Errorf(
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
		return false, nil
	}

	newStatus.VerifiedIn[stageName] = kargoapi.VerifiedStage{}
	if err = r.patchFreightStatusFn(ctx, freight, newStatus); err != nil {
		return false, err
	}

	logger.Debug("marked Freight as verified in Stage")
	return true, nil
}

func (r *reconciler) patchFreightStatus(
	ctx context.Context,
	freight *kargoapi.Freight,
	newStatus kargoapi.FreightStatus,
) error {
	if err := kubeclient.PatchStatus(
		ctx,
		r.kargoClient,
		freight,
		func(status *kargoapi.FreightStatus) {
			*status = newStatus
		},
	); err != nil {
		return fmt.Errorf(
			"error patching Freight %q status in namespace %q: %w",
			freight.Name,
			freight.Namespace,
			err,
		)
	}
	return nil
}

func (r *reconciler) isAutoPromotionPermitted(
	ctx context.Context,
	namespace string,
	stageName string,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)
	project, err := r.getProjectFn(ctx, r.kargoClient, namespace)
	if err != nil {
		return false, fmt.Errorf("error finding Project %q: %w", namespace, err)
	}
	if project == nil {
		return false, fmt.Errorf("Project %q not found", namespace)
	}
	if project.Spec == nil || len(project.Spec.PromotionPolicies) == 0 {
		logger.Debug("found no PromotionPolicy associated with the Stage")
		return false, nil
	}
	for _, policy := range project.Spec.PromotionPolicies {
		if policy.Stage == stageName {
			logger.Debug(
				"found PromotionPolicy associated with the Stage",
				"autoPromotionEnabled", policy.AutoPromotionEnabled,
			)
			return policy.AutoPromotionEnabled, nil
		}
	}
	return false, nil
}

func (r *reconciler) getPromotionsForStage(
	ctx context.Context,
	stageNamespace string,
	stageName string,
) ([]kargoapi.Promotion, error) {
	var promos kargoapi.PromotionList
	if err := r.listPromosFn(
		ctx,
		&promos,
		&client.ListOptions{
			Namespace: stageNamespace,
			FieldSelector: fields.OneTermEqualSelector(
				kubeclient.PromotionsByStageIndexField,
				stageName,
			),
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing Promotions for Stage %q in namespace %q: %w",
			stageName,
			stageNamespace,
			err,
		)
	}
	return promos.Items, nil
}

func (r *reconciler) getAvailableFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	includeApproved bool,
) ([]kargoapi.Freight, error) {
	var availableFreight []kargoapi.Freight
	for _, req := range stage.Spec.RequestedFreight {
		// Get Freight direct from Warehouses if allowed
		if req.Sources.Direct {
			var freight kargoapi.FreightList
			if err := r.listFreightFn(
				ctx,
				&freight,
				&client.ListOptions{
					Namespace: stage.Namespace,
					FieldSelector: fields.OneTermEqualSelector(
						kubeclient.FreightByWarehouseIndexField,
						req.Origin,
					),
				},
			); err != nil {
				return nil, fmt.Errorf(
					"error listing Freight from Warehouse %q in namespace %q: %w",
					req.Origin,
					stage.Namespace,
					err,
				)
			}
			availableFreight = append(availableFreight, freight.Items...)
		}
		// Get Freight verified in upstream Stages
		for _, upstream := range req.Sources.Stages {
			var verifiedFreight kargoapi.FreightList
			if err := r.listFreightFn(
				ctx,
				&verifiedFreight,
				&client.ListOptions{
					Namespace: stage.Namespace,
					FieldSelector: fields.OneTermEqualSelector(
						kubeclient.FreightByVerifiedStagesIndexField,
						upstream,
					),
				},
			); err != nil {
				return nil, fmt.Errorf(
					"error listing Freight verified in Stage %q in namespace %q: %w",
					upstream,
					stage.Namespace,
					err,
				)
			}
			availableFreight = append(availableFreight, verifiedFreight.Items...)
		}
	}

	if includeApproved {
		var approvedFreight kargoapi.FreightList
		if err := r.listFreightFn(
			ctx,
			&approvedFreight,
			&client.ListOptions{
				Namespace: stage.Namespace,
				FieldSelector: fields.OneTermEqualSelector(
					kubeclient.FreightApprovedForStagesIndexField,
					stage.Name,
				),
			},
		); err != nil {
			return nil, fmt.Errorf(
				"error listing Freight approved for Stage %q in namespace %q: %w",
				stage,
				stage.Namespace,
				err,
			)
		}
		availableFreight = append(availableFreight, approvedFreight.Items...)
	}

	// De-dupe the Freight
	slices.SortFunc(availableFreight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	availableFreight = slices.CompactFunc(availableFreight, func(lhs, rhs kargoapi.Freight) bool {
		return lhs.Name == rhs.Name
	})

	return availableFreight, nil
}

func (r *reconciler) recordFreightVerificationEvent(
	s *kargoapi.Stage,
	fr *kargoapi.Freight,
	vi *kargoapi.VerificationInfo,
	ar *rollouts.AnalysisRun,
) {
	annotations := map[string]string{
		kargoapi.AnnotationKeyEventActor:             kargoapi.FormatEventControllerActor(r.cfg.Name()),
		kargoapi.AnnotationKeyEventProject:           s.Namespace,
		kargoapi.AnnotationKeyEventStageName:         s.Name,
		kargoapi.AnnotationKeyEventFreightAlias:      fr.Alias,
		kargoapi.AnnotationKeyEventFreightName:       fr.Name,
		kargoapi.AnnotationKeyEventFreightCreateTime: fr.CreationTimestamp.Format(time.RFC3339),
	}
	if vi.StartTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationStartTime] = vi.StartTime.Format(time.RFC3339)
	}
	if vi.FinishTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationFinishTime] = vi.FinishTime.Format(time.RFC3339)
	}

	// Extract metadata from the AnalysisRun if available
	if ar != nil {
		annotations[kargoapi.AnnotationKeyEventAnalysisRunName] = ar.Name
		// AnalysisRun that triggered by a Promotion contains the Promotion name
		if promoName, ok := ar.Labels[kargoapi.PromotionLabelKey]; ok {
			annotations[kargoapi.AnnotationKeyEventPromotionName] = promoName
		}
	}

	// If the verification is manually triggered (e.g. reverify),
	// override the actor with the one who triggered the verification.
	if vi.Actor != "" {
		annotations[kargoapi.AnnotationKeyEventActor] = vi.Actor
	}

	reason := kargoapi.EventReasonFreightVerificationUnknown
	message := vi.Message

	switch vi.Phase {
	case kargoapi.VerificationPhaseSuccessful:
		reason = kargoapi.EventReasonFreightVerificationSucceeded
		message = "Freight verification succeeded"
	case kargoapi.VerificationPhaseFailed:
		reason = kargoapi.EventReasonFreightVerificationFailed
	case kargoapi.VerificationPhaseError:
		reason = kargoapi.EventReasonFreightVerificationErrored
	case kargoapi.VerificationPhaseAborted:
		reason = kargoapi.EventReasonFreightVerificationAborted
	case kargoapi.VerificationPhaseInconclusive:
		reason = kargoapi.EventReasonFreightVerificationInconclusive
	}

	r.recorder.AnnotatedEventf(fr, annotations, corev1.EventTypeNormal, reason, message)
}
