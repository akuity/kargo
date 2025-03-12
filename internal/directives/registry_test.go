package directives

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunnerRegistry_register(t *testing.T) {
	t.Run("registers", func(t *testing.T) {
		registry := runnerRegistry{}
		promoter := &mockPromoter{}
		registry.register(promoter)
		assert.Same(t, promoter, registry[promoter.Name()])
	})

	t.Run("overwrites registration", func(t *testing.T) {
		registry := runnerRegistry{}
		promoter1 := &mockPromoter{}
		registry.register(promoter1)
		promoter2 := &mockPromoter{
			promoteErr: fmt.Errorf("error"),
		}
		registry.register(promoter2)
		assert.NotSame(t, promoter1, registry[promoter2.Name()])
		assert.Same(t, promoter2, registry[promoter2.Name()])
	})
}

func TestRunnerRegistry_getPromoter(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := runnerRegistry{}
		promoter := &mockPromoter{}
		registry.register(promoter)
		r := registry.getPromoter(promoter.Name())
		assert.Same(t, promoter, r)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		promoter := runnerRegistry{}.getPromoter("nonexistent")
		assert.Nil(t, promoter)
	})
}

func TestRunnerRegistry_getHealthChecker(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := runnerRegistry{}
		healthChecker := &mockHealthChecker{}
		registry.register(healthChecker)
		r := registry.getHealthChecker(healthChecker.Name())
		assert.Same(t, r, healthChecker)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		healthChecker := runnerRegistry{}.getHealthChecker("nonexistent")
		assert.Nil(t, healthChecker)
	})
}
