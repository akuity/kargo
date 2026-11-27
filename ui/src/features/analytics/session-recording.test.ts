import { describe, expect, test } from 'vitest';

import { maskAttribute, sessionRecordingOptions } from './session-recording';

// The tests run without a DOM, so elements are stood in for by the two
// things maskAttribute asks of one.
const element = (tagName: string, insideDefs = false) =>
  ({
    tagName: tagName.toUpperCase(),
    closest: (selector: string) => (insideDefs && selector === 'defs' ? {} : null)
  }) as unknown as Element;

describe('maskAttribute', () => {
  test.for([
    // What an element shows or is called.
    ['title', 'Stage prod', 'span'],
    ['alt', 'avatar of kilgore', 'img'],
    ['placeholder', 'ghcr.io/my-org', 'input'],
    ['value', 'my-project', 'input'],
    ['label', 'prod', 'option'],
    ['name', 'repoURL', 'input'],
    ['content', 'my-project', 'meta'],
    ['download', 'my-project.yaml', 'a'],
    // Where it points.
    ['href', '/project/my-project/stage/prod', 'a'],
    ['xlink:href', '/project/my-project', 'a'],
    // IDs, which component libraries derive from row keys and tab names.
    ['id', 'rc-tabs-0-tab-my-project', 'div'],
    ['for', 'form_my-project', 'label'],
    ['data-row-key', 'my-project/prod', 'tr'],
    ['data-id', 'stage/prod', 'div'],
    ['aria-label', 'open my-project', 'button'],
    ['aria-labelledby', 'rc-tabs-0-tab-my-project', 'div'],
    // Attribute names are matched whatever their case.
    ['TITLE', 'Stage prod', 'span'],
    ['Data-Row-Key', 'my-project/prod', 'tr']
  ])('masks %s', ([name, value, tagName]) => {
    expect(maskAttribute(name, value, element(tagName))).toBe('*');
  });

  test.for([
    ['class', 'ant-btn ant-btn-primary', 'button'],
    ['style', 'transform: translate(10px, 20px)', 'div'],
    ['d', 'M0 0L10 10', 'path'],
    ['marker-end', 'url(#arrow-1)', 'path'],
    ['src', 'kargo-logo.png', 'img'],
    ['type', 'password', 'input'],
    ['viewBox', '0 0 20 20', 'svg'],
    ['data-theme', 'dark', 'html']
  ])('keeps %s', ([name, value, tagName]) => {
    expect(maskAttribute(name, value, element(tagName))).toBe(value);
  });

  test('keeps the href of a stylesheet and of the base URL', () => {
    expect(maskAttribute('href', '/assets/index.css', element('link'))).toBe('/assets/index.css');
    expect(maskAttribute('href', '/kargo/', element('base'))).toBe('/kargo/');
  });

  test('keeps an href that refers to an element of the same document', () => {
    expect(maskAttribute('href', '#arrow-1', element('use'))).toBe('#arrow-1');
    expect(maskAttribute('xlink:href', '#arrow-1', element('use'))).toBe('#arrow-1');
  });

  test('keeps the ID of an SVG definition, which shapes refer to', () => {
    expect(maskAttribute('id', 'arrow-1', element('marker', true))).toBe('arrow-1');
  });

  test('masks without an element to judge by', () => {
    expect(maskAttribute('id', 'arrow-1')).toBe('*');
    expect(maskAttribute('href', '/assets/index.css')).toBe('*');
  });
});

describe('sessionRecordingOptions', () => {
  test('masks all text, all input and naming attributes, and records no payloads', () => {
    expect(sessionRecordingOptions).toEqual({
      maskTextSelector: '*',
      maskAllInputs: true,
      maskAttributeFn: maskAttribute,
      recordHeaders: false,
      recordBody: false
    });
  });
});
