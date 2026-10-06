package api

import (
	"fmt"
	"slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ArgoCDAppsOutputKey is the key under which Argo CD-aware promotion steps
// report the Argo CD Applications they resolved as step output. The value is a
// list of ArgoCDAppRef in the JSON-native form produced by
// ArgoCDAppRefsToOutput.
const ArgoCDAppsOutputKey = "apps"

const (
	argoCDAppNameKey      = "name"
	argoCDAppNamespaceKey = "namespace"
)

// argoCDStepKinds are the kinds of promotion steps that report the Argo CD
// Applications they resolved under ArgoCDAppsOutputKey.
var argoCDStepKinds = []string{"argocd-update", "argocd-wait"}

// ArgoCDAppRef identifies a single Argo CD Application associated with a Stage.
// A list of these is JSON-encoded into the AnnotationKeyArgoCDContext
// annotation, from which the UI builds deep links into Argo CD.
type ArgoCDAppRef struct {
	// Name is the name of the Argo CD Application.
	Name string `json:"name"`
	// Namespace is the namespace of the Argo CD Application.
	Namespace string `json:"namespace"`
}

// ArgoCDAppRefsToOutput converts refs into the representation an Argo CD-aware
// step must report them as.
//
// Step output is shared state, and shared state is deep-copied between steps
// with runtime.DeepCopyJSON, which panics on any value that is not one of the
// types JSON decodes to. A []ArgoCDAppRef would therefore take the Promotion
// controller down on the next step, so refs are flattened to the maps they
// would have been decoded into.
func ArgoCDAppRefsToOutput(refs []ArgoCDAppRef) []any {
	out := make([]any, len(refs))
	for i, ref := range refs {
		out[i] = map[string]any{
			argoCDAppNameKey:      ref.Name,
			argoCDAppNamespaceKey: ref.Namespace,
		}
	}
	return out
}

// argoCDAppRefsFromStepOutputs extracts ArgoCDAppRefs from the outputs of any
// Argo CD-aware steps among the ones provided. state is the aggregated state of
// a Promotion, keyed by step alias.
//
// The steps are expected to have been inflated -- i.e. every step has an alias
// assigned and any steps belonging to a PromotionTask have been expanded --
// which is the case for the steps of any Promotion admitted by the Promotion
// webhook.
//
// Malformed or absent output is not an error. Any step whose output does not
// have the expected shape simply contributes nothing.
func argoCDAppRefsFromStepOutputs(
	steps []kargoapi.PromotionStep,
	state map[string]any,
) []ArgoCDAppRef {
	var refs []ArgoCDAppRef
	for _, step := range steps {
		if !slices.Contains(argoCDStepKinds, step.Uses) || step.As == "" {
			continue
		}
		stepOutput, ok := state[step.As].(map[string]any)
		if !ok {
			continue
		}
		refs = append(refs, argoCDAppRefsFromRawApps(stepOutput[ArgoCDAppsOutputKey])...)
	}
	return refs
}

// argoCDAppRefsFromHealthChecks extracts ArgoCDAppRefs from the configuration of
// the provided health check steps.
//
// Argo CD-aware steps report the Applications they resolved as step output, but
// only as of a recent version of Kargo. Health check configuration remains the
// sole source of Argo CD context for any Promotion whose argocd-update step
// executed before the upgrade to such a version.
//
// Malformed or absent configuration is not an error. Any health check whose
// configuration does not have the expected shape simply contributes nothing.
func argoCDAppRefsFromHealthChecks(
	healthChecks []kargoapi.HealthCheckStep,
) []ArgoCDAppRef {
	var refs []ArgoCDAppRef
	for _, healthCheck := range healthChecks {
		refs = append(
			refs,
			argoCDAppRefsFromRawApps(healthCheck.GetConfig()[ArgoCDAppsOutputKey])...,
		)
	}
	return refs
}

// argoCDAppRefsFromRawApps converts the unstructured representation of a list
// of Argo CD Applications into ArgoCDAppRefs. Anything that does not have the
// expected shape is skipped.
func argoCDAppRefsFromRawApps(rawApps any) []ArgoCDAppRef {
	appList, ok := rawApps.([]any)
	if !ok {
		return nil
	}
	refs := make([]ArgoCDAppRef, 0, len(appList))
	for _, rawApp := range appList {
		app, ok := rawApp.(map[string]any)
		if !ok {
			continue
		}
		name, _ := app[argoCDAppNameKey].(string)
		namespace, _ := app[argoCDAppNamespaceKey].(string)
		refs = append(refs, ArgoCDAppRef{Name: name, Namespace: namespace})
	}
	return refs
}

// dedupeArgoCDAppRefs returns the provided ArgoCDAppRefs with duplicates and
// refs lacking a name removed, preserving the order of first occurrence.
//
// Deduplication matters because a single Argo CD Application is commonly
// reported by more than one contributor -- e.g. by both an argocd-update step's
// output and the health check that same step registered, or by an argocd-update
// step and a subsequent argocd-wait step targeting the same Application.
func dedupeArgoCDAppRefs(refs []ArgoCDAppRef) []ArgoCDAppRef {
	deduped := make([]ArgoCDAppRef, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.Name == "" {
			// A ref without a name is not actionable, and would break the UI's
			// parsing of the annotation for the Stage as a whole.
			continue
		}
		key := fmt.Sprintf("%s/%s", ref.Namespace, ref.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, ref)
	}
	return deduped
}
