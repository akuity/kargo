package builtin

import (
	"context"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/expr-lang/expr"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindYAMLParse = "yaml-parse"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindYAMLParse,
			Value: newYAMLParser,
		},
	)
}

// yamlParser is an implementation of the promotion.StepRunner interface that
// parses a YAML file and extracts specified outputs.
type yamlParser struct {
	schemaLoader gojsonschema.JSONLoader
}

// newYAMLParser returns a new instance of yamlParser.
func newYAMLParser(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &yamlParser{schemaLoader: getConfigSchemaLoader(stepKindYAMLParse)}
}

// Run implements the promotion.StepRunner interface.
func (yp *yamlParser) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := yp.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return yp.run(ctx, stepCtx, cfg)
}

// convert validates yamlParser configuration against a YAML schema and
// converts it into a builtin.YAMLParseConfig struct.
func (yp *yamlParser) convert(cfg promotion.Config) (builtin.YAMLParseConfig, error) {
	return validateAndConvert[builtin.YAMLParseConfig](yp.schemaLoader, cfg, stepKindYAMLParse)
}

func (yp *yamlParser) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.YAMLParseConfig,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	if cfg.Path == "" {
		return failure, fmt.Errorf("YAML file path cannot be empty")
	}

	if len(cfg.Outputs) == 0 {
		return failure, fmt.Errorf("invalid %s config: outputs is required", stepKindYAMLParse)
	}

	data, err := yp.readAndParseYAML(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return failure, err
	}

	extractedValues, err := yp.extractValues(data, cfg.Outputs)
	if err != nil {
		return failure, fmt.Errorf("failed to extract outputs: %w", err)
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: extractedValues,
	}, nil
}

// readAndParseYAML reads and unmarshals a YAML document and returns the result.
func (yp *yamlParser) readAndParseYAML(
	workDir string,
	path string,
) (any, error) {
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
	var data any
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("could not parse YAML file: %w", err)
	}
	return data, nil
}

// extractValues returns select data extracted from the provided data by
// evaluating it against expressions contained within the provided
// []builtin.YAMLParse.
func (yp *yamlParser) extractValues(
	data any,
	outputs []builtin.YAMLParse,
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
