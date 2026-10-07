package verification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/logging"
)

type Verifier struct {
	controllerName             string
	rolloutsIntegrationEnabled bool
	client                     client.Client
	eventSender                kargoEvent.Sender
	analysisRunners            map[kargoapi.AnalysisRunGVK]AnalysisRunner
}

func NewVerifier(
	controllerName string,
	rolloutsIntegrationEnabled bool,
	cl client.Client,
	eventSender kargoEvent.Sender,
	analysisRunners map[kargoapi.AnalysisRunGVK]AnalysisRunner,
) Verifier {
	return Verifier{
		controllerName:             controllerName,
		rolloutsIntegrationEnabled: rolloutsIntegrationEnabled,
		client:                     cl,
		eventSender:                eventSender,
		analysisRunners:            analysisRunners,
	}
}

func (ver Verifier) VerifyStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	startTime time.Time,
	endTime func() time.Time,
) (newStatus kargoapi.StageStatus, err error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus = *stage.Status.DeepCopy()

	// If there is no current Freight, then we have nothing to verify.
	curFreight := stage.Status.FreightHistory.Current()
	if curFreight == nil {
		logger.Debug("Stage has no current Freight: no verification to perform")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeVerified,
			Status:             metav1.ConditionUnknown,
			Reason:             "NoFreight",
			Message:            "Stage has no current Freight to verify",
			ObservedGeneration: stage.Generation,
		})
		return newStatus, nil
	}

	// If we are currently promoting Freight, then we are not in a stable state
	// and should wait until the promotion is complete.
	if stage.PromotionInProgress() {
		logger.Debug("Stage is currently promoting Freight: skipping verification")
		return newStatus, nil
	}

	defer func() {
		curFreight = newStatus.FreightHistory.Current()
		if curFreight == nil || len(curFreight.VerificationHistory) == 0 {
			return
		}

		for _, vi := range curFreight.VerificationHistory {
			if vi.Phase == kargoapi.VerificationPhaseSuccessful {
				// If the Freight has at least one successful verification,
				// then we can consider the Freight to be verified.
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionTrue,
					Reason:             "Verified",
					Message:            "Freight has been verified",
					ObservedGeneration: stage.Generation,
				})
				return
			}
		}

		// If the Freight has no successful verification, then we should look
		// for the most recent verification and set the status accordingly.
		lastVerification := curFreight.VerificationHistory.Current()
		if lastVerification != nil {
			switch lastVerification.Phase {
			case kargoapi.VerificationPhasePending:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionUnknown,
					Reason:             "VerificationPending",
					Message:            "Freight is pending verification",
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseRunning:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionUnknown,
					Reason:             "VerificationRunning",
					Message:            "Freight is currently being verified",
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseFailed, kargoapi.VerificationPhaseError:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionFalse,
					Reason:             fmt.Sprintf("Verification%s", lastVerification.Phase),
					Message:            lastVerification.Message,
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseAborted:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionFalse,
					Reason:             "VerificationAborted",
					Message:            lastVerification.Message,
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseInconclusive:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionUnknown,
					Reason:             "VerificationInconclusive",
					Message:            lastVerification.Message,
					ObservedGeneration: stage.Generation,
				})
			default:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:    kargoapi.ConditionTypeVerified,
					Status:  metav1.ConditionUnknown,
					Reason:  "UnknownVerificationPhase",
					Message: fmt.Sprintf("Freight verification is in an unknown phase: %s", lastVerification.Phase),
				})
			}
		}
	}()

	// Get the re-verification request, if any.
	reverifyReq, _ := api.ReverifyAnnotationValue(stage.GetAnnotations())

	// Check if the current Freight has already been verified.
	var newVI *kargoapi.VerificationInfo
	if lastVerification := curFreight.VerificationHistory.Current(); lastVerification != nil {
		// If the last verification is not terminal, then we should check if
		// we need to abort the verification, or if we need to get the verification
		// result.
		if !lastVerification.Phase.IsTerminal() {
			// Check if we need to abort the verification.
			abortReq, _ := api.AbortVerificationAnnotationValue(stage.GetAnnotations())
			if abortReq.ForID(lastVerification.ID) {
				logger.Debug("aborting verification of Stage Freight")

				// Abort the verification.
				newVI, err = ver.abortVerification(ctx, *curFreight, abortReq, endTime)
				if newVI != nil {
					newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)
				}

				// Issue an event for the aborted verification.
				for _, ref := range curFreight.Freight {
					ver.recordFreightVerificationEvent(stage, ref, newVI)
				}

				return newStatus, err
			}

			// Get the latest result of the verification.
			newVI, err = ver.getVerificationResult(ctx, *curFreight, endTime)
			if newVI != nil {
				newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)

				// If the verification is terminal, we should issue an event for
				// each Freight that was verified.
				if newVI.Phase.IsTerminal() {
					for _, ref := range curFreight.Freight {
						ver.recordFreightVerificationEvent(stage, ref, newVI)
					}
				}
			}
			return newStatus, err
		}

		// If the last verification is terminal, and we are not re-verifying
		// the Freight, then we have nothing to do.
		if !reverifyReq.ForID(lastVerification.ID) {
			logger.Debug("Stage Freight has already been verified")
			return newStatus, nil
		}
	}

	// If the Stage is not passed any health checks (yet), then we should not
	// verify the Freight.
	if stage.Status.Health == nil || stage.Status.Health.Status != kargoapi.HealthStateHealthy {
		logger.Debug("Stage has not passed health checks: skipping verification")
		return newStatus, nil
	}

	// If we have no specific verification configuration, then we can mark the
	// verification as successful.
	if stage.Spec.Verification == nil {
		newVI := kargoapi.VerificationInfo{
			ID:         uuid.NewString(),
			StartTime:  ptr.To(metav1.NewTime(startTime)),
			FinishTime: ptr.To(metav1.NewTime(endTime())),
			Phase:      kargoapi.VerificationPhaseSuccessful,
		}
		newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(newVI)

		// Issue an event for each Freight that was verified.
		for _, ref := range curFreight.Freight {
			ver.recordFreightVerificationEvent(stage, ref, &newVI)
		}
		return newStatus, nil
	}

	// Start a new (re-)verification.
	newVI, err = ver.startVerification(ctx, stage, *curFreight, reverifyReq, startTime, endTime)
	if newVI != nil {
		newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)

		// There is a chance for the verification to be terminal immediately
		// after starting it. For example, if the rollouts integration is not
		// enabled. In this case, we should issue an event for the verification.
		if newVI.Phase.IsTerminal() {
			for _, ref := range curFreight.Freight {
				ver.recordFreightVerificationEvent(stage, ref, newVI)
			}
		}
	}
	return newStatus, err
}

