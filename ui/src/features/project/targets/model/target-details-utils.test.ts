import { describe, expect, test } from 'vitest';

import { Freight } from '@ui/gen/api/v2/models';

import { describeArtifact, flattenParams, freightArtifacts } from './target-details-utils';

describe('flattenParams()', () => {
  test('nests into dotted keys and sorts them', () => {
    expect(
      flattenParams({
        replicas: 10,
        cluster: { server: 'kubernetes.example.com', region: 'us-center-5' },
        branch: 'env/prod-us'
      })
    ).toEqual([
      { key: 'branch', value: 'env/prod-us' },
      { key: 'cluster.region', value: 'us-center-5' },
      { key: 'cluster.server', value: 'kubernetes.example.com' },
      { key: 'replicas', value: '10' }
    ]);
  });

  test('arrays and other values print as JSON, since a step reads them whole', () => {
    expect(flattenParams({ zones: ['a', 'b'], enabled: true, none: null })).toEqual([
      { key: 'enabled', value: 'true' },
      { key: 'none', value: 'null' },
      { key: 'zones', value: '["a","b"]' }
    ]);
  });

  test('no params, no rows', () => {
    expect(flattenParams()).toEqual([]);
  });

  test('an empty object still names its key, rather than vanishing', () => {
    expect(flattenParams({ branch: 'main', featureFlags: {} })).toEqual([
      { key: 'branch', value: 'main' },
      { key: 'featureFlags', value: '{}' }
    ]);
  });
});

describe('describeArtifact()', () => {
  test.each([
    [
      { type: 'image', repoURL: 'ghcr.io/example/otel-app', tag: '0.2.0' },
      { kind: 'Image', text: 'otel-app:0.2.0' }
    ],
    [
      { type: 'helm', repoURL: 'oci://example.com/charts', name: 'otel', version: '1.4.0' },
      { kind: 'Chart', text: 'otel:1.4.0' }
    ],
    [
      { type: 'git', repoURL: 'https://github.com/example/repo', id: '0123456789abcdef' },
      { kind: 'Commit', text: 'repo@0123456' }
    ]
  ])('%j -> %j', (source, expected) => {
    expect(describeArtifact(source as Parameters<typeof describeArtifact>[0])).toEqual(expected);
  });

  test('a trailing slash does not swallow the repository name', () => {
    expect(
      describeArtifact({ type: 'image', repoURL: 'ghcr.io/example/otel-app/', tag: 'v1' })
    ).toEqual({ kind: 'Image', text: 'otel-app:v1' });
  });
});

describe('freightArtifacts()', () => {
  const freightMap: Record<string, Freight> = {
    f1: {
      metadata: { name: 'f1' },
      origin: { kind: 'Warehouse', name: 'w1' },
      images: [{ repoURL: 'ghcr.io/example/app', tag: 'v1.2.3' }]
    },
    f2: {
      metadata: { name: 'f2' },
      origin: { kind: 'Warehouse', name: 'w2' },
      commits: [{ repoURL: 'https://github.com/example/config', id: '0123456789abcdef' }]
    }
  };

  test("describes each named Freight's artifacts, as the Project records them", () => {
    expect(freightArtifacts(['f1', 'f2'], freightMap)).toEqual([
      { kind: 'Image', text: 'app:v1.2.3' },
      { kind: 'Commit', text: 'config@0123456' }
    ]);
  });

  test('skips Freight the Project no longer has', () => {
    expect(freightArtifacts(['gone', 'f1'], freightMap)).toEqual([
      { kind: 'Image', text: 'app:v1.2.3' }
    ]);
  });
});
