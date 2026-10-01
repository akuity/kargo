import { describe, expect, test } from 'vitest';

import {
  deepLinkFormSchema,
  DeepLinkFormValues,
  deepLinkFromFormValues,
  formValuesFromDeepLink
} from './deep-link-form-utils';
import { deepLinkKinds } from './deep-link-kinds';

const values = (overrides: Partial<DeepLinkFormValues> = {}): DeepLinkFormValues => ({
  title: 'Runbook',
  url: 'https://example.com/{{ .stage.metadata.name }}',
  description: '',
  if: '',
  ...overrides
});

const issuePaths = (candidate: DeepLinkFormValues) =>
  deepLinkFormSchema.safeParse(candidate).error?.issues.map((issue) => issue.path) ?? [];

describe('deepLinkFormSchema', () => {
  test('accepts a title and URL alone', () => {
    expect(deepLinkFormSchema.safeParse(values()).success).toBe(true);
  });

  test('requires a title and a URL', () => {
    expect(issuePaths(values({ title: '', url: '' }))).toEqual([['title'], ['url']]);
  });

  test('rejects a URL whose template delimiters do not pair up', () => {
    // the server drops a link it cannot parse, so the typo is invisible later
    expect(issuePaths(values({ url: 'https://example.com/{{ .stage.metadata.name' }))).toEqual([
      ['url']
    ]);
  });

  test('accepts a URL with several template expressions', () => {
    const url = 'https://example.com/{{ .stage.metadata.namespace }}/{{ .stage.metadata.name }}';

    expect(deepLinkFormSchema.safeParse(values({ url })).success).toBe(true);
  });

  test('rejects a condition written as a template', () => {
    expect(issuePaths(values({ if: '{{ .stage.metadata.name }}' }))).toEqual([['if']]);
  });

  test('accepts a condition written as an expression', () => {
    expect(
      deepLinkFormSchema.safeParse(values({ if: 'stage.spec.verification != nil' })).success
    ).toBe(true);
  });
});

describe('formValuesFromDeepLink', () => {
  test('starts blank for a new link', () => {
    expect(formValuesFromDeepLink()).toEqual({ title: '', url: '', description: '', if: '' });
  });

  test('fills every field from an existing link', () => {
    expect(
      formValuesFromDeepLink({
        title: 'Runbook',
        url: 'https://example.com',
        description: 'How to operate this',
        if: 'stage.metadata.name != ""'
      })
    ).toEqual({
      title: 'Runbook',
      url: 'https://example.com',
      description: 'How to operate this',
      if: 'stage.metadata.name != ""'
    });
  });
});

describe('deepLinkFromFormValues', () => {
  test('omits the optional fields when they are blank', () => {
    expect(deepLinkFromFormValues(values())).toEqual({
      title: 'Runbook',
      url: 'https://example.com/{{ .stage.metadata.name }}'
    });
  });

  test('trims every field and keeps the optional ones that are set', () => {
    expect(
      deepLinkFromFormValues(
        values({
          title: '  Runbook  ',
          url: '  https://example.com  ',
          description: '  How to operate this  ',
          if: '  stage.metadata.name != ""  '
        })
      )
    ).toEqual({
      title: 'Runbook',
      url: 'https://example.com',
      description: 'How to operate this',
      if: 'stage.metadata.name != ""'
    });
  });

  test('drops optional fields that hold only whitespace', () => {
    expect(deepLinkFromFormValues(values({ description: '   ', if: '  ' }))).toEqual({
      title: 'Runbook',
      url: 'https://example.com/{{ .stage.metadata.name }}'
    });
  });
});

describe('the examples the form offers', () => {
  const kinds = Object.entries(deepLinkKinds);

  test.each(kinds)('%s URL examples satisfy the form it suggests them in', (_, kind) => {
    for (const url of kind.urlExamples) {
      expect(deepLinkFormSchema.safeParse(values({ url })).success).toBe(true);
    }
  });

  test.each(kinds)('%s condition examples satisfy the form', (_, kind) => {
    for (const condition of kind.conditionExamples) {
      expect(deepLinkFormSchema.safeParse(values({ if: condition })).success).toBe(true);
    }
  });

  // the context is the resource as JSON, so an unset `omitempty` field is
  // absent rather than null. `index`, `[...]` and `len()` raise on an absent
  // field, and the server drops the whole link when they do, which is
  // invisible in the UI. An example has to hold for a resource with none of
  // its optional fields set, so it reaches them with `get` or `?.` instead.
  test.each(kinds)('%s examples avoid constructs that raise on an unset field', (_, kind) => {
    for (const example of [...kind.urlExamples, ...kind.conditionExamples]) {
      expect(example).not.toMatch(/\bindex\b|\blen\(|\["/);
    }
  });
});
