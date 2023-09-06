package promotion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/bookkeeper"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
		*kargoapi.Stage,
		kargoapi.Freight,
	) (kargoapi.Freight, error)
}

// GetName implements the Mechanism interface.
func (f *FakeMechanism) GetName() string {
	return f.Name
}

// Promote implements the Mechanism interface.
func (f *FakeMechanism) Promote(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight kargoapi.Freight,
) (kargoapi.Freight, error) {
	return f.PromoteFn(ctx, stage, freight)
}
