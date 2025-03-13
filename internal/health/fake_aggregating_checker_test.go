package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestFakeAggregatingChecker_Check(t *testing.T) {
	t.Run("without function injection", func(t *testing.T) {
		checker := &FakeAggregatingChecker{}
		res := checker.Check(context.Background(), "fake-project", "fake-stage", nil)
		assert.Equal(t, kargoapi.HealthStateHealthy, res.Status)
	})

	t.Run("with function injection", func(t *testing.T) {
		ctx := context.Background()
		const testProject = "fake-project"
		const testStage = "fake-stage"
		criteria := []Criteria{{Kind: "mock"}}
		checker := &FakeAggregatingChecker{
			CheckFn: func(givenCtx context.Context, _, _ string, givenCriteria []Criteria) kargoapi.Health {
				assert.Equal(t, ctx, givenCtx)
				assert.Equal(t, criteria, givenCriteria)
				return kargoapi.Health{Status: kargoapi.HealthStateUnhealthy}
			},
		}
		res := checker.Check(ctx, testProject, testStage, criteria)
		assert.Equal(t, kargoapi.HealthStateUnhealthy, res.Status)
	})
}
