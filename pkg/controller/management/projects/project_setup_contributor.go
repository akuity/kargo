package projects

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/component"
)

type (
	// projectSetupContributorPredicate returns true if the contributor has
	// setup work to perform for the given Project.
	projectSetupContributorPredicate = func(context.Context, *kargoapi.Project) (bool, error)

	// projectSetupContributorFunc performs additional setup for a Project
	// during reconciliation.
	projectSetupContributorFunc = func(ctx context.Context, project *kargoapi.Project) error

	// ProjectSetupContributorRegistration associates a predicate with a
	// contributor function for project setup.
	ProjectSetupContributorRegistration = component.PredicateBasedRegistration[
		*kargoapi.Project,
		projectSetupContributorPredicate,
		projectSetupContributorFunc,
		struct{},
	]

	// projectCleanupContributorPredicate returns true if the contributor has
	// cleanup work to perform for the given Project.
	projectCleanupContributorPredicate = func(context.Context, *kargoapi.Project) (bool, error)

	// projectCleanupContributorFunc performs cleanup for a Project when it is
	// deleted.
	projectCleanupContributorFunc = func(ctx context.Context, project *kargoapi.Project) error

	// ProjectCleanupContributorRegistration associates a predicate with a
	// contributor function for project cleanup.
	ProjectCleanupContributorRegistration = component.PredicateBasedRegistration[
		*kargoapi.Project,
		projectCleanupContributorPredicate,
		projectCleanupContributorFunc,
		struct{},
	]
)

var defaultProjectSetupContributorRegistry = component.MustNewPredicateBasedRegistry[
	*kargoapi.Project,
	projectSetupContributorPredicate,
	projectSetupContributorFunc,
	struct{},
]()

var defaultProjectCleanupContributorRegistry = component.MustNewPredicateBasedRegistry[
	*kargoapi.Project,
	projectCleanupContributorPredicate,
	projectCleanupContributorFunc,
	struct{},
]()

// RegisterProjectSetupContributor adds a contributor to the global registry
// used by the project reconciler to perform additional setup during project
// reconciliation. It should be called before SetupReconcilerWithManager (e.g.
// at program startup).
func RegisterProjectSetupContributor(reg ProjectSetupContributorRegistration) {
	defaultProjectSetupContributorRegistry.MustRegister(reg)
}

// RegisterProjectCleanupContributor adds a contributor to the global registry
// used by the project reconciler to perform cleanup when a project is deleted.
// It should be called before SetupReconcilerWithManager (e.g. at program
// startup).
func RegisterProjectCleanupContributor(reg ProjectCleanupContributorRegistration) {
	defaultProjectCleanupContributorRegistry.MustRegister(reg)
}
