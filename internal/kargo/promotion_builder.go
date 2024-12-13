package kargo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api/user"
)

const (
	// nameSeparator is the separator used in the Promotion name.
	nameSeparator = "."

	// aliasSeparator is the separator used in the Promotion step alias
	// to separate the task alias from the step alias.
	aliasSeparator = "::"

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
		annotations[kargoapi.AnnotationKeyCreateActor] = kargoapi.FormatEventUserActor(u)
	}

	// Build steps
	steps, err := b.buildSteps(ctx, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to build promotion steps: %w", err)
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
			Steps:   steps,
		},
	}
	return &promotion, nil
}

// buildSteps processes the Promotion steps from the PromotionTemplate of the
// given Stage. If a PromotionStep references a PromotionTask, the task is
// retrieved and its steps are inflated with the given task inputs.
func (b *PromotionBuilder) buildSteps(ctx context.Context, stage kargoapi.Stage) ([]kargoapi.PromotionStep, error) {
	steps := make([]kargoapi.PromotionStep, 0, len(stage.Spec.PromotionTemplate.Spec.Steps))
	for i, step := range stage.Spec.PromotionTemplate.Spec.Steps {
		switch {
		case step.Task != nil:
			alias := step.GetAlias(i)
			taskSteps, err := b.inflateTaskSteps(ctx, stage.Namespace, alias, step)
			if err != nil {
				return nil, fmt.Errorf("inflate tasks steps for task %q (%q): %w", step.Task.Name, alias, err)
			}
			steps = append(steps, taskSteps...)
		default:
			steps = append(steps, step)
		}
	}
	return steps, nil
}

// inflateTaskSteps inflates the PromotionSteps for the given PromotionStep
// that references a (Cluster)PromotionTask. The task is retrieved and its
// steps are inflated with the given task inputs.
func (b *PromotionBuilder) inflateTaskSteps(
	ctx context.Context,
	project, taskAlias string,
	taskStep kargoapi.PromotionStep,
) ([]kargoapi.PromotionStep, error) {
	task, err := b.getTaskSpec(ctx, project, taskStep.Task)
	if err != nil {
		return nil, err
	}

	inputs, err := promotionTaskInputsToStepInputs(task.Inputs, taskStep.Config)
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

		// With the inputs validated and mapped, they are now available to
		// the Config of the step during the Promotion execution.
		step.Inputs = inputs

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
	return fmt.Sprintf("%s%s%s", taskAlias, aliasSeparator, stepAlias)
}

// promotionTaskInputsToStepInputs validates the task step config against the task
// inputs, and maps the config to inputs for the inflated steps.
func promotionTaskInputsToStepInputs(
	taskInputs []kargoapi.PromotionTaskInput,
	stepConfig *apiextensionsv1.JSON,
) ([]kargoapi.PromotionStepInput, error) {
	if len(taskInputs) == 0 {
		return nil, nil
	}

	if stepConfig == nil {
		return nil, errors.New("missing step config")
	}

	config := make(map[string]any, len(taskInputs))
	if err := yaml.Unmarshal(stepConfig.Raw, &config); err != nil {
		return nil, fmt.Errorf("unmarshal step config: %w", err)
	}

	inputs := make([]kargoapi.PromotionStepInput, 0, len(taskInputs))
	for _, input := range taskInputs {
		iv := input.Default
		if cv, exists := config[input.Name]; exists {
			strVal, ok := cv.(string)
			if !ok {
				return nil, fmt.Errorf("input %q must be a string", input.Name)
			}
			iv = strVal
		}
		if iv == "" {
			return nil, fmt.Errorf("missing required input %q", input.Name)
		}
		inputs = append(inputs, kargoapi.PromotionStepInput{
			Name:  input.Name,
			Value: iv,
		})
	}
	return inputs, nil
}
