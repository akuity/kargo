package stages

import (
	"context" // nolint: gosec
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// startVerification starts a verification for the given Stage. If the Stage
// does not have a reverification annotation, it checks if there is an existing
// AnalysisRun for the Stage and Freight. If there is, it returns the status of
// this AnalysisRun. If there is not, it creates a new AnalysisRun for the Stage
// and Freight.
//
// In case of an error, it returns a VerificationInfo with the error message and
// the phase set to Error. If the error may be due to a transient issue, it is
// returned, so that the caller can retry the operation.
func (r *reconciler) startVerification(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightCol *kargoapi.FreightCollection,
) (*kargoapi.VerificationInfo, error) {
	startTime := r.nowFn()

	newInfo := &kargoapi.VerificationInfo{
		ID:        uuid.NewString(),
		StartTime: ptr.To(metav1.NewTime(startTime)),
	}

	// Gather information about the verification being a re-verification request,
	// and extract the actor who requested the re-verification to be used in the
	// new verification info.
	var isReverify bool
	curVer := freightCol.VerificationHistory.Current()
	if curVer != nil {
		if req, _ := kargoapi.ReverifyAnnotationValue(stage.GetAnnotations()); req.ForID(curVer.ID) {
			isReverify = true
			newInfo.Actor = req.Actor
		}
	}

	if !r.cfg.RolloutsIntegrationEnabled {
		newInfo.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
		newInfo.Phase = kargoapi.VerificationPhaseError
		newInfo.Message = "Rollouts integration is disabled on this controller; cannot start verification"
		return newInfo, nil
	}

	logger := logging.LoggerFromContext(ctx)

	// If this is not a re-verification request, check if there is an existing
	// AnalysisRun for the Stage and Freight. If there is, return the status of
	// this AnalysisRun.
	if !isReverify {
		analysisRuns := rollouts.AnalysisRunList{}
		if err := r.listAnalysisRunsFn(
			ctx,
			&analysisRuns,
			&client.ListOptions{
				Namespace: stage.Namespace,
				LabelSelector: labels.SelectorFromSet(
					map[string]string{
						kargoapi.StageLabelKey:             stage.Name,
						kargoapi.FreightCollectionLabelKey: freightCol.ID,
					},
				),
			},
		); err != nil {
			newInfo.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
			newInfo.Phase = kargoapi.VerificationPhaseError
			newInfo.Message = fmt.Errorf(
				"error listing AnalysisRuns for Stage %q and FreightCollection %q in namespace %q: %w",
				stage.Name,
				freightCol.ID,
				stage.Namespace,
				err,
			).Error()
			return newInfo, err
		}
		if len(analysisRuns.Items) > 0 {
			// Sort the AnalysisRuns by creation timestamp, so that the most recent
			// one is first.
			slices.SortFunc(analysisRuns.Items, func(lhs, rhs rollouts.AnalysisRun) int {
				return rhs.CreationTimestamp.Time.Compare(lhs.CreationTimestamp.Time)
			})

			logger.Debug("AnalysisRun already exists for Freight")
			latestAnalysisRun := analysisRuns.Items[0]

			newInfo.FinishTime = latestAnalysisRun.Status.CompletedAt()
			newInfo.Phase = kargoapi.VerificationPhase(latestAnalysisRun.Status.Phase)
			newInfo.AnalysisRun = &kargoapi.AnalysisRunReference{
				Name:      latestAnalysisRun.Name,
				Namespace: latestAnalysisRun.Namespace,
				Phase:     string(latestAnalysisRun.Status.Phase),
			}
			return newInfo, nil
		}
	}

	ver := stage.Spec.Verification

	templates := make([]*rollouts.AnalysisTemplate, len(ver.AnalysisTemplates))
	for i, templateRef := range ver.AnalysisTemplates {
		template, err := r.getAnalysisTemplateFn(
			ctx,
			r.kargoClient,
			types.NamespacedName{
				Namespace: stage.Namespace,
				Name:      templateRef.Name,
			},
		)
		if err != nil {
			newInfo.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
			newInfo.Phase = kargoapi.VerificationPhaseError
			newInfo.Message = fmt.Errorf(
				"error getting AnalysisTemplate %q in namespace %q: %w",
				templateRef.Name,
				stage.Namespace,
				err,
			).Error()
			return newInfo, err
		}
		if template == nil {
			newInfo.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
			newInfo.Phase = kargoapi.VerificationPhaseError
			newInfo.Message = fmt.Errorf(
				"AnalysisTemplate %q in namespace %q not found",
				templateRef.Name,
				stage.Namespace,
			).Error()
			return newInfo, nil
		}
		templates[i] = template
	}

	run, err := r.buildAnalysisRunFn(ctx, stage, curVer, freightCol, templates)
	if err != nil {
		newInfo.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
		newInfo.Phase = kargoapi.VerificationPhaseError
		newInfo.Message = fmt.Errorf(
			"error building AnalysisRun for Stage %q and Freight %q in namespace %q: %w",
			stage.Name,
			freightCol.ID,
			stage.Namespace,
			err,
		).Error()
		return newInfo, nil
	}

	if err := r.createAnalysisRunFn(ctx, run); err != nil {
		newInfo.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
		newInfo.Phase = kargoapi.VerificationPhaseError
		newInfo.Message = fmt.Errorf(
			"error creating AnalysisRun %q in namespace %q: %w",
			run.Name,
			run.Namespace,
			err,
		).Error()
		return newInfo, kubeclient.IgnoreInvalid(err) // Ignore errors which are due to validation issues
	}

	newInfo.StartTime = ptr.To(run.CreationTimestamp)
	newInfo.Phase = kargoapi.VerificationPhasePending
	newInfo.AnalysisRun = &kargoapi.AnalysisRunReference{
		Name:      run.Name,
		Namespace: run.Namespace,
	}
	return newInfo, nil
}

// getVerificationInfo returns the status of the AnalysisRun for the given Stage.
//
// In case of an error, it returns a VerificationInfo with the error message and
// the phase set to Error. If the error may be due to a transient issue, it is
// returned, so that the caller can retry the operation.
//
// If an error is returned, the AnalysisRun reference in the VerificationInfo
// will always be set to the AnalysisRun that was being checked. This is to
// ensure the caller can continue to track the status of the AnalysisRun.
func (r *reconciler) getVerificationInfo(
	ctx context.Context,
	stage *kargoapi.Stage,
	verificationInfo *kargoapi.VerificationInfo,
) (*kargoapi.VerificationInfo, error) {
	if !r.cfg.RolloutsIntegrationEnabled {
		return &kargoapi.VerificationInfo{
			ID:         verificationInfo.ID,
			Actor:      verificationInfo.Actor,
			StartTime:  verificationInfo.StartTime,
			FinishTime: verificationInfo.FinishTime,
			Phase:      kargoapi.VerificationPhaseError,
			Message: "Rollouts integration is disabled on this controller; cannot " +
				"get verification info",
		}, nil
	}

	analysisRunName := verificationInfo.AnalysisRun.Name
	analysisRun, err := r.getAnalysisRunFn(
		ctx,
		r.kargoClient,
		types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      analysisRunName,
		},
	)
	if err != nil {
		return &kargoapi.VerificationInfo{
			ID:         verificationInfo.ID,
			Actor:      verificationInfo.Actor,
			StartTime:  verificationInfo.StartTime,
			FinishTime: verificationInfo.FinishTime,
			Phase:      kargoapi.VerificationPhaseError,
			Message: fmt.Errorf(
				"error getting AnalysisRun %q in namespace %q: %w",
				analysisRunName,
				stage.Namespace,
				err,
			).Error(),
			AnalysisRun: verificationInfo.AnalysisRun.DeepCopy(),
		}, err
	}
	if analysisRun == nil {
		return &kargoapi.VerificationInfo{
			ID:         verificationInfo.ID,
			Actor:      verificationInfo.Actor,
			StartTime:  verificationInfo.StartTime,
			FinishTime: verificationInfo.FinishTime,
			Phase:      kargoapi.VerificationPhaseError,
			Message: fmt.Errorf(
				"AnalysisRun %q in namespace %q not found",
				analysisRunName,
				stage.Namespace,
			).Error(),
		}, nil
	}

	return &kargoapi.VerificationInfo{
		ID:         verificationInfo.ID,
		Actor:      verificationInfo.Actor,
		StartTime:  ptr.To(analysisRun.CreationTimestamp),
		FinishTime: analysisRun.Status.CompletedAt(),
		Phase:      kargoapi.VerificationPhase(analysisRun.Status.Phase),
		Message:    analysisRun.Status.Message,
		AnalysisRun: &kargoapi.AnalysisRunReference{
			Name:      analysisRun.Name,
			Namespace: analysisRun.Namespace,
			Phase:     string(analysisRun.Status.Phase),
		},
	}, nil
}

