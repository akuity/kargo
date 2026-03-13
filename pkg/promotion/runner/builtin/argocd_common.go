package builtin

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// authorizeArgoCDAppAccess returns an error if the Argo CD Application
// represented by appMeta does not explicitly permit access by the Kargo Stage
// represented by stepCtx.
func authorizeArgoCDAppAccess(
	stepCtx *promotion.StepContext,
	appMeta metav1.ObjectMeta,
) error {
	permErr := fmt.Errorf( // nolint:staticcheck
		"Argo CD Application %q in namespace %q does not permit access by "+
			"Kargo Stage %s in namespace %s",
		appMeta.Name,
		appMeta.Namespace,
		stepCtx.Stage,
		stepCtx.Project,
	)

	allowedStage, ok := appMeta.Annotations[kargoapi.AnnotationKeyAuthorizedStage]
	if !ok {
		return permErr
	}

	tokens := strings.SplitN(allowedStage, ":", 2)
	if len(tokens) != 2 {
		return fmt.Errorf(
			"unable to parse value of annotation %q (%q) on "+
				"Argo CD Application %q in namespace %q",
			kargoapi.AnnotationKeyAuthorizedStage,
			allowedStage,
			appMeta.Name,
			appMeta.Namespace,
		)
	}

	projectName, stageName := tokens[0], tokens[1]
	if strings.Contains(projectName, "*") || strings.Contains(stageName, "*") {
		return fmt.Errorf( // nolint:staticcheck
			"Argo CD Application %q in namespace %q has deprecated glob "+
				"expression in annotation %q (%q)",
			appMeta.Name,
			appMeta.Namespace,
			kargoapi.AnnotationKeyAuthorizedStage,
			allowedStage,
		)
	}
	if projectName != stepCtx.Project || stageName != stepCtx.Stage {
		return permErr
	}
	return nil
}

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
