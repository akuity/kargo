package subscription

import (
	"embed"
	"fmt"
	"net/http"
	"slices"

	"github.com/xeipuuv/gojsonschema"
)

// additionalPropertyNotAllowed is the gojsonschema error type for a field that
// a schema with "additionalProperties": false doesn't define.
const additionalPropertyNotAllowed = "additional_property_not_allowed"

var (
	//go:embed schemas/*.json
	embeddedSchemasFS embed.FS
	schemasFS         = http.FS(embeddedSchemasFS)
)

// UnknownFields validates the raw, JSON-encoded configuration of a
// subscription of the given type (e.g. "image") against the type's JSON schema
// and returns, sorted, the name of each field the schema doesn't define.
// Decoding a subscription into its typed struct silently drops such fields.
// Other schema violations are left to each Subscriber's ValidateSubscription.
// Types without a schema have no unknown fields.
func UnknownFields(subType string, cfg []byte) ([]string, error) {
	if _, err := embeddedSchemasFS.Open(schemaPath(subType)); err != nil {
		return nil, nil
	}
	res, err := gojsonschema.Validate(
		gojsonschema.NewReferenceLoaderFileSystem(
			"file:///"+schemaPath(subType),
			schemasFS,
		),
		gojsonschema.NewBytesLoader(cfg),
	)
	if err != nil {
		return nil, fmt.Errorf("error validating %s subscription: %w", subType, err)
	}
	var fields []string
	for _, resErr := range res.Errors() {
		if resErr.Type() != additionalPropertyNotAllowed {
			continue
		}
		if name, ok := resErr.Details()["property"].(string); ok {
			fields = append(fields, name)
		}
	}
	slices.Sort(fields)
	return fields, nil
}

func schemaPath(subType string) string {
	return fmt.Sprintf("schemas/%s.json", subType)
}
