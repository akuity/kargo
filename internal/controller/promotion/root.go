package promotion

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/bookkeeper"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// Mechanism provides a consistent interface for all promotion mechanisms.
type Mechanism interface {
	// GetName returns the name of a promotion mechanism.
	GetName() string
	// Promote consults rules in the provided Stage to perform some portion of the
	// transition into the specified Freight. It returns the Freight, which may
	// possibly be updated by the process.
	Promote(
		context.Context,
		*kargoapi.Stage,
		kargoapi.SimpleFreight,
	) (kargoapi.SimpleFreight, error)
}

// NewMechanisms returns the entrypoint to a hierarchical tree of promotion
// mechanisms.
func NewMechanisms(
	argoClient client.Client,
	credentialsDB credentials.Database,
	bookkeeperService bookkeeper.Service,
) Mechanism {
	return newCompositeMechanism(
		"promotion mechanisms",
		newCompositeMechanism(
			"Git-based promotion mechanisms",
			newGenericGitMechanism(credentialsDB),
			newBookkeeperMechanism(credentialsDB, bookkeeperService),
			newKustomizeMechanism(credentialsDB),
			newHelmMechanism(credentialsDB),
		),
		newArgoCDMechanism(argoClient),
	)
}
