import { describe, expect, test } from 'vitest';

import { Freight } from '@ui/gen/api/v2/models';

import { getChangedArtifacts, getPreviousFreight } from './freight-artifact-changes-utils';

const makeFreight = (f: {
  name?: string;
  warehouse?: string;
  discoveredAt?: string;
  images?: Freight['images'];
  commits?: Freight['commits'];
  charts?: Freight['charts'];
  artifacts?: Freight['artifacts'];
}) =>
  ({
    metadata: { name: f.name || 'freight' },
    origin: { kind: 'Warehouse', name: f.warehouse || 'guestbook' },
    discoveredAt: f.discoveredAt,
    images: f.images,
    commits: f.commits,
    charts: f.charts,
    artifacts: f.artifacts
  }) as Freight;

describe('getPreviousFreight', () => {
  test('pairs each piece of freight with the one before it from the same Warehouse', () => {
    const freights = [
      makeFreight({ name: 'gb-1', warehouse: 'guestbook', discoveredAt: '2026-01-01T00:00:00Z' }),
      makeFreight({ name: 'api-1', warehouse: 'api', discoveredAt: '2026-01-02T00:00:00Z' }),
      makeFreight({ name: 'gb-3', warehouse: 'guestbook', discoveredAt: '2026-01-05T00:00:00Z' }),
      makeFreight({ name: 'api-2', warehouse: 'api', discoveredAt: '2026-01-04T00:00:00Z' }),
      makeFreight({ name: 'gb-2', warehouse: 'guestbook', discoveredAt: '2026-01-03T00:00:00Z' })
    ];

    const previous = getPreviousFreight(freights);

    expect(
      Object.fromEntries(Object.entries(previous).map(([name, f]) => [name, f.metadata?.name]))
    ).toEqual({ 'gb-3': 'gb-2', 'gb-2': 'gb-1', 'api-2': 'api-1' });
  });

  test("a Warehouse's oldest freight has none before it", () => {
    expect(getPreviousFreight([makeFreight({ name: 'only' })])).toEqual({});
  });
});

describe('getChangedArtifacts', () => {
  const repo = 'ghcr.io/akuity/guestbook';
  const config = 'https://github.com/akuity/config.git';

  test('a changed image is flagged and an unchanged commit is not', () => {
    const previous = makeFreight({
      images: [{ repoURL: repo, tag: 'v1', digest: 'sha256:1' }],
      commits: [{ repoURL: config, id: 'abc' }]
    });
    const freight = makeFreight({
      images: [{ repoURL: repo, tag: 'v2', digest: 'sha256:2' }],
      commits: [{ repoURL: config, id: 'abc' }]
    });

    const isChanged = getChangedArtifacts(freight, previous);

    expect(isChanged.has(freight.images![0])).toBe(true);
    expect(isChanged.has(freight.commits![0])).toBe(false);
  });

  test('an image pushed again under the same tag is changed', () => {
    const previous = makeFreight({
      images: [{ repoURL: repo, tag: 'latest', digest: 'sha256:1' }]
    });
    const freight = makeFreight({ images: [{ repoURL: repo, tag: 'latest', digest: 'sha256:2' }] });

    expect(getChangedArtifacts(freight, previous).has(freight.images![0])).toBe(true);
  });

  test('the same image under another tag is changed', () => {
    const previous = makeFreight({
      images: [{ repoURL: repo, tag: '1.2.0-rc.1', digest: 'sha256:1' }]
    });
    const freight = makeFreight({ images: [{ repoURL: repo, tag: '1.2.0', digest: 'sha256:1' }] });

    expect(getChangedArtifacts(freight, previous).has(freight.images![0])).toBe(true);
  });

  test('the same commit under another tag is changed', () => {
    const previous = makeFreight({ commits: [{ repoURL: config, id: 'abc', tag: 'v1.0.0-rc.1' }] });
    const freight = makeFreight({ commits: [{ repoURL: config, id: 'abc', tag: 'v1.0.0' }] });

    expect(getChangedArtifacts(freight, previous).has(freight.commits![0])).toBe(true);
  });

  test('an artifact the previous freight did not carry is changed', () => {
    const previous = makeFreight({ images: [{ repoURL: repo, tag: 'v1' }] });
    const freight = makeFreight({
      images: [{ repoURL: repo, tag: 'v1' }],
      charts: [{ repoURL: 'oci://ghcr.io/akuity/charts', name: 'guestbook', version: '1.0.0' }]
    });

    const isChanged = getChangedArtifacts(freight, previous);

    expect(isChanged.has(freight.charts![0])).toBe(true);
    expect(isChanged.has(freight.images![0])).toBe(false);
  });

  test('of two charts from one repository, only the one that differs is changed', () => {
    const charts = 'oci://ghcr.io/akuity/charts';
    const previous = makeFreight({
      charts: [
        { repoURL: charts, name: 'api', version: '1.0.0' },
        { repoURL: charts, name: 'web', version: '1.0.0' }
      ]
    });
    const freight = makeFreight({
      charts: [
        { repoURL: charts, name: 'api', version: '1.0.0' },
        { repoURL: charts, name: 'web', version: '1.1.0' }
      ]
    });

    const isChanged = getChangedArtifacts(freight, previous);

    expect(isChanged.has(freight.charts![0])).toBe(false);
    expect(isChanged.has(freight.charts![1])).toBe(true);
  });

  test('a different version of another kind of artifact is changed', () => {
    const previous = makeFreight({ artifacts: [{ subscriptionName: 'assets', version: 'v1' }] });
    const freight = makeFreight({ artifacts: [{ subscriptionName: 'assets', version: 'v2' }] });

    expect(getChangedArtifacts(freight, previous).has(freight.artifacts![0])).toBe(true);
  });

  test('an older version is changed too: it only has to differ', () => {
    const previous = makeFreight({ images: [{ repoURL: repo, tag: 'v2', digest: 'sha256:2' }] });
    const freight = makeFreight({ images: [{ repoURL: repo, tag: 'v1', digest: 'sha256:1' }] });

    expect(getChangedArtifacts(freight, previous).has(freight.images![0])).toBe(true);
  });

  test('with no previous freight, every artifact counts as changed', () => {
    const freight = makeFreight({
      images: [{ repoURL: repo, tag: 'v1' }],
      commits: [{ repoURL: config, id: 'abc' }]
    });

    expect(getChangedArtifacts(freight, undefined).size).toBe(2);
  });
});
