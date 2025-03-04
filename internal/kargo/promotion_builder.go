package kargo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/server/user"
)

const (
	// PromotionAliasSeparator is the separator used in the Promotion step alias
	// to separate the task alias from the step alias.
	PromotionAliasSeparator = "::"

	// nameSeparator is the separator used in the Promotion name.
	nameSeparator = "."

	// ulidLength is the length of the ULID string.
	ulidLength = ulid.EncodedSize

	// shortHashLength is the length of the short hash.
	shortHashLength = 7

	// maxStageNamePrefixLength is the maximum length of the Stage name
	// used in the Promotion name prefix before it exceeds the Kubernetes
	// resource name limit of 253.
	maxStageNamePrefixLength = 253 - len(nameSeparator) - ulidLength - len(nameSeparator) - shortHashLength
)

type PromotionBuilder struct {
	client client.Client
}

// NewPromotionBuilder creates a new PromotionBuilder with the given client.
func NewPromotionBuilder(c client.Client) *PromotionBuilder {
	return &PromotionBuilder{
		client: c,
	}
}

// Build creates a new Promotion for the Freight based on the PromotionTemplate
// of the given Stage.
func (b *PromotionBuilder) Build(
	ctx context.Context,
	stage kargoapi.Stage,
	freight string,
) (*kargoapi.Promotion, error) {
	if stage.Name == "" {
		return nil, fmt.Errorf("stage is required")
	}

	if stage.Spec.PromotionTemplate == nil {
		return nil, fmt.Errorf("stage %q has no promotion template", stage.Name)
	}

	if freight == "" {
		return nil, fmt.Errorf("freight is required")
	}

	// Build metadata
	annotations := make(map[string]string)
	if u, ok := user.InfoFromContext(ctx); ok {
		annotations[kargoapi.AnnotationKeyCreateActor] = api.FormatEventUserActor(u)
	}

	promotion := kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:        generatePromotionName(stage.Name, freight),
			Namespace:   stage.Namespace,
			Annotations: annotations,
		},
		Spec: kargoapi.PromotionSpec{
			Stage:   stage.Name,
			Freight: freight,
			Vars:    stage.Spec.PromotionTemplate.Spec.Vars,
			Steps:   stage.Spec.PromotionTemplate.Spec.Steps,
		},
	}
	return &promotion, nil
}

// InflateSteps inflates the Promotion steps by resolving any references to
// PromotionTasks and expanding them into their individual steps. The inflated
// steps are then set on the Promotion, replacing the original steps.
func (b *PromotionBuilder) InflateSteps(ctx context.Context, promo *kargoapi.Promotion) error {
	steps := make([]kargoapi.PromotionStep, 0, len(promo.Spec.Steps))
	for i, step := range promo.Spec.Steps {
		switch {
		case step.Task != nil:
			alias := step.GetAlias(i)
			taskSteps, err := b.inflateTaskSteps(
				ctx,
				promo.Namespace,
				alias,
				promo.Spec.Vars,
				step,
			)
			if err != nil {
				return fmt.Errorf("inflate tasks steps for task %q (%q): %w", step.Task.Name, alias, err)
			}
			steps = append(steps, taskSteps...)
		default:
			steps = append(steps, step)
		}
	}
	promo.Spec.Steps = steps
	return nil
}

// inflateTaskSteps inflates the PromotionSteps for the given PromotionStep
// that references a (Cluster)PromotionTask. The task is retrieved and its
// steps are inflated with the given task inputs.
func (b *PromotionBuilder) inflateTaskSteps(
	ctx context.Context,
	project, taskAlias string,
	promoVars []kargoapi.PromotionVariable,
	taskStep kargoapi.PromotionStep,
) ([]kargoapi.PromotionStep, error) {
	task, err := b.getTaskSpec(ctx, project, taskStep.Task)
	if err != nil {
		return nil, err
	}

	vars, err := promotionTaskVarsToStepVars(task.Vars, promoVars, taskStep.Vars)
	if err != nil {
		return nil, err
	}

	var steps []kargoapi.PromotionStep
	for i := range task.Steps {
		// Copy the step as-is.
		step := &task.Steps[i]

		// Ensures we have a unique alias for each step within the context of
		// the Promotion.
		step.As = generatePromotionTaskStepAlias(taskAlias, step.GetAlias(i))

		// With the variables validated and mapped, they are now available to
		// the Config of the step during the Promotion execution.
		step.Vars = append(vars, step.Vars...)

		// Append the inflated step to the list of steps.
		steps = append(steps, *step)
	}
	return steps, nil
}

