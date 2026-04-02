package projects

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/akuity/kargo/pkg/component"
)

type (
	// roleRulesContributorPredicate returns true if the contributor has
	// PolicyRules to contribute for the given role name.
	roleRulesContributorPredicate = func(context.Context, string) (bool, error)

	// roleRulesContributorFunc returns additional PolicyRules for a given role
	// name.
	roleRulesContributorFunc = func(roleName string) []rbacv1.PolicyRule

	// RoleRulesContributorRegistration associates a predicate with a
	// contributor function.
	RoleRulesContributorRegistration = component.PredicateBasedRegistration[
		string,
		roleRulesContributorPredicate,
		roleRulesContributorFunc,
		struct{},
	]
)

var defaultRoleRulesContributorRegistry = component.MustNewPredicateBasedRegistry[
	string,
	roleRulesContributorPredicate,
	roleRulesContributorFunc,
	struct{},
]()

// RegisterRoleRulesContributor adds a contributor to the global registry used
// by the project reconciler when creating default project roles. It should be
// called before SetupReconcilerWithManager (e.g. at program startup).
func RegisterRoleRulesContributor(reg RoleRulesContributorRegistration) {
	defaultRoleRulesContributorRegistry.MustRegister(reg)
}