func (r *reconciler) abortVerification(
	ctx context.Context,
	stage *kargoapi.Stage,
	currentVI *kargoapi.VerificationInfo,
) *kargoapi.VerificationInfo {
	newVI := &kargoapi.VerificationInfo{
		ID:        currentVI.ID,
		StartTime: currentVI.StartTime,
	}

	// Extract the actor who requested the abort to be used in the new
	// verification info.
	if req, _ := kargoapi.AbortAnnotationValue(stage.GetAnnotations()); req.ForID(currentVI.ID) {
		newVI.Actor = req.Actor
	}

	if !r.cfg.RolloutsIntegrationEnabled {
		newVI.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = "Rollouts integration is disabled on this controller; cannot abort verification"
		return newVI
	}

	ar := &rollouts.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      currentVI.AnalysisRun.Name,
			Namespace: currentVI.AnalysisRun.Namespace,
		},
	}
	if err := r.patchAnalysisRunFn(
		ctx,
		ar,
		client.RawPatch(types.MergePatchType, []byte(`{"spec":{"terminate":true}}`)),
	); err != nil {
		newVI.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = fmt.Errorf(
			"error terminating AnalysisRun %q in namespace %q: %w",
			ar.Name,
			ar.Namespace,
			err,
		).Error()
		newVI.AnalysisRun = currentVI.AnalysisRun.DeepCopy()
		return newVI
	}

	// Return a new VerificationInfo with the same ID and a message indicating
	// that the verification was aborted. The Phase will be set to Failed, as
	// the verification was not successful.
	// We do not use the further information from the AnalysisRun, as this
	// will indicate a "Succeeded" phase due to Argo Rollouts behavior.
	newVI.StartTime = ptr.To(ar.CreationTimestamp)
	newVI.FinishTime = ptr.To(metav1.NewTime(r.nowFn()))
	newVI.Phase = kargoapi.VerificationPhaseAborted
	newVI.Message = "Verification aborted by user"
	newVI.AnalysisRun = currentVI.AnalysisRun.DeepCopy()
	return newVI
}

