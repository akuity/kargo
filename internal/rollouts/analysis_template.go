package rollouts

import (
	"fmt"

	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

// flattenTemplates combines multiple analysis templates into a single
// template. It merges metrics, dry-run metrics, measurement retention
// metrics, and arguments.
func flattenTemplates(templates []*rolloutsapi.AnalysisTemplate) (*rolloutsapi.AnalysisTemplate, error) {
	metrics, err := flattenMetrics(templates)
	if err != nil {
		return nil, fmt.Errorf("flatten metrics: %w", err)
	}

	dryRun, err := flattenDryRunMetrics(templates)
	if err != nil {
		return nil, fmt.Errorf("flatten dry-run metrics: %w", err)
	}

	retention, err := flattenMeasurementRetentionMetrics(templates)
	if err != nil {
		return nil, fmt.Errorf("flatten measurement retention metrics: %w", err)
	}

	args, err := flattenArgs(templates)
	if err != nil {
		return nil, fmt.Errorf("flatten arguments: %w", err)
	}

	return &rolloutsapi.AnalysisTemplate{
		Spec: rolloutsapi.AnalysisTemplateSpec{
			Metrics:              metrics,
			DryRun:               dryRun,
			MeasurementRetention: retention,
			Args:                 args,
		},
	}, nil
}

// flattenMetrics combines metrics from multiple templates while ensuring
// unique names.
func flattenMetrics(templates []*rolloutsapi.AnalysisTemplate) ([]rolloutsapi.Metric, error) {
	metrics := make([]rolloutsapi.Metric, 0, len(templates))
	seen := make(map[string]struct{})

	for _, tmpl := range templates {
		for _, metric := range tmpl.Spec.Metrics {
			if _, exists := seen[metric.Name]; exists {
				return nil, fmt.Errorf("duplicate metric name: %q", metric.Name)
			}
			seen[metric.Name] = struct{}{}
			metrics = append(metrics, metric)
		}
	}

	return metrics, nil
}

// flattenDryRunMetrics combines dry-run metrics from multiple templates.
func flattenDryRunMetrics(templates []*rolloutsapi.AnalysisTemplate) ([]rolloutsapi.DryRun, error) {
	dryRun := make([]rolloutsapi.DryRun, 0, len(templates))

	for _, tmpl := range templates {
		dryRun = append(dryRun, tmpl.Spec.DryRun...)
	}

	if err := validateUniqueDryRunMetrics(dryRun); err != nil {
		return nil, fmt.Errorf("validate dry-run metrics: %w", err)
	}

	return dryRun, nil
}

// flattenMeasurementRetentionMetrics combines measurement retention metrics
// from multiple templates.
func flattenMeasurementRetentionMetrics(
	templates []*rolloutsapi.AnalysisTemplate,
) ([]rolloutsapi.MeasurementRetention, error) {
	retention := make([]rolloutsapi.MeasurementRetention, 0, len(templates))

	for _, tmpl := range templates {
		retention = append(retention, tmpl.Spec.MeasurementRetention...)
	}

	if err := validateUniqueMeasurementRetentionMetrics(retention); err != nil {
		return nil, fmt.Errorf("validate measurement retention metrics: %w", err)
	}

	return retention, nil
}

// flattenArgs combines arguments from multiple templates, handling conflicts
// and updates.
func flattenArgs(templates []*rolloutsapi.AnalysisTemplate) ([]rolloutsapi.Argument, error) {
	var combinedArgs []rolloutsapi.Argument

	updateOrAppend := func(newArg rolloutsapi.Argument) error {
		for i, existingArg := range combinedArgs {
			if existingArg.Name == newArg.Name {
				if err := validateAndUpdateArg(&existingArg, newArg); err != nil {
					return err
				}
				combinedArgs[i] = existingArg
				return nil
			}
		}
		combinedArgs = append(combinedArgs, newArg)
		return nil
	}

	for _, tmpl := range templates {
		for _, arg := range tmpl.Spec.Args {
			if err := updateOrAppend(arg); err != nil {
				return nil, err
			}
		}
	}

	return combinedArgs, nil
}

// validateAndUpdateArg checks for conflicts between arguments and updates the
// existing argument if needed.
func validateAndUpdateArg(existing *rolloutsapi.Argument, updated rolloutsapi.Argument) error {
	if existing.Value != nil && updated.Value != nil && *existing.Value != *updated.Value {
		return fmt.Errorf(
			"conflicting values for argument %q: %q and %q",
			existing.Name, *existing.Value, *updated.Value,
		)
	}

	// Update existing argument only if it has no value and new argument has
	// one.
	if existing.Value == nil && updated.Value != nil {
		existing.Value = updated.Value
	}

	return nil
}

// validateUniqueDryRunMetrics ensures no duplicate metric names exist in dry-run
// metrics.
func validateUniqueDryRunMetrics(metrics []rolloutsapi.DryRun) error {
	seen := make(map[string]struct{})

	for _, metric := range metrics {
		if _, exists := seen[metric.MetricName]; exists {
			return fmt.Errorf("duplicate dry-run metric name: %q", metric.MetricName)
		}
		seen[metric.MetricName] = struct{}{}
	}

	return nil
}

// validateUniqueMeasurementRetentionMetrics ensures no duplicate metric names
// exist in measurement retention metrics.
func validateUniqueMeasurementRetentionMetrics(metrics []rolloutsapi.MeasurementRetention) error {
	seen := make(map[string]struct{})

	for _, metric := range metrics {
		if _, exists := seen[metric.MetricName]; exists {
			return fmt.Errorf("duplicate measurement retention metric name: %q", metric.MetricName)
		}
		seen[metric.MetricName] = struct{}{}
	}

	return nil
}

// mergeArgs combines incoming arguments with template arguments, giving
// precedence to incoming values. It returns a new slice containing the merged
// arguments or an error if argument resolution fails.
func mergeArgs(incomingArgs, templateArgs []rolloutsapi.Argument) ([]rolloutsapi.Argument, error) {
	// Create a copy of template args with exact capacity needed
	merged := make([]rolloutsapi.Argument, len(templateArgs))
	copy(merged, templateArgs)

	// Update or append incoming args
	for _, incoming := range incomingArgs {
		idx := findArgIndex(merged, incoming.Name)
		if idx >= 0 {
			updateArg(&merged[idx], incoming)
		}
	}

	if err := validateArgsResolution(merged); err != nil {
		return nil, fmt.Errorf("validate merged arguments: %w", err)
	}

	return merged, nil
}

// updateArg modifies the target argument with values from the source argument.
// It updates either Value or ValueFrom, giving precedence to Value if both are
// present.
func updateArg(target *rolloutsapi.Argument, source rolloutsapi.Argument) {
	if source.Value != nil {
		target.Value = source.Value
	} else if source.ValueFrom != nil {
		target.ValueFrom = source.ValueFrom
	}
}

// findArgIndex returns the index of the argument with the given name in the
// slice, or -1 if not found.
func findArgIndex(args []rolloutsapi.Argument, name string) int {
	for i := range args {
		if args[i].Name == name {
			return i
		}
	}
	return -1
}

// validateArgsResolution ensures all arguments have either Value or ValueFrom
// set. It returns an error for the first unresolved argument found.
func validateArgsResolution(args []rolloutsapi.Argument) error {
	for i := range args {
		if args[i].Value == nil && args[i].ValueFrom == nil {
			return fmt.Errorf("unresolved argument %q: neither Value nor ValueFrom is set", args[i].Name)
		}
	}
	return nil
}
