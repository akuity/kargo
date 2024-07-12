package stages

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type mockAppHealthEvaluator struct {
	Health *kargoapi.Health
}

func (m *mockAppHealthEvaluator) EvaluateHealth(
	context.Context,
	*kargoapi.Stage,
) *kargoapi.Health {
	return m.Health
}
