package directives

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepRunnerRegistry_RegisterPromotionStepRunner(t *testing.T) {
	t.Run("registers", func(t *testing.T) {
		registry := NewStepRunnerRegistry()
		runner := &mockPromotionStepRunner{}
		registry.RegisterPromotionStepRunner(runner, nil)
		assert.Same(t, runner, registry.promotionStepRunners[runner.Name()].Runner)
	})

	t.Run("overwrites registration", func(t *testing.T) {
		registry := NewStepRunnerRegistry()
		runner1 := &mockPromotionStepRunner{}
		registry.RegisterPromotionStepRunner(runner1, nil)
		runner2 := &mockPromotionStepRunner{
			runErr: fmt.Errorf("error"),
		}
		registry.RegisterPromotionStepRunner(runner2, nil)
		assert.NotSame(t, runner1, registry.promotionStepRunners[runner2.Name()].Runner)
		assert.Same(t, runner2, registry.promotionStepRunners[runner2.Name()].Runner)
	})
}

func TestStepRunnerRegistry_GetPromotionStepRunnerRegistration(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := NewStepRunnerRegistry()
		runner := &mockPromotionStepRunner{}
		registry.RegisterPromotionStepRunner(runner, nil)
		reg, err := registry.GetPromotionStepRunnerRegistration(runner.Name())
		assert.NoError(t, err)
		assert.Same(t, runner, reg.Runner)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		_, err := NewStepRunnerRegistry().
			GetPromotionStepRunnerRegistration("nonexistent")
		assert.ErrorContains(t, err, "not found")
	})
}

func TestStepRunnerRegistry_RegisterHealthCheckStepRunner(t *testing.T) {
	t.Run("registers", func(t *testing.T) {
		registry := NewStepRunnerRegistry()
		runner := &mockHealthCheckStepRunner{}
		registry.RegisterHealthCheckStepRunner(runner, nil)
		assert.Same(t, runner, registry.healthCheckStepRunners[runner.Name()].Runner)
	})

	t.Run("overwrites registration", func(t *testing.T) {
		registry := NewStepRunnerRegistry()
		runner1 := &mockHealthCheckStepRunner{}
		registry.RegisterHealthCheckStepRunner(runner1, nil)
		runner2 := &mockHealthCheckStepRunner{}
		registry.RegisterHealthCheckStepRunner(runner2, nil)
		assert.NotSame(t, runner1, registry.healthCheckStepRunners[runner2.Name()].Runner)
		assert.Same(t, runner2, registry.healthCheckStepRunners[runner2.Name()].Runner)
	})
}

func TestStepRunnerRegistry_GetHealthCheckStepRunnerRegistration(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := NewStepRunnerRegistry()
		runner := &mockHealthCheckStepRunner{}
		registry.RegisterHealthCheckStepRunner(runner, nil)
		reg, err := registry.GetHealthCheckStepRunnerRegistration(runner.Name())
		assert.NoError(t, err)
		assert.Same(t, runner, reg.Runner)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		_, err := NewStepRunnerRegistry().
			GetHealthCheckStepRunnerRegistration("nonexistent")
		assert.ErrorContains(t, err, "not found")
	})
}
