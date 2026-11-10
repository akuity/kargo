package deeplinks

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
)

// legacyTemplateRegex matches a "{{" that does not open a "${{" expression,
// which means it is left over from the Go template syntax deep links used
// before.
var legacyTemplateRegex = regexp.MustCompile(`(^|[^$]){{`)

// ResolvedLink is a DeepLink whose URL expressions have been evaluated against
// a specific resource.
type ResolvedLink struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
} // @name ResolvedLink

// EvaluateLinks evaluates each DeepLink against ctx, applying any If
// conditions and evaluating expressions in URLs. Links whose conditions
// evaluate to false are silently omitted. Non-fatal evaluation errors are
// collected and returned alongside resolved links so callers can surface them
// without aborting the whole response.
//
// Expressions are evaluated with no Kargo-specific functions. In particular,
// functions that read Secrets must never be made available here: resolved
// links are shown to anyone who can view the resource.
func EvaluateLinks(
	links []kargoapi.DeepLink,
	ctx map[string]any,
) ([]ResolvedLink, []string) {
	resolved := make([]ResolvedLink, 0, len(links))
	var errs []string

	for _, link := range links {
		// Check for legacy syntax before the condition so that a link that was
		// only partially migrated still fails loudly when its condition is false.
		if err := checkLegacyURLTemplate(link.URL); err != nil {
			errs = append(errs, fmt.Sprintf(
				"invalid URL template for link %q: %v", link.Title, err,
			))
			continue
		}

		show, err := evaluateCondition(link.If, ctx)
		if err != nil {
			errs = append(errs, fmt.Sprintf(
				"error evaluating condition for link %q: %v", link.Title, err,
			))
			continue
		}
		if !show {
			continue
		}

		url, err := evaluateURLTemplate(link.URL, ctx)
		if err != nil {
			errs = append(errs, fmt.Sprintf(
				"error evaluating URL template for link %q: %v", link.Title, err,
			))
			continue
		}

		resolved = append(resolved, ResolvedLink{
			Title:       link.Title,
			URL:         url,
			Description: link.Description,
		})
	}

	return resolved, errs
}

// checkLegacyURLTemplate returns an error if tmpl contains Go template syntax.
// Without this check, such a template contains no expressions and would be
// returned verbatim, braces and all. Under the old engine every "{{" in a URL
// template was a template action, so this rejects nothing that used to work.
func checkLegacyURLTemplate(tmpl string) error {
	if legacyTemplateRegex.MatchString(tmpl) {
		return errors.New(
			"URL template uses Go template syntax " +
				"(e.g. {{ .freight.metadata.name }}), which is no longer " +
				"supported; use expr-lang expressions offset by ${{ }} instead " +
				"(e.g. ${{ freight.metadata.name }})",
		)
	}
	return nil
}

// evaluateCondition evaluates cond against ctx. An empty condition is true.
func evaluateCondition(cond string, ctx map[string]any) (bool, error) {
	// Text around an expression is kept in the result, so surrounding
	// whitespace would turn a boolean into a string.
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return true, nil
	}
	out, err := expressions.EvaluateTemplate(cond, ctx)
	if err != nil {
		return false, err
	}
	switch res := out.(type) {
	case bool:
		return res, nil
	case string:
		if !strings.Contains(cond, "${{") {
			// A condition with no expression in it comes back unevaluated.
			return false, errors.New(
				"condition must be a ${{ }} expression that evaluates to a boolean",
			)
		}
		return false, fmt.Errorf(
			"condition must evaluate to a boolean, got string %q", res,
		)
	default:
		return false, fmt.Errorf(
			"condition must evaluate to a boolean, got %T", out,
		)
	}
}

// evaluateURLTemplate evaluates any expressions in tmpl against ctx and
// returns the resulting URL.
func evaluateURLTemplate(tmpl string, ctx map[string]any) (string, error) {
	out, err := expressions.EvaluateTemplate(tmpl, ctx)
	if err != nil {
		return "", err
	}
	res, ok := out.(string)
	if !ok {
		// A URL template made of a single expression can evaluate to a number,
		// bool, list or object, none of which is a usable href.
		return "", fmt.Errorf("URL template evaluated to %T, not a string", out)
	}
	return res, nil
}

// FreightContext converts a Freight resource into an expression context map with
// a single "freight" key for use with EvaluateLinks.
func FreightContext(freight *kargoapi.Freight) (map[string]any, error) {
	return toContextMap("freight", freight)
}

// StageContext converts a Stage resource into an expression context map with a
// single "stage" key for use with EvaluateLinks.
func StageContext(stage *kargoapi.Stage) (map[string]any, error) {
	return toContextMap("stage", stage)
}

func toContextMap(key string, obj any) (map[string]any, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("error marshaling %s: %w", key, err)
	}
	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("error building context for %s: %w", key, err)
	}
	return map[string]any{key: m}, nil
}