// getTaskSpec retrieves the PromotionTaskSpec for the given PromotionTaskReference.
func (b *PromotionBuilder) getTaskSpec(
	ctx context.Context,
	project string,
	ref *kargoapi.PromotionTaskReference,
) (*kargoapi.PromotionTaskSpec, error) {
	var spec kargoapi.PromotionTaskSpec

	if ref == nil {
		return nil, errors.New("missing task reference")
	}

	switch ref.Kind {
	case "PromotionTask", "":
		task := &kargoapi.PromotionTask{}
		if err := b.client.Get(ctx, client.ObjectKey{Namespace: project, Name: ref.Name}, task); err != nil {
			return nil, err
		}
		spec = task.Spec
	case "ClusterPromotionTask":
		task := &kargoapi.ClusterPromotionTask{}
		if err := b.client.Get(ctx, client.ObjectKey{Name: ref.Name}, task); err != nil {
			return nil, err
		}
		spec = task.Spec
	default:
		return nil, fmt.Errorf("unknown task reference kind %q", ref.Kind)
	}

	return &spec, nil
}

// generatePromotionName generates a name for the Promotion by combining the
// Stage name, a ULID, and a short hash of the Freight.
//
// The name has the format of:
//
//	<stage-name>.<ulid>.<short-hash>
func generatePromotionName(stageName, freight string) string {
	if stageName == "" || freight == "" {
		return ""
	}

	shortHash := freight
	if len(shortHash) > shortHashLength {
		shortHash = shortHash[0:shortHashLength]
	}

	shortStageName := stageName
	if len(stageName) > maxStageNamePrefixLength {
		shortStageName = shortStageName[0:maxStageNamePrefixLength]
	}

	parts := []string{shortStageName, ulid.Make().String(), shortHash}
	return strings.ToLower(strings.Join(parts, nameSeparator))
}

// generatePromotionTaskStepAlias generates an alias for a PromotionTask step
// by combining the task alias and the step alias.
func generatePromotionTaskStepAlias(taskAlias, stepAlias string) string {
	return fmt.Sprintf("%s%s%s", taskAlias, PromotionAliasSeparator, stepAlias)
}

// promotionTaskVarsToStepVars validates the presence of the PromotionTask
// variables and maps them to variables which can be used by the inflated
// PromotionStep.
func promotionTaskVarsToStepVars(
	taskVars, promoVars, stepVars []kargoapi.PromotionVariable,
) ([]kargoapi.PromotionVariable, error) {
	// Promotion variables can be used to set (or override) the variables
	// required by the PromotionTask, but they are not inflated into the
	// variables for the step. This map is used to check if a variable is
	// set on the Promotion, to avoid overriding it with the default value
	// and to validate that the variable is set.
	promoVarsMap := make(map[string]struct{}, len(promoVars))
	for _, v := range promoVars {
		if v.Value != "" {
			promoVarsMap[v.Name] = struct{}{}
		}
	}

	// Step variables are inflated into the variables for the step. This map
	// is used to ensure all variables required by the PromotionTask without
	// a default value are set.
	stepVarsMap := make(map[string]struct{}, len(stepVars))
	for _, v := range stepVars {
		if v.Value != "" {
			stepVarsMap[v.Name] = struct{}{}
		}
	}

	var vars []kargoapi.PromotionVariable

	// Set the PromotionTask variable default values, but only if the variable
	// is not set on the Promotion.
	for _, v := range taskVars {
		// Variable is set on the Promotion, we do not need to set the default.
		if _, ok := promoVarsMap[v.Name]; ok {
			continue
		}

		// Set the variable if it has a default value.
		if v.Value != "" {
			vars = append(vars, v)
			continue
		}

		// If not, the variable must be set in the step variables.
		if _, ok := stepVarsMap[v.Name]; !ok {
			return nil, fmt.Errorf("missing value for variable %q", v.Name)
		}
	}

	// Set the step variables.
	vars = append(vars, stepVars...)

	return vars, nil
}