// recordFreightVerificationEvent records an event for the verification of a
// Freight. The event contains information about the Freight, the verification,
// and the Stage that triggered the verification.
func (ver Verifier) recordFreightVerificationEvent(
	stage *kargoapi.Stage,
	freightRef kargoapi.FreightReference,
	vi *kargoapi.VerificationInfo,
) {
	freight := &kargoapi.Freight{}
	if err := ver.client.Get(context.Background(), types.NamespacedName{
		Namespace: stage.Namespace,
		Name:      freightRef.Name,
	}, freight); err != nil {
		logging.LoggerFromContext(context.Background()).Error(
			err, "failed to get Freight for verification event",
			"freight", freightRef.Name,
		)
		return
	}

	var analysisTriggeredByPromotion *string
	// Extract metadata from the AnalysisRun if available
	if vi.HasAnalysisRun() {
		promoName, err := ver.getAnalysisRunPromotionName(context.Background(), *vi.AnalysisRun)
		if err != nil {
			// Log the error but do not fail the event recording.
			logging.LoggerFromContext(context.Background()).Error(
				err, "failed to get AnalysisRun for verification event",
				"analysisRun", vi.AnalysisRun.Name, "freight", freightRef.Name,
			)
		}

		if promoName != "" {
			analysisTriggeredByPromotion = &promoName
		}
	}

	evtActor := api.FormatEventControllerActor(ver.controllerName)

	// If the verification is manually triggered (e.g. reverify),
	// override the actor with the one who triggered the verification.
	if vi.Actor != "" {
		evtActor = vi.Actor
	}

	var evt kargoEvent.FreightVerificationEventMeta

	switch vi.Phase {
	case kargoapi.VerificationPhaseSuccessful:
		e := kargoEvent.NewFreightVerificationSucceeded(evtActor, stage.Name, freight, vi)
		e.Message = "Freight verification succeeded"
		evt = e
	case kargoapi.VerificationPhaseFailed:
		evt = kargoEvent.NewFreightVerificationFailed(evtActor, stage.Name, freight, vi)
	case kargoapi.VerificationPhaseError:
		evt = kargoEvent.NewFreightVerificationErrored(evtActor, stage.Name, freight, vi)
	case kargoapi.VerificationPhaseAborted:
		evt = kargoEvent.NewFreightVerificationAborted(evtActor, stage.Name, freight, vi)
	case kargoapi.VerificationPhaseInconclusive:
		evt = kargoEvent.NewFreightVerificationInconclusive(evtActor, stage.Name, freight, vi)
	default:
		evt = kargoEvent.NewFreightVerificationUnknown(evtActor, stage.Name, freight, vi)
	}

	evt.SetTriggeredByPromotion(analysisTriggeredByPromotion)

	if err := ver.eventSender.Send(context.Background(), evt); err != nil {
		logging.LoggerFromContext(context.Background()).Error(
			err, "failed to send verification event",
			"freight", freightRef.Name,
		)
	}
}

