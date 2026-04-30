package builtin

import (
	"errors"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	"github.com/akuity/kargo/pkg/promotion"
)

// validateAndConvert validates the given configuration document against the
// provided JSON schema loader and converts it into the specified type T. It
// returns an error if the validation fails or if there is an error during
// conversion. If the validation is successful and the conversion is successful,
// it returns the converted configuration of type T.
func validateAndConvert[T any](
	schemaLoader gojsonschema.JSONLoader,
	config promotion.Config,
	stepKind string,
) (T, error) {
	var zero T

	if err := validate(schemaLoader, gojsonschema.NewGoLoader(config), stepKind); err != nil {
		return zero, err
	}

	cfg, err := promotion.ConfigToStruct[T](config)
	if err != nil {
		return zero, fmt.Errorf("could not convert config into %s config: %w", stepKind, err)
	}

	return cfg, nil
}

// validate validates the given configuration document against the provided JSON
// schema loader. It returns an error if the validation fails or if there is an
// error during validation. If the validation is successful, it returns nil.
func validate(
	schemaLoader gojsonschema.JSONLoader,
	docLoader gojsonschema.JSONLoader,
	stepKind string,
) error {
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return fmt.Errorf("could not validate %s config: %w", stepKind, err)
	}
	if !result.Valid() {
		errs := make([]error, len(result.Errors()))
		for i, err := range result.Errors() {
			errs[i] = errors.New(err.String())
		}
		return fmt.Errorf("invalid %s config: %w", stepKind, errors.Join(errs...))
	}
	return nil
}
