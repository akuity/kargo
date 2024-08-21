package directives

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStepRegistry_RegisterStep(t *testing.T) {
	t.Run("registers step", func(t *testing.T) {
		r := make(StepRegistry)
		s := &mockStep{}
		r.RegisterStep(s)

		assert.Equal(t, s, r[s.Name()])
	})

	t.Run("overwrites step", func(t *testing.T) {
		r := make(StepRegistry)
		s := &mockStep{}
		r.RegisterStep(s)
		s2 := &mockStep{
			runErr: fmt.Errorf("error"),
		}
		r.RegisterStep(s2)

		assert.NotEqual(t, s, r[s2.Name()])
		assert.Equal(t, s2, r[s2.Name()])
	})
}

func TestStepRegistry_GetStep(t *testing.T) {
	t.Run("step exists", func(t *testing.T) {
		r := make(StepRegistry)
		s := &mockStep{}
		r.RegisterStep(s)

		step, err := r.GetStep(s.Name())
		assert.NoError(t, err)
		assert.Equal(t, s, step)
	})

	t.Run("step does not exist", func(t *testing.T) {
		r := make(StepRegistry)
		_, err := r.GetStep("nonexistent")
		assert.ErrorContains(t, err, "not found")
	})
}
