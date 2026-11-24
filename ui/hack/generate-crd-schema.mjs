#!/usr/bin/env zx

/* eslint-disable no-undef */

import jsonStringify from 'json-stable-stringify';

import { crdSchemaToJSONSchema } from './crd-to-json-schema.mjs';

const crdDir = path.join(__dirname, '../../charts/kargo/resources/crds');
const crdFiles = (await $`ls '${crdDir}'`.quiet()).stdout
  .split('\n')
  .filter((f) => f.endsWith('.yaml'));
const outDir = path.join(__dirname, '../src/gen/schema');
await $`mkdir -p '${outDir}'`.quiet();
await $`rm -r '${outDir}/'`.quiet();

for (const crdFile of crdFiles) {
  const crd = YAML.parse(await fs.readFile(path.join(crdDir, crdFile), 'utf8'));
  const name = crd.metadata.name;
  for (const version of crd.spec.versions) {
    const outputPath = path.join(outDir, `${name}_${version.name}.json`);
    const schema = crdSchemaToJSONSchema(version.schema.openAPIV3Schema, `${name}/${version.name}`);
    await fs.outputFile(outputPath, jsonStringify(schema, { space: 2 }));
  }
}
