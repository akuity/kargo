package directives

import (
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestPromotionStep_GetConfig(t *testing.T) {
	promoCtx := PromotionContext{
		Project:   "fake-project",
		Stage:     "fake-stage",
		Promotion: "fake-promotion",
		Vars: []kargoapi.PromotionVariable{
			{
				Name:  "strVar",
				Value: "foo",
			},
			{
				Name:  "concatStrVar",
				Value: "${{ vars.strVar }}bar",
			},
			{
				Name:  "boolVar",
				Value: "true",
			},
			{
				Name: "boolStrVar",
				// Prove boolVar evaluated as a boolean
				Value: "${{ quote(!vars.boolVar) }}",
			},
			{
				Name:  "numVar",
				Value: "42",
			},
			{
				Name: "numStrVar",
				// Prove numVar evaluated as a number
				Value: "${{ quote(vars.numVar + 1) }}",
			},
		},
		Secrets: map[string]map[string]string{
			"secret1": {
				"key1": "value1",
				"key2": "value2",
			},
			"secret2": {
				"key3": "value3",
				"key4": "value4",
			},
		},
	}
	promoState := State{
		"strOutput":  "foo",
		"boolOutput": true,
		"numOutput":  42,
	}
	promoStep := PromotionStep{
		Config: []byte(`{
			"project": "${{ ctx.project }}",
			"stage": "${{ ctx.stage }}",
			"promotion": "${{ ctx.promotion }}",
			"staticStr": "foo",
			"staticBool": true,
			"staticNum": 42,
			"strVar": "${{ vars.strVar }}",
			"concatStrVar": "${{ vars.concatStrVar }}",
			"boolVar": "${{ vars.boolVar }}",
			"boolStrVar": "${{ quote(vars.boolStrVar) }}",
			"numVar": "${{ vars.numVar }}",
			"numStrVar": "${{ quote(vars.numStrVar) }}",
			"secret1-1": "${{ secrets.secret1.key1 }}",
			"secret1-2": "${{ secrets.secret1.key2 }}",
			"secret2-3": "${{ secrets.secret2.key3 }}",
			"secret2-4": "${{ secrets.secret2.key4 }}",
			"strOutput": "${{ outputs.strOutput }}",
			"strOutputConcat": "${{ outputs.strOutput }}${{ outputs.strOutput }}",
			"boolOutput": "${{ outputs.boolOutput }}",
			"boolStrOutput": "${{ quote(!outputs.boolOutput) }}",
			"numOutput": "${{ outputs.numOutput }}",
			"numStrOutput": "${{ quote(outputs.numOutput + 1) }}"
		}`),
	}
	stepCfg, err := promoStep.GetConfig(promoCtx, promoState)
	require.NoError(t, err)
	require.Equal(
		t,
		Config{
			"project":         "fake-project",
			"stage":           "fake-stage",
			"promotion":       "fake-promotion",
			"staticStr":       "foo",
			"staticBool":      true,
			"staticNum":       42,
			"strVar":          "foo",
			"concatStrVar":    "foobar",
			"boolVar":         true,
			"boolStrVar":      "false",
			"numVar":          42,
			"numStrVar":       "43",
			"secret1-1":       "value1",
			"secret1-2":       "value2",
			"secret2-3":       "value3",
			"secret2-4":       "value4",
			"strOutput":       "foo",
			"strOutputConcat": "foofoo",
			"boolOutput":      true,
			"boolStrOutput":   "false",
			"numOutput":       42,
			"numStrOutput":    "43",
		},
		stepCfg,
	)
}
