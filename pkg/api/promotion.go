package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/user"
)

// PromotionAliasSeparator is the separator used in the alias of an inflated
// PromotionTask step to separate the task alias from the step alias.
const PromotionAliasSeparator = "::"

const (
	// promotionNameSeparator separates components of a Promotion name.
	promotionNameSeparator = "."

	// promotionShortHashLength is the length of the short Freight hash
	// embedded in a generated Promotion name.
	promotionShortHashLength = 7

	// maxStageNamePrefixForPromotionName is the maximum length of the Stage
	// name used as the prefix of a generated Promotion name before it would
	// exceed the Kubernetes resource name limit of 253.
	maxStageNamePrefixForPromotionName = 253 -
		len(promotionNameSeparator) - ulid.EncodedSize -
		len(promotionNameSeparator) - promotionShortHashLength
)

// NewMinimalPromotion constructs a Promotion containing only the fields that
// callers (API server endpoints, the Stage controller's auto-promote loop) are
// responsible for setting. The Promotion defaulting webhook fills in the rest:
// name, steps copied from the Stage's PromotionTemplate, etc.
func NewMinimalPromotion(
	stage *kargoapi.Stage,
	freightName string,
) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: stage.Namespace,
			// The defaulting webhook overwrites this. We set it here only so that the
			// Kubernetes API server has a name to work with before admission runs.
			GenerateName: "promo-",
		},
		Spec: kargoapi.PromotionSpec{
			Stage:   stage.Name,
			Freight: freightName,
		},
	}
}

// GeneratePromotionName generates a name for a Promotion by combining the
// Stage name, a ULID, and a short hash of the Freight.
//
// The name has the format of:
//
//	<stage-name>.<ulid>.<short-hash>
//
// Promotion sorting and comparison logic elsewhere in Kargo relies on names
// in this format -- the embedded ULID makes lex order match creation order.
// Callers that need a Promotion name should always use this function.
func GeneratePromotionName(stageName, freight string) string {
	if stageName == "" || freight == "" {
		return ""
	}

	shortHash := freight
	if len(shortHash) > promotionShortHashLength {
		shortHash = shortHash[0:promotionShortHashLength]
	}

	shortStageName := stageName
	if len(stageName) > maxStageNamePrefixForPromotionName {
		shortStageName = shortStageName[0:maxStageNamePrefixForPromotionName]
	}

	parts := []string{shortStageName, ulid.Make().String(), shortHash}
	return strings.ToLower(strings.Join(parts, promotionNameSeparator))
}

// GetPromotion returns a pointer to the Promotion resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func GetPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.Promotion, error) {
	promo := kargoapi.Promotion{}
	if err := c.Get(ctx, namespacedName, &promo); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"error getting Promotion %q in namespace %q: %w",
			namespacedName.Name,
			namespacedName.Namespace,
			err,
		)
	}
	return &promo, nil
}

// RefreshPromotion forces reconciliation of a Promotion by setting an annotation
// on the Promotion, causing the controller to reconcile it. Currently, the
// annotation value is the timestamp of the request, but might in the
// future include additional metadata/context necessary for the request.
func RefreshPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
) (*kargoapi.Promotion, error) {
	promo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespacedName.Namespace,
			Name:      namespacedName.Name,
		},
	}
	if err := patchAnnotation(ctx, c, promo, kargoapi.AnnotationKeyRefresh, time.Now().Format(time.RFC3339)); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}
	return promo, nil
}

// AbortPromotion forces aborting the Promotion by setting an annotation on the
// object, causing the controller to abort the Promotion. The annotation value
// is the action to take on the Promotion to abort it.
func AbortPromotion(
	ctx context.Context,
	c client.Client,
	namespacedName types.NamespacedName,
	action kargoapi.AbortAction,
) error {
	promotion, err := GetPromotion(ctx, c, namespacedName)
	if err != nil || promotion == nil {
		if promotion == nil {
			// nolint:staticcheck
			err = fmt.Errorf(
				"Promotion %q in namespace %q not found",
				namespacedName.Name,
				namespacedName.Namespace,
			)
		}
		return err
	}

	if promotion.Status.Phase.IsTerminal() {
		// The Promotion is already in a terminal phase, so we can skip the
		// abort request.
		return nil
	}

	ar := kargoapi.AbortPromotionRequest{
		Action: action,
	}
	// Put actor information to track on the controller side
	if u, ok := user.InfoFromContext(ctx); ok {
		ar.Actor = FormatEventUserActor(u)
	}
	return patchAnnotation(ctx, c, promotion, kargoapi.AnnotationKeyAbort, ar.String())
}

