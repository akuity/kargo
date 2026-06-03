package promotions

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/argocd"
	"github.com/akuity/kargo/pkg/expressions"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
)

// promotionSelectorsMatchApp reports whether any label-selector-based
// argocd-update or argocd-wait step in the (running) Promotion targets the Argo
// CD Application identified by appNamespace and appLabels.
//
// This is the forward half of the scoped-forward-scan that handles
// label-selector targeting: the Promotion is the selector-bearing (intrinsic)
// side, and the changed Application is the query. Selectors are evaluated
// against the Application's labels exactly as the argocd-update/argocd-wait
// steps would, so the result cannot go stale relative to the Stage.
func promotionSelectorsMatchApp(
	ctx context.Context,
	cl client.Client,
	promo *kargoapi.Promotion,
	appNamespace string,
	appLabels map[string]string,
) bool {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"promotion", promo.Name,
		"namespace", promo.Namespace,
	)

	// The Stage is required to build the expression context for the step config.
	stage := &kargoapi.Stage{}
	if err := cl.Get(
		ctx,
		client.ObjectKey{Namespace: promo.Namespace, Name: promo.Spec.Stage},
		stage,
	); err != nil {
		logger.Error(
			err, "failed to get Stage for Promotion",
			"stage", promo.Spec.Stage,
		)
		return false
	}
	promoCtx := promotion.NewContext(promo, stage)

	for i, step := range promo.Spec.Steps {
		if int64(i) > promo.Status.CurrentStep {
			// We are only interested in steps that have already been executed or
			// are about to be.
			break
		}
		if (step.Uses != "argocd-update" && step.Uses != "argocd-wait") ||
			step.Config == nil {
			continue
		}

		dirStep := promotion.Step{
			Kind:   step.Uses,
			Alias:  step.As,
			Vars:   step.Vars,
			Config: step.Config.Raw,
		}
		evaluator := promotion.NewStepEvaluator(cl, nil)
		vars, err := evaluator.Vars(ctx, promoCtx, dirStep)
		if err != nil {
			logger.Error(err, "error evaluating step vars; ignoring step", "step", i)
			continue
		}

		// Unpack only the fields we care about. We deliberately avoid unmarshaling
		// the entire config into a struct, both because we lack the context to
		// evaluate every expression and because templated fields may not be
		// strings in the struct.
		cfgMap := map[string]any{}
		if err = json.Unmarshal(step.Config.Raw, &cfgMap); err != nil {
			logger.Error(err, "error unmarshaling step config; ignoring step", "step", i)
			continue
		}
		appsList, ok := cfgMap["apps"].([]any)
		if !ok {
			continue
		}

		for _, app := range appsList {
			appMap, ok := app.(map[string]any)
			if !ok {
				continue
			}
			selMap, ok := appMap["selector"].(map[string]any)
			if !ok {
				// Not a selector-based entry (or malformed); name-based entries are
				// handled by the name index.
				continue
			}

			env := evaluator.BuildExprEnv(
				promoCtx,
				promotion.ExprEnvWithOutputs(promoCtx.State),
				promotion.ExprEnvWithTaskOutputs(dirStep.Alias, promoCtx.State),
				promotion.ExprEnvWithVars(vars),
			)

			// The targeted namespace defaults to the Argo CD controller namespace.
			namespace := libargocd.Namespace()
			if nsTemplate, ok := appMap["namespace"].(string); ok {
				evaluated, err := expressions.EvaluateTemplate(nsTemplate, env)
				if err != nil {
					logger.Error(err, "error evaluating app namespace; ignoring entry", "step", i)
					continue
				}
				if s, ok := evaluated.(string); ok {
					namespace = s
				}
			}
			if namespace != appNamespace {
				continue
			}

			selector, err := evaluateAppSelector(selMap, env)
			if err != nil {
				logger.Error(err, "error building app selector; ignoring entry", "step", i)
				continue
			}
			if selector.Matches(labels.Set(appLabels)) {
				return true
			}
		}
	}
	return false
}

// evaluateAppSelector converts an argocd step's selector config (as an
// unstructured map), evaluating any expressions in its values against env, into
// a labels.Selector. The selector config's JSON shape matches
// metav1.LabelSelector.
func evaluateAppSelector(
	selMap map[string]any,
	env map[string]any,
) (labels.Selector, error) {
	labelSelector := &metav1.LabelSelector{}

	if matchLabels, ok := selMap["matchLabels"].(map[string]any); ok {
		labelSelector.MatchLabels = make(map[string]string, len(matchLabels))
		for key, value := range matchLabels {
			tmpl, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("matchLabels value for %q is not a string", key)
			}
			evaluated, err := expressions.EvaluateTemplate(tmpl, env)
			if err != nil {
				return nil, fmt.Errorf("error evaluating matchLabels value for %q: %w", key, err)
			}
			labelSelector.MatchLabels[key] = fmt.Sprintf("%v", evaluated)
		}
	}

	if matchExpressions, ok := selMap["matchExpressions"].([]any); ok {
		for _, expr := range matchExpressions {
			exprMap, ok := expr.(map[string]any)
			if !ok {
				continue
			}
			key, _ := exprMap["key"].(string)
			operator, _ := exprMap["operator"].(string)
			req := metav1.LabelSelectorRequirement{
				Key:      key,
				Operator: metav1.LabelSelectorOperator(operator),
			}
			if values, ok := exprMap["values"].([]any); ok {
				for _, value := range values {
					tmpl, ok := value.(string)
					if !ok {
						return nil, fmt.Errorf("matchExpressions value for key %q is not a string", key)
					}
					evaluated, err := expressions.EvaluateTemplate(tmpl, env)
					if err != nil {
						return nil, fmt.Errorf("error evaluating matchExpressions value for key %q: %w", key, err)
					}
					req.Values = append(req.Values, fmt.Sprintf("%v", evaluated))
				}
			}
			labelSelector.MatchExpressions = append(labelSelector.MatchExpressions, req)
		}
	}

	return metav1.LabelSelectorAsSelector(labelSelector)
}
