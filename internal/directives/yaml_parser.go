package directives

import (
	"context"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/expr-lang/expr"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func init() {
	builtins.RegisterPromotionStepRunner(newYAMLParser(), nil)
}

// yamlParser is an implementation of the PromotionStepRunner interface that
// parses a YAML file and extracts specified outputs.
type yamlParser struct {
	schemaLoader gojsonschema.JSONLoader
}

// newYAMLParser returns a new instance of yamlParser.
func newYAMLParser() PromotionStepRunner {
	r := &yamlParser{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (yp *yamlParser) Name() string {
	return "yaml-parse"
}

func (yp *yamlParser) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := yp.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	cfg, err := ConfigToStruct[YAMLParseConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", yp.Name(), err)
	}

	return yp.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates yamlParser configuration against a YAML schema.
func (yp *yamlParser) validate(cfg Config) error {
	return validate(yp.schemaLoader, gojsonschema.NewGoLoader(cfg), yp.Name())
}

func (yp *yamlParser) runPromotionStep(
	_ context.Context,
	stepCtx *PromotionStepContext,
	cfg YAMLParseConfig,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if cfg.Path == "" {
		return failure, fmt.Errorf("YAML file path cannot be empty")
	}

	if len(cfg.Outputs) == 0 {
		return failure, fmt.Errorf("invalid yaml-parse config: outputs is required")
	}

	data, err := yp.readAndParseYAML(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return failure, err
	}

	extractedValues, err := yp.extractValues(data, cfg.Outputs)
	if err != nil {
		return failure, fmt.Errorf("failed to extract outputs: %w", err)
	}

	return PromotionStepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: extractedValues,
	}, nil
}

// readAndParseYAML reads a YAML file and unmarshals it into a map.
func (yp *yamlParser) readAndParseYAML(workDir string, path string) (map[string]any, error) {

	absFilePath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return nil, fmt.Errorf("error joining path %q: %w", path, err)
	}

	yamlData, err := os.ReadFile(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file %q: %w", absFilePath, err)
	}

	if len(yamlData) == 0 {
		return nil, fmt.Errorf("could not parse empty YAML file: %q", absFilePath)
	}

	var data map[string]any
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("could not parse YAML file: %w", err)
	}

	return data, nil
}

// extractValues evaluates JSONPath expressions using expr and returns extracted values.
func (yp *yamlParser) extractValues(data map[string]any, outputs []YAMLParse) (map[string]any, error) {
	results := make(map[string]any)

	for _, output := range outputs {
		program, err := expr.Compile(output.FromExpression, expr.Env(data))
		if err != nil {
			return nil, fmt.Errorf("error compiling expression %s: %w", output.FromExpression, err)
		}

		value, err := expr.Run(program, data)
		if err != nil {
			return nil, fmt.Errorf("error evaluating expression %s: %w", output.FromExpression, err)
		}

		results[output.Name] = value
	}

	return results, nil
}
