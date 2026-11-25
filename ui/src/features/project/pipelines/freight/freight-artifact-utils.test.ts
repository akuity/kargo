import { describe, expect, test } from 'vitest';

import { Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { getArtifactDisplay } from './freight-artifact-utils';

const commit = (c: Partial<GitCommit>): GitCommit => ({
  repoURL: 'https://github.com/akuity/kargo.git',
  id: '0123456789abcdef',
  ...c
});

const image = (i: Partial<Image>): Image => ({
  repoURL: 'ghcr.io/akuity/guestbook',
  tag: 'v1.0.0',
  digest: 'sha256:abc',
  ...i
});

const chart: Chart = { repoURL: 'oci://ghcr.io/akuity/charts/guestbook', version: '1.2.3' };

describe('getArtifactDisplay', () => {
  test('a commit shows its tag and links to its Git host', () => {
    expect(getArtifactDisplay(commit({ tag: 'v2.0.0' }))).toEqual({
      value: 'v2.0.0',
      title: 'https://github.com/akuity/kargo.git',
      url: 'https://github.com/akuity/kargo/commit/0123456789abcdef'
    });
  });

  test('a commit without a tag shows its short SHA', () => {
    expect(getArtifactDisplay(commit({})).value).toBe('0123456');
  });

  test('a long tag is shortened in the middle', () => {
    expect(getArtifactDisplay(commit({ tag: 'release-2026-11-25-0123' })).value).toBe(
      'releas...5-0123'
    );
  });

  test('a chart shows its version and links nowhere', () => {
    expect(getArtifactDisplay(chart)).toEqual({
      value: '1.2.3',
      title: 'oci://ghcr.io/akuity/charts/guestbook:1.2.3',
      url: ''
    });
  });

  test('an image links to the source its annotations name', () => {
    const annotated = image({
      annotations: {
        'org.opencontainers.image.source': 'https://github.com/akuity/guestbook',
        'org.opencontainers.image.revision': 'abc'
      }
    });

    expect(getArtifactDisplay(annotated)).toEqual({
      value: 'v1.0.0',
      title: 'ghcr.io/akuity/guestbook:v1.0.0',
      url: 'https://github.com/akuity/guestbook/commit/abc'
    });
  });

  test('an image without a source links nowhere', () => {
    expect(getArtifactDisplay(image({})).url).toBe('');
  });
});
