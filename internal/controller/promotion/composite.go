package promotion

import (
	"context"

	"github.com/pkg/errors"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// compositeMechanism is an implementation of the Mechanism interface that is
// composed only of other Mechanisms. Executing Promote() or CheckHealth() on a
// compositeMechanism will execute that same function on each of its child
// Mechanisms in turn.
type compositeMechanism struct {
	name            string
	childMechanisms []Mechanism
}

// newCompositeMechanism returns an implementation of the Mechanism interface
// that is composed only of other Mechanisms. Executing Promote() or
// CheckHealth() on a compositeMechanism will execute that same function on each
// of its child Mechanisms in turn.
func newCompositeMechanism(
	name string,
	childPromotionMechanisms ...Mechanism,
) Mechanism {
	return &compositeMechanism{
		name:            name,
		childMechanisms: childPromotionMechanisms,
	}
}

// GetName implements the Mechanism interface.
func (c *compositeMechanism) GetName() string {
	return c.name
}

// Promote implements the Mechanism interface.
func (c *compositeMechanism) Promote(
	ctx context.Context,
	stage *api.Stage,
	newState api.StageState,
) (api.StageState, error) {
	newState = *newState.DeepCopy()

	logger := logging.LoggerFromContext(ctx)
	logger.Debugf("executing %s", c.name)

	for _, childMechanism := range c.childMechanisms {
		var err error
		newState, err = childMechanism.Promote(ctx, stage, newState)
		if err != nil {
			return newState, errors.Wrapf(
				err,
				"error executing %s",
				childMechanism.GetName(),
			)
		}
	}

	logger.Debug("done executing promotion mechanisms")

	return newState, nil
}