// startVerification starts a new verification for the Freight that is associated
// with the Stage. If the Freight has already been verified, then no verification
// is started unless a re-verification is requested.
//
// If there is no verification configuration for the Stage, then the verification
// is automatically considered successful and no verification is started.
//
// If the Rollouts integration is disabled, then the verification is marked as
// failed with an appropriate message.
//
// To start a verification, the Stage must be healthy.
func (ver Verifier) startVerification(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight kargoapi.FreightCollection,
	req *kargoapi.VerificationRequest,
	startTime time.Time,
	endTime func() time.Time,
) (*kargoapi.VerificationInfo, error) {
	newVI := &kargoapi.VerificationInfo{
		ID:        uuid.NewString(),
		StartTime: &metav1.Time{Time: startTime},
	}

	// If we have a verification request, we should enrich the information
	// with the actor who requested the verification.
	curVI := freight.VerificationHistory.Current()
	if curVI != nil && req.ForID(curVI.ID) {
		newVI.Actor = req.Actor
	}

	// Return early, as we cannot start the verification if the Rollouts
	// integration is disabled.
	if !ver.rolloutsIntegrationEnabled {
		newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = "Rollouts integration is disabled on this controller: cannot start verification"
		return newVI, nil
	}

	logger := logging.LoggerFromContext(ctx)

	// If this is not a re-verification request, check if there is an existing
	// AnalysisRun for the Stage and Freight. If there is, return the status
	// of the existing AnalysisRun.
	if req == nil {
		existingAnalysisRun, err := ver.findExistingAnalysisRun(
			ctx,
			types.NamespacedName{
				Namespace: stage.Namespace,
				Name:      stage.Name,
			},
			freight.ID)
		if err != nil {
			newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
			newVI.Phase = kargoapi.VerificationPhaseError
			newVI.Message = err.Error()
			return newVI, nil
		}

		logger.Info("EXISTING: ", "analysisRun", existingAnalysisRun)

		if existingAnalysisRun != nil {
			logger.Debug("AnalysisRun already exists for FreightCollection")

			newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
			newVI.Phase = kargoapi.VerificationPhase(existingAnalysisRun.Status.Phase)
			newVI.AnalysisRun = &kargoapi.AnalysisRunReference{
				Name:      existingAnalysisRun.Name,
				Namespace: existingAnalysisRun.Namespace,
				Phase:     existingAnalysisRun.Status.Phase,
			}
			return newVI, nil
		}
	}

	// At this point, we know that we need to start a new AnalysisRun for the
	// verification.
	lastPromoName := ""
	if curVI == nil || (req.ForID(curVI.ID) && req.ControlPlane && req.Actor != "") {
		lastPromoName = stage.LastPromotionName()
	}
	run, errorMessage, err := ver.createAnalysisRun(ctx, *stage, freight, lastPromoName)
	// Distinguish by error message because we might want to fail newVI without bubbling the error up.
	if errorMessage != "" {
		newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = errorMessage
		return newVI, err
	}

	// Mark the verification as pending.
	newVI.FinishTime = ptr.To(run.CreationTimestamp)
	newVI.Phase = kargoapi.VerificationPhasePending
	newVI.AnalysisRun = &kargoapi.AnalysisRunReference{
		Name:      run.Name,
		Namespace: run.Namespace,
		Phase:     run.Status.Phase,
		GVK:       run.GVK,
	}
	return newVI, nil
}

