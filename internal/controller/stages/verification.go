package stages

import (
	"context"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

func (r *reconciler) startVerification(
	ctx context.Context,
	stage *kargoapi.Stage,
) *kargoapi.VerificationInfo {
	if r.rolloutsClient == nil {
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhaseError,
			Message: "Rollouts integration is disabled on this controller; " +
				"cannot start verification",
		}
	}

	logger := logging.LoggerFromContext(ctx)

	namespace := r.getAnalysisRunNamespace(stage)

	// Check for existing AnalysisRun
	analysisRuns := rollouts.AnalysisRunList{}
	if err := r.listAnalysisRunsFn(
		ctx,
		&analysisRuns,
		&client.ListOptions{
			Namespace: namespace,
			LabelSelector: labels.SelectorFromSet(
				map[string]string{
					kargoapi.StageLabelKey:   stage.Name,
					kargoapi.FreightLabelKey: stage.Status.CurrentFreight.ID,
				},
			),
		},
	); err != nil {
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhaseError,
			Message: errors.Wrapf(
				err,
				"error listing AnalysisRuns for Stage %q and Freight %q in namespace %q",
				stage.Name,
				stage.Status.CurrentFreight.ID,
				namespace,
			).Error(),
		}
	}
	if len(analysisRuns.Items) > 0 {
		logger.Debug("AnalysisRun already exists for Freight")
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhase(analysisRuns.Items[0].Status.Phase),
			AnalysisRun: &kargoapi.AnalysisRunReference{
				Name:      analysisRuns.Items[0].Name,
				Namespace: analysisRuns.Items[0].Namespace,
				Phase:     string(analysisRuns.Items[0].Status.Phase),
			},
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
			return &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseError,
				Message: errors.Wrapf(
					err,
					"error getting AnalysisTemplate %q in namespace %q",
					templateRef.Name,
					stage.Namespace,
				).Error(),
			}
		}
		if template == nil {
			return &kargoapi.VerificationInfo{
				Phase: kargoapi.VerificationPhaseError,
				Message: errors.Errorf(
					"AnalysisTemplate %q in namespace %q not found",
					templateRef.Name,
					stage.Namespace,
				).Error(),
			}
		}
		templates[i] = template
	}

	run, err := r.buildAnalysisRunFn(stage, templates)
	if err != nil {
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhaseError,
			Message: errors.Wrapf(
				err,
				"error building AnalysisRun for Stage %q and Freight %q in namespace %q",
				stage.Name,
				stage.Status.CurrentFreight.ID,
				stage.Namespace,
			).Error(),
		}
	}
	if err := r.createAnalysisRunFn(ctx, run); err != nil {
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhaseError,
			Message: errors.Wrapf(
				err,
				"error creating AnalysisRun %q in namespace %q",
				run.Name,
				run.Namespace,
			).Error(),
		}
	}

	return &kargoapi.VerificationInfo{
		Phase: kargoapi.VerificationPhasePending,
		AnalysisRun: &kargoapi.AnalysisRunReference{
			Name:      run.Name,
			Namespace: run.Namespace,
		},
	}
}

func (r *reconciler) getVerificationInfo(
	ctx context.Context,
	stage *kargoapi.Stage,
) *kargoapi.VerificationInfo {
	if r.rolloutsClient == nil {
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhaseError,
			Message: "Rollouts integration is disabled on this controller; cannot " +
				"get verification info",
		}
	}

	namespace := r.getAnalysisRunNamespace(stage)
	analysisRunName := stage.Status.CurrentFreight.VerificationInfo.AnalysisRun.Name
	analysisRun, err := r.getAnalysisRunFn(
		ctx,
		r.rolloutsClient,
		types.NamespacedName{
			Namespace: namespace,
			Name:      analysisRunName,
		},
	)
	if err != nil {
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhaseError,
			Message: errors.Wrapf(
				err,
				"error getting AnalysisRun %q in namespace %q",
				analysisRunName,
				namespace,
			).Error(),
		}
	}
	if analysisRun == nil {
		return &kargoapi.VerificationInfo{
			Phase: kargoapi.VerificationPhaseError,
			Message: errors.Errorf(
				"AnalysisRun %q in namespace %q not found",
				analysisRunName,
				namespace,
			).Error(),
		}
	}
	return &kargoapi.VerificationInfo{
		Phase: kargoapi.VerificationPhase(analysisRun.Status.Phase),
		AnalysisRun: &kargoapi.AnalysisRunReference{
			Name:      analysisRun.Name,
			Namespace: analysisRun.Namespace,
			Phase:     string(analysisRun.Status.Phase),
		},
	}
}

