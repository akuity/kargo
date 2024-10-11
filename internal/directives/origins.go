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
func getDesiredOrigin(update any, targetUpdate any) *kargoapi.FreightOrigin {
	origin := getDesiredOriginInternal(update, targetUpdate, nil)
	if origin == nil {
		return nil
	}
	return &kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKind(origin.Kind),
		Name: origin.Name,
	}
}

func getDesiredOriginInternal(
	update any,
	targetUpdate any,
	defaultOrigin *AppFromOrigin,
) *AppFromOrigin {
	// As a small sanity check, verify that update and targetUpdate are both
	// pointers.
	if reflect.ValueOf(update).Kind() != reflect.Ptr {
		panic("update must be a pointer")
	}
	if reflect.ValueOf(targetUpdate).Kind() != reflect.Ptr {
		panic("targetUpdate must be a pointer")
	}

	var origin *AppFromOrigin
	var subUpdates []any
	switch m := update.(type) {
	// Begin root
	case *ArgoCDUpdateConfig:
		origin = m.FromOrigin
		subUpdates = make([]any, len(m.Apps))
		for i := range m.Apps {
			subUpdates[i] = &m.Apps[i]
		}
	case *ArgoCDAppUpdate:
		origin = m.FromOrigin
		subUpdates = make([]any, len(m.Sources))
		for i := range m.Sources {
			subUpdates[i] = &m.Sources[i]
		}
	case *ArgoCDAppSourceUpdate:
		origin = m.FromOrigin
		subUpdates = []any{m.Kustomize, m.Helm}
	// Begin Kustomize-based
	case *ArgoCDKustomizeImageUpdates:
		origin = m.FromOrigin
		subUpdates = make([]any, len(m.Images))
		for i := range m.Images {
			subUpdates[i] = &m.Images[i]
		}
	case *ArgoCDKustomizeImageUpdate:
		origin = m.FromOrigin
	case *ArgoCDHelmParameterUpdates:
		origin = m.FromOrigin
		subUpdates = make([]any, len(m.Images))
		for i := range m.Images {
			subUpdates[i] = &m.Images[i]
		}
	case *ArgoCDHelmImageUpdate:
		origin = m.FromOrigin
	}
	if origin == nil {
		origin = defaultOrigin
	}
	if update == targetUpdate {
		return origin
	}
	for _, ts := range subUpdates {
		if reflect.ValueOf(ts).IsNil() {
			continue
		}
		result := getDesiredOriginInternal(ts, targetUpdate, origin)
		if result != nil {
			return result
		}
	}
	return nil
}