func (r *reconciler) buildAnalysisRun(
	ctx context.Context,
	stage *kargoapi.Stage,
	verificationInfo *kargoapi.VerificationInfo,
	freightCol *kargoapi.FreightCollection,
	templates []*rollouts.AnalysisTemplate,
) (*rollouts.AnalysisRun, error) {
	// maximum length of the stage name used in the promotion name prefix before it exceeds
	// kubernetes resource name limit of 253
	// 253 - 1 (.) - 26 (ulid) - 1 (.) - 7 (sha) = 218
	const maxStageNamePrefixLength = 218

	// Build the name of the AnalysisRun
	shortStageName := stage.Name
	if len(stage.Name) > maxStageNamePrefixLength {
		shortStageName = shortStageName[0:maxStageNamePrefixLength]
	}
	analysisRunName := strings.ToLower(
		fmt.Sprintf(
			"%s.%s.%s",
			shortStageName,
			ulid.Make(),
			freightCol.ID[0:7], // nolint: gosec
		),
	)

	// Build the labels and annotations for the AnalysisRun
	var numLabels int
	var numAnnotations int
	if stage.Spec.Verification.AnalysisRunMetadata != nil {
		numLabels = len(stage.Spec.Verification.AnalysisRunMetadata.Labels)
		numAnnotations = len(stage.Spec.Verification.AnalysisRunMetadata.Annotations)
	}
	// Kargo will add up to three labels of its own, so size the map accordingly
	lbls := make(map[string]string, numLabels+4)
	annotations := make(map[string]string, numAnnotations+1)
	if stage.Spec.Verification.AnalysisRunMetadata != nil {
		for k, v := range stage.Spec.Verification.AnalysisRunMetadata.Labels {
			lbls[k] = v
		}
		for k, v := range stage.Spec.Verification.AnalysisRunMetadata.Annotations {
			annotations[k] = v
		}
	}
	lbls[kargoapi.StageLabelKey] = stage.Name
	lbls[kargoapi.FreightCollectionLabelKey] = freightCol.ID

	// Add Promotion name if the AnalysisRun is triggered by Promotion.
	// This is the case when there is no existing verification information,
	// or the re-verification request has been made by the control plane
	// without an actor specified.
	reverifyReq, _ := kargoapi.ReverifyAnnotationValue(stage.GetAnnotations())
	if verificationInfo == nil ||
		reverifyReq.ForID(verificationInfo.ID) && reverifyReq.ControlPlane && reverifyReq.Actor == "" {
		// Add Promotion name if the AnalysisRun is triggered by Promotion.
		if stage.Status.LastPromotion != nil {
			lbls[kargoapi.PromotionLabelKey] = stage.Status.LastPromotion.Name
		}
	}
	if r.cfg.RolloutsControllerInstanceID != "" {
		lbls["argo-rollouts.argoproj.io/controller-instance-id"] = r.cfg.RolloutsControllerInstanceID
	}

	// Flatten templates into a single template
	template, err := flattenTemplates(templates)
	if err != nil {
		return nil, fmt.Errorf("error flattening templates: %w", err)
	}

	// Merge the args from the template with the args from the Stage
	rolloutsArgs := make([]rollouts.Argument, len(stage.Spec.Verification.Args))
	for i, argument := range stage.Spec.Verification.Args {
		arg := argument // Avoid implicit memory aliasing
		rolloutsArgs[i] = rollouts.Argument{
			Name:  arg.Name,
			Value: &arg.Value,
		}
	}
	mergedArgs, err := mergeArgs(rolloutsArgs, template.Spec.Args)
	if err != nil {
		return nil, fmt.Errorf("error merging arguments: %w", err)
	}

	ar := &rollouts.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        analysisRunName,
			Namespace:   stage.Namespace,
			Labels:      lbls,
			Annotations: annotations,
		},
		Spec: rollouts.AnalysisRunSpec{
			Metrics:              template.Spec.Metrics,
			DryRun:               template.Spec.DryRun,
			MeasurementRetention: template.Spec.MeasurementRetention,
			Args:                 mergedArgs,
		},
	}

	gvk := kargoapi.GroupVersion.WithKind("Freight")
	for _, freightRef := range freightCol.Freight {
		f, err := r.getFreightFn(
			ctx,
			r.kargoClient,
			types.NamespacedName{
				Namespace: stage.Namespace,
				Name:      freightRef.Name,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"error getting Freight %q in namespace %q: %w",
				freightRef.Name,
				stage.Namespace,
				err,
			)
		}
		if f == nil {
			return nil, fmt.Errorf(
				"Freight %q in namespace %q not found",
				freightRef.Name,
				stage.Namespace,
			)
		}
		// Mark the Freight as an owner of the AnalysisRun
		ar.OwnerReferences = append(
			ar.OwnerReferences,
			metav1.OwnerReference{
				APIVersion:         gvk.GroupVersion().String(),
				Kind:               gvk.Kind,
				Name:               f.GetName(),
				UID:                f.GetUID(),
				BlockOwnerDeletion: ptr.To(true),
			},
		)
	}

	return ar, nil
}

