package promotion

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/kargo/pkg/promotion"
)

func TestStepRunnerRegistry_register(t *testing.T) {
	t.Run("registers", func(t *testing.T) {
		registry := stepRunnerRegistry{}
		runner := &promotion.MockStepRunner{}
		registry.register(runner)
		assert.Same(t, runner, registry[runner.Name()])
	})

	t.Run("overwrites registration", func(t *testing.T) {
		registry := stepRunnerRegistry{}
		runner1 := &promotion.MockStepRunner{}
		registry.register(runner1)
		runner2 := &promotion.MockStepRunner{
			RunErr: fmt.Errorf("error"),
		}
		registry.register(runner2)
		assert.NotSame(t, runner1, registry[runner2.Name()])
		assert.Same(t, runner2, registry[runner2.Name()])
	})
}

func TestStepRunnerRegistry_getStepRunner(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := stepRunnerRegistry{}
		runner := &promotion.MockStepRunner{}
		registry.register(runner)
		r := registry.getStepRunner(runner.Name())
		assert.Same(t, runner, r)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		assert.Nil(t, stepRunnerRegistry{}.getStepRunner("nonexistent"))
	})
}
