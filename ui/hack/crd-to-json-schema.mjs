// Converts the OpenAPI v3 schema embedded in a Kubernetes CRD to JSON Schema
// draft-04, the dialect the UI's YAML editor and form generators consume.
//
// The input is never arbitrary OpenAPI. The API server accepts only the field
// set that apiextensions.k8s.io/v1 JSONSchemaProps defines, so every keyword
// that can legally appear is enumerated below. Anything outside that set is
// rejected rather than copied blindly, so a CRD that grows a construct this
// does not understand fails codegen loudly instead of yielding a schema that
// is quietly wrong.

export const jsonSchemaDraft04 = 'http://json-schema.org/draft-04/schema#';

// Keywords draft-04 shares with OpenAPI v3, carried over verbatim. `default`
// and `enum` hold arbitrary user-supplied JSON, so they are copied rather than
// walked: their contents are values, not schemas. `exclusiveMinimum` and
// `exclusiveMaximum` are booleans in both dialects and so need no translation.
const passthroughKeywords = [
  '$ref',
  '$schema',
  'default',
  'description',
  'enum',
  'exclusiveMaximum',
  'exclusiveMinimum',
  'format',
  'id',
  'maxItems',
  'maxLength',
  'maxProperties',
  'maximum',
  'minItems',
  'minLength',
  'minProperties',
  'minimum',
  'multipleOf',
  'pattern',
  'required',
  'title',
  'type',
  'uniqueItems'
];

// Kubernetes' own extensions, plus the two OpenAPI annotations JSONSchemaProps
// permits. None carry meaning for a draft-04 validator, which ignores keywords
// it does not know, so they are kept rather than stripped: that leaves the
// generated schemas a faithful record of the CRD they came from.
const annotationKeywords = [
  'example',
  'externalDocs',
  'x-kubernetes-embedded-resource',
  'x-kubernetes-int-or-string',
  'x-kubernetes-list-map-keys',
  'x-kubernetes-list-type',
  'x-kubernetes-map-type',
  'x-kubernetes-preserve-unknown-fields',
  'x-kubernetes-validations'
];

// Keywords whose value is a schema, an array of schemas, or a boolean.
const subSchemaKeywords = [
  'additionalItems',
  'additionalProperties',
  'allOf',
  'anyOf',
  'items',
  'not',
  'oneOf'
];

// Keywords whose value maps a user-chosen name to a schema. Their keys are
// field names rather than keywords, so they are never validated as keywords --
// a CRD is free to declare a property called "nullable" or "items".
const subSchemaMapKeywords = ['definitions', 'patternProperties', 'properties'];

// An integer format implies bounds that draft-04 states outright.
const integerBounds = {
  int32: { minimum: -2147483648, maximum: 2147483647 },
  // int64's true maximum, 2**63 - 1, has no exact double representation, so the
  // bound widens by one to the nearest value JSON can carry. The minimum is exact.
  int64: { minimum: -(2 ** 63), maximum: 2 ** 63 }
};

const convertSchemaMap = (schemas, schemaPath) =>
  Object.fromEntries(
    Object.entries(schemas).map(([name, schema]) => [
      name,
      convertSchema(schema, `${schemaPath}/${name}`)
    ])
  );

const convertSchema = (schema, schemaPath) => {
  if (Array.isArray(schema)) {
    return schema.map((item, i) => convertSchema(item, `${schemaPath}[${i}]`));
  }
  // `additionalProperties: true` and `additionalItems: false` are booleans
  // standing in for a schema.
  if (schema === null || typeof schema !== 'object') {
    return schema;
  }

  const converted = {};
  for (const [keyword, value] of Object.entries(schema)) {
    const keywordPath = `${schemaPath}/${keyword}`;
    if (keyword === 'nullable') {
      // Folded into `type` below. draft-04 has no `nullable` of its own.
      continue;
    } else if (passthroughKeywords.includes(keyword) || annotationKeywords.includes(keyword)) {
      converted[keyword] = value;
    } else if (subSchemaKeywords.includes(keyword)) {
      converted[keyword] = convertSchema(value, keywordPath);
    } else if (subSchemaMapKeywords.includes(keyword)) {
      converted[keyword] = convertSchemaMap(value, keywordPath);
    } else if (keyword === 'dependencies') {
      converted[keyword] = Object.fromEntries(
        Object.entries(value).map(([name, dependency]) => [
          name,
          // A dependency is either a schema or a list of co-required field names.
          Array.isArray(dependency)
            ? dependency
            : convertSchema(dependency, `${keywordPath}/${name}`)
        ])
      );
    } else {
      throw new Error(
        `${keywordPath}: unrecognized schema keyword; ` +
          'teach hack/crd-to-json-schema.mjs how to convert it'
      );
    }
  }

  // draft-04 spells nullability as a union with the null type. A schema that
  // names no type already admits null, so there is nothing to widen.
  if (schema.nullable === true && typeof converted.type === 'string') {
    converted.type = [converted.type, 'null'];
  }

  const bounds = integerBounds[converted.format];
  if (bounds) {
    // Never clobber a bound the CRD declares for itself.
    if (converted.minimum === undefined) {
      converted.minimum = bounds.minimum;
    }
    if (converted.maximum === undefined) {
      converted.maximum = bounds.maximum;
    }
  }

  return converted;
};

export const crdSchemaToJSONSchema = (schema, schemaPath) => ({
  ...convertSchema(schema, schemaPath),
  // Declared last so the dialect the output actually conforms to always wins,
  // even over an `$schema` the CRD set for itself.
  $schema: jsonSchemaDraft04
});
