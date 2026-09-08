package stages

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	gocache "github.com/patrickmn/go-cache"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	exprfn "github.com/akuity/kargo/pkg/expressions/function"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/kubernetes"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/rollouts"
)

// verifyStageFreight verifies the current Freight of a Stage. If the Stage has
// no current Freight, or the Freight has already been verified, then no action
// is taken. If the Freight has not been verified yet, then a new verification
// is started.
//
// An annotation can be set on the Stage to request the verification to be
// aborted. This is useful if the verification is taking too long, or if the
// Freight is no longer needed.
//
// In addition, an annotation can be set on the Stage to request re-verification
// of the Freight. This can be useful to ensure that the current Freight is
// still in a good state.
//
// When the Stage is unhealthy, or a Promotion is currently running, then the
// verification is skipped.
func (r *RegularStageReconciler) verifyStageFreight(
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
	curPromotion := getCurrentPromoObject(stage)
	// If we are currently promoting Freight, then we are not in a stable state
	// and should wait until the promotion is complete.
	if curPromotion != nil {
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
				newVI, err = r.abortVerification(ctx, *curFreight, abortReq, endTime)
				if newVI != nil {
					newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)
				}

				// Issue an event for the aborted verification.
				for _, ref := range curFreight.Freight {
					r.recordFreightVerificationEvent(stage, ref, newVI)
				}

				return newStatus, err
			}

			// Get the latest result of the verification.
			newVI, err = r.getVerificationResult(ctx, *curFreight, endTime)
			if newVI != nil {
				newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)

				// If the verification is terminal, we should issue an event for
				// each Freight that was verified.
				if newVI.Phase.IsTerminal() {
					for _, ref := range curFreight.Freight {
						r.recordFreightVerificationEvent(stage, ref, newVI)
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
			r.recordFreightVerificationEvent(stage, ref, &newVI)
		}
		return newStatus, nil
	}

	// Start a new (re-)verification.
	newVI, err = r.startVerification(ctx, stage, *curFreight, reverifyReq, startTime, endTime)
	if newVI != nil {
		newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)

		// There is a chance for the verification to be terminal immediately
		// after starting it. For example, if the rollouts integration is not
		// enabled. In this case, we should issue an event for the verification.
		if newVI.Phase.IsTerminal() {
			for _, ref := range curFreight.Freight {
				r.recordFreightVerificationEvent(stage, ref, newVI)
			}
		}
	}
	return newStatus, err
}

// markFreightVerifiedForStage marks the Freight that is associated with the
// Stage as verified. If the Freight has already been verified, then no action
// is taken.
func (r *RegularStageReconciler) markFreightVerifiedForStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

	// If the Stage is unhealthy, then we should not verify the Freight.
	if stage.Status.Health == nil || stage.Status.Health.Status != kargoapi.HealthStateHealthy {
		return newStatus, nil
	}

	// If there is no current Freight, or the Stage has not been verified yet
	// after the last Promotion, then we are not ready to verify the Freight.
	curFreight := stage.Status.FreightHistory.Current()
	if curFreight == nil ||
		len(curFreight.VerificationHistory) == 0 ||
		curFreight.HasNonTerminalVerification() ||
		curFreight.VerificationHistory.Current().Phase != kargoapi.VerificationPhaseSuccessful {
		return newStatus, nil
	}

	// At this point, all preconditions for verifying the Freight have been met,
	// and we can proceed with the verification.
	for _, ref := range curFreight.Freight {
		freight := &kargoapi.Freight{}
		if err := r.client.Get(ctx, types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      ref.Name,
		}, freight); err != nil {
			return newStatus, fmt.Errorf(
				"error getting Freight %q in namespace %q: %w",
				ref.Name, stage.Namespace, err,
			)
		}

		// If the Freight has already been verified, then there is no need to
		// verify it again.
		if freight.IsVerifiedIn(stage.Name) {
			logger.Debug("Freight has already been verified in Stage")
			continue
		}

		// Verify the Freight.
		if err := kubeclient.PatchStatus(ctx, r.client, freight, func(status *kargoapi.FreightStatus) {
			if status.VerifiedIn == nil {
				status.VerifiedIn = make(map[string]kargoapi.VerifiedStage)
			}
			status.AddVerifiedStage(stage.Name, curFreight.VerificationHistory.Current().FinishTime.Time)
		}); err != nil {
			return newStatus, fmt.Errorf(
				"error marking Freight %q as verified in Stage: %w",
				freight.Name, err,
			)
		}
		logger.Debug("marked Freight as verified in Stage", "freight", freight.Name)
	}

	return newStatus, nil
}

