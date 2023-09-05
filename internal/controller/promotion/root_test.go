package promotion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/bookkeeper"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewMechanisms(t *testing.T) {
	promoMechs := NewMechanisms(
		fake.NewClientBuilder().Build(),
		credentials.NewKubernetesDatabase("", nil, nil),
		bookkeeper.NewService(nil),
	)
	require.IsType(t, &compositeMechanism{}, promoMechs)
}

// FakeMechanism is a fake implementation of the Mechanism interface used for
// testing.
type FakeMechanism struct {
	Name      string
	PromoteFn func(
		context.Context,
		*api.Stage,
		api.Freight,
	) (api.Freight, error)
}

// GetName implements the Mechanism interface.
func (f *FakeMechanism) GetName() string {
	return f.Name
}

// Promote implements the Mechanism interface.
func (f *FakeMechanism) Promote(
	ctx context.Context,
	stage *api.Stage,
	freight api.Freight,
) (api.Freight, error) {
	return f.PromoteFn(ctx, stage, freight)
}
