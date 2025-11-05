package external

import (
	"encoding/json"
	"fmt"
	"slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"

	"k8s.io/apimachinery/pkg/selection"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
)

type conditionEvalSummary struct {
	errors  map[string]error
	results map[string]bool
}

func evaluateConditions(conditions []kargoapi.ConditionSelector, env map[string]any) (string, bool) {
	summary := &conditionEvalSummary{
		errors:  make(map[string]error),
		results: make(map[string]bool),
	}
	for _, condition := range conditions {
		conditionMet, err := evaluateConditionSelector(condition, env)
		if err != nil {
			summary.addError(condition.Key, err)
			continue
		}
		summary.addResult(condition.Key, conditionMet)
	}
	return summary.String(), summary.allConditionsMet()
}

func (s *conditionEvalSummary) String() string {
	b, _ := json.Marshal(map[string]string{
		"errors":  s.Errors(),
		"results": s.Results(),
	})
	return string(b)
}

func (s *conditionEvalSummary) Errors() string {
	var errs []error
	for conditionKey, err := range s.errors {
		errs = append(errs,
			fmt.Errorf("failed to evaluate condition-key %q: %w; ", conditionKey, err),
		)
	}
	return kerrors.Flatten(kerrors.NewAggregate(errs)).Error()
}

func (s *conditionEvalSummary) Results() string {
	var results []string
	for conditionKey, met := range s.results {
		msg := fmt.Sprintf("condition-key %q met: %t", conditionKey, met)
		results = append(results, msg)
	}
	return fmt.Sprintf("conditionEvalSummary{results: %v}", results)
}

func (s *conditionEvalSummary) addError(conditionKey string, err error) {
	s.errors[conditionKey] = err
}

func (s *conditionEvalSummary) addResult(conditionKey string, result bool) {
	s.results[conditionKey] = result
}

func (s *conditionEvalSummary) allConditionsMet() bool {
	if len(s.errors) > 0 {
		return false
	}
	for _, conditionMet := range s.results {
		if !conditionMet {
			return false
		}
	}
	return true
}

func evaluateConditionSelector(cs kargoapi.ConditionSelector, env map[string]any) (bool, error) {
	// Evaluate the key expression to get the actual value to check against.
	// If the key is not an expression, it will be returned as is.
	result, err := expressions.EvaluateTemplate(cs.Key, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	strResult, ok := result.(string)
	if !ok {
		return false, fmt.Errorf("expression result %q evaluated to %T; not a string", result, result)
	}

	// The only operators supported are In and NotIn.
	// If we're dealing with NotIn, we need to invert the result.
	contains := slices.Contains(cs.Values, strResult)
	if cs.Operator == selection.NotIn {
		return !contains, nil
	}
	return contains, nil
}