// getVerificationResult gets the result of the verification for the current
// Freight of a Stage.
//
// If the Stage does not have an AnalysisRun associated with the verification,
// an error is returned.
//
// If the Rollouts integration is disabled, then the verification is marked as
// failed with an appropriate message.
func (ver Verifier) getVerificationResult(
	ctx context.Context,
	freight kargoapi.FreightCollection,
	endTime func() time.Time,
) (*kargoapi.VerificationInfo, error) {
	// Ensure all necessary information is available to get the verification.
	currentVI := freight.VerificationHistory.Current()
	if currentVI == nil {
		return nil, fmt.Errorf("no current verification info for Freight collection %q", freight.ID)
	}
	if currentVI.AnalysisRun == nil {
		return nil, fmt.Errorf(
			"no AnalysisRun reference in current verification info for Freight collection %q",
			freight.ID,
		)
	}

	// If the Rollouts integration is disabled, then we cannot get the
	// verification.
	if !ver.rolloutsIntegrationEnabled {
		return &kargoapi.VerificationInfo{
			ID:         currentVI.ID,
			StartTime:  currentVI.StartTime,
			FinishTime: ptr.To(metav1.NewTime(endTime())),
			Phase:      kargoapi.VerificationPhaseError,
			Message:    "Rollouts integration is disabled on this controller: cannot get verification result",
		}, nil
	}

	arStatus, err := ver.getAnalysisRunStatus(ctx, *currentVI.AnalysisRun)
	if err != nil {
		return &kargoapi.VerificationInfo{
			ID:         currentVI.ID,
			Actor:      currentVI.Actor,
			StartTime:  currentVI.StartTime,
			FinishTime: currentVI.FinishTime,
			Phase:      kargoapi.VerificationPhaseError,
			Message: fmt.Errorf(
				"error getting AnalysisRun %q in namespace %q: %w",
				currentVI.AnalysisRun.Name,
				currentVI.AnalysisRun.Namespace,
				err,
			).Error(),
			AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
		}, err
	}

	// Return a new VerificationInfo with the same ID and the information from
	// the current state of the AnalysisRun.
	return &kargoapi.VerificationInfo{
		ID:         currentVI.ID,
		Actor:      currentVI.Actor,
		StartTime:  currentVI.StartTime,
		FinishTime: arStatus.CompletedAt,
		Phase:      kargoapi.VerificationPhase(arStatus.Phase),
		Message:    arStatus.Message,
		AnalysisRun: &kargoapi.AnalysisRunReference{
			Name:      currentVI.AnalysisRun.Name,
			Namespace: currentVI.AnalysisRun.Namespace,
			Phase:     arStatus.Phase,
		},
	}, nil
}

// abortVerification aborts the verification for the current Freight of a Stage.
func (ver Verifier) abortVerification(
	ctx context.Context,
	freight kargoapi.FreightCollection,
	req *kargoapi.VerificationRequest,
	endTime func() time.Time,
) (*kargoapi.VerificationInfo, error) {
	// Ensure all necessary information is available to abort the verification.
	currentVI := freight.VerificationHistory.Current()
	if currentVI == nil {
		return nil, fmt.Errorf("no current verification info for Freight collection %q", freight.ID)
	}
	if currentVI.AnalysisRun == nil {
		return nil, fmt.Errorf(
			"no AnalysisRun reference in current verification info for Freight collection %q",
			freight.ID,
		)
	}

	// If the current verification is already terminal, then there is no need
	// to abort it.
	if currentVI.Phase.IsTerminal() {
		return currentVI, nil
	}

	// Determine the actor who requested the abort.
	actor := currentVI.Actor
	if req.ForID(currentVI.ID) {
		actor = req.Actor
	}

	// If the Rollouts integration is disabled, then we cannot abort the
	// verification.
	if !ver.rolloutsIntegrationEnabled {
		return &kargoapi.VerificationInfo{
			ID:          currentVI.ID,
			Actor:       actor,
			StartTime:   currentVI.StartTime,
			FinishTime:  ptr.To(metav1.NewTime(endTime())),
			Phase:       kargoapi.VerificationPhaseError,
			Message:     "Rollouts integration is disabled on this controller: cannot abort verification",
			AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
		}, nil
	}

	err := ver.abortAnalysisRun(ctx, *currentVI.AnalysisRun)
	if err != nil {
		// TODO(hidde): we should consider better error handling here to e.g.
		// retry an abort request when the Kubernetes API server is under heavy
		// load.
		return &kargoapi.VerificationInfo{
			ID:         currentVI.ID,
			Actor:      actor,
			StartTime:  currentVI.StartTime,
			FinishTime: ptr.To(metav1.NewTime(endTime())),
			Phase:      kargoapi.VerificationPhaseError,
			Message: fmt.Errorf(
				"error terminating AnalysisRun %q in namespace %q: %w",
				currentVI.AnalysisRun.Name, currentVI.AnalysisRun.Namespace, err,
			).Error(),
			AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
		}, nil
	}

	// Return a new VerificationInfo with the same ID and a message indicating
	// that the verification was aborted. The Phase will be set to Failed, as
	// the verification was not successful.
	// We do not use the further information from the AnalysisRun, as this
	// will indicate a "Succeeded" phase due to Argo Rollouts behavior.
	return &kargoapi.VerificationInfo{
		ID:          currentVI.ID,
		Actor:       actor,
		StartTime:   currentVI.StartTime,
		FinishTime:  ptr.To(metav1.NewTime(endTime())),
		Phase:       kargoapi.VerificationPhaseFailed,
		Message:     "Verification aborted by user",
		AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
	}, nil
}
