import { describe, expect, test } from 'vitest';

import { parseArgoCDContext } from './argocd-link-utils';

describe('parseArgoCDContext', () => {
  test('returns an empty list when the annotation is absent', () => {
    expect(parseArgoCDContext(undefined)).toEqual([]);
    expect(parseArgoCDContext('')).toEqual([]);
  });

  test('removes duplicate apps, preserving first-seen order', () => {
    const raw = JSON.stringify([
      { name: 'example-app', namespace: 'argocd' },
      { name: 'other-app', namespace: 'argocd' },
      { name: 'example-app', namespace: 'argocd' }
    ]);

    expect(parseArgoCDContext(raw)).toEqual([
      { name: 'example-app', namespace: 'argocd' },
      { name: 'other-app', namespace: 'argocd' }
    ]);
  });

  test('keeps apps with the same name in different namespaces', () => {
    const raw = JSON.stringify([
      { name: 'example-app', namespace: 'argocd' },
      { name: 'example-app', namespace: 'team-a' }
    ]);

    expect(parseArgoCDContext(raw)).toHaveLength(2);
  });

  test('throws on malformed annotation', () => {
    expect(() => parseArgoCDContext('not-json')).toThrow();
    expect(() => parseArgoCDContext(JSON.stringify([{ name: 'x' }]))).toThrow();
  });
});
