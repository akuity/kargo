package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckerRegistry_register(t *testing.T) {
	t.Run("registers", func(t *testing.T) {
		registry := checkerRegistry{}
		checker := &mockChecker{}
		registry.register(checker)
		assert.Same(t, checker, registry[checker.Name()])
	})

	t.Run("overwrites registration", func(t *testing.T) {
		registry := checkerRegistry{}
		checker1 := &mockChecker{}
		registry.register(checker1)
		checker2 := &mockChecker{}
		registry.register(checker2)
		assert.NotSame(t, checker1, registry[checker2.Name()])
		assert.Same(t, checker2, registry[checker2.Name()])
	})
}

func TestCheckerRegistry_getChecker(t *testing.T) {
	t.Run("registration exists", func(t *testing.T) {
		registry := checkerRegistry{}
		checker := &mockChecker{}
		registry.register(checker)
		c := registry.getChecker(checker.Name())
		assert.Same(t, checker, c)
	})

	t.Run("registration does not exist", func(t *testing.T) {
		assert.Nil(t, checkerRegistry{}.getChecker("nonexistent"))
	})
}
