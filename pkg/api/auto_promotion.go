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
	"github.com/akuity/kargo/pkg/pattern"
)

// IsAutoPromotionEnabled returns whether the ProjectConfig enables
// auto-promotion for the supplied Stage metadata.
func IsAutoPromotionEnabled(
	ctx context.Context,
	c client.Client,
	stage metav1.ObjectMeta,
) (bool, error) {
	projectCfg := &kargoapi.ProjectConfig{}
	if err := c.Get(ctx, types.NamespacedName{
		Name:      stage.Namespace,
		Namespace: stage.Namespace,
	}, projectCfg); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("error getting ProjectConfig for Project %q: %w", stage.Namespace, err)
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
				return false, fmt.Errorf("error parsing PromotionPolicy name pattern %q: %w", nameSelector, err)
			}
			if !m.Matches(stage.Name) {
				continue
			}
		}

		if labelSelector := policy.StageSelector.LabelSelector; labelSelector != nil {
			s, err := metav1.LabelSelectorAsSelector(labelSelector)
			if err != nil {
				return false, fmt.Errorf("error parsing PromotionPolicy label selector %q: %w", labelSelector, err)
			}
			if !s.Matches(labels.Set(stage.Labels)) {
				continue
			}
		}

		return policy.AutoPromotionEnabled, nil
	}

	return false, nil
}

// SelectAutoPromotionCandidates returns the Freight selected by each requested
// origin's auto-promotion selection policy. This is the candidate selection
// decision only; callers still own write-side guards such as current-Freight,
// existing-Promotion, and hold checks.
func SelectAutoPromotionCandidates(
	stage *kargoapi.Stage,
	availableFreight []kargoapi.Freight,
) (map[string]kargoapi.Freight, error) {
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
			return nil, fmt.Errorf(
				"unexpectedly found %d available Freight running immediately "+
					"upstream from Stage %q in namespace %q; this should not be possible",
				len(freight), stage.Name, stage.Namespace,
			)
		}

		slices.SortFunc(freight, func(lhs, rhs kargoapi.Freight) int {
			cmp := rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time)
			if cmp != 0 {
				return cmp
			}
			return strings.Compare(rhs.Name, lhs.Name)
		})
		candidates[origin] = freight[0]
	}
	return candidates, nil
}

// AutoPromotionHoldIdentityMatches returns whether hold still identifies the
// same auto-promotion hold as expected.
func AutoPromotionHoldIdentityMatches(
	hold kargoapi.AutoPromotionHold,
	expected kargoapi.AutoPromotionHold,
) bool {
	return hold.Freight.Name == expected.Freight.Name &&
		hold.Freight.Origin.Equals(&expected.Freight.Origin) &&
		hold.PromotionName == expected.PromotionName &&
		hold.PromotionUID == expected.PromotionUID &&
		autoPromotionHoldTimesEqual(hold.CreatedAt, expected.CreatedAt)
}

func autoPromotionHoldTimesEqual(lhs *metav1.Time, rhs *metav1.Time) bool {
	switch {
	case lhs == nil && rhs == nil:
		return true
	case lhs == nil || rhs == nil:
		return false
	default:
		return lhs.Time.Equal(rhs.Time)
	}
}
