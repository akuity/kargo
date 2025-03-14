package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
)

func TestMockAggregatingChecker_Check(t *testing.T) {
	t.Run("without function injection", func(t *testing.T) {
		checker := &MockAggregatingChecker{}
		res := checker.Check(context.Background(), "fake-project", "fake-stage", nil)
		assert.Equal(t, kargoapi.HealthStateHealthy, res.Status)
	})

	t.Run("with function injection", func(t *testing.T) {
		ctx := context.Background()
		const testProject = "fake-project"
		const testStage = "fake-stage"
		criteria := []health.Criteria{{Kind: "mock"}}
		checker := &MockAggregatingChecker{
			CheckFn: func(givenCtx context.Context, _, _ string, givenCriteria []health.Criteria) kargoapi.Health {
				assert.Equal(t, ctx, givenCtx)
				assert.Equal(t, criteria, givenCriteria)
				return kargoapi.Health{Status: kargoapi.HealthStateUnhealthy}
			},
		}
		res := checker.Check(ctx, testProject, testStage, criteria)
		assert.Equal(t, kargoapi.HealthStateUnhealthy, res.Status)
	})
}
