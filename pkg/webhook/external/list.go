package external

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
)

func (g *genericWebhookReceiver) listTargetObjects(
	ctx context.Context,
	targetSelectionCriteria kargoapi.GenericWebhookTargetSelectionCriteria,
	actionEnv map[string]any,
) ([]client.Object, error) {
	listOpts, err := buildListOptionsForTarget(g.project, targetSelectionCriteria, actionEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to build list options: %w", err)
	}
	switch targetSelectionCriteria.Kind {
	case kargoapi.GenericWebhookTargetKindWarehouse:
		warehouses := new(kargoapi.WarehouseList)
		if err := g.client.List(ctx, warehouses, listOpts...); err != nil {
			return nil, fmt.Errorf("error listing %s targets: %w", targetSelectionCriteria.Kind, err)
		}
		if targetSelectionCriteria.Name == "" {
			return itemsToObjects(warehouses.Items), nil
		}
		name, err := evalAsString(targetSelectionCriteria.Name, actionEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate target name as string: %w", err)
		}
		for _, wh := range warehouses.Items {
			if wh.Name == name {
				return []client.Object{&wh}, nil
			}
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported target kind: %q", targetSelectionCriteria.Kind)
	}
}

// buildListOptionsForTarget builds a list of client.ListOption based on the
// provided GenericWebhookTarget's selectors. The returned ListOptions can be
// used to list Kubernetes resources that match the target's selection criteria.
func buildListOptionsForTarget(
	project string,
	t kargoapi.GenericWebhookTargetSelectionCriteria,
	env map[string]any,
) ([]client.ListOption, error) {
	listOpts := []client.ListOption{client.InNamespace(project)}
	indexSelectorListOpts, err := newListOptionsForIndexSelector(t.IndexSelector, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create field selector: %w", err)
	}
	listOpts = append(listOpts, indexSelectorListOpts...)
	labelSelectorListOpts, err := newListOptionsForLabelSelector(t.LabelSelector, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create label selector: %w", err)
	}
	listOpts = append(listOpts, labelSelectorListOpts...)
	return listOpts, nil
}

// newListOptionsForIndexSelector creates a list of client.ListOption based on
// the provided IndexSelector and environment for expression evaluation.
func newListOptionsForIndexSelector(
	is kargoapi.IndexSelector,
	env map[string]any,
) ([]client.ListOption, error) {
	var listOpts []client.ListOption
	for _, expr := range is.MatchIndices {
		resultStr, err := evalAsString(expr.Value, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate values expression as string: %w", err)
		}
		var s fields.Selector
		switch expr.Operator {
		case kargoapi.IndexSelectorOperatorEqual:
			s = fields.OneTermEqualSelector(expr.Key, resultStr)
		case kargoapi.IndexSelectorOperatorNotEqual:
			s = fields.OneTermNotEqualSelector(expr.Key, resultStr)
		default:
			return nil, fmt.Errorf("unsupported operator %q in index selector expression", expr.Operator)
		}
		listOpts = append(listOpts, client.MatchingFieldsSelector{Selector: s})
	}
	return listOpts, nil
}

// newListOptionsForLabelSelector creates a list of client.ListOption based on
// the provided LabelSelector.
func newListOptionsForLabelSelector(ls metav1.LabelSelector, env map[string]any) ([]client.ListOption, error) {
	var labelReqs []labels.Requirement
	matchExpressionRequirements, err := newLabelRequirementsForMatchExpressions(ls.MatchExpressions, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create label requirements for match expressions: %w", err)
	}
	labelReqs = append(labelReqs, matchExpressionRequirements...)
	matchLabelRequirements, err := newLabelRequirementsForMatchLabels(ls.MatchLabels, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create label requirements for match labels: %w", err)
	}
	labelReqs = append(labelReqs, matchLabelRequirements...)
	if len(labelReqs) == 0 {
		return nil, nil
	}
	return []client.ListOption{
		client.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(labelReqs...),
		},
	}, nil
}

func newLabelRequirementsForMatchExpressions(
	matchExpressions []metav1.LabelSelectorRequirement,
	env map[string]any,
) ([]labels.Requirement, error) {
	var labelReqs []labels.Requirement
	for _, expr := range matchExpressions {
		op, err := labelOpToSelectionOp(expr.Operator)
		if err != nil {
			return nil, fmt.Errorf("failed to convert label selector operator: %w", err)
		}
		values, err := evalValues(expr.Values, env)
		if err != nil {
			return nil, fmt.Errorf("failed to parse matchExpression values: %w", err)
		}
		labelReq, err := labels.NewRequirement(expr.Key, op, values)
		if err != nil {
			return nil, fmt.Errorf("failed to create label requirement: %w", err)
		}
		labelReqs = append(labelReqs, *labelReq)
	}
	return labelReqs, nil
}

func newLabelRequirementsForMatchLabels(
	matchLabels map[string]string,
	env map[string]any,
) ([]labels.Requirement, error) {
	var labelReqs []labels.Requirement
	for k, v := range matchLabels {
		strValue, err := evalAsString(v, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate matchLabel value as string: %w", err)
		}
		req, err := labels.NewRequirement(k, selection.Equals, []string{strValue})
		if err != nil {
			return nil, fmt.Errorf("failed to create label requirement: %w", err)
		}
		labelReqs = append(labelReqs, *req)
	}
	return labelReqs, nil
}

// labelOpToSelectionOp converts a metav1.LabelSelectorOperator
// into a selection.Operator, which is used to build label requirements.
// Returns an error if the operator is not supported. Unsupported operators are
// GT, LT, Exists, and NotExists.
func labelOpToSelectionOp(op metav1.LabelSelectorOperator) (selection.Operator, error) {
	switch op {
	case metav1.LabelSelectorOpIn:
		return selection.In, nil
	case metav1.LabelSelectorOpNotIn:
		return selection.NotIn, nil
	case metav1.LabelSelectorOpExists:
		return selection.Exists, nil
	case metav1.LabelSelectorOpDoesNotExist:
		return selection.DoesNotExist, nil
	default:
		return "", fmt.Errorf("unsupported LabelSelectorOperator: %q", op)
	}
}

// itemsToObjects converts a slice of Kubernetes resources to []client.Object.
// This generic helper works for any type T where *T implements client.Object.
func itemsToObjects[T any, PT interface {
	*T
	client.Object
}](items []T) []client.Object {
	objs := make([]client.Object, len(items))
	for i := range items {
		objs[i] = PT(&items[i])
	}
	return objs
}

func evalValues(vals []string, env map[string]any) ([]string, error) {
	values := make([]string, len(vals))
	for i, v := range vals {
		s, err := evalAsString(v, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate value %q as string: %w", v, err)
		}
		values[i] = s
	}
	return values, nil
}

func evalAsString(expr string, env map[string]any) (string, error) {
	result, err := expressions.EvaluateTemplate(expr, env)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression: %w", err)
	}
	s, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("expression result %q evaluated to %T; not a string", result, result)
	}
	return s, nil
}
