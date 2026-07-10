package api

import (
	"context"
	"fmt"
	"slices"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/pattern"
)

// findMatchingPromotionPolicy returns the first PromotionPolicy in the
// Project's ProjectConfig whose stage selector matches the supplied Stage
// metadata. Policies are evaluated in order and the first match wins, even
// when a later policy also matches. It returns nil when the Project has no
// ProjectConfig or no policy matches.
func findMatchingPromotionPolicy(
	ctx context.Context,
	c client.Client,
	stage metav1.ObjectMeta,
) (*kargoapi.PromotionPolicy, error) {
	projectCfg := &kargoapi.ProjectConfig{}
	if err := c.Get(ctx, types.NamespacedName{
		Name:      stage.Namespace,
		Namespace: stage.Namespace,
	}, projectCfg); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting ProjectConfig for Project %q: %w", stage.Namespace, err)
	}

	for _, policy := range projectCfg.Spec.PromotionPolicies {
		if policy.StageSelector == nil {
			// Maintain backward compatibility with older versions of the
			// PromotionPolicy where the selector was not available.
			policy.StageSelector = &kargoapi.PromotionPolicySelector{
				Name: policy.Stage, // nolint:staticcheck
			}
		}

		if nameSelector := policy.StageSelector.Name; nameSelector != "" {
			m, err := pattern.ParseNamePattern(nameSelector)
			if err != nil {
				return nil, fmt.Errorf("error parsing PromotionPolicy name pattern %q: %w", nameSelector, err)
			}
			if !m.Matches(stage.Name) {
				continue
			}
		}

		if labelSelector := policy.StageSelector.LabelSelector; labelSelector != nil {
			s, err := metav1.LabelSelectorAsSelector(labelSelector)
			if err != nil {
				return nil, fmt.Errorf("error parsing PromotionPolicy label selector %q: %w", labelSelector, err)
			}
			if !s.Matches(labels.Set(stage.Labels)) {
				continue
			}
		}

		return &policy, nil
	}

	return nil, nil
}

// IsAutoPromotionEnabled returns whether the ProjectConfig enables
// auto-promotion for the supplied Stage metadata.
func IsAutoPromotionEnabled(
	ctx context.Context,
	c client.Client,
	stage metav1.ObjectMeta,
) (bool, error) {
	policy, err := findMatchingPromotionPolicy(ctx, c, stage)
	if err != nil {
		return false, err
	}
	return policy != nil && policy.AutoPromotionEnabled, nil
}

// SelectAutoPromotionCandidates returns, for each origin in the Stage's
// requested Freight, the available Freight that origin's auto-promotion
// selection policy would pick. Selection is all this does: it never decides
// whether the pick should actually be promoted. Before acting on a candidate,
// callers apply their own checks -- e.g. that the candidate is not already the
// Stage's current Freight, that no Promotion for it already exists, and that
// no auto-promotion hold blocks its origin.
func SelectAutoPromotionCandidates(
	ctx context.Context,
	stage *kargoapi.Stage,
	availableFreight []kargoapi.Freight,
) map[string]kargoapi.Freight {
	availableByOrigin := make(map[string][]kargoapi.Freight)
	for _, freight := range availableFreight {
		origin := freight.Origin.String()
		availableByOrigin[origin] = append(availableByOrigin[origin], freight)
	}

	candidates := make(map[string]kargoapi.Freight)
	for _, req := range stage.Spec.RequestedFreight {
		origin := req.Origin.String()
		freight := availableByOrigin[origin]
		if len(freight) == 0 {
			continue
		}
		if req.Sources.AutoPromotionOptions != nil &&
			req.Sources.AutoPromotionOptions.SelectionPolicy == kargoapi.AutoPromotionSelectionPolicyMatchUpstream &&
			len(freight) > 1 {
			// This is a transient race: the Stage reconciler updates two Freight
			// resources after every Promotion completes (the incoming gains the
			// Stage, the outgoing loses it), so momentarily both report the same
			// Stage. Treat the origin as having no candidate this pass; the next
			// reconciliation will see only one.
			logging.LoggerFromContext(ctx).Debug(
				"transiently found multiple Freight upstream; skipping origin",
				"origin", origin,
				"count", len(freight),
			)
			continue
		}

		slices.SortFunc(freight, func(lhs, rhs kargoapi.Freight) int {
			cmp := rhs.EffectiveDiscoveredAt().Compare(lhs.EffectiveDiscoveredAt())
			if cmp != 0 {
				return cmp
			}
			return strings.Compare(rhs.Name, lhs.Name)
		})
		candidates[origin] = freight[0]
	}
	return candidates
}

// SetAutoPromotionHoldAnnotation stamps promo with AnnotationKeyAutoPromotionHold
// set to origin's canonical key. The Stage controller reads a succeeded
// Promotion carrying this annotation to establish a hold for that origin.
func SetAutoPromotionHoldAnnotation(promo *kargoapi.Promotion, origin kargoapi.FreightOrigin) {
	if promo.Annotations == nil {
		promo.Annotations = make(map[string]string, 1)
	}
	delete(promo.Annotations, kargoapi.AnnotationKeyAutoPromotionResume)
	promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold] = origin.String()
}

// SetAutoPromotionResumeAnnotation stamps promo with
// AnnotationKeyAutoPromotionResume set to origin's canonical key. The Stage
// controller reads a succeeded Promotion carrying this annotation to resume
// auto-promotion for that origin by clearing any active hold.
func SetAutoPromotionResumeAnnotation(promo *kargoapi.Promotion, origin kargoapi.FreightOrigin) {
	if promo.Annotations == nil {
		promo.Annotations = make(map[string]string, 1)
	}
	delete(promo.Annotations, kargoapi.AnnotationKeyAutoPromotionHold)
	promo.Annotations[kargoapi.AnnotationKeyAutoPromotionResume] = origin.String()
}
