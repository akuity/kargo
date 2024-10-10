// source schemas from internal/directives/schemas/*.json
// expand the references such that UI can parse easily
// output to ui/src/gen/directives

import { copyFileSync, mkdirSync, readdirSync, rmSync, writeFileSync } from 'fs';
import path, { dirname } from 'path';
import { fileURLToPath } from 'url';

import $RefParser from '@apidevtools/json-schema-ref-parser';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 *
 * @param {JSONSchema7} schema
 * @param {string[]} props
 */
const removePropertiesRecursively = (schema, props) => {
  // remove keys
  for (const prop of props) {
    if (schema?.[prop]) {
      delete schema[prop];
    }
  }

  // recurse
  for (const [key, value] of Object.entries(schema || {})) {
    if (typeof value === 'object') {
      schema[key] = removePropertiesRecursively(value, props);
    }
  }

  return schema;
};

const main = async () => {
  const UIDirectivesDir = path.resolve(__dirname, '../src/gen/directives');
  const BackendDirectivesDir = path.resolve(__dirname, '../../internal/directives/schemas');
  rmSync(UIDirectivesDir, { recursive: true, force: true });
  const source = readdirSync(path.resolve(__dirname, BackendDirectivesDir));

  if (source.length > 0) {
    mkdirSync(UIDirectivesDir);
  }

  for (const file of source) {
    const fileBase = path.basename(file);
    if (file.endsWith('.json')) {
      copyFileSync(
        path.resolve(BackendDirectivesDir, file),
        path.resolve(UIDirectivesDir, fileBase)
      );
    }
  }

  const UISources = readdirSync(UIDirectivesDir);

  for (const uiSource of UISources) {
    // $ref: any-file.json#./.. -> expand
    let referenced = await $RefParser.dereference(path.resolve(UIDirectivesDir, uiSource));

    // remove 'anyOf', 'oneOf', 'allOf'
    referenced = removePropertiesRecursively(referenced, [
      'anyOf',
      'oneOf',
      'allOf',
      'required',
      'minItems'
    ]);

    writeFileSync(path.resolve(UIDirectivesDir, uiSource), JSON.stringify(referenced, null, ' '));
  }
};

main();
