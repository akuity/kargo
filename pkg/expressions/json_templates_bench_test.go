package expressions

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/require"
)

// BenchmarkEvaluateTemplate covers single template strings without caller
// functions, as the Promotion indexer, Promotion events and deep links evaluate
// them.
func BenchmarkEvaluateTemplate(b *testing.B) {
	env := benchEnv()
	testCases := []struct {
		name     string
		template string
	}{
		// No delimiter: the fast path most config values take.
		{name: "literal", template: "main"},
		{name: "single", template: "${{ vars.imageRepo }}"},
		{
			name:     "multi",
			template: "${{ ctx.project }}/${{ ctx.stage }}/${{ vars.env }}-values.yaml",
		},
		{name: "function", template: "${{ quote(outputs['commit-1'].commit) }}"},
	}
	for _, testCase := range testCases {
		b.Run(testCase.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := EvaluateTemplate(testCase.template, env); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkEvaluateJSONTemplate covers a whole step config: one expression per
// YAML update plus the step's path and commit message. It passes a caller
// function, as the Promotion evaluator does.
func BenchmarkEvaluateJSONTemplate(b *testing.B) {
	env := benchEnv()
	opt := expr.Function(
		"upper",
		func(a ...any) (any, error) {
			s, _ := a[0].(string)
			return strings.ToUpper(s), nil
		},
		new(func(string) string),
	)
	for _, updates := range []int{1, 10} {
		b.Run(fmt.Sprintf("updates-%d", updates), func(b *testing.B) {
			cfg := benchStepConfig(b, updates)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := EvaluateJSONTemplate(cfg, env, opt); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// benchEnv returns an environment shaped like the one Promotion steps see.
func benchEnv() map[string]any {
	outputs := map[string]any{}
	for i := range 10 {
		outputs[fmt.Sprintf("commit-%d", i)] = map[string]any{
			"commit": fmt.Sprintf("%040d", i),
		}
	}
	return map[string]any{
		"ctx": map[string]any{
			"project":   "fake-project",
			"promotion": "fake-promotion",
			"stage":     "fake-stage",
			"targetFreight": map[string]any{
				"name":   "fake-freight",
				"origin": map[string]any{"name": "fake-warehouse"},
			},
		},
		"vars": map[string]any{
			"env":       "prod",
			"imageRepo": "ghcr.io/example/app",
			"gitRepo":   "https://github.com/example/gitops.git",
		},
		"outputs": outputs,
	}
}

// benchStepConfig returns a yaml-update step config with the given number of
// templated updates.
func benchStepConfig(b *testing.B, updates int) []byte {
	b.Helper()
	cfgUpdates := make([]map[string]any, updates)
	for i := range updates {
		cfgUpdates[i] = map[string]any{
			"key":   fmt.Sprintf("services.svc-%d.image.tag", i),
			"value": fmt.Sprintf("${{ outputs['commit-%d'].commit }}", i),
		}
	}
	cfg, err := json.Marshal(map[string]any{
		"path":    "./out/${{ ctx.stage }}/${{ vars.env }}-values.yaml",
		"message": "Update ${{ ctx.stage }} to ${{ ctx.targetFreight.name }}",
		"updates": cfgUpdates,
	})
	require.NoError(b, err)
	return cfg
}
