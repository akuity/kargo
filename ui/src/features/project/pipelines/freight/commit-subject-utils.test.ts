import { describe, expect, test } from 'vitest';

import { commitSubject } from './commit-subject-utils';

describe('commitSubject', () => {
  test('returns the first line of the message', () => {
    expect(commitSubject('fix: bump tenant DB (#694)\n\nlonger body')).toBe(
      'fix: bump tenant DB (#694)'
    );
  });

  test('returns empty string for missing message', () => {
    expect(commitSubject(undefined)).toBe('');
  });

  test('leaves subjects within the limit untouched', () => {
    expect(commitSubject('a'.repeat(20), 20)).toBe('a'.repeat(20));
  });

  test('truncates long subjects with ellipsis', () => {
    expect(commitSubject('fix: something very long here', 20)).toBe('fix: something ve...');
  });

  test('keeps a trailing GitHub PR reference', () => {
    expect(commitSubject('fix(akp/test): bump tenant DB to db.t4g.large (#694)', 30)).toBe(
      'fix(akp/test): bump... (#694)'
    );
  });

  test('keeps a trailing GitLab MR reference', () => {
    expect(commitSubject('feat: add a very long gitlab change (!42)', 25)).toBe(
      'feat: add a very... (!42)'
    );
  });

  test('result never exceeds the limit', () => {
    const msg = 'x'.repeat(500) + ' (#12345)';
    expect(commitSubject(msg).length).toBeLessThanOrEqual(100);
  });
});
