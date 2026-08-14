import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { expect, test } from 'vitest';

import { useDiscoverPromotionDirectivesRegistries } from './use-discover-registries';

const SCHEMA_DIR = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../../gen/directives'
);

// steps are <step>-config.json; common.json and argocd-common.json are shared defs
const schemaStepNames = () =>
  fs
    .readdirSync(SCHEMA_DIR)
    .filter((file) => file.endsWith('-config.json'))
    .map((file) => file.replace(/-config\.json$/, ''))
    .sort();

const registeredIdentifiers = () =>
  useDiscoverPromotionDirectivesRegistries()
    .runners.map((runner) => runner.identifier)
    .sort();

// Every generated step schema must appear in the wizard registry exactly once.
test('every promotion step schema has a registry entry', () => {
  const stepNames = schemaStepNames();

  // catch a wrong SCHEMA_DIR instead of comparing two empty lists
  expect(stepNames.length).toBeGreaterThan(0);

  expect(registeredIdentifiers()).toEqual(stepNames);
});

test('registry does not register the same step twice', () => {
  const identifiers = registeredIdentifiers();

  expect(identifiers).toEqual([...new Set(identifiers)]);
});
