package rbac

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

var allVerbs = []string{
	"create",
	"delete",
	"deletecollection",
	"get",
	"list",
	"patch",
	"update",
	"watch",
}

// NormalizePolicyRules returns a predictably ordered slice of Normalized
// PolicyRules built from the the provided slice of PolicyRules. If the provided
// PolicyRules include wildcards in their APIGroups or Resources, this function
// will produce an error. Provided PolicyRules will be split or combined as
// necessary such that each rule references a single APIGroup, a single
// Resource, and at most a single ResourceName, with wildcard verbs expanded and
// all applicable verbs de-duplicated and sorted.
func NormalizePolicyRules(rules []rbacv1.PolicyRule) ([]rbacv1.PolicyRule, error) {
	rulesMap, err := BuildNormalizedPolicyRulesMap(rules)
	if err != nil {
		return nil, err
	}
	return PolicyRulesMapToSlice(rulesMap), nil
}

// BuildNormalizedPolicyRulesMap returns a map of Normalized PolicyRules built
// from the the provided slice of PolicyRules. The map is keyed by the
// combination of group, resource, and if applicable, resourceName, as derived
// by the RuleKey() function. If the provided PolicyRules include wildcards in
// their APIGroups or Resources, this function will produce an error. Provided
// PolicyRules will be split or combined as necessary such that each rule
// references a single APIGroup, a single Resource, and at most a single
// ResourceName, with wildcard verbs expanded and all applicable verbs
// de-duplicated and sorted.
func BuildNormalizedPolicyRulesMap(
	rules []rbacv1.PolicyRule,
) (map[string]rbacv1.PolicyRule, error) {
	rulesMap := make(map[string]rbacv1.PolicyRule)
	for _, rule := range rules {
		for _, resource := range rule.Resources {
			if err := validateResourceTypeName(resource); err != nil {
				return nil, err
			}
			// We ignore the group in the rule and use what we know to be the correct
			// group for the resource type.
			group := getGroupName(resource)
			if len(rule.ResourceNames) == 0 {
				rule.ResourceNames = append(rule.ResourceNames, "")
			}
			for _, resourceName := range rule.ResourceNames {
				verbs := rule.Verbs
				key := RuleKey(group, resource, resourceName)
				if existingRule, ok := rulesMap[key]; ok {
					verbs = append(existingRule.Verbs, verbs...)
				}
				rulesMap[key] = buildRule(group, resource, resourceName, verbs)
			}
		}
	}
	return rulesMap, nil
}

// PolicyRulesMapToSlice returns a slice of PolicyRules built from the provided
// map.
func PolicyRulesMapToSlice(rulesMap map[string]rbacv1.PolicyRule) []rbacv1.PolicyRule {
	ruleKeys := make([]string, 0, len(rulesMap))
	for key := range rulesMap {
		ruleKeys = append(ruleKeys, key)
	}
	slices.Sort(ruleKeys)
	rules := make([]rbacv1.PolicyRule, len(ruleKeys))
	for i, key := range ruleKeys {
		rules[i] = rulesMap[key]
	}
	return rules
}

// RuleKey returns a single string that combines the provided group, resource,
// and if non-empty, resourceName. This key is suitable for use as a key in a
// map of RBAC PolicyRules.
func RuleKey(group, resource, resourceName string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		group = "core"
	}
	resource = strings.TrimSpace(resource)
	tokens := []string{group, resource}
	resourceName = strings.TrimSpace(resourceName)
	if resourceName != "" {
		tokens = append(tokens, resourceName)
	}
	return strings.Join(tokens, "/")
}

// buildRule builds a single rule from the provided group, resource,
// resourceName, and verbs. Wildcards in verbs are expanded and verbs are also
// de-duplicated and sorted.
func buildRule(
	group string,
	resource string,
	resourceName string,
	verbs []string,
) rbacv1.PolicyRule {
	// De-dupe verbs and expand verb wildcards
	verbsMap := make(map[string]struct{})
	for _, verb := range verbs {
		verb = strings.TrimSpace(verb)
		if verb == "*" {
			verbsMap = make(map[string]struct{})
			for _, verb := range allVerbs {
				verbsMap[verb] = struct{}{}
			}
		} else {
			verbsMap[verb] = struct{}{}
		}
	}
	verbs = make([]string, 0, len(verbsMap))
	for verb := range verbsMap {
		verbs = append(verbs, verb)
	}
	// Sort them
	slices.Sort(verbs)
	// Build the rule
	rule := rbacv1.PolicyRule{
		APIGroups: []string{strings.TrimSpace(group)},
		Resources: []string{strings.TrimSpace(resource)},
		Verbs:     verbs,
	}
	resourceName = strings.TrimSpace(resourceName)
	if resourceName != "" {
		rule.ResourceNames = []string{resourceName}
	}
	return rule
}

// nolint: goconst
func validateResourceTypeName(resource string) error {
	switch resource {
	case "analysisruns", "analysistemplates", "events", "freights", "freights/status", "roles",
		"rolebindings", "promotions", "secrets", "serviceaccounts", "stages", "warehouses":
		return nil
	case "analysisrun", "analysistemplate", "event", "freight", "role",
		"rolebinding", "promotion", "secret", "serviceaccount", "stage", "warehouse":
		return kubeerr.NewBadRequest(
			fmt.Sprintf(`unrecognized resource type %q; did you mean "%ss"?`, resource, resource),
		)
	case "freight/status":
		return kubeerr.NewBadRequest(
			`unrecognized resource type "freight/status"; did you mean "freights/status"?`,
		)
	default:
		return kubeerr.NewBadRequest(fmt.Sprintf(`unrecognized resource type %q`, resource))
	}
}

// nolint: goconst
func getGroupName(resourceType string) string {
	// resourceType must already be validated
	switch resourceType {
	case "events", "secrets", "serviceaccounts":
		return ""
	case "rolebindings", "roles":
		return rbacv1.SchemeGroupVersion.Group
	case "freights", "freights/status", "promotions", "stages", "warehouses":
		return kargoapi.GroupVersion.Group
	case "analysisruns", "analysistemplates":
		return rolloutsapi.GroupVersion.Group
	default:
		return "" // If the resourceType was validated, this will never happen
	}
}
