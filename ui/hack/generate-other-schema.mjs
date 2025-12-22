// source schemas from input directory (supplied as first argument)
// expand the references such that UI can parse easily
// output to output directory (supplied as second argument)

import { copyFileSync, mkdirSync, readdirSync, rmSync, writeFileSync } from 'fs';
import path from 'path';

import $RefParser from '@apidevtools/json-schema-ref-parser';

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
  if (process.argv.length < 4) {
    // eslint-disable-next-line no-console
    console.error('Usage: node generate-directive-schema.mjs <input-directory> <output-directory>');
    process.exit(1);
  }

  const inputDir = path.resolve(process.argv[2]);
  const outputDir = path.resolve(process.argv[3]);

  rmSync(outputDir, { recursive: true, force: true });
  const source = readdirSync(inputDir);

  if (source.length > 0) {
    mkdirSync(outputDir);
  }

  for (const file of source) {
    const fileBase = path.basename(file);
    if (file.endsWith('.json')) {
      copyFileSync(path.resolve(inputDir, file), path.resolve(outputDir, fileBase));
    }
  }

  const UISources = readdirSync(outputDir);

  for (const uiSource of UISources) {
    // $ref: any-file.json#./.. -> expand
    let referenced = await $RefParser.dereference(path.resolve(outputDir, uiSource));

    // remove 'anyOf', 'oneOf', 'allOf'
    referenced = removePropertiesRecursively(referenced, [
      'anyOf',
      'oneOf',
      'allOf',
      'required',
      'minItems'
    ]);

    writeFileSync(path.resolve(outputDir, uiSource), JSON.stringify(referenced, null, ' '));
  }
};

main();
