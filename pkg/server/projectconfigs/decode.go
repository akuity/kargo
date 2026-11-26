package projectconfigs

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// requiredSpecKeys lists, for every object under spec that the ProjectConfig
// CRD schema gives a "required" list, the path to that object and its required
// keys. A path is relative to spec; a "[]" suffix means "each element of this
// list". TestDecodeEnforcesEveryCRDRequiredKey checks this table against the
// CRD.
var requiredSpecKeys = []struct {
	path string
	keys []string
}{
	{path: "promotionPolicies[].stageSelector.matchExpressions[]", keys: []string{"key", "operator"}},
	{path: "webhookReceivers[]", keys: []string{"name"}},
	{path: "webhookReceivers[].bitbucket", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].dockerhub", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].github", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].gitlab", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].harbor", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].quay", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].artifactory", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].azure", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].gitea", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].generic", keys: []string{"secretRef"}},
	{path: "webhookReceivers[].generic.actions[]", keys: []string{"action"}},
	{
		path: "webhookReceivers[].generic.actions[].targetSelectionCriteria[]",
		keys: []string{"kind"},
	},
	{
		path: "webhookReceivers[].generic.actions[].targetSelectionCriteria[].labelSelector.matchExpressions[]",
		keys: []string{"key", "operator"},
	},
	{
		path: "webhookReceivers[].generic.actions[].targetSelectionCriteria[].indexSelector.matchIndices[]",
		keys: []string{"key", "operator", "value"},
	},
	{path: "freightLinks[]", keys: []string{"title", "url"}},
	{path: "stageLinks[]", keys: []string{"title", "url"}},
	{path: "promotionWindows[]", keys: []string{"kind", "name"}},
	{path: "promotionWindows[].stageSelector.matchExpressions[]", keys: []string{"key", "operator"}},
	{path: "promotionWindows[].projectSelector.matchExpressions[]", keys: []string{"key", "operator"}},
}

// Decode converts a ProjectConfig decoded from JSON into a generic map (as
// encoding/json produces) into a typed ProjectConfig. Before converting, it
// enforces the rules that depend on whether a key is present, which the
// typed form cannot express: every key the CRD schema marks required under
// spec, and the PromotionPolicy one-of rule for an explicitly empty stage.
// Errors use the field paths the CRD reports today.
//
// Field names are matched case-sensitively, as Kubernetes does. Unknown
// fields are dropped, not rejected, matching Kubernetes pruning. Callers
// should call Validate on the result when Decode returns no errors.
func Decode(obj map[string]any) (*kargoapi.ProjectConfig, field.ErrorList) {
	var errs field.ErrorList
	specPath := field.NewPath("spec")
	if spec, ok := obj["spec"].(map[string]any); ok {
		for _, req := range requiredSpecKeys {
			errs = append(errs, requireKeys(
				spec,
				specPath,
				strings.Split(req.path, "."),
				req.keys,
			)...)
		}
		errs = append(errs, validateExplicitStages(specPath.Child("promotionPolicies"), spec)...)
	}

	cfg := &kargoapi.ProjectConfig{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj, cfg); err != nil {
		return nil, append(errs, field.Invalid(
			nil,
			field.OmitValueType{},
			fmt.Sprintf("error decoding ProjectConfig: %v", err),
		))
	}
	return cfg, errs
}

// requireKeys walks obj along tokens (a "[]" suffix descends into each list
// element) and reports a Required error for each key absent, or null, in every
// object it reaches. Values of an unexpected type are skipped; conversion
// reports them.
func requireKeys(
	obj map[string]any,
	f *field.Path,
	tokens []string,
	keys []string,
) field.ErrorList {
	if len(tokens) == 0 {
		var errs field.ErrorList
		for _, key := range keys {
			if obj[key] == nil {
				errs = append(errs, field.Required(f.Child(key), ""))
			}
		}
		return errs
	}
	name, isList := strings.CutSuffix(tokens[0], "[]")
	if !isList {
		child, ok := obj[name].(map[string]any)
		if !ok {
			return nil
		}
		return requireKeys(child, f.Child(name), tokens[1:], keys)
	}
	items, ok := obj[name].([]any)
	if !ok {
		return nil
	}
	var errs field.ErrorList
	for i, item := range items {
		if child, ok := item.(map[string]any); ok {
			errs = append(errs, requireKeys(child, f.Child(name).Index(i), tokens[1:], keys)...)
		}
	}
	return errs
}

// validateExplicitStages covers a PromotionPolicy whose stage key is present
// but empty, which the typed form cannot tell from an absent stage. Today the
// CRD rejects it by the stage pattern and, when stageSelector is also present,
// by the CEL one-of rule, which tests key presence.
func validateExplicitStages(f *field.Path, spec map[string]any) field.ErrorList {
	policies, ok := spec["promotionPolicies"].([]any)
	if !ok {
		return nil
	}
	var errs field.ErrorList
	for i, item := range policies {
		policy, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if stage, ok := policy["stage"].(string); !ok || stage != "" {
			continue
		}
		errs = append(errs, validatePattern(f.Index(i).Child("stage"), "")...)
		if policy["stageSelector"] != nil {
			errs = append(errs, field.Invalid(f.Index(i), field.OmitValueType{}, policyOneOfMessage))
		}
	}
	return errs
}
