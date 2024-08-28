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
		r.RegisterDirective(s, nil)
		assert.Same(t, s, r[s.Name()].Directive)
	})

	t.Run("overwrites directive", func(t *testing.T) {
		r := make(DirectiveRegistry)
		s := &mockDirective{}
		r.RegisterDirective(s, nil)
		s2 := &mockDirective{
			runErr: fmt.Errorf("error"),
		}
		r.RegisterDirective(s2, nil)
		assert.NotSame(t, s, r[s2.Name()].Directive)
		assert.Same(t, s2, r[s2.Name()].Directive)
	})
}

func TestDirectiveRegistry_GetDirectiveRegistration(t *testing.T) {
	t.Run("directive exists", func(t *testing.T) {
		r := make(DirectiveRegistry)
		s := &mockDirective{}
		r.RegisterDirective(s, nil)
		reg, err := r.GetDirectiveRegistration(s.Name())
		assert.NoError(t, err)
		assert.Same(t, s, reg.Directive)
	})

	t.Run("directive does not exist", func(t *testing.T) {
		r := make(DirectiveRegistry)
		_, err := r.GetDirectiveRegistration("nonexistent")
		assert.ErrorContains(t, err, "not found")
	})
}
