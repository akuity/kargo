package expressions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/valyala/fasttemplate"
)

// EvaluateJSONTemplate evaluates a JSON byte slice, which is presumed to be a
// template containing expr-lang expressions offset by ${{ and }}, using the
// provided environment as context. The evaluated JSON is returned as a new byte
// slice, ready for unmarshaling.
//
// Only expressions contained within values are evaluated. i.e. Any expressions
// within keys are NOT evaluated.
//
// Since the template itself must be valid JSON, all expressions MUST be
// enclosed in quotes.
//
// If, after evaluating all expressions in a single value (multiples are
// permitted), the result can be parsed as a bool, float64, or other valid
// non-string JSON, it will be treated as such. This ensures the possibility of
// expressions being used to construct any valid JSON value, despite the fact
// that expressions must, themselves, be contained within a string value. This
// does mean that for expressions which may evaluate as something resembling a
// valid non-string JSON value, the user must take care to ensure that the
// expression evaluates to a string enclosed in quotes. e.g. ${{ true }} will
// evaluated as a bool, but ${{ quote(true) }} will be evaluated as a string.
// This behavior should be intuitive to anyone familiar with YAML.
func EvaluateJSONTemplate(jsonBytes []byte, env map[string]any) ([]byte, error) {
	if _, ok := env["quote"]; ok {
		return nil, fmt.Errorf(
			`"quote" is a forbidden key in the environment map; it is reserved for internal use`,
		)
	}
	env = maps.Clone(env) // We don't want to add the quote function to the user's map.
	env["quote"] = func(a any) string { return fmt.Sprintf(`"%v"`, a) }
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		return nil,
			fmt.Errorf("input is not valid JSON; are all expressions enclosed in quotes? %w", err)
	}
	if err := evaluateExpressions(parsed, env); err != nil {
		return nil, err
	}
	return json.Marshal(parsed)
}

// evaluateExpressions recursively evaluates all expressions contained within
// elements of a map[string]any or []any, updating those elements in place.
// Passing any other type to this function will have no effect. Expressions are
// evaluated using the provided environment map as context.
func evaluateExpressions(collection any, env map[string]any) error {
	switch col := collection.(type) {
	case map[string]any:
		for key, val := range col {
			switch v := val.(type) {
			case map[string]any:
				if err := evaluateExpressions(v, env); err != nil {
					return err
				}
			case []any:
				if err := evaluateExpressions(v, env); err != nil {
					return err
				}
			case string:
				var err error
				if col[key], err = evaluateTemplate(v, env); err != nil {
					return err
				}
			}
		}
	case []any:
		for i, val := range col {
			switch v := val.(type) {
			case map[string]any:
				if err := evaluateExpressions(v, env); err != nil {
					return err
				}
			case []any:
				if err := evaluateExpressions(v, env); err != nil {
					return err
				}
			case string:
				var err error
				if col[i], err = evaluateTemplate(v, env); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// evaluateTemplate evaluates a single template string with the provided
// environment. Note that a single template string can contain multiple
// expressions.
func evaluateTemplate(template string, env map[string]any) (any, error) {
	t := fasttemplate.New(template, "${{", "}}")
	out := &bytes.Buffer{}
	if _, err := t.ExecuteFunc(out, getExpressionEvaluator(env)); err != nil {
		return nil, err
	}
	result := out.String()
	// If the result is enclosed in quotes, this is probably the result of an
	// expression that deliberately enclosed the result in quotes to prevent it
	// from being mistaken for a number, bool, etc. e.g. ${{ quote(true) }}
	// instead of ${{ true }}. Strip the quotes and make no attempt to parse the
	// result as any other type.
	//
	// Note: There's an edge case where this is NOT the reason for the leading and
	// trailing quotes, but the likelihood of this occurring in the context in
	// which we are using this function is so low that it's not worth sacrificing
	// the convenience of this behavior.
	if len(result) > 1 && strings.HasPrefix(result, `"`) && strings.HasSuffix(result, `"`) {
		return result[1 : len(result)-1], nil
	}
	// If the result is parseable as a bool return that.
	if resBool, err := strconv.ParseBool(result); err == nil {
		return resBool, nil
	}
	// If the result is parseable as a float64, return that. float64 is used
	// because it can represent all JSON numbers.
	if resNum, err := strconv.ParseFloat(result, 64); err == nil {
		return resNum, nil
	}
	// If the result is valid JSON, return its unmarshaled value.
	var resMap any
	if err := json.Unmarshal([]byte(result), &resMap); err == nil {
		return resMap, nil
	}
	// If we get to here, just return the string.
	return result, nil
}

// getExpressionEvaluator returns a fasttemplate.TagFunc that evaluates input
// as a single expr-lang expression with the provided map as the environment.
func getExpressionEvaluator(env map[string]any) fasttemplate.TagFunc {
	return func(out io.Writer, expression string) (int, error) {
		program, err := expr.Compile(expression, expr.Env(env))
		if err != nil {
			return 0, err
		}
		result, err := expr.Run(program, env)
		if err != nil {
			return 0, err
		}
		if resStr, ok := result.(string); ok {
			// A string result can be written directly to the output as is.
			return out.Write([]byte(resStr))
		}
		// For non-string results, which could include nils, bools, numbers of any
		// type, structs, collections, etc. the result must be marshaled to JSON
		// before being written to the output.
		resJSON, err := json.Marshal(result)
		if err != nil {
			return 0, err
		}
		return out.Write(resJSON)
	}
}
