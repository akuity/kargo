package rbac

import (
	"fmt"
	"slices"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	allVerbs = []string{
		"create",
		"delete",
		"deletecollection",
		"get",
		"list",
		"patch",
		"update",
		"watch",
	}

	allStagesVerbs = append(allVerbs, "promote")
)

func init() {
	slices.Sort(allVerbs)
	slices.Sort(allStagesVerbs)
}

type PolicyRuleNormalizationOptions struct {
	// IncludeCustomVerbsInExpansion indicates whether custom verbs (like
	// "promote" for Stages) should be included in the expansion of the "*"
	// wildcard verb. This is optional because when normalizing PolicyRules with
	// the intent to create or update a Role, this is how we would like "*" to be
	// interpreted. However, when normalizing PolicyRules with the intent to
	// display them to the user, we will not want to expand "*" to include custom
	// verbs because Kubernetes own interpretation of "*" does not include custom
	// verbs.
	IncludeCustomVerbsInExpansion bool
}

// NormalizePolicyRules returns a predictably ordered slice of Normalized
// PolicyRules built from the the provided slice of PolicyRules. If the provided
// PolicyRules include wildcards in their APIGroups or Resources, this function
// will produce an error. Provided PolicyRules will be split or combined as
// necessary such that each rule references a single APIGroup, a single
// Resource, and at most a single ResourceName, with wildcard verbs expanded and
// all applicable verbs de-duplicated and sorted.
func NormalizePolicyRules(
	rules []rbacv1.PolicyRule,
	opts *PolicyRuleNormalizationOptions,
) ([]rbacv1.PolicyRule, error) {
	rulesMap, err := BuildNormalizedPolicyRulesMap(rules, opts)
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
	opts *PolicyRuleNormalizationOptions,
) (map[string]rbacv1.PolicyRule, error) {
	rulesMap := make(map[string]rbacv1.PolicyRule)
	for _, rule := range rules {
		// Preserve the group(s) the rule specifies rather than overwriting them;
		// an absent group means the core ("") group. (Previously the group was
		// replaced using a hardcoded table, which could not account for resource
		// types from Kargo's enterprise APIs or other CRDs. Callers that start from
		// a group-less request resolve the group up front -- see groupResolver.)
		groups := rule.APIGroups
		if len(groups) == 0 {
			groups = []string{""}
		}
		names := rule.ResourceNames
		if len(names) == 0 {
			names = []string{""}
		}
		for _, resource := range rule.Resources {
			if err := validateResourceTypeName(resource); err != nil {
				return nil, err
			}
			for _, group := range groups {
				for _, resourceName := range names {
					verbs := rule.Verbs
					key := RuleKey(group, resource, resourceName)
					if existingRule, ok := rulesMap[key]; ok {
						verbs = append(existingRule.Verbs, verbs...)
					}
					rulesMap[key] = buildRule(group, resource, resourceName, verbs, opts)
				}
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
	opts *PolicyRuleNormalizationOptions,
) rbacv1.PolicyRule {
	if opts == nil {
		opts = &PolicyRuleNormalizationOptions{}
	}
	// De-dupe verbs and expand verb wildcards
	verbsMap := make(map[string]struct{})
	for _, verb := range verbs {
		verb = strings.TrimSpace(verb)
		if verb == "*" {
			verbsMap = make(map[string]struct{})
			for _, verb := range allVerbsFor(resource, opts.IncludeCustomVerbsInExpansion) {
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

// validateResourceTypeName performs a best-effort check that resource is the plural form of a
// resource type. It deliberately does not validate against a fixed set of known types because
// otherwise we have to maintain a hardcoded table of all known resource types. Kubernetes resource
// names are conventionally the lowercase plural of a Kind and therefore end in "s"; an input that
// does not is almost certainly a singular fat-finger (e.g. "stage" instead of "stages"), so we
// reject it with a suggestion rather than silently creating a rule that matches nothing.
func validateResourceTypeName(resource string) error {
	// Subresources (e.g. "freights/status") are validated on their resource part.
	base, subresource, hasSubresource := strings.Cut(resource, "/")
	if base == "" {
		return apierrors.NewBadRequest(fmt.Sprintf("unrecognized resource type %q", resource))
	}
	if strings.HasSuffix(base, "s") {
		return nil
	}
	// Looks singular; suggest the plural form using Kubernetes' own guess.
	plural, _ := meta.UnsafeGuessKindToResource(schema.GroupVersionKind{Kind: base})
	suggestion := plural.Resource
	if hasSubresource {
		suggestion += "/" + subresource
	}
	return apierrors.NewBadRequest(fmt.Sprintf(
		"unrecognized resource type %q; did you mean %q?", resource, suggestion,
	))
}

// groupResolver resolves the API group that serves a (plural) resource type. It
// is consulted only when a request carries no explicit group -- the Grant/Revoke
// ResourceDetails flow, and group-less rules submitted via Create/Update (see
// resolveRuleGroups).
type groupResolver func(resource string) (group string, err error)

// newRESTMapperGroupResolver returns a groupResolver backed by a RESTMapper. The
// API server's RESTMapper is discovery-backed, so this resolves the group for
// any resource type the cluster actually serves -- core, CRDs, and Kargo's
// enterprise APIs alike -- without a hardcoded table or configuration. Known
// resources are served from the mapper's in-memory cache; it returns a
// bad-request error for an unknown or ambiguous resource type.
func newRESTMapperGroupResolver(mapper meta.RESTMapper) groupResolver {
	return func(resource string) (string, error) {
		// The group is a property of the resource, not the subresource.
		res, _, _ := strings.Cut(resource, "/")
		gvrs, err := mapper.ResourcesFor(schema.GroupVersionResource{Resource: res})
		if err != nil {
			if meta.IsNoMatchError(err) {
				return "", apierrors.NewBadRequest(fmt.Sprintf("unrecognized resource type %q", resource))
			}
			return "", fmt.Errorf("error resolving API group for resource type %q: %w", resource, err)
		}
		groups := make(map[string]struct{}, len(gvrs))
		for _, gvr := range gvrs {
			groups[gvr.Group] = struct{}{}
		}
		switch len(groups) {
		case 1:
			var group string
			// NOTE(thomastaylor312): I know this looks weird, but it's the only way to get the
			// single key out of a map in Go.
			for g := range groups {
				group = g
			}
			return group, nil
		case 0:
			return "", apierrors.NewBadRequest(fmt.Sprintf("unrecognized resource type %q", resource))
		default:
			return "", apierrors.NewBadRequest(fmt.Sprintf(
				"ambiguous resource type %q; it is served by multiple API groups", resource,
			))
		}
	}
}

// resolveRuleGroups fills in the API group of any rule that does not specify
// one, by resolving it from the rule's resource type(s). Rules that already
// specify a group are returned unchanged. A group-less rule that lists multiple
// resources is split so each resource is paired with its own resolved group,
// since different resources may live in different groups. A nil resolver leaves
// rules unchanged.
//
// This is the entry point for group-less rules submitted through Create/Update
// (notably the UI, which omits apiGroups and relies on the server to resolve
// them); NormalizePolicyRules itself only preserves groups, never resolves them.
func resolveRuleGroups(
	rules []rbacv1.PolicyRule,
	resolve groupResolver,
) ([]rbacv1.PolicyRule, error) {
	if resolve == nil {
		return rules, nil
	}
	resolved := make([]rbacv1.PolicyRule, 0, len(rules))
	for _, rule := range rules {
		if len(rule.APIGroups) > 0 || len(rule.Resources) == 0 {
			resolved = append(resolved, rule)
			continue
		}
		for _, resource := range rule.Resources {
			group, err := resolve(resource)
			if err != nil {
				return nil, err
			}
			r := *rule.DeepCopy()
			r.APIGroups = []string{group}
			r.Resources = []string{resource}
			resolved = append(resolved, r)
		}
	}
	return resolved, nil
}

func allVerbsFor(resourceType string, includeCustom bool) []string {
	if !includeCustom {
		return allVerbs
	}
	switch resourceType {
	case "stages":
		return allStagesVerbs
	default:
		return allVerbs
	}
}