func flattenTemplates(
	templates []*rollouts.AnalysisTemplate,
) (*rollouts.AnalysisTemplate, error) {
	metrics, err := flattenMetrics(templates)
	if err != nil {
		return nil, err
	}
	dryRunMetrics, err := flattenDryRunMetrics(templates)
	if err != nil {
		return nil, err
	}
	measurementRetentionMetrics, err :=
		flattenMeasurementRetentionMetrics(templates)
	if err != nil {
		return nil, err
	}
	args, err := flattenArgs(templates)
	if err != nil {
		return nil, err
	}
	return &rollouts.AnalysisTemplate{
		Spec: rollouts.AnalysisTemplateSpec{
			Metrics:              metrics,
			DryRun:               dryRunMetrics,
			MeasurementRetention: measurementRetentionMetrics,
			Args:                 args,
		},
	}, nil
}

func flattenMetrics(
	templates []*rollouts.AnalysisTemplate,
) ([]rollouts.Metric, error) {
	var combinedMetrics []rollouts.Metric
	for _, template := range templates {
		combinedMetrics = append(combinedMetrics, template.Spec.Metrics...)
	}
	metricMap := map[string]bool{}
	for _, metric := range combinedMetrics {
		if _, ok := metricMap[metric.Name]; ok {
			return nil, fmt.Errorf("two metrics have the same name '%s'", metric.Name)
		}
		metricMap[metric.Name] = true
	}
	return combinedMetrics, nil
}

