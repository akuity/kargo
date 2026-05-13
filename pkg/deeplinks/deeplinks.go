package deeplinks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/expr-lang/expr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

var sprigFuncMap template.FuncMap

func init() {
	sprigFuncMap = sprig.GenericFuncMap()
	// Prevent templates from reading host environment or performing DNS lookups.
	delete(sprigFuncMap, "env")
	delete(sprigFuncMap, "expandenv")
	delete(sprigFuncMap, "getHostByName")
}

// ResolvedLink is a DeepLink whose URL template has been evaluated against a
// specific resource.
type ResolvedLink struct {
	Title       string
	URL         string
	Description string
}

// EvaluateLinks evaluates each DeepLink against ctx, applying any If
// conditions and resolving URL templates. Links whose conditions evaluate to
// false are silently omitted. Non-fatal evaluation errors are collected and
// returned alongside resolved links so callers can surface them without
// aborting the whole response.
func EvaluateLinks(
	links []kargoapi.DeepLink,
	ctx map[string]any,
) ([]ResolvedLink, []string) {
	resolved := make([]ResolvedLink, 0, len(links))
	var errs []string

	for _, link := range links {
		if link.If != "" {
			out, err := expr.Eval(link.If, ctx)
			if err != nil {
				errs = append(errs, fmt.Sprintf(
					"error evaluating condition for link %q: %v", link.Title, err,
				))
				continue
			}
			condResult, ok := out.(bool)
			if !ok {
				errs = append(errs, fmt.Sprintf(
					"condition for link %q evaluated to non-boolean value", link.Title,
				))
				continue
			}
			if !condResult {
				continue
			}
		}

		t, err := template.New("").Funcs(sprigFuncMap).Parse(link.URL)
		if err != nil {
			errs = append(errs, fmt.Sprintf(
				"error parsing URL template for link %q: %v", link.Title, err,
			))
			continue
		}
		var buf bytes.Buffer
		if err = t.Execute(&buf, ctx); err != nil {
			errs = append(errs, fmt.Sprintf(
				"error evaluating URL template for link %q: %v", link.Title, err,
			))
			continue
		}

		resolved = append(resolved, ResolvedLink{
			Title:       link.Title,
			URL:         buf.String(),
			Description: link.Description,
		})
	}

	return resolved, errs
}

// FreightContext converts a Freight resource into a template context map with
// a single "freight" key for use with EvaluateLinks.
func FreightContext(freight *kargoapi.Freight) (map[string]any, error) {
	return toContextMap("freight", freight)
}

// StageContext converts a Stage resource into a template context map with a
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
