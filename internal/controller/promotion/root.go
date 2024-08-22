package promotion

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

// Mechanism provides a consistent interface for all promotion mechanisms.
type Mechanism interface {
	// GetName returns the name of a promotion mechanism.
	GetName() string
	// Promote consults rules in the provided Stage to perform some portion of the
	// transition to using artifacts from the provided FreightReferences. It
	// may modify the provided Promotion's status.
	Promote(context.Context, *kargoapi.Stage, *kargoapi.Promotion) error
}

// NewMechanisms returns the entrypoint to a hierarchical tree of promotion
// mechanisms.
func NewMechanisms(
	kargoClient client.Client,
	argocdClient client.Client,
	credentialsDB credentials.Database,
) Mechanism {
	return newCompositeMechanism(
		"promotion mechanisms",
		newCompositeMechanism(
			"Git-based promotion mechanisms",
			newGenericGitMechanism(kargoClient, credentialsDB),
			newKargoRenderMechanism(kargoClient, credentialsDB),
			newKustomizeMechanism(kargoClient, credentialsDB),
			newHelmMechanism(kargoClient, credentialsDB),
		),
		newArgoCDMechanism(kargoClient, argocdClient),
	)
}
