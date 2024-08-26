package directives

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirectiveRegistry_RegisterDirective(t *testing.T) {
	t.Run("registers directive", func(t *testing.T) {
		r := make(DirectiveRegistry)
		s := &mockDirective{}
		r.RegisterDirective(s)

		assert.Equal(t, s, r[s.Name()])
	})

	t.Run("overwrites directive", func(t *testing.T) {
		r := make(DirectiveRegistry)
		s := &mockDirective{}
		r.RegisterDirective(s)
		s2 := &mockDirective{
			runErr: fmt.Errorf("error"),
		}
		r.RegisterDirective(s2)

		assert.NotEqual(t, s, r[s2.Name()])
		assert.Equal(t, s2, r[s2.Name()])
	})
}

func TestDirectiveRegistry_GetDirective(t *testing.T) {
	t.Run("directive exists", func(t *testing.T) {
		r := make(DirectiveRegistry)
		s := &mockDirective{}
		r.RegisterDirective(s)

		step, err := r.GetDirective(s.Name())
		assert.NoError(t, err)
		assert.Equal(t, s, step)
	})

	t.Run("directive does not exist", func(t *testing.T) {
		r := make(DirectiveRegistry)
		_, err := r.GetDirective("nonexistent")
		assert.ErrorContains(t, err, "not found")
	})
}
