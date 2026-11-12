package subscription

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// gojsonschema error types that Schema.Validate handles specially
const (
	errTypeRequired           = "required"
	errTypeAdditionalProperty = "additional_property_not_allowed"
	errTypeEnum               = "enum"
	errTypeConditionThen      = "condition_then"
	errTypeConditionElse      = "condition_else"
	errTypeAllOf              = "number_all_of"
)

var (
	//go:embed schemas/*.json
	embeddedSchemasFS embed.FS
	schemasFS         = http.FS(embeddedSchemasFS)
)

// Schema is a compiled JSON schema for the configuration of one type of
// subscription.
type Schema struct {
	schema *gojsonschema.Schema
}

// mustLoadSchema compiles the embedded JSON schema for subscriptions of the
// given type (e.g. "image"). It panics if the schema can't be compiled.
func mustLoadSchema(subType string) *Schema {
	schema, err := gojsonschema.NewSchema(
		gojsonschema.NewReferenceLoaderFileSystem(
			fmt.Sprintf("file:///schemas/%s.json", subType),
			schemasFS,
		),
	)
	if err != nil {
		panic(fmt.Errorf("error compiling %s subscription schema: %w", subType, err))
	}
	return &Schema{schema: schema}
}

// Validate validates the raw, JSON-encoded configuration of a subscription
// against the Schema and returns an error for each violation, with f as the
// path to the configuration.
func (s *Schema) Validate(f *field.Path, cfg []byte) field.ErrorList {
	res, err := s.schema.Validate(gojsonschema.NewBytesLoader(cfg))
	if err != nil {
		return field.ErrorList{field.InternalError(f, err)}
	}
	var errs field.ErrorList
	for _, resErr := range res.Errors() {
		if fieldErr := toFieldError(f, resErr); fieldErr != nil {
			errs = append(errs, fieldErr)
		}
	}
	return errs
}

// toFieldError converts a schema violation into a field error, with f as the
// path to the validated configuration. It returns nil for errors that only
// summarize others, such as an unsatisfied "then", whose causes are reported
// separately.
func toFieldError(f *field.Path, resErr gojsonschema.ResultError) *field.Error {
	path := toFieldPath(f, resErr.Context())
	switch resErr.Type() {
	case errTypeConditionThen, errTypeConditionElse, errTypeAllOf:
		return nil
	case errTypeRequired:
		return field.Required(path.Child(propertyDetail(resErr)), "")
	case errTypeAdditionalProperty:
		return field.Forbidden(path.Child(propertyDetail(resErr)), "unknown field")
	case errTypeEnum:
		return field.NotSupported(path, badValue(resErr), allowedDetail(resErr))
	default:
		return field.Invalid(path, badValue(resErr), resErr.Description())
	}
}

// badValue returns the value behind a schema violation. gojsonschema decodes
// numbers as json.Number, which would be reported as a quoted string, so they
// are converted to int64 or float64.
func badValue(resErr gojsonschema.ResultError) any {
	num, ok := resErr.Value().(json.Number)
	if !ok {
		return resErr.Value()
	}
	if i, err := num.Int64(); err == nil {
		return i
	}
	if f, err := num.Float64(); err == nil {
		return f
	}
	return num.String()
}

// toFieldPath converts the location of a schema violation, such as
// "(root).includePaths.1", into a path relative to f.
func toFieldPath(f *field.Path, ctx *gojsonschema.JsonContext) *field.Path {
	// The first element is always the root
	elems := strings.Split(ctx.String(), ".")[1:]
	for _, elem := range elems {
		if i, err := strconv.Atoi(elem); err == nil {
			f = f.Index(i)
		} else {
			f = f.Child(elem)
		}
	}
	return f
}

// propertyDetail returns the name of the property that a schema violation
// concerns, such as a missing, required property.
func propertyDetail(resErr gojsonschema.ResultError) string {
	property, _ := resErr.Details()["property"].(string)
	return property
}

// allowedDetail returns the values allowed by the "enum" behind a schema
// violation. gojsonschema provides them as a single, comma-separated list of
// JSON-encoded values.
func allowedDetail(resErr gojsonschema.ResultError) []string {
	allowed, _ := resErr.Details()["allowed"].(string)
	values := strings.Split(allowed, ", ")
	for i, value := range values {
		var str string
		if err := json.Unmarshal([]byte(value), &str); err == nil {
			values[i] = str
		}
	}
	return values
}
