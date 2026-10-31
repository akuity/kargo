import { expect, test } from 'vitest';

import { artifactURL } from './artifact-parts-utils';

test('artifactURL links oci:// repositories over https', () => {
  expect(artifactURL('oci://ghcr.io/akuity/charts')).toBe('https://ghcr.io/akuity/charts');
  expect(artifactURL('https://ghcr.io/akuity/charts')).toBe('https://ghcr.io/akuity/charts');
});
