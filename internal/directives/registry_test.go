package directives

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepRunnerRegistry_register(t *testing.T) {
	t.Run("registers", func(t *testing.T) {
		registry := stepRunnerRegistry{}
		runner := &mockPromotionStepRunner{}
		registry.register(runner)
		assert.Same(t, runner, registry[runner.Name()])
	})

	t.Run("overwrites registration", func(t *testing.T) {
		registry := stepRunnerRegistry{}
		runner1 := &mockPromotionStepRunner{}
		registry.register(runner1)
		runner2 := &mockPromotionStepRunner{
			runErr: fmt.Errorf("error"),
		}
		registry.register(runner2)
		assert.NotSame(t, runner1, registry[runner2.Name()])
		assert.Same(t, runner2, registry[runner2.Name()])
	})
}

func TestStepRunnerRegistry_getPromotionStepRunner(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := stepRunnerRegistry{}
		runner := &mockPromotionStepRunner{}
		registry.register(runner)
		r := registry.getPromotionStepRunner(runner.Name())
		assert.Same(t, runner, r)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		runner := stepRunnerRegistry{}.getPromotionStepRunner("nonexistent")
		assert.Nil(t, runner)
	})
}

func TestStepRunnerRegistry_getHealthCheckStepRunner(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := stepRunnerRegistry{}
		runner := &mockHealthCheckStepRunner{}
		registry.register(runner)
		r := registry.getHealthCheckStepRunner(runner.Name())
		assert.Same(t, r, runner)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		runner := stepRunnerRegistry{}.getHealthCheckStepRunner("nonexistent")
		assert.Nil(t, runner)
	})
}
