import { describe, expect, test } from 'vitest';

import { Freight, Image, ImageAnnotationMappings } from '@ui/gen/api/v2/models';

import { pairArtifacts } from '../project/pipelines/promotion/pair-artifacts';

import { getImageReleaseContext, pairImageReleaseContexts } from './release-context-utils';

describe('getImageReleaseContext', () => {
  const mappings: ImageAnnotationMappings = {
    commitSubject: 'com.example.image.commit.subject',
    commitAuthor: 'com.example.image.commit.author',
    commitCommitter: 'com.example.image.commit.committer',
    commitCreatedAt: 'com.example.image.commit.created'
  };
  const image: Image = {
    repoURL: 'ghcr.io/example/api',
    tag: 'v1.5.0',
    annotations: {
      'org.opencontainers.image.source': 'https://github.com/example/api',
      'org.opencontainers.image.revision': 'abc123',
      'org.opencontainers.image.created': '2026-08-10T16:38:17Z',
      'org.opencontainers.image.description': 'Example API',
      'com.example.image.commit.subject': 'Improve request handling',
      'com.example.image.commit.author': 'Ada',
      'com.example.image.commit.committer': 'Grace',
      'com.example.image.commit.created': '2026-08-10T16:35:01Z'
    }
  };

  test('interprets exact configured keys without changing OCI meanings or raw annotations', () => {
    const context = getImageReleaseContext(image, mappings);

    expect(context).toMatchObject({
      image,
      source: 'https://github.com/example/api',
      sourceURL: 'https://github.com/example/api',
      commitURL: 'https://github.com/example/api/commit/abc123',
      revision: 'abc123',
      createdAt: '2026-08-10T16:38:17Z',
      subject: 'Improve request handling',
      author: 'Ada',
      committer: 'Grace',
      commitCreatedAt: '2026-08-10T16:35:01Z'
    });
    expect(Object.fromEntries(context.annotations.map(({ key, value }) => [key, value]))).toEqual(
      image.annotations
    );
    expect(context.annotations.map(({ key }) => key)).toEqual(
      Object.keys(image.annotations || {}).sort((a, b) => a.localeCompare(b))
    );
  });

  test.each([undefined, {}])('does not infer custom semantics without mappings: %j', (mapping) => {
    const context = getImageReleaseContext(image, mapping);

    expect(context.subject).toBeUndefined();
    expect(context.author).toBeUndefined();
    expect(context.committer).toBeUndefined();
    expect(context.commitCreatedAt).toBeUndefined();
    expect(context.revision).toBe('abc123');
    expect(context.annotations).toHaveLength(8);
  });

  test('supports organization-owned keys without requiring a suffix convention', () => {
    const context = getImageReleaseContext(
      {
        annotations: {
          'dev.example.summary': 'Ship it',
          'com.example.image.commit.subject': 'Do not infer this',
          'dev.example.author': 'Ada'
        }
      },
      { commitSubject: 'dev.example.summary' }
    );

    expect(context.subject).toBe('Ship it');
    expect(context.author).toBeUndefined();
  });

  test('does not reinterpret reserved OCI keys even with an invalid mapping', () => {
    const context = getImageReleaseContext(image, {
      commitSubject: 'org.opencontainers.image.description',
      commitCreatedAt: 'org.opencontainers.image.created'
    });

    expect(context.subject).toBeUndefined();
    expect(context.commitCreatedAt).toBeUndefined();
    expect(context.createdAt).toBe('2026-08-10T16:38:17Z');
  });

  test('omits missing and empty values without reading inherited object properties', () => {
    const context = getImageReleaseContext(
      { annotations: { 'com.example.subject': '' } },
      { commitSubject: 'com.example.subject', commitAuthor: 'toString', commitCommitter: 'missing' }
    );

    expect(context.subject).toBeUndefined();
    expect(context.author).toBeUndefined();
    expect(context.committer).toBeUndefined();
    expect(getImageReleaseContext({}).annotations).toEqual([]);
  });

  test.each([
    'javascript:alert(1)',
    'data:text/html,example',
    'vscode://example.com/repo',
    '//example.com/repo',
    'not a URL'
  ])('retains unsupported source %s as text, not as a clickable link', (source) => {
    const context = getImageReleaseContext({
      annotations: { 'org.opencontainers.image.source': source }
    });

    expect(context.source).toBe(source);
    expect(context.sourceURL).toBeUndefined();
    expect(context.commitURL).toBeUndefined();
  });

  test('allows an HTTPS commit link derived from an SSH repository without linking SSH', () => {
    const context = getImageReleaseContext({
      annotations: {
        'org.opencontainers.image.source': 'git@github.com:example/api.git',
        'org.opencontainers.image.revision': 'abc123'
      }
    });

    expect(context.sourceURL).toBeUndefined();
    expect(context.commitURL).toBe('https://github.com/example/api/commit/abc123');
  });
});

describe('pairImageReleaseContexts', () => {
  test('uses the same comparison contract as the Freight tab', () => {
    const current: Freight = {
      origin: { kind: 'Warehouse', name: 'images' },
      images: [
        { repoURL: 'ghcr.io/example/api', tag: 'v1', digest: 'sha256:abc' },
        { repoURL: 'ghcr.io/example/worker', tag: 'v1' }
      ],
      commits: [{ repoURL: 'https://github.com/example/config', id: 'abc' }]
    };
    const incoming: Freight = {
      origin: { kind: 'Warehouse', name: 'images' },
      images: [
        {
          subscriptionName: 'api',
          repoURL: 'ghcr.io/example/api',
          tag: 'v1',
          digest: 'sha256:def',
          annotations: { 'com.example.subject': 'Ship it' }
        },
        { repoURL: 'ghcr.io/example/web', tag: 'v1' }
      ]
    };
    const pairs = pairImageReleaseContexts(current, incoming, {
      commitSubject: 'com.example.subject'
    });

    expect(pairs.map(({ key, status }) => ({ key, status }))).toEqual(
      pairArtifacts(current, incoming)
        .filter((pair) => pair.current?.type === 'image' || pair.incoming?.type === 'image')
        .map(({ key, status }) => ({ key, status }))
    );
    expect(pairs.map(({ status }) => status)).toEqual(['CHANGED', 'NEW', 'REMOVED']);
    expect(pairs[0].current?.image.digest).toBe('sha256:abc');
    expect(pairs[0].incoming?.subject).toBe('Ship it');
    expect(pairs[0].incoming?.image.subscriptionName).toBe('api');
  });

  test('handles first promotions, unchanged images, and Freight without images', () => {
    const freight: Freight = {
      origin: { kind: 'Warehouse', name: 'images' },
      images: [{ repoURL: 'ghcr.io/example/api', tag: 'v1' }]
    };

    expect(pairImageReleaseContexts(undefined, freight)[0].status).toBe('NEW');
    expect(pairImageReleaseContexts(freight, freight)[0].status).toBe('UNCHANGED');
    expect(pairImageReleaseContexts(undefined, { ...freight, images: [] })).toEqual([]);
  });
});
