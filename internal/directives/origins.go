package directives

import (
	"reflect"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// getDesiredOrigin recursively walks a the object graph of an
// ArgoCDUpdateConfig, taking note of the origin of node (if specified) until it
// finds the "target" node. At each step, if no origin is found, the parent's
// origin is inherited. This function essentially permits children to inherit or
// override origins specified by their ancestors despite the fact that they
// never have any back-reference to their parent.
func getDesiredOrigin(mechanism any, targetMechanism any) *kargoapi.FreightOrigin {
	origin := getDesiredOriginInternal(mechanism, targetMechanism, nil)
	if origin == nil {
		return nil
	}
	return &kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKind(origin.Kind),
		Name: origin.Name,
	}
}

func getDesiredOriginInternal(
	mechanism any,
	targetMechanism any,
	defaultOrigin *AppFromOrigin,
) *AppFromOrigin {
	// As a small sanity check, verify that mechanism and targetMechanism are both
	// pointers.
	if reflect.ValueOf(mechanism).Kind() != reflect.Ptr {
		panic("mechanism must be a pointer")
	}
	if reflect.ValueOf(targetMechanism).Kind() != reflect.Ptr {
		panic("targetMechanism must be a pointer")
	}

	var origin *AppFromOrigin
	var subMechs []any
	switch m := mechanism.(type) {
	// Begin root
	case *ArgoCDUpdateConfig:
		origin = m.FromOrigin
		subMechs = make([]any, len(m.Apps))
		for i := range m.Apps {
			subMechs[i] = &m.Apps[i]
		}
	case *ArgoCDAppUpdate:
		origin = m.FromOrigin
		subMechs = make([]any, len(m.Sources))
		for i := range m.Sources {
			subMechs[i] = &m.Sources[i]
		}
	case *ArgoCDAppSourceUpdate:
		origin = m.FromOrigin
		subMechs = []any{m.Kustomize, m.Helm}
	// Begin Kustomize-based
	case *ArgoCDKustomizeImageUpdates:
		origin = m.FromOrigin
		subMechs = make([]any, len(m.Images))
		for i := range m.Images {
			subMechs[i] = &m.Images[i]
		}
	case *ArgoCDKustomizeImageUpdate:
		origin = m.FromOrigin
	case *ArgoCDHelmParameterUpdates:
		origin = m.FromOrigin
		subMechs = make([]any, len(m.Images))
		for i := range m.Images {
			subMechs[i] = &m.Images[i]
		}
	case *ArgoCDHelmImageUpdate:
		origin = m.FromOrigin
	}
	if origin == nil {
		origin = defaultOrigin
	}
	if mechanism == targetMechanism {
		return origin
	}
	for _, ts := range subMechs {
		if reflect.ValueOf(ts).IsNil() {
			continue
		}
		result := getDesiredOriginInternal(ts, targetMechanism, origin)
		if result != nil {
			return result
		}
	}
	return nil
}
