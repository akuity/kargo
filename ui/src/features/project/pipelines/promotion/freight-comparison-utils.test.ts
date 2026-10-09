import { describe, expect, test } from 'vitest';

import { repoLabel } from './freight-comparison-utils';

describe('repoLabel', () => {
  test('appends chart name for HTTP Helm repositories', () => {
    expect(
      repoLabel({
        type: 'helm',
        repoURL: 'https://charts.example.com',
        name: 'my-chart',
        version: '1.0.0'
      })
    ).toBe('https://charts.example.com/my-chart');
  });

  test('uses repoURL as-is for OCI Helm charts', () => {
    expect(
      repoLabel({
        type: 'helm',
        repoURL: 'oci://ghcr.io/example/my-chart',
        name: '',
        version: '1.0.0'
      })
    ).toBe('oci://ghcr.io/example/my-chart');
  });
});
