package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/expr-lang/expr"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindJSONParse = "json-parse"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindJSONParse,
			Value: newJSONParser,
		},
	)
}

// jsonParser is an implementation of the promotion.StepRunner interface that
// parses a JSON file and extracts specified outputs.
type jsonParser struct {
	schemaLoader gojsonschema.JSONLoader
}

// newJSONParser returns a new instance of jsonParser.
func newJSONParser(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &jsonParser{schemaLoader: getConfigSchemaLoader(stepKindJSONParse)}
}

// Run implements the promotion.StepRunner interface.
func (jp *jsonParser) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := jp.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return jp.run(ctx, stepCtx, cfg)
}

// convert validates jsonParser configuration against a JSON schema and
// converts it into a builtin.JSONParseConfig struct.
func (jp *jsonParser) convert(cfg promotion.Config) (builtin.JSONParseConfig, error) {
	return validateAndConvert[builtin.JSONParseConfig](jp.schemaLoader, cfg, stepKindJSONParse)
}

func (jp *jsonParser) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.JSONParseConfig,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	if cfg.Path == "" {
		return failure, fmt.Errorf("JSON file path cannot be empty")
	}

	if len(cfg.Outputs) == 0 {
		return failure, fmt.Errorf("invalid %s config: outputs is required", stepKindJSONParse)
	}

	data, err := jp.readAndParseJSON(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return failure, err
	}

	extractedValues, err := jp.extractValues(data, cfg.Outputs)
	if err != nil {
		return failure, fmt.Errorf("failed to extract outputs: %w", err)
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: extractedValues,
	}, nil
}

// readAndParseJSON reads and unmarshals a JSON document and returns the result.
func (jp *jsonParser) readAndParseJSON(
	workDir string,
	path string,
) (any, error) {
	absFilePath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return nil, fmt.Errorf("error joining path %q: %w", path, err)
	}
	jsonData, err := os.ReadFile(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading JSON file %q: %w", absFilePath, err)
	}
	var data any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("could not parse JSON file: %w", err)
	}
	return data, nil
}

// extractValues returns select data extracted from the provided data by
// evaluating it against expressions contained within the provided
// []builtin.JSONParse.
func (jp *jsonParser) extractValues(
	data any,
	outputs []builtin.JSONParse,
) (map[string]any, error) {
	results := make(map[string]any, len(outputs))
	for _, output := range outputs {
		program, err := expr.Compile(output.FromExpression)
		if err != nil {
			return nil, fmt.Errorf(
				"error compiling expression %q: %w",
				output.FromExpression, err,
			)
		}
		value, err := expr.Run(program, data)
		if err != nil {
			return nil, fmt.Errorf(
				"error evaluating expression %q: %w",
				output.FromExpression, err,
			)
		}
		results[output.Name] = value
	}
	return results, nil
}
