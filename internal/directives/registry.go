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

// DirectiveRegistry is a map of directive names to DirectiveRegistrations. It
// is used to register and retrieve directives by name.
type DirectiveRegistry map[string]DirectiveRegistration

// DirectiveRegistration is a registration for a single Directive. It includes
// the Directive itself and a set of permissions that indicate capabilities
// the Directive execution engine should enable for the Directive.
type DirectiveRegistration struct {
	// Permissions is a set of permissions that indicate capabilities the
	// Directive execution engine should enable for the Directive.
	Permissions DirectivePermissions
	// Directive is a Directive that performs a discrete action in the context
	// of a Promotion.
	Directive Directive
}

// DirectivePermissions is a set of permissions that indicate capabilities the
// Directive execution engine should enable for a Directive.
type DirectivePermissions struct {
	// AllowCredentialsDB indicates whether the Directive execution engine may
	// provide the Directive with access to the credentials database.
	AllowCredentialsDB bool
	// AllowKargoClient indicates whether the Directive execution engine may
	// provide the Directive with access to a Kubernetes client for the Kargo
	// control plane.
	AllowKargoClient bool
	// AllowArgoCDClient indicates whether the Directive execution engine may
	// provide the Directive with access to a Kubernetes client for the Argo CD
	// control plane.
	AllowArgoCDClient bool
}

// RegisterDirective registers a Directive with the given name. If a Directive
// with the same name has already been registered, it will be overwritten.
func (r DirectiveRegistry) RegisterDirective(
	directive Directive,
	permissions *DirectivePermissions,
) {
	if permissions == nil {
		permissions = &DirectivePermissions{}
	}
	r[directive.Name()] = DirectiveRegistration{
		Permissions: *permissions,
		Directive:   directive,
	}
}

// GetDirectiveRegistration returns the DirectiveRegistration for the Directive
// with the given name, or an error if no such Directive is registered.
func (r DirectiveRegistry) GetDirectiveRegistration(
	name string,
) (DirectiveRegistration, error) {
	step, ok := r[name]
	if !ok {
		return DirectiveRegistration{}, fmt.Errorf("directive %q not found", name)
	}
	return step, nil
}
