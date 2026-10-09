import { expect, test } from 'vitest';

import { stepDescriptionForManifest } from './utils';

test('stepDescriptionForManifest', () => {
  expect(stepDescriptionForManifest('does a thing')).toBe('does a thing');
  expect(stepDescriptionForManifest('  does a thing \n')).toBe('does a thing');

  // blank values are omitted: the API rejects an empty description
  expect(stepDescriptionForManifest(undefined)).toBeUndefined();
  expect(stepDescriptionForManifest('')).toBeUndefined();
  expect(stepDescriptionForManifest('   ')).toBeUndefined();
});
