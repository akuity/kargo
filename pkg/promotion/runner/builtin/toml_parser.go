package builtin

import (
	"context"
	"fmt"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/expr-lang/expr"
	tomlv2 "github.com/pelletier/go-toml/v2"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const stepKindTOMLParse = "toml-parse"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name:  stepKindTOMLParse,
			Value: newTOMLParser,
		},
	)
}

// tomlParser is an implementation of the promotion.StepRunner interface that
// parses a TOML file and extracts specified outputs.
type tomlParser struct {
	schemaLoader gojsonschema.JSONLoader
}

// newTOMLParser returns a new instance of tomlParser.
func newTOMLParser(promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &tomlParser{schemaLoader: getConfigSchemaLoader(stepKindTOMLParse)}
}

// Run implements the promotion.StepRunner interface.
func (tp *tomlParser) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := tp.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return tp.run(ctx, stepCtx, cfg)
}

// convert validates tomlParser configuration against a JSON schema and
// converts it into a builtin.TOMLParseConfig struct.
func (tp *tomlParser) convert(cfg promotion.Config) (builtin.TOMLParseConfig, error) {
	return validateAndConvert[builtin.TOMLParseConfig](tp.schemaLoader, cfg, stepKindTOMLParse)
}

func (tp *tomlParser) run(
	_ context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.TOMLParseConfig,
) (promotion.StepResult, error) {
	failure := promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}

	if cfg.Path == "" {
		return failure, fmt.Errorf("TOML file path cannot be empty")
	}

	if len(cfg.Outputs) == 0 {
		return failure, fmt.Errorf("invalid %s config: outputs is required", stepKindTOMLParse)
	}

	data, err := tp.readAndParseTOML(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return failure, err
	}

	extractedValues, err := tp.extractValues(data, cfg.Outputs)
	if err != nil {
		return failure, fmt.Errorf("failed to extract outputs: %w", err)
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: extractedValues,
	}, nil
}

// readAndParseTOML reads and unmarshals a TOML document and returns the result.
func (tp *tomlParser) readAndParseTOML(workDir string, path string) (any, error) {
	absFilePath, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return nil, fmt.Errorf("error joining path %q: %w", path, err)
	}
	tomlData, err := os.ReadFile(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading TOML file %q: %w", absFilePath, err)
	}
	if len(tomlData) == 0 {
		return nil, fmt.Errorf("could not parse empty TOML file: %q", absFilePath)
	}
	var data any
	if err := tomlv2.Unmarshal(tomlData, &data); err != nil {
		return nil, fmt.Errorf("could not parse TOML file: %w", err)
	}
	return data, nil
}

// extractValues returns select data extracted from the provided data by
// evaluating it against expressions contained within the provided
// []builtin.TOMLParse.
func (tp *tomlParser) extractValues(
	data any,
	outputs []builtin.TomlParse,
) (map[string]any, error) {
	results := make(map[string]any, len(outputs))
	for _, output := range outputs {
		program, err := expr.Compile(output.FromExpression)
		if err != nil {
			return nil, fmt.Errorf(
				"error compiling expression %q: %w",
				output.FromExpression,
				err,
			)
		}
		value, err := expr.Run(program, data)
		if err != nil {
			return nil, fmt.Errorf(
				"error evaluating expression %q: %w",
				output.FromExpression,
				err,
			)
		}
		results[output.Name] = value
	}
	return results, nil
}
