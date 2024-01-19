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
	// transition into the specified Freight. It returns current promo status
	// and Freight, which may possibly be updated by the process.
	Promote(
		context.Context,
		*kargoapi.Stage,
		*kargoapi.Promotion,
		kargoapi.FreightReference,
	) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error)
}

// NewMechanisms returns the entrypoint to a hierarchical tree of promotion
// mechanisms.
func NewMechanisms(
	argocdClient client.Client,
	credentialsDB credentials.Database,
) Mechanism {
	return newCompositeMechanism(
		"promotion mechanisms",
		newCompositeMechanism(
			"Git-based promotion mechanisms",
			newGenericGitMechanism(credentialsDB),
			newKargoRenderMechanism(credentialsDB),
			newKustomizeMechanism(credentialsDB),
			newHelmMechanism(credentialsDB),
		),
		newArgoCDMechanism(argocdClient),
	)
}
