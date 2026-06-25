package rbac

import (
	"context"
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/server/kubernetes"
)

// splitResourceType splits a resource type such as "freights/status" into its
// resource ("freights") and subresource ("status"). Resource types without a
// "/" have an empty subresource.
func splitResourceType(resourceType string) (resource, subresource string) {
	resource, subresource, _ = strings.Cut(resourceType, "/")
	return resource, subresource
}

// verifyRulesNotEscalating returns an error unless the user bound to the context
// already holds every permission described by the provided rules in the given
// namespace. It is used when creating or replacing a Role's rules, when granting
// a permission to a Role, and (via verifyBindingNotEscalating) when binding a
// user to a Role. A nil authorizer (tests, or non-authorizing local mode)
// disables the check. As a convenience, it also returns the normalized rules to
// avoid the caller having to normalize them again for storage in the Role.
//
// The rules are normalized first, which reduces them to atomic permissions -- one
// API group, one resource, at most one resource name, with "*" verbs expanded --
// so the authorization check is a flat loop over concrete permissions.
func verifyRulesNotEscalating(
	ctx context.Context,
	authz kubernetes.Authorizer,
	namespace string,
	rules []rbacv1.PolicyRule,
) ([]rbacv1.PolicyRule, error) {
	normalized, err := NormalizePolicyRules(
		rules,
		&PolicyRuleNormalizationOptions{IncludeCustomVerbsInExpansion: true},
	)
	if err != nil {
		return nil, err
	}
	if authz == nil {
		return normalized, nil
	}

	for _, rule := range normalized {
		// Normalized rules carry exactly one API group and one resource.
		resourceType := rule.Resources[0]
		resource, subresource := splitResourceType(resourceType)
		gvr := schema.GroupVersionResource{Group: rule.APIGroups[0], Resource: resource}
		var name string
		if len(rule.ResourceNames) > 0 {
			name = rule.ResourceNames[0]
		}
		key := client.ObjectKey{Namespace: namespace, Name: name}
		for _, verb := range rule.Verbs {
			if err := authorizeGrant(
				ctx, authz, verb, gvr, subresource, key, resourceType,
			); err != nil {
				return nil, err
			}
		}
	}
	return normalized, nil
}

// verifyBindingNotEscalating verifies that the requester already holds every
// permission conferred by the Roles bound to the given Kargo Role's
// ServiceAccount. It is used when binding a user's identity to a Kargo Role via
// claim annotations (GrantRoleToUsers): doing so maps that identity onto the
// ServiceAccount and grants it all of the Role's permissions, so a requester
// must not be able to confer more than they themselves hold. A nil authorizer
// disables the check.
//
// Only the namespaced Roles are considered: callers reach this after
// manageableResources, which rejects ServiceAccounts bound to ClusterRoles, so
// any ClusterRoles are excluded here.
func verifyBindingNotEscalating(
	ctx context.Context,
	authz kubernetes.Authorizer,
	namespace string,
	roles []rbacv1.Role,
) error {
	if authz == nil {
		return nil
	}
	var rules []rbacv1.PolicyRule
	for _, role := range roles {
		rules = append(rules, role.Rules...)
	}
	_, err := verifyRulesNotEscalating(ctx, authz, namespace, rules)
	return err
}

// authorizeGrant checks a single granted permission against the user's own
// authority. A forbidden result is rewritten as an escalation error; any other
// error is propagated so the operation fails closed.
func authorizeGrant(
	ctx context.Context,
	authz kubernetes.Authorizer,
	verb string,
	gvr schema.GroupVersionResource,
	subresource string,
	key client.ObjectKey,
	resourceType string,
) error {
	err := authz.Authorize(ctx, verb, gvr, subresource, key)
	if err == nil {
		return nil
	}
	if apierrors.IsForbidden(err) {
		return apierrors.NewForbidden(
			gvr.GroupResource(),
			key.Name,
			fmt.Errorf(
				"requester may not grant permissions it does not hold: verb %q on resource %q",
				verb, resourceType,
			),
		)
	}
	return fmt.Errorf(
		"error verifying permission to grant verb %q on resource %q: %w",
		verb, resourceType, err,
	)
}
