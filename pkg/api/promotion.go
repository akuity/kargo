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

	// maxPrefixForPromotionName is the maximum length of the descriptive prefix
	// of a generated Promotion name -- the Stage name, or the Stage and Target
	// names together -- before the name would exceed the Kubernetes resource
	// name limit of 253.
	maxPrefixForPromotionName = 253 -
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

// NewMinimalPromotionForOrigin constructs a Promotion that, unlike
// NewMinimalPromotion which is explicit about the exact Freight to promote,
// specifies only an origin. The mutating webhook resolves the origin to the
// auto-promotion candidate Freight at admission time.
func NewMinimalPromotionForOrigin(
	stage *kargoapi.Stage,
	origin kargoapi.FreightOrigin,
) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    stage.Namespace,
			GenerateName: "promo-",
		},
		Spec: kargoapi.PromotionSpec{
			Stage:  stage.Name,
			Origin: &origin,
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
	return generatePromotionStyleName(stageName, freight)
}

// generatePromotionStyleName generates a name of the form
// <prefix>.<ulid>.<short-hash> for a resource that promotes a piece of Freight.
// The embedded ULID makes lex order match creation order, which sorting logic
// elsewhere in Kargo relies on.
func generatePromotionStyleName(prefix, freight string) string {
	if prefix == "" || freight == "" {
		return ""
	}

	shortHash := freight
	if len(shortHash) > promotionShortHashLength {
		shortHash = shortHash[0:promotionShortHashLength]
	}

	if len(prefix) > maxPrefixForPromotionName {
		// Truncation can land on a separator or hyphen, neither of which a
		// Kubernetes resource name may end with.
		prefix = strings.TrimRight(
			prefix[0:maxPrefixForPromotionName],
			promotionNameSeparator+"-",
		)
	}

	parts := []string{prefix, ulid.Make().String(), shortHash}
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
		return comparePromotionNamesByULID(a.Name, b.Name)
	default:
		// Terminal Promotions are ordered in descending order based on the
		// ULID in the Promotion name. This ensures that the most recent
		// Promotion is at the top, limiting the number of Promotions which
		// have to be further inspected to collect the "new" Promotions.
		return comparePromotionNamesByULID(b.Name, a.Name)
	}
}

// comparePromotionNamesByULID orders two generated Promotion names by the ULID
// they embed, rather than by comparing the names whole.
//
// Comparing whole names is only equivalent to comparing ULIDs while everything
// to the left of the ULID is identical, which stopped being true once a
// Promotion could be named after the Target it promotes to. Two Promotions of
// the same Stage then differ at the Target segment, the comparison resolves
// there, and it never reaches the ULID at all -- so an older Promotion to
// "us-west" would sort after a newer one to "us-east", and a Stage would start
// promoting to the second while the first was still outstanding.
//
// Both generated name formats put the ULID in the second-to-last
// dot-separated segment:
//
//	<stage>.<ulid>.<short-hash>
//	<stage>.<target>.<ulid>.<short-hash>
//
// Indexing from the end is what makes that reliable: a Kubernetes name may
// contain dots, so a Stage or Target named "foo.bar" adds segments on the left,
// while the short hash never contains one.
//
// Names that do not carry a parseable ULID -- a Promotion created before Kargo
// generated names, or one named by hand -- are compared whole, preserving the
// previous behavior rather than declaring them equal and leaving the sort
// unstable.
//
// A ULID and an arbitrary string are not on the same comparison axis, so a pair
// where only one name carries a ULID is decided by which one does, not by
// comparing the names. Choosing the axis per pair is what would break
// transitivity, and slices.SortFunc gives no guarantees about the order it
// produces for a comparator that is not a strict weak ordering. Consider:
//
//	a = "b.7zzzzzzzzzzzzzzzzzzzzzzzzz.abc1234"
//	b = "m"
//	c = "z.00000000000000000000000000.abc1234"
//
// A per-pair axis compares a and b, and b and c, as whole names, giving a < b
// and b < c, but compares a and c as ULIDs, giving a > c. No order satisfies
// all three, so a single hand-named Promotion alongside generated ones would be
// enough to make the Stage's choice of current Promotion arbitrary.
func comparePromotionNamesByULID(a, b string) int {
	aULID, aOK := promotionNameULID(a)
	bULID, bOK := promotionNameULID(b)
	switch {
	case aOK && bOK:
		return strings.Compare(aULID, bULID)
	case aOK != bOK:
		// Bucket by whether a ULID is present. Names without one sort first,
		// which -- because the caller inverts the arguments for terminal
		// Promotions -- puts them ahead of generated names among non-terminal
		// Promotions and behind them among terminal ones.
		if aOK {
			return 1
		}
		return -1
	default:
		return strings.Compare(a, b)
	}
}

// promotionNameULID returns the ULID embedded in a generated Promotion name,
// and whether one was found. The returned value keeps the case it appears in:
// generated names are lowercased, and lowercasing a ULID preserves ordering,
// since Crockford base32 digits sort before its letters in ASCII either way.
func promotionNameULID(name string) (string, bool) {
	parts := strings.Split(name, promotionNameSeparator)
	// A generated name has at least a prefix, a ULID, and a hash.
	if len(parts) < 3 {
		return "", false
	}
	candidate := parts[len(parts)-2]
	if len(candidate) != ulid.EncodedSize {
		return "", false
	}
	if _, err := ulid.ParseStrict(candidate); err != nil {
		return "", false
	}
	return candidate, true
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
					"inflate task steps for task %q (%q): %w",
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
