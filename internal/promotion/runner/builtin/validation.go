package builtin

import (
	"errors"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

func validate(
	schemaLoader gojsonschema.JSONLoader,
	docLoader gojsonschema.JSONLoader,
	configKind string,
) error {
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return fmt.Errorf("could not validate %s config: %w", configKind, err)
	}
	if !result.Valid() {
		errs := make([]error, len(result.Errors()))
		for i, err := range result.Errors() {
			errs[i] = errors.New(err.String())
		}
		return fmt.Errorf("invalid %s config: %w", configKind, errors.Join(errs...))
	}
	return nil
}
