package external

import (
	"context"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
)

func (g *genericWebhookReceiver) listUniqueObjects(
	ctx context.Context,
	action kargoapi.GenericWebhookAction,
	actionEnv map[string]any,
) ([]client.Object, error) {
	var resources []client.Object
	for i, tsc := range action.TargetSelectionCriteria {
		objects, err := g.listTargetObjects(ctx, tsc, actionEnv)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects for targetSelectionCriteria at index %d: %w", i, err)
		}
		resources = append(resources, objects...)
	}
	slices.SortFunc(resources, func(a, b client.Object) int {
		if comp := strings.Compare(a.GetNamespace(), b.GetNamespace()); comp != 0 {
			return comp
		}
		return strings.Compare(a.GetName(), b.GetName())
	})
	resources = slices.CompactFunc(resources, func(a, b client.Object) bool {
		return a.GetNamespace() == b.GetNamespace() && a.GetName() == b.GetName()
	})
	return resources, nil
}

func (g *genericWebhookReceiver) listTargetObjects(
	ctx context.Context,
	targetSelectionCriteria kargoapi.GenericWebhookTargetSelectionCriteria,
	actionEnv map[string]any,
) ([]client.Object, error) {
	listOpts, err := g.buildListOptionsForTarget(targetSelectionCriteria, actionEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to build list options: %w", err)
	}
	var objects []client.Object
	switch targetSelectionCriteria.Kind {
	case kargoapi.GenericWebhookTargetKindWarehouse:
		warehouses := new(kargoapi.WarehouseList)
		if err = g.client.List(ctx, warehouses, listOpts...); err != nil {
			return nil, fmt.Errorf("error listing %s targets: %w", targetSelectionCriteria.Kind, err)
		}
		objects = itemsToObjects(warehouses.Items)
	default:
		return nil, fmt.Errorf("unsupported target kind: %q", targetSelectionCriteria.Kind)
	}
	if targetSelectionCriteria.Name == "" {
		return objects, nil
	}
	name, err := evalAsString(targetSelectionCriteria.Name, actionEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate target name as string: %w", err)
	}
	var filtered []client.Object
	for _, o := range objects {
		if o.GetName() == name {
			filtered = append(filtered, o)
		}
	}
	return filtered, nil
}

// buildListOptionsForTarget builds a list of client.ListOption based on the
// provided GenericWebhookTarget's selectors. The returned ListOptions can be
// used to list Kubernetes resources that match the target's selection criteria.
func (g *genericWebhookReceiver) buildListOptionsForTarget(
	t kargoapi.GenericWebhookTargetSelectionCriteria,
	env map[string]any,
) ([]client.ListOption, error) {
	var listOpts []client.ListOption
	if g.project != "" {
		listOpts = append(listOpts, client.InNamespace(g.project))
	}
	indexSelectorListOpts, err := newListOptionsForIndexSelector(t.IndexSelector, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create index selector list options: %w", err)
	}
	listOpts = append(listOpts, indexSelectorListOpts...)
	labelSelectorListOpts, err := newListOptionsForLabelSelector(t.LabelSelector, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create label selector list options: %w", err)
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
		value, err := evalAsString(expr.Value, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate values expression as string: %w", err)
		}
		var s fields.Selector
		switch expr.Operator {
		case kargoapi.IndexSelectorOperatorEqual:
			s = fields.OneTermEqualSelector(expr.Key, value)
		case kargoapi.IndexSelectorOperatorNotEqual:
			s = fields.OneTermNotEqualSelector(expr.Key, value)
		default:
			return nil, fmt.Errorf("unsupported operator %q in index selector expression", expr.Operator)
		}
		listOpts = append(listOpts, client.MatchingFieldsSelector{Selector: s})
	}
	return listOpts, nil
}

// newListOptionsForLabelSelector creates a list of client.ListOption based on
// the provided LabelSelector.
func newListOptionsForLabelSelector(
	ls metav1.LabelSelector,
	env map[string]any,
) ([]client.ListOption, error) {
	ls = *ls.DeepCopy()
	if ls.Size() == 0 {
		return nil, nil
	}
	var err error
	for labelKey, labelVal := range ls.MatchLabels {
		if ls.MatchLabels[labelKey], err = evalAsString(labelVal, env); err != nil {
			return nil, err
		}
	}
	for i, req := range ls.MatchExpressions {
		for j, val := range req.Values {
			ls.MatchExpressions[i].Values[j], err = evalAsString(val, env)
			if err != nil {
				return nil, err
			}
		}
	}
	selector, err := metav1.LabelSelectorAsSelector(&ls)
	if err != nil {
		return nil, fmt.Errorf("failed to convert label selector: %w", err)
	}
	return []client.ListOption{
		client.MatchingLabelsSelector{Selector: selector},
	}, nil
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