// recordFreightVerificationEvent records an event for the verification of a
// Freight. The event contains information about the Freight, the verification,
// and the Stage that triggered the verification.
func (r *RegularStageReconciler) recordFreightVerificationEvent(
	stage *kargoapi.Stage,
	freightRef kargoapi.FreightReference,
	vi *kargoapi.VerificationInfo,
) {
	freight := &kargoapi.Freight{}
	if err := r.client.Get(context.Background(), types.NamespacedName{
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
		ar := &rolloutsapi.AnalysisRun{}
		if err := r.client.Get(context.Background(), types.NamespacedName{
			Namespace: vi.AnalysisRun.Namespace,
			Name:      vi.AnalysisRun.Name,
		}, ar); err != nil {
			// Log the error but do not fail the event recording.
			logging.LoggerFromContext(context.Background()).Error(
				err, "failed to get AnalysisRun for verification event",
				"analysisRun", vi.AnalysisRun.Name, "freight", freightRef.Name,
			)
		}
		// AnalysisRun that triggered by a Promotion contains the Promotion name
		if promoName, ok := ar.Annotations[kargoapi.AnnotationKeyPromotion]; ok {
			analysisTriggeredByPromotion = &promoName
		}
	}

	evtActor := api.FormatEventControllerActor(r.cfg.Name())

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

	if err := r.eventSender.Send(context.Background(), evt); err != nil {
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
func (r *RegularStageReconciler) startVerification(
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
	if !r.cfg.RolloutsIntegrationEnabled {
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
		existingAnalysisRun, err := r.findExistingAnalysisRun(ctx, types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      stage.Name,
		}, freight.ID)
		if err != nil {
			newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
			newVI.Phase = kargoapi.VerificationPhaseError
			newVI.Message = err.Error()
			return newVI, nil
		}

		if existingAnalysisRun != nil {
			logger.Debug("AnalysisRun already exists for FreightCollection")

			newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
			newVI.Phase = kargoapi.VerificationPhase(existingAnalysisRun.Status.Phase)
			newVI.AnalysisRun = &kargoapi.AnalysisRunReference{
				Name:      existingAnalysisRun.Name,
				Namespace: existingAnalysisRun.Namespace,
				Phase:     string(existingAnalysisRun.Status.Phase),
			}
			return newVI, nil
		}
	}

	// At this point, we know that we need to start a new AnalysisRun for the
	// verification.
	shortStageName := kubernetes.ShortenLabelValue(stage.Name)
	builder := rollouts.NewAnalysisRunBuilder(r.client, rollouts.Config{
		ControllerInstanceID: r.cfg.RolloutsControllerInstanceID,
	})
	builderOpts := []rollouts.AnalysisRunOption{
		rollouts.WithNamePrefix(stage.Name),
		rollouts.WithNameSuffix(freight.ID),
		rollouts.WithExtraLabels(map[string]string{
			kargoapi.LabelKeyStage:             shortStageName,
			kargoapi.LabelKeyFreightCollection: freight.ID,
		}),
		rollouts.WithArgumentEvaluationConfig{
			Env: map[string]any{
				"ctx": map[string]any{
					"project": stage.Namespace,
					"stage":   stage.Name,
				},
			},
			Options: slices.Concat(
				exprfn.DataOperations(
					ctx,
					r.client,
					gocache.New(gocache.NoExpiration, gocache.NoExpiration),
					stage.Namespace,
				),
				exprfn.FreightOperations(
					ctx,
					r.client,
					stage.Namespace,
					stage.Spec.RequestedFreight,
					freight.References(),
				),
				exprfn.UtilityOperations(),
			),
			Vars: stage.Spec.Vars,
		},
	}

	if stage.Name != shortStageName {
		builderOpts = append(builderOpts, rollouts.WithExtraAnnotations(map[string]string{
			kargoapi.AnnotationKeyStage: stage.Name,
		}))
	}

	for _, freightRef := range freight.Freight {
		builderOpts = append(builderOpts, rollouts.WithOwner{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Freight",
			Reference:  types.NamespacedName{Namespace: stage.Namespace, Name: freightRef.Name},
		})
	}
	if curVI == nil || (req.ForID(curVI.ID) && req.ControlPlane && req.Actor != "") {
		lastPromo := getLastPromoObject(stage)
		if lastPromo != nil {
			builderOpts = append(builderOpts, rollouts.WithExtraAnnotations{
				kargoapi.AnnotationKeyPromotion: lastPromo.GetName(),
			})
		}
	}
	ar, err := builder.Build(ctx, stage.Namespace, stage.Spec.Verification, builderOpts...)
	if err != nil {
		newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = fmt.Errorf(
			"error building AnalysisRun for Stage %q and Freight collection %q in namespace %q: %w",
			stage.Name,
			freight.ID,
			stage.Namespace,
			err,
		).Error()
		return newVI, nil
	}
	if err = r.client.Create(ctx, ar); err != nil {
		newVI.FinishTime = ptr.To(metav1.NewTime(endTime()))
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = fmt.Errorf(
			"error creating AnalysisRun %q in namespace %q: %w",
			ar.Name,
			ar.Namespace,
			err,
		).Error()
		return newVI, kubeclient.IgnoreInvalid(err) // Ignore errors which are due to validation issues
	}

	// Mark the verification as pending.
	newVI.FinishTime = ptr.To(ar.CreationTimestamp)
	newVI.Phase = kargoapi.VerificationPhasePending
	newVI.AnalysisRun = &kargoapi.AnalysisRunReference{
		Name:      ar.Name,
		Namespace: ar.Namespace,
		Phase:     string(ar.Status.Phase),
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
func (r *RegularStageReconciler) getVerificationResult(
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
	if !r.cfg.RolloutsIntegrationEnabled {
		return &kargoapi.VerificationInfo{
			ID:         currentVI.ID,
			StartTime:  currentVI.StartTime,
			FinishTime: ptr.To(metav1.NewTime(endTime())),
			Phase:      kargoapi.VerificationPhaseError,
			Message:    "Rollouts integration is disabled on this controller: cannot get verification result",
		}, nil
	}

	// TODO(hidde): This retry logic has been put in place because we have
	// observed the cache not being up-to-date with the API server in some
	// edge case scenarios. While this is not a long-term solution, it cures
	// the symptoms for now. We should investigate the root cause of this
	// issue and remove this retry logic when the root cause has been resolved.
	ar := rolloutsapi.AnalysisRun{}
	if err := retry.OnError(r.backoffCfg, func(err error) bool {
		return apierrors.IsNotFound(err)
	}, func() error {
		return r.client.Get(ctx, types.NamespacedName{
			Namespace: currentVI.AnalysisRun.Namespace,
			Name:      currentVI.AnalysisRun.Name,
		}, &ar)
	}); err != nil {
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
		FinishTime: ar.Status.CompletedAt,
		Phase:      kargoapi.VerificationPhase(ar.Status.Phase),
		Message:    ar.Status.Message,
		AnalysisRun: &kargoapi.AnalysisRunReference{
			Name:      ar.Name,
			Namespace: ar.Namespace,
			Phase:     string(ar.Status.Phase),
		},
	}, nil
}

// abortVerification aborts the verification for the current Freight of a Stage.
func (r *RegularStageReconciler) abortVerification(
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
	if !r.cfg.RolloutsIntegrationEnabled {
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

	// Patch the AnalysisRun to request the abort.
	ar := &rolloutsapi.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: currentVI.AnalysisRun.Namespace,
			Name:      currentVI.AnalysisRun.Name,
		},
	}
	if err := r.client.Patch(
		ctx,
		ar,
		client.RawPatch(types.MergePatchType, []byte(`{"spec":{"terminate":true}}`)),
	); err != nil {
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
				"error terminating AnalysisRun %q in namespace %q: %w", ar.Name, ar.Namespace, err,
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

// findExistingAnalysisRun finds the most recent AnalysisRun for a Stage and
// Freight collection in the namespace of the Stage. If no AnalysisRun is found,
// it returns nil.
func (r *RegularStageReconciler) findExistingAnalysisRun(
	ctx context.Context,
	stage types.NamespacedName,
	freightColID string,
) (*rolloutsapi.AnalysisRun, error) {
	analysisRuns := &rolloutsapi.AnalysisRunList{}
	if err := r.client.List(
		ctx,
		analysisRuns,
		client.InNamespace(stage.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(map[string]string{
				kargoapi.LabelKeyStage:             kubernetes.ShortenLabelValue(stage.Name),
				kargoapi.LabelKeyFreightCollection: freightColID,
			}),
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing AnalysisRuns for Stage %q and Freight collection %q in namespace %q: %w",
			stage.Name, freightColID, stage.Namespace, err,
		)
	}

	if len(analysisRuns.Items) == 0 {
		return nil, nil
	}

	// Sort the AnalysisRuns by creation timestamp, so that the most recent
	// one is first.
	slices.SortFunc(analysisRuns.Items, func(lhs, rhs rolloutsapi.AnalysisRun) int {
		return rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time)
	})
	return &analysisRuns.Items[0], nil
}