// getAnalysisRunNamespace determines the namespace in which to create the
// AnalysisRun resources.
func (r *reconciler) getAnalysisRunNamespace(stage *kargoapi.Stage) string {
	if r.cfg.AnalysisRunsNamespace == "" {
		return stage.Namespace
	}
	return r.cfg.AnalysisRunsNamespace
}

func (r *reconciler) buildAnalysisRun(
	stage *kargoapi.Stage,
	templates []*rollouts.AnalysisTemplate,
) (*rollouts.AnalysisRun, error) {
	// maximum length of the stage name used in the promotion name prefix before it exceeds
	// kubernetes resource name limit of 253
	// 253 - 1 (.) - 26 (ulid) - 1 (.) - 7 (sha) = 218
	const maxStageNamePrefixLength = 218

	// Build the name of the AnalysisRun
	shortHash := stage.Status.CurrentFreight.ID
	if len(shortHash) > 7 {
		shortHash = shortHash[0:7]
	}
	shortStageName := stage.Name
	if len(stage.Name) > maxStageNamePrefixLength {
		shortStageName = shortStageName[0:maxStageNamePrefixLength]
	}
	analysisRunName := strings.ToLower(fmt.Sprintf("%s.%s.%s", shortStageName, ulid.Make(), shortHash))

	// Build the labels and annotations for the AnalysisRun
	var numLabels int
	var numAnnotations int
	if stage.Spec.Verification.AnalysisRunMetadata != nil {
		numLabels = len(stage.Spec.Verification.AnalysisRunMetadata.Labels)
		numAnnotations = len(stage.Spec.Verification.AnalysisRunMetadata.Annotations)
	}
	// Kargo will add up to three labels of its own, so size the map accordingly
	lbls := make(map[string]string, numLabels+3)
	annotations := make(map[string]string, numAnnotations)
	if stage.Spec.Verification.AnalysisRunMetadata != nil {
		for k, v := range stage.Spec.Verification.AnalysisRunMetadata.Labels {
			lbls[k] = v
		}
		for k, v := range stage.Spec.Verification.AnalysisRunMetadata.Annotations {
			annotations[k] = v
		}
	}
	lbls[kargoapi.StageLabelKey] = stage.Name
	lbls[kargoapi.FreightLabelKey] = stage.Status.CurrentFreight.ID
	if r.cfg.RolloutsControllerInstanceID != "" {
		lbls["argo-rollouts.argoproj.io/controller-instance-id"] = r.cfg.RolloutsControllerInstanceID
	}

	// Flatten templates into a single template
	template, err := flattenTemplates(templates)
	if err != nil {
		return nil, errors.Wrap(err, "error flattening templates")
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
		return nil, errors.Errorf("error merging arguments")
	}

	return &rollouts.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        analysisRunName,
			Namespace:   r.getAnalysisRunNamespace(stage),
			Labels:      lbls,
			Annotations: annotations,
		},
		Spec: rollouts.AnalysisRunSpec{
			Metrics:              template.Spec.Metrics,
			DryRun:               template.Spec.DryRun,
			MeasurementRetention: template.Spec.MeasurementRetention,
			Args:                 mergedArgs,
		},
	}, nil
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
