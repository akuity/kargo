import { describe, expect, it } from 'vitest';

import { ChartDiscoveryResult } from '@ui/gen/api/v2/models';

import { constructFreight } from './construct-freight';

const annotations = {
  'org.opencontainers.image.revision': 'abc123',
  'example.com/release-notes': 'https://example.com/releases/1.2.3'
};

describe('constructFreight', () => {
  it.each([
    {
      name: 'selected OCI version',
      artifact: {
        repoURL: 'oci://registry.example.com/charts/app',
        versions: ['1.2.3', '1.2.2'],
        annotations: {
          '1.2.3': annotations,
          '1.2.2': { 'org.opencontainers.image.revision': 'old' }
        }
      },
      expected: annotations
    },
    {
      name: 'OCI chart without annotations',
      artifact: {
        repoURL: 'oci://registry.example.com/charts/app',
        versions: ['1.2.3']
      },
      expected: undefined
    },
    {
      name: 'classic HTTP chart',
      artifact: {
        repoURL: 'https://charts.example.com',
        name: 'app',
        versions: ['1.2.3']
      },
      expected: undefined
    }
  ])('preserves annotations for $name', ({ artifact, expected }) => {
    const freight = constructFreight(
      { chart: { artifact: artifact as ChartDiscoveryResult, info: '1.2.3' } },
      'charts'
    );
    expect(freight.charts).toEqual([
      {
        repoURL: artifact.repoURL,
        name: artifact.name,
        version: '1.2.3',
        annotations: expected
      }
    ]);
    expect(freight.origin).toEqual({ kind: 'Warehouse', name: 'charts' });
  });

  it('preserves other artifact types', () => {
    const freight = constructFreight(
      {
        image: {
          artifact: { repoURL: 'registry.example.com/app', references: [] },
          info: { tag: '1.2.3', digest: 'sha256:abc', annotations }
        },
        git: {
          artifact: { repoURL: 'https://github.com/example/app', commits: [] },
          info: { id: 'abc123', subject: 'release', branch: 'main' }
        },
        other: {
          artifact: { name: 'other', artifactReferences: [] },
          info: { subscriptionName: 'other', version: '1.2.3' }
        }
      },
      'mixed'
    );
    expect(freight.images?.[0]).toMatchObject({ tag: '1.2.3', annotations });
    expect(freight.commits?.[0]).toMatchObject({ id: 'abc123', message: 'release' });
    expect(freight.artifacts).toEqual([{ subscriptionName: 'other', version: '1.2.3' }]);
  });
});