// ComparePromotionByPhaseAndCreationTime compares two Promotions by their
// phase and creation timestamp. It returns a negative value if Promotion `a`
// should come before Promotion `b`, a positive value if Promotion `a` should
// come after Promotion `b`, or zero if they are considered equal for sorting
// purposes. It can be used in conjunction with slices.SortFunc to sort a list
// of Promotions.
//
// The order of Promotions is as follows:
//  1. Running Promotions
//  2. Non-terminal Promotions (ordered by ULID in ascending order)
//  3. Terminal Promotions (ordered by ULID in descending order)
func ComparePromotionByPhaseAndCreationTime(a, b kargoapi.Promotion) int {
	// Compare the phases of the Promotions first.
	if phaseCompare := ComparePromotionPhase(a.Status.Phase, b.Status.Phase); phaseCompare != 0 {
		return phaseCompare
	}

	switch {
	case !a.Status.Phase.IsTerminal():
		// Non-terminal Promotions are ordered in ascending order based on the
		// ULID in the Promotion name. This ensures that the Promotion which
		// was (or will be) enqueued first is at the top.
		return strings.Compare(a.Name, b.Name)
	default:
		// Terminal Promotions are ordered in descending order based on the
		// ULID in the Promotion name. This ensures that the most recent
		// Promotion is at the top, limiting the number of Promotions which
		// have to be further inspected to collect the "new" Promotions.
		return strings.Compare(b.Name, a.Name)
	}
}

// ComparePromotionPhase compares two Promotion phases. It returns a negative
// value if phase `a` should come before phase `b`, a positive value if phase
// `a` should come after phase `b`, or zero if they are considered equal for
// sorting purposes. It can be used in combination with slices.SortFunc to sort
// a list of Promotion phases.
//
// The order of Promotion phases is as follows:
//  1. Running
//  2. Non-terminal phases
//  3. Terminal phases
func ComparePromotionPhase(a, b kargoapi.PromotionPhase) int {
	aRunning, bRunning := a == kargoapi.PromotionPhaseRunning, b == kargoapi.PromotionPhaseRunning
	aTerminal, bTerminal := a.IsTerminal(), b.IsTerminal()

	// NB: The order of the cases here is important, as "Running" is a special
	// case that should always come before any other phase.
	switch {
	case aRunning && !bRunning:
		return -1
	case !aRunning && bRunning:
		return 1
	case !aTerminal && bTerminal:
		return -1
	case aTerminal && !bTerminal:
		return 1
	default:
		return 0
	}
}

// IsCurrentStepRunning returns true if the promotion is in a running state and the current step is also running.
func IsCurrentStepRunning(promo *kargoapi.Promotion) bool {
	return promo.Status.Phase == kargoapi.PromotionPhaseRunning &&
		int64(len(promo.Status.StepExecutionMetadata)) == promo.Status.CurrentStep+1 &&
		promo.Status.StepExecutionMetadata[promo.Status.CurrentStep].Status == kargoapi.PromotionStepStatusRunning
}

// InflateSteps inflates the given Promotion's steps in place by resolving any
// references to (Cluster)PromotionTasks and expanding them into their
// individual steps.
func InflateSteps(
	ctx context.Context,
	c client.Client,
	promo *kargoapi.Promotion,
) error {
	steps := make([]kargoapi.PromotionStep, 0, len(promo.Spec.Steps))
	for i, step := range promo.Spec.Steps {
		switch {
		case step.Task != nil:
			alias := step.GetAlias(i)
			taskSteps, err := inflateTaskSteps(
				ctx,
				c,
				promo.Namespace,
				alias,
				promo.Spec.Vars,
				step,
			)
			if err != nil {
				return fmt.Errorf(
					"inflate tasks steps for task %q (%q): %w",
					step.Task.Name, alias, err,
				)
			}
			steps = append(steps, taskSteps...)
		default:
			step.As = step.GetAlias(i)
			steps = append(steps, step)
		}
	}
	promo.Spec.Steps = steps
	return nil
}

// inflateTaskSteps inflates the PromotionSteps for the given PromotionStep
// that references a (Cluster)PromotionTask. The task is retrieved and its
// steps are inflated with the given task inputs.
func inflateTaskSteps(
	ctx context.Context,
	c client.Client,
	project, taskAlias string,
	promoVars []kargoapi.ExpressionVariable,
	taskStep kargoapi.PromotionStep,
) ([]kargoapi.PromotionStep, error) {
	task, err := getPromotionTaskSpec(ctx, c, project, taskStep.Task)
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

// getPromotionTaskSpec retrieves the PromotionTaskSpec for the given
// PromotionTaskReference.
func getPromotionTaskSpec(
	ctx context.Context,
	c client.Client,
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
		if err := c.Get(ctx, client.ObjectKey{Namespace: project, Name: ref.Name}, task); err != nil {
			return nil, err
		}
		spec = task.Spec
	case "ClusterPromotionTask":
		task := &kargoapi.ClusterPromotionTask{}
		if err := c.Get(ctx, client.ObjectKey{Name: ref.Name}, task); err != nil {
			return nil, err
		}
		spec = task.Spec
	default:
		return nil, fmt.Errorf("unknown task reference kind %q", ref.Kind)
	}

	return &spec, nil
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
	taskVars, promoVars, stepVars []kargoapi.ExpressionVariable,
) ([]kargoapi.ExpressionVariable, error) {
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

	var vars []kargoapi.ExpressionVariable

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
