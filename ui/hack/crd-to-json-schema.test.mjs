import { describe, expect, test } from 'vitest';

import { crdSchemaToJSONSchema, jsonSchemaDraft04 } from './crd-to-json-schema.mjs';

const convert = (schema) => crdSchemaToJSONSchema(schema, 'test');

// Every field apiextensions.k8s.io/v1 JSONSchemaProps defines, with a
// representative value. Taken from k8s.io/apiextensions-apiserver v0.37.0,
// pkg/apis/apiextensions/v1/types_jsonschema.go. This is the complete set of
// keywords a CRD can legally present, so a Kubernetes bump that adds a field
// should add it here and teach the converter what to do with it.
const everyKeyword = {
  id: 'some-id',
  $schema: 'http://json-schema.org/draft-07/schema#',
  $ref: '#/definitions/Thing',
  description: 'a description',
  type: 'object',
  format: 'date-time',
  title: 'a title',
  default: { a: 1 },
  maximum: 10,
  exclusiveMaximum: true,
  minimum: 1,
  exclusiveMinimum: true,
  maxLength: 5,
  minLength: 1,
  pattern: '^a$',
  maxItems: 3,
  minItems: 1,
  uniqueItems: true,
  multipleOf: 2,
  enum: ['a', 'b'],
  maxProperties: 4,
  minProperties: 1,
  required: ['a'],
  items: { type: 'string' },
  allOf: [{ type: 'string' }],
  oneOf: [{ type: 'string' }],
  anyOf: [{ type: 'string' }],
  not: { type: 'string' },
  properties: { a: { type: 'string' } },
  additionalProperties: true,
  patternProperties: { '^a': { type: 'string' } },
  dependencies: { a: ['b'] },
  additionalItems: false,
  definitions: { Thing: { type: 'string' } },
  externalDocs: { url: 'https://example.com' },
  example: { a: 1 },
  nullable: false,
  'x-kubernetes-preserve-unknown-fields': true,
  'x-kubernetes-embedded-resource': true,
  'x-kubernetes-int-or-string': true,
  'x-kubernetes-list-map-keys': ['a'],
  'x-kubernetes-list-type': 'map',
  'x-kubernetes-map-type': 'granular',
  'x-kubernetes-validations': [{ rule: 'self > 0', message: 'must be positive' }]
};

describe('keyword coverage', () => {
  test.each(Object.keys(everyKeyword))('accepts %s', (keyword) => {
    expect(() => convert({ [keyword]: everyKeyword[keyword] })).not.toThrow();
  });

  test('accepts every keyword at once', () => {
    expect(() => convert(everyKeyword)).not.toThrow();
  });

  test('rejects a keyword it does not recognize', () => {
    expect(() => convert({ type: 'object', discriminator: { propertyName: 'kind' } })).toThrow(
      /\/discriminator: unrecognized schema keyword/
    );
  });

  test('names the path to an unrecognized keyword', () => {
    expect(() =>
      convert({ properties: { spec: { properties: { a: { xml: { name: 'a' } } } } } })
    ).toThrow('test/properties/spec/properties/a/xml: unrecognized schema keyword');
  });

  test('rejects an unrecognized keyword nested in a list', () => {
    expect(() => convert({ anyOf: [{ type: 'string' }, { writeOnly: true }] })).toThrow(
      'test/anyOf[1]/writeOnly: unrecognized schema keyword'
    );
  });
});

describe('dialect', () => {
  test('declares draft-04', () => {
    expect(convert({ type: 'object' }).$schema).toBe(jsonSchemaDraft04);
  });

  test('overrides a dialect the CRD declares for itself', () => {
    expect(convert({ $schema: 'http://json-schema.org/draft-07/schema#' }).$schema).toBe(
      jsonSchemaDraft04
    );
  });
});

describe('integer bounds', () => {
  test('int32 gains the bounds its format implies', () => {
    expect(convert({ type: 'integer', format: 'int32' })).toMatchObject({
      minimum: -2147483648,
      maximum: 2147483647
    });
  });

  test('int64 gains the bounds its format implies', () => {
    expect(convert({ type: 'integer', format: 'int64' })).toMatchObject({
      minimum: -(2 ** 63),
      maximum: 2 ** 63
    });
  });

  test('leaves a bound the CRD declares for itself', () => {
    expect(convert({ type: 'integer', format: 'int64', minimum: 0 })).toMatchObject({
      minimum: 0,
      maximum: 2 ** 63
    });
  });

  test('leaves other formats alone', () => {
    const converted = convert({ type: 'string', format: 'date-time' });
    expect(converted.minimum).toBeUndefined();
    expect(converted.maximum).toBeUndefined();
  });

  test('reaches integers nested anywhere', () => {
    const converted = convert({
      properties: {
        list: { type: 'array', items: { type: 'integer', format: 'int32' } }
      }
    });
    expect(converted.properties.list.items.maximum).toBe(2147483647);
  });
});

