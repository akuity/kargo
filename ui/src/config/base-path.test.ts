import { afterEach, describe, expect, test, vi } from 'vitest';

import { withBasePath, withoutBasePath } from './base-path';

describe('withoutBasePath', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  test.for([
    ['', '/project/foo', '/project/foo'],
    ['/kargo', '/kargo/project/foo', '/project/foo'],
    ['/kargo', '/kargo/', '/'],
    ['/kargo', '/kargo', '/'],
    ['/kargo', '/kargofoo/project', '/kargofoo/project'],
    ['/kargo', '/project/foo', '/project/foo']
  ])('with base %j turns %j into %j', ([base, path, expected]) => {
    vi.stubGlobal('window', { __KARGO_BASE_PATH__: base });
    expect(withoutBasePath(path)).toBe(expected);
  });

  test('round-trips with withBasePath', () => {
    vi.stubGlobal('window', { __KARGO_BASE_PATH__: '/kargo' });
    expect(withBasePath(withoutBasePath('/kargo/project/foo'))).toBe('/kargo/project/foo');
  });
});
