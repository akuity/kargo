import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import { ArtifactMetadata } from './artifact-metadata';

describe('ArtifactMetadata', () => {
  it('displays OCI source and revision links for Helm charts', () => {
    const html = renderToStaticMarkup(
      <ArtifactMetadata
        type='helm'
        repoURL='oci://registry.example.com/charts/app'
        name=''
        version='1.2.3'
        annotations={{
          'org.opencontainers.image.source': 'https://github.com/example/app',
          'org.opencontainers.image.revision': 'abc123'
        }}
      />
    );
    expect(html).toContain('href="https://github.com/example/app/commit/abc123"');
    expect(html).toContain('Annotations');
  });

  it('makes custom chart annotations available in the annotations panel', () => {
    const html = renderToStaticMarkup(
      <ArtifactMetadata
        type='helm'
        repoURL='oci://registry.example.com/charts/app'
        name=''
        version='1.2.3'
        annotations={{ 'example.com/release-notes': 'https://example.com/releases/1.2.3' }}
      />
    );
    expect(html).toContain('Annotations');
    expect(html).toContain('Only OCI Annotation');
  });

  it('handles charts without annotations', () => {
    expect(
      renderToStaticMarkup(
        <ArtifactMetadata
          type='helm'
          repoURL='https://charts.example.com'
          name='app'
          version='1.2.3'
        />
      )
    ).toBe('-');
  });
});
