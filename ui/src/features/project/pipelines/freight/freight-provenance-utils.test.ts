import { describe, expect, test } from 'vitest';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { artifactName, getFreightProvenanceRows } from './freight-provenance-utils';

const freight = (input: Partial<Freight>) =>
  ({
    alias: '',
    metadata: { name: 'fb95df6e1234567890' },
    commits: [],
    charts: [],
    images: [],
    artifacts: [],
    ...input
  }) as Freight;

describe('artifactName', () => {
  test('uses the last repo segment and trims .git', () => {
    expect(artifactName('https://github.com/fixico/deployment-hub.git')).toBe('deployment-hub');
    expect(
      artifactName('europe-docker.pkg.dev/prj-platform-mgmt-eu/ar-platform-workload-apps/fixicore')
    ).toBe('fixicore');
  });
});

describe('getFreightProvenanceRows', () => {
  test('formats alias, git commit, image tag, id, and age', () => {
    const rows = getFreightProvenanceRows(
      freight({
        alias: 'release-2026.06.12-588d48a-fb95df6e',
        commits: [
          {
            repoURL: 'https://github.com/fixico/deployment-hub.git',
            id: '0423f94abcdef'
          }
        ],
        images: [
          {
            repoURL:
              'europe-docker.pkg.dev/prj-platform-mgmt-eu/ar-platform-workload-apps/fixicore',
            tag: 'release-latest'
          }
        ]
      }),
      { showAlias: true, age: '18 minutes' }
    );

    expect(rows.map((row) => [row.label, row.value])).toEqual([
      ['alias', 'release-2026.06.12-588d48a-fb95df6e'],
      ['git', 'deployment-hub @ 0423f94'],
      ['image', 'fixicore:release-latest'],
      ['id', 'fb95df6'],
      ['age', '18 minutes']
    ]);
  });

  test('hides alias when requested', () => {
    const rows = getFreightProvenanceRows(freight({ alias: 'release-2026.06.12-588d48a' }), {
      showAlias: false
    });

    expect(rows.map((row) => row.label)).toEqual(['id']);
  });

  test('uses git tags before commit IDs', () => {
    const rows = getFreightProvenanceRows(
      freight({
        commits: [
          {
            repoURL: 'https://github.com/acme/app.git',
            id: 'abcdef123456',
            tag: 'v1.2.3'
          }
        ]
      })
    );

    expect(rows.find((row) => row.label === 'git')?.value).toBe('app @ v1.2.3');
  });
});