describe('nullable', () => {
  test('becomes a union with the null type', () => {
    expect(convert({ type: 'string', nullable: true }).type).toEqual(['string', 'null']);
  });

  test('is dropped when the schema names no type', () => {
    const converted = convert({ nullable: true, description: 'anything goes' });
    expect(converted.nullable).toBeUndefined();
    expect(converted.type).toBeUndefined();
  });

  test('widens nothing when false', () => {
    const converted = convert({ type: 'string', nullable: false });
    expect(converted.type).toBe('string');
    expect(converted.nullable).toBeUndefined();
  });

  test('is never emitted, since draft-04 has no such keyword', () => {
    expect(convert({ type: 'string', nullable: true }).nullable).toBeUndefined();
  });
});

describe('field names that collide with keywords', () => {
  // A CRD is free to declare a field called "nullable" or "items". The Kargo
  // CRDs already declare "items" and "type".
  const collidingNames = ['nullable', 'items', 'type', 'example', 'properties', 'not', '$ref'];

  test.each(collidingNames)('a property named %s is a field, not a keyword', (name) => {
    const converted = convert({ type: 'object', properties: { [name]: { type: 'string' } } });
    expect(converted.properties[name]).toEqual({ type: 'string' });
  });

  test.each(collidingNames)('a definition named %s is a field, not a keyword', (name) => {
    const converted = convert({ definitions: { [name]: { type: 'integer', format: 'int32' } } });
    expect(converted.definitions[name].maximum).toBe(2147483647);
  });

  test('a pattern property named like a keyword is a field, not a keyword', () => {
    const converted = convert({ patternProperties: { nullable: { type: 'string' } } });
    expect(converted.patternProperties.nullable).toEqual({ type: 'string' });
  });
});

describe('values that are data rather than schemas', () => {
  test('default is copied, not walked', () => {
    const value = { nullable: true, discriminator: 'not a keyword here' };
    expect(convert({ type: 'object', default: value }).default).toEqual(value);
  });

  test('enum entries are copied, not walked', () => {
    const value = [{ xml: 'still just data' }];
    expect(convert({ enum: value }).enum).toEqual(value);
  });

  test('example is copied, not walked', () => {
    const value = { readOnly: true };
    expect(convert({ example: value }).example).toEqual(value);
  });
});

describe('subschema shapes', () => {
  test('items as a single schema', () => {
    expect(convert({ type: 'array', items: { type: 'integer', format: 'int32' } }).items).toEqual({
      type: 'integer',
      format: 'int32',
      minimum: -2147483648,
      maximum: 2147483647
    });
  });

  test('items as a list of schemas', () => {
    const converted = convert({ type: 'array', items: [{ type: 'integer', format: 'int32' }] });
    expect(converted.items[0].maximum).toBe(2147483647);
  });

  test('additionalProperties as a boolean', () => {
    expect(convert({ additionalProperties: true }).additionalProperties).toBe(true);
    expect(convert({ additionalProperties: false }).additionalProperties).toBe(false);
  });

  test('additionalProperties as a schema', () => {
    const converted = convert({ additionalProperties: { type: 'integer', format: 'int64' } });
    expect(converted.additionalProperties.maximum).toBe(2 ** 63);
  });

  test('additionalItems as a schema', () => {
    const converted = convert({ additionalItems: { type: 'integer', format: 'int32' } });
    expect(converted.additionalItems.maximum).toBe(2147483647);
  });

  test('not, allOf, anyOf and oneOf are converted', () => {
    const int32 = { type: 'integer', format: 'int32' };
    const converted = convert({ not: int32, allOf: [int32], anyOf: [int32], oneOf: [int32] });
    expect(converted.not.maximum).toBe(2147483647);
    expect(converted.allOf[0].maximum).toBe(2147483647);
    expect(converted.anyOf[0].maximum).toBe(2147483647);
    expect(converted.oneOf[0].maximum).toBe(2147483647);
  });
});

describe('dependencies', () => {
  test('a schema dependency is converted', () => {
    const converted = convert({
      dependencies: { a: { properties: { b: { type: 'integer', format: 'int32' } } } }
    });
    expect(converted.dependencies.a.properties.b.maximum).toBe(2147483647);
  });

  test('a list of co-required field names is preserved', () => {
    expect(convert({ dependencies: { a: ['b', 'c'] } }).dependencies.a).toEqual(['b', 'c']);
  });

  test('an unrecognized keyword inside a schema dependency is still caught', () => {
    expect(() => convert({ dependencies: { a: { readOnly: true } } })).toThrow(
      'test/dependencies/a/readOnly: unrecognized schema keyword'
    );
  });
});

describe('Kubernetes extensions', () => {
  test.each([
    ['x-kubernetes-preserve-unknown-fields', true],
    ['x-kubernetes-embedded-resource', true],
    ['x-kubernetes-int-or-string', true],
    ['x-kubernetes-list-map-keys', ['name']],
    ['x-kubernetes-list-type', 'map'],
    ['x-kubernetes-map-type', 'granular'],
    ['x-kubernetes-validations', [{ rule: 'self > 0' }]]
  ])('%s survives conversion', (keyword, value) => {
    expect(convert({ type: 'object', [keyword]: value })[keyword]).toEqual(value);
  });
});

describe('purity', () => {
  test('leaves the input untouched', () => {
    const input = {
      type: 'object',
      properties: { a: { type: 'integer', format: 'int32' }, b: { type: 'string', nullable: true } }
    };
    const before = structuredClone(input);
    convert(input);
    expect(input).toEqual(before);
  });
});
