import { describe, expect, it } from 'vitest';

import { flattenFreightOrigin } from './flatten-freight-origin-utils';

describe('flattenFreightOrigin', () => {
  it('preserves OCI chart annotations for artifact metadata', () => {
    const annotations = {
      'org.opencontainers.image.source': 'https://github.com/example/app',
      'org.opencontainers.image.revision': 'abc123',
      'example.com/release-notes': 'https://example.com/releases/1.2.3'
    };
    expect(
      flattenFreightOrigin({
        charts: [
          {
            repoURL: 'oci://registry.example.com/charts/app',
            version: '1.2.3',
            annotations,
            subscriptionName: 'app'
          }
        ]
      })
    ).toEqual([
      {
        type: 'helm',
        repoURL: 'oci://registry.example.com/charts/app',
        name: '',
        version: '1.2.3',
        annotations,
        subscriptionName: 'app'
      }
    ]);
  });

  it('defaults missing chart annotations to an empty map', () => {
    expect(flattenFreightOrigin({ charts: [{ version: '1.2.3' }] })[0]).toMatchObject({
      type: 'helm',
      annotations: {}
    });
  });
});
