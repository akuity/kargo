package directives

import (
	"fmt"
	"maps"
)

// builtins is the registry of built-in directives.
var builtins = DirectiveRegistry{}

// BuiltinsRegistry returns a registry of built-in directives.
func BuiltinsRegistry() DirectiveRegistry {
	return maps.Clone(builtins)
}

// DirectiveRegistry is a map of directive names to directives. It is used to
// register and retrieve directives by name.
type DirectiveRegistry map[string]Directive


// RegisterDirective registers a Directive with the given name. If a Directive
// with the same name has already been registered, it will be overwritten.
func (r DirectiveRegistry) RegisterDirective(directive Directive) {
	r[directive.Name()] = directive
}

// GetDirective returns the Directive with the given name, or an error if no
// such Directive exists.
func (r DirectiveRegistry) GetDirective(name string) (Directive, error) {
	step, ok := r[name]
	if !ok {
		return nil, fmt.Errorf("directive %q not found", name)
	}
	return step, nil
}
