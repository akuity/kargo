package directives

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/expr-lang/expr"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/x/directive/builtin"
)

func init() {
	builtinsReg.RegisterPromotionStepRunner(newJSONParser(), nil)
}

// jsonParser is an implementation of the PromotionStepRunner interface that
// parses a JSON file and extracts specified outputs.
type jsonParser struct {
	schemaLoader gojsonschema.JSONLoader
}

// newJSONParser returns a new instance of jsonParser.
func newJSONParser() PromotionStepRunner {
	r := &jsonParser{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (jp *jsonParser) Name() string {
	return "json-parse"
}

func (jp *jsonParser) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := jp.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	cfg, err := ConfigToStruct[builtin.JSONParseConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", jp.Name(), err)
	}

	return jp.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates jsonParser configuration against a JSON schema.
func (jp *jsonParser) validate(cfg Config) error {
	return validate(jp.schemaLoader, gojsonschema.NewGoLoader(cfg), jp.Name())
}

func (jp *jsonParser) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg builtin.JSONParseConfig,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if cfg.Path == "" {
		return failure, fmt.Errorf("JSON file path cannot be empty")
	}

	if len(cfg.Outputs) == 0 {
		return failure, fmt.Errorf("invalid %s config: outputs is required", jp.Name())
	}

	data, err := jp.readAndParseJSON(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return failure, err
	}

	extractedValues, err := jp.extractValues(data, cfg.Outputs)
	if err != nil {
		return failure, fmt.Errorf("failed to extract outputs: %w", err)
	}

	return PromotionStepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: extractedValues,
	}, nil
}

// readAndParseJSON reads a JSON file and unmarshals it into a map.
func (jp *jsonParser) readAndParseJSON(workDir string, path string) (map[string]any, error) {

	absFilePath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return nil, fmt.Errorf("error joining path %q: %w", path, err)
	}

	jsonData, err := os.ReadFile(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file %q: %w", absFilePath, err)
	}

	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("could not parse JSON file: %w", err)
	}

	return data, nil
}

// extractValues evaluates JSONPath expressions using expr and returns extracted values.
func (jp *jsonParser) extractValues(data map[string]any, outputs []builtin.JSONParse) (map[string]any, error) {
	results := make(map[string]any)

	for _, output := range outputs {
		program, err := expr.Compile(output.FromExpression, expr.Env(data))
		if err != nil {
			return nil, fmt.Errorf("error compiling expression %q: %w", output.FromExpression, err)
		}

		value, err := expr.Run(program, data)
		if err != nil {
			return nil, fmt.Errorf("error evaluating expression %q: %w", output.FromExpression, err)
		}

		results[output.Name] = value
	}

	return results, nil
}