func flattenDryRunMetrics(
	templates []*rollouts.AnalysisTemplate,
) ([]rollouts.DryRun, error) {
	var combinedDryRunMetrics []rollouts.DryRun
	for _, template := range templates {
		combinedDryRunMetrics = append(combinedDryRunMetrics, template.Spec.DryRun...)
	}
	err := validateDryRunMetrics(combinedDryRunMetrics)
	if err != nil {
		return nil, err
	}
	return combinedDryRunMetrics, nil
}

func flattenMeasurementRetentionMetrics(
	templates []*rollouts.AnalysisTemplate,
) ([]rollouts.MeasurementRetention, error) {
	var combinedMeasurementRetentionMetrics []rollouts.MeasurementRetention
	for _, template := range templates {
		combinedMeasurementRetentionMetrics =
			append(combinedMeasurementRetentionMetrics, template.Spec.MeasurementRetention...)
	}
	err := validateMeasurementRetentionMetrics(combinedMeasurementRetentionMetrics)
	if err != nil {
		return nil, err
	}
	return combinedMeasurementRetentionMetrics, nil
}

func flattenArgs(
	templates []*rollouts.AnalysisTemplate,
) ([]rollouts.Argument, error) {
	var combinedArgs []rollouts.Argument
	appendOrUpdate := func(newArg rollouts.Argument) error {
		for i, prevArg := range combinedArgs {
			if prevArg.Name == newArg.Name {
				// found two args with same name. verify they have the same value,
				// otherwise update the combined args with the new non-nil value
				if prevArg.Value != nil &&
					newArg.Value != nil &&
					*prevArg.Value != *newArg.Value {
					return fmt.Errorf(
						"Argument `%s` specified multiple times with different "+
							"values: '%s', '%s'",
						prevArg.Name,
						*prevArg.Value,
						*newArg.Value,
					)
				}
				// If previous arg value is already set (not nil), it should not be
				// replaced by a new arg with a nil value
				if prevArg.Value == nil {
					combinedArgs[i] = newArg
				}
				return nil
			}
		}
		combinedArgs = append(combinedArgs, newArg)
		return nil
	}
	for _, template := range templates {
		for _, arg := range template.Spec.Args {
			if err := appendOrUpdate(arg); err != nil {
				return nil, err
			}
		}
	}
	return combinedArgs, nil
}

func validateDryRunMetrics(dryRunMetrics []rollouts.DryRun) error {
	metricMap := map[string]bool{}
	for _, dryRun := range dryRunMetrics {
		if _, ok := metricMap[dryRun.MetricName]; ok {
			return fmt.Errorf(
				"two Dry-Run metric rules have the same name '%s'",
				dryRun.MetricName,
			)
		}
		metricMap[dryRun.MetricName] = true
	}
	return nil
}

func validateMeasurementRetentionMetrics(
	measurementRetentionMetrics []rollouts.MeasurementRetention,
) error {
	metricMap := map[string]bool{}
	for _, measurementRetention := range measurementRetentionMetrics {
		if _, ok := metricMap[measurementRetention.MetricName]; ok {
			return fmt.Errorf(
				"two Measurement Retention metric rules have the same name '%s'",
				measurementRetention.MetricName,
			)
		}
		metricMap[measurementRetention.MetricName] = true
	}
	return nil
}

// MergeArgs merges two lists of arguments, the incoming and the templates. If
// there are any unresolved arguments that have no value, raises an error.
func mergeArgs(
	incomingArgs []rollouts.Argument,
	templateArgs []rollouts.Argument,
) ([]rollouts.Argument, error) {
	newArgs := append(templateArgs[:0:0], templateArgs...)
	for _, arg := range incomingArgs {
		i := findArg(arg.Name, newArgs)
		if i >= 0 {
			if arg.Value != nil {
				newArgs[i].Value = arg.Value
			} else if arg.ValueFrom != nil {
				newArgs[i].ValueFrom = arg.ValueFrom
			}
		}
	}
	err := resolveArgs(newArgs)
	if err != nil {
		return nil, err
	}
	return newArgs, nil
}

func findArg(name string, args []rollouts.Argument) int {
	for i, arg := range args {
		if arg.Name == name {
			return i
		}
	}
	return -1
}

func resolveArgs(args []rollouts.Argument) error {
	for _, arg := range args {
		if arg.Value == nil && arg.ValueFrom == nil {
			return fmt.Errorf("args.%s was not resolved", arg.Name)
		}
	}
	return nil
}
