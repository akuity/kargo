package promotions

import (
	"context"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libargocd "github.com/akuity/kargo/pkg/argocd"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	rbuiltin "github.com/akuity/kargo/pkg/promotion/runner/builtin"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// argoCDSelectorStepConfig is the minimal subset of an argocd-update or
// argocd-wait step config needed to determine whether the step targets an Argo
// CD Application by label selector. It reuses builtin.ArgoCDAppSelector so the
// selector shape stays in sync with the step runners.
type argoCDSelectorStepConfig struct {
	Apps []struct {
		Namespace string                     `json:"namespace,omitempty"`
		Selector  *builtin.ArgoCDAppSelector `json:"selector,omitempty"`
	} `json:"apps"`
}

// promotionSelectorsMatchApp reports whether any label-selector-based
// argocd-update or argocd-wait step in the (running) Promotion targets the Argo
// CD Application identified by appNamespace and appLabels.
//
// This is the forward half of the scoped-forward-scan that handles
// label-selector targeting: the Promotion is the selector-bearing (intrinsic)
// side, and the changed Application is the query. The step config is evaluated
// exactly as the engine evaluates it for the step runner, and the selector is
// built with the same helper the runner uses, so the result cannot drift
// relative to what the step would actually match.
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
	evaluator := promotion.NewStepEvaluator(cl, nil, nil)

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

		// Evaluate the step config exactly as the engine does for the step
		// runner, so any expressions in the selector or namespace are resolved
		// identically.
		dirStep := promotion.Step{
			Kind:   step.Uses,
			Alias:  step.As,
			Vars:   step.Vars,
			Config: step.Config.Raw,
		}
		evaledCfg, err := evaluator.Config(ctx, promoCtx, dirStep)
		if err != nil {
			logger.Error(err, "error evaluating step config; ignoring step", "step", i)
			continue
		}
		cfg, err := promotion.ConfigToStruct[argoCDSelectorStepConfig](evaledCfg)
		if err != nil {
			logger.Error(err, "error converting step config; ignoring step", "step", i)
			continue
		}

		for _, app := range cfg.Apps {
			if app.Selector == nil {
				// Name-based entries are handled by the name index.
				continue
			}

			// The targeted namespace defaults to the Argo CD controller namespace.
			namespace := app.Namespace
			if namespace == "" {
				namespace = libargocd.Namespace()
			}
			if namespace != appNamespace {
				continue
			}

			selector, err := rbuiltin.BuildArgoCDAppLabelSelector(app.Selector)
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
