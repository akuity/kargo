import { describe, expect, test } from 'vitest';

import { Freight, Image } from '@ui/gen/api/v2/models';

import { pairArtifacts } from './pair-artifacts';

const f = (input: {
  images?: Image[];
  commits?: { repoURL: string; id: string; branch?: string; message?: string }[];
  charts?: { repoURL: string; name?: string; version: string }[];
}) =>
  ({
    images: input.images || [],
    commits: input.commits || [],
    charts: input.charts || [],
    artifacts: []
  }) as unknown as Freight;

describe('pairArtifacts', () => {
  test.each([
    { name: 'adding a name', currentName: undefined, incomingName: 'api' },
    { name: 'renaming', currentName: 'old-api', incomingName: 'api' },
    { name: 'removing a name', currentName: 'api', incomingName: undefined }
  ])('$name preserves image continuity', ({ currentName, incomingName }) => {
    const image = { repoURL: 'ghcr.io/example/api', tag: 'v1', digest: 'sha256:abc' };
    const rows = pairArtifacts(
      f({ images: [{ ...image, subscriptionName: currentName }] }),
      f({ images: [{ ...image, subscriptionName: incomingName }] })
    );

    expect(rows).toHaveLength(1);
    expect(rows[0].status).toBe('UNCHANGED');
  });

  test.each([
    { tag: 'v2', digest: 'sha256:abc' },
    { tag: 'v1', digest: 'sha256:def' },
    { tag: undefined, digest: 'sha256:def' }
  ])('detects image version changes: $tag $digest', (incomingVersion) => {
    const repoURL = 'ghcr.io/example/api';
    const rows = pairArtifacts(
      f({ images: [{ repoURL, tag: 'v1', digest: 'sha256:abc' }] }),
      f({ images: [{ repoURL, ...incomingVersion }] })
    );

    expect(rows).toHaveLength(1);
    expect(rows[0].status).toBe('CHANGED');
  });

  test('multi-artifact: image changed, git changed, chart unchanged', () => {
    const current = f({
      images: [{ repoURL: 'ghcr.io/acme/guestbook', tag: 'v0.0.87' }],
      commits: [{ repoURL: 'github.com/acme/kargo-advanced.git', id: 'a1f3e2c0' }],
      charts: [{ repoURL: 'charts.example.com/guestbook', version: '2.4.1' }]
    });
    const incoming = f({
      images: [{ repoURL: 'ghcr.io/acme/guestbook', tag: 'v0.0.88' }],
      commits: [{ repoURL: 'github.com/acme/kargo-advanced.git', id: '206b2e87' }],
      charts: [{ repoURL: 'charts.example.com/guestbook', version: '2.4.1' }]
    });

    const rows = pairArtifacts(current, incoming);
    expect(rows.map((r) => [r.key, r.status])).toEqual([
      ['image:ghcr.io/acme/guestbook', 'CHANGED'],
      ['git:github.com/acme/kargo-advanced.git', 'CHANGED'],
      ['helm:charts.example.com/guestbook/', 'UNCHANGED']
    ]);
  });

  test('single-artifact incoming, identical current → UNCHANGED', () => {
    const freight = f({
      images: [{ repoURL: 'ghcr.io/x/y', tag: 'v1.0.0' }]
    });

    const rows = pairArtifacts(freight, freight);
    expect(rows).toHaveLength(1);
    expect(rows[0].status).toBe('UNCHANGED');
  });

  test('no current freight → every incoming row is NEW', () => {
    const incoming = f({
      images: [{ repoURL: 'ghcr.io/x/y', tag: 'v1.0.0' }],
      commits: [{ repoURL: 'github.com/x/y.git', id: 'abc' }]
    });

    const rows = pairArtifacts(undefined, incoming);
    expect(rows.map((r) => r.status)).toEqual(['NEW', 'NEW']);
  });

  test('artifact only in incoming → NEW; artifact only in current → REMOVED', () => {
    const current = f({
      images: [{ repoURL: 'ghcr.io/keep', tag: 'v1' }],
      charts: [{ repoURL: 'charts.example.com/old', version: '1.0.0' }]
    });
    const incoming = f({
      images: [{ repoURL: 'ghcr.io/keep', tag: 'v1' }],
      commits: [{ repoURL: 'github.com/new.git', id: 'sha1' }]
    });

    const rows = pairArtifacts(current, incoming);
    expect(rows.map((r) => [r.key, r.status])).toEqual([
      ['image:ghcr.io/keep', 'UNCHANGED'],
      ['git:github.com/new.git', 'NEW'],
      ['helm:charts.example.com/old/', 'REMOVED']
    ]);
  });

  test('classic helm repo: charts with same repoURL but different names pair independently', () => {
    const current = f({
      charts: [
        { repoURL: 'charts.example.com', name: 'guestbook', version: '1.0.0' },
        { repoURL: 'charts.example.com', name: 'frontend', version: '2.0.0' }
      ]
    });
    const incoming = f({
      charts: [
        { repoURL: 'charts.example.com', name: 'guestbook', version: '1.1.0' },
        { repoURL: 'charts.example.com', name: 'frontend', version: '2.0.0' }
      ]
    });

    const rows = pairArtifacts(current, incoming);
    expect(rows.map((r) => [r.key, r.status])).toEqual([
      ['helm:charts.example.com/guestbook', 'CHANGED'],
      ['helm:charts.example.com/frontend', 'UNCHANGED']
    ]);
  });

  test('promoting freight already on stage → all rows UNCHANGED', () => {
    const freight = f({
      images: [{ repoURL: 'ghcr.io/x/y', tag: 'v1' }],
      commits: [{ repoURL: 'github.com/x/y.git', id: 'abc' }],
      charts: [{ repoURL: 'charts.example.com/c', version: '1.0' }]
    });

    const rows = pairArtifacts(freight, freight);
    expect(rows.every((r) => r.status === 'UNCHANGED')).toBe(true);
  });
});
