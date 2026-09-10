import { describe, expect, test } from 'vitest';

import { DEFAULT_STAGE_TAB, StageTab, extensionTabKey, resolveStageTab } from './stage-tabs';

describe('resolveStageTab()', () => {
  test('no segment falls back to the default tab', () => {
    expect(resolveStageTab(undefined)).toBe(DEFAULT_STAGE_TAB);
    expect(resolveStageTab('')).toBe(DEFAULT_STAGE_TAB);
  });

  test.each(Object.values(StageTab))('built-in tab %s resolves to itself', (key) => {
    expect(resolveStageTab(key)).toBe(key);
  });

  test('an extension tab resolves when its key is known', () => {
    expect(resolveStageTab('argo-cd', ['argo-cd'])).toBe('argo-cd');
  });

  test('an unknown segment falls back to the default tab', () => {
    expect(resolveStageTab('nope')).toBe(DEFAULT_STAGE_TAB);
    expect(resolveStageTab('Promotions')).toBe(DEFAULT_STAGE_TAB);
  });
});

describe('extensionTabKey()', () => {
  test('slugifies the label', () => {
    expect(extensionTabKey('Argo CD', new Set())).toBe('argo-cd');
    expect(extensionTabKey('  Cost / Usage!  ', new Set())).toBe('cost-usage');
  });

  test('never collides with a built-in or earlier key', () => {
    const taken = new Set<string>([...Object.values(StageTab), 'argo-cd']);
    expect(extensionTabKey('Settings', taken)).toBe('settings-2');
    expect(extensionTabKey('Argo CD', taken)).toBe('argo-cd-2');
    taken.add('argo-cd-2');
    expect(extensionTabKey('Argo CD', taken)).toBe('argo-cd-3');
  });

  test('a label with no usable characters still yields a key', () => {
    expect(extensionTabKey('!!!', new Set())).toBe('tab');
  });
});
