package builtin

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// buildArgoCDAppLabelSelector converts an ArgoCDAppSelector into a Kubernetes
// labels.Selector.
func buildArgoCDAppLabelSelector(
	selector *builtin.ArgoCDAppSelector,
) (labels.Selector, error) {
	if len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0 {
		return nil, fmt.Errorf("selector must have at least one match criterion")
	}

	labelSelector := labels.NewSelector()

	for key, value := range selector.MatchLabels {
		req, err := labels.NewRequirement(key, selection.Equals, []string{value})
		if err != nil {
			return nil, fmt.Errorf("invalid matchLabel %s=%s: %w", key, value, err)
		}
		labelSelector = labelSelector.Add(*req)
	}

	for _, expr := range selector.MatchExpressions {
		var op selection.Operator
		switch expr.Operator {
		case builtin.In:
			op = selection.In
		case builtin.NotIn:
			op = selection.NotIn
		case builtin.Exists:
			op = selection.Exists
		case builtin.DoesNotExist:
			op = selection.DoesNotExist
		default:
			return nil, fmt.Errorf("invalid operator: %s", expr.Operator)
		}

		req, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, fmt.Errorf("invalid matchExpression: %w", err)
		}
		labelSelector = labelSelector.Add(*req)
	}

	return labelSelector, nil
}
