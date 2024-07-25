package freight

import (
	"context"
	"reflect"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// GetDesiredOrigin recursively walks a graph of promotion mechanisms, taking
// note of the origin of each mechanism (if specified) until it finds the
// "target" promotion mechanism. At each step, if no origin is found, the
// parent's origin is inherited. This function essentially permits child
// promotion mechanisms to inherit or override origins specified by their
// ancestors despites the fact that child promotion mechanisms never have any
// back-reference to their parent.
func GetDesiredOrigin(ctx context.Context, mechanism any, targetMechanism any) *kargoapi.FreightOrigin {
	return getDesiredOriginInternal(ctx, mechanism, targetMechanism, nil)
}

func getDesiredOriginInternal(
	ctx context.Context,
	mechanism any,
	targetMechanism any,
	defaultOrigin *kargoapi.FreightOrigin,
) *kargoapi.FreightOrigin {
	logger := logging.LoggerFromContext(ctx)
	logger.Debug("Finding desired origin",
		"mechanism", mechanism,
		"targetMechanism", targetMechanism,
		"defaultOrigin", defaultOrigin)
	// As a small sanity check, verify that mechanism and targetMechanism are both
	// pointers.
	if reflect.ValueOf(mechanism).Kind() != reflect.Ptr {
		panic("mechanism must be a pointer")
	}
	if reflect.ValueOf(targetMechanism).Kind() != reflect.Ptr {
		panic("targetMechanism must be a pointer")
	}

	var origin *kargoapi.FreightOrigin
	var subMechs []any
	switch m := mechanism.(type) {
	// Begin root
	case *kargoapi.Stage:
		// Stage is not technically a promotion mechanism, but it is a convenient
		// entry point for the recursion.
		subMechs = []any{m.Spec.PromotionMechanisms}
	case *kargoapi.PromotionMechanisms:
		origin = m.Origin
		subMechs = make([]any, len(m.GitRepoUpdates)+len(m.ArgoCDAppUpdates))
		for i := range m.GitRepoUpdates {
			subMechs[i] = &m.GitRepoUpdates[i]
		}
		for i := range m.ArgoCDAppUpdates {
			subMechs[i+len(m.GitRepoUpdates)] = &m.ArgoCDAppUpdates[i]
		}
	// Begin git-based
	case *kargoapi.GitRepoUpdate:
		origin = m.Origin
		subMechs = []any{m.Kustomize, m.Helm, m.Render}
	// Begin kustomize-based
	case *kargoapi.KustomizePromotionMechanism:
		origin = m.Origin
		subMechs = make([]any, len(m.Images))
		for i := range m.Images {
			subMechs[i] = &m.Images[i]
		}
	case *kargoapi.KustomizeImageUpdate:
		origin = m.Origin
	// End kustomize-based
	// Begin helm-based
	case *kargoapi.HelmPromotionMechanism:
		origin = m.Origin
		subMechs = make([]any, len(m.Images)+len(m.Charts))
		for i := range m.Images {
			subMechs[i] = &m.Images[i]
		}
		for i := range m.Charts {
			subMechs[i+len(m.Images)] = &m.Charts[i]
		}
	case *kargoapi.HelmImageUpdate:
		origin = m.Origin
	case *kargoapi.HelmChartDependencyUpdate:
		origin = m.Origin
	// End helm-based
	// Begin Kargo Render-based
	case *kargoapi.KargoRenderPromotionMechanism:
		origin = m.Origin
		subMechs = make([]any, len(m.Images))
		for i := range m.Images {
			subMechs[i] = &m.Images[i]
		}
	case *kargoapi.KargoRenderImageUpdate:
		origin = m.Origin
	// End Kargo Render-based
	// End git-based
	// Begin ArgoCD-based
	case *kargoapi.ArgoCDAppUpdate:
		origin = m.Origin
		subMechs = make([]any, len(m.SourceUpdates))
		for i := range m.SourceUpdates {
			subMechs[i] = &m.SourceUpdates[i]
		}
	case *kargoapi.ArgoCDSourceUpdate:
		origin = m.Origin
		subMechs = []any{m.Kustomize, m.Helm}
	// Begin Kustomize-based
	case *kargoapi.ArgoCDKustomize:
		origin = m.Origin
		subMechs = make([]any, len(m.Images))
		for i := range m.Images {
			subMechs[i] = &m.Images[i]
		}
	case *kargoapi.ArgoCDKustomizeImageUpdate:
		origin = m.Origin
	// End Kustomize-based
	// Begin Helm-based
	case *kargoapi.ArgoCDHelm:
		origin = m.Origin
		subMechs = make([]any, len(m.Images))
		for i := range m.Images {
			subMechs[i] = &m.Images[i]
		}
	case *kargoapi.ArgoCDHelmImageUpdate:
		origin = m.Origin
		// End Helm-based
		// End ArgoCD-based
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
		result := getDesiredOriginInternal(ctx, ts, targetMechanism, origin)
		if result != nil {
			return result
		}
	}
	return nil
}
