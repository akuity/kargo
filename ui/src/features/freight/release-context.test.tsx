import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, test, vi } from 'vitest';

import { Freight } from '@ui/gen/api/v2/models';

import { ReleaseContext } from './release-context';

const { getConfig } = vi.hoisted(() => ({ getConfig: vi.fn() }));

vi.mock('@ui/gen/api/v2/core/core', () => ({
  useGetFreightReleaseContextConfig: getConfig
}));

describe('ReleaseContext', () => {
  const freight: Freight = {
    metadata: { namespace: 'example', name: 'release' },
    origin: { kind: 'Warehouse', name: 'images' },
    images: [
      {
        repoURL: 'ghcr.io/example/api',
        tag: 'v1',
        digest: 'sha256:abc',
        annotations: {
          'com.example.subject': 'Improve requests',
          'com.example.author': '<script>alert(1)</script>',
          'org.opencontainers.image.revision': 'abc123',
          'org.opencontainers.image.source': 'https://github.com/example/api'
        }
      }
    ]
  };

  beforeEach(() => {
    getConfig.mockReset();
    getConfig.mockReturnValue({
      data: { data: { imageAnnotations: { commitSubject: 'com.example.subject' } } },
      isLoading: false,
      isError: false,
      refetch: vi.fn()
    });
  });

  test('uses the authorized Freight identity and configured mapping in the detail view', () => {
    const html = renderToStaticMarkup(<ReleaseContext freight={freight} />);

    expect(getConfig).toHaveBeenCalledWith('example', 'release', {
      query: { enabled: true, refetchOnWindowFocus: true }
    });
    expect(html).toContain('Container images in this Freight');
    expect(html).toContain('Improve requests');
    expect(html).toContain('href="https://github.com/example/api/commit/abc123"');
    expect(html).toContain('Image annotations');
  });

  test('compares images across a subscription-name transition using both tag and digest', () => {
    const incoming: Freight = {
      ...freight,
      images: [{ ...freight.images?.[0], subscriptionName: 'api', digest: 'sha256:def' }]
    };
    const html = renderToStaticMarkup(
      <ReleaseContext freight={incoming} currentFreight={freight} comparison />
    );

    expect(html).toContain('CHANGED');
    expect(html).not.toContain('REMOVED');
    expect(html).toContain('sha256:abc');
    expect(html).toContain('sha256:def');
    expect(html).toContain('Current');
    expect(html).toContain('Incoming');
  });

  test('shows a loading indicator without hiding standard OCI context', () => {
    getConfig.mockReturnValue({ isLoading: true, isError: false });
    const html = renderToStaticMarkup(<ReleaseContext freight={freight} />);

    expect(html).toContain('ant-skeleton');
    expect(html).toContain('abc123');
    expect(html).not.toContain('Improve requests');
  });

  test('does not use stale custom mappings after a configuration read fails', () => {
    getConfig.mockReturnValue({
      data: { data: { imageAnnotations: { commitSubject: 'com.example.subject' } } },
      isLoading: false,
      isError: true,
      refetch: vi.fn()
    });
    const html = renderToStaticMarkup(<ReleaseContext freight={freight} />);

    expect(html).toContain('Custom annotation configuration could not be loaded');
    expect(html).toContain('Retry');
    expect(html).toContain('abc123');
    expect(html).toContain('Image annotations');
    expect(html).not.toContain('Improve requests');
  });

  test('keeps unsupported source URLs as text and escapes custom annotation content', () => {
    getConfig.mockReturnValue({
      data: { data: { imageAnnotations: { commitAuthor: 'com.example.author' } } }
    });
    const html = renderToStaticMarkup(
      <ReleaseContext
        freight={{
          ...freight,
          images: [
            {
              ...freight.images?.[0],
              annotations: {
                ...freight.images?.[0].annotations,
                'org.opencontainers.image.source': 'vscode://example.com/api'
              }
            }
          ]
        }}
      />
    );

    expect(html).toContain('vscode://example.com/api');
    expect(html).not.toContain('href="vscode:');
    expect(html).toContain('&lt;script&gt;alert(1)&lt;/script&gt;');
    expect(html).not.toContain('<script>');
  });

  test('identifies image-free Freight without implying Git or chart metadata is missing', () => {
    const html = renderToStaticMarkup(
      <ReleaseContext
        freight={{
          ...freight,
          images: [],
          commits: [{ repoURL: 'https://github.com/example/config', id: 'abc' }]
        }}
      />
    );

    expect(html).toContain('This Freight contains no container images.');
  });

  test.each([
    '2026-02-30T16:35:01Z',
    '01/02/2026',
    'not a timestamp',
    '2026-08-10T16:35:01+99:00',
    '2026-08-10T16:35:01Zgarbage',
    '2026-08-10T16:35:01'
  ])('preserves invalid timestamp %s as raw text', (timestamp) => {
    getConfig.mockReturnValue({
      data: { data: { imageAnnotations: { commitCreatedAt: 'com.example.commit.created' } } }
    });
    const html = renderToStaticMarkup(
      <ReleaseContext
        freight={{
          ...freight,
          images: [
            {
              ...freight.images?.[0],
              annotations: {
                'org.opencontainers.image.created': timestamp,
                'com.example.commit.created': timestamp
              }
            }
          ]
        }}
      />
    );

    expect(html.split(timestamp)).toHaveLength(3);
  });

  test.each(['2026-08-10T16:35:01Z', '2026-08-10T16:35:01.123+02:00'])(
    'formats valid timestamp %s',
    (timestamp) => {
      const html = renderToStaticMarkup(
        <ReleaseContext
          freight={{
            ...freight,
            images: [
              {
                ...freight.images?.[0],
                annotations: { 'org.opencontainers.image.created': timestamp }
              }
            ]
          }}
        />
      );

      expect(html).toContain('Built');
      expect(html).not.toContain(timestamp);
    }
  );
});
