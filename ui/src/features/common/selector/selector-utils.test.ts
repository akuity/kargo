import { describe, expect, it } from 'vitest';

import { selectorLines } from './selector-utils';

describe('selectorLines', () => {
  it('is empty for an absent selector', () => {
    expect(selectorLines()).toEqual([]);
  });

  it('is empty for a selector with no constraints', () => {
    expect(selectorLines({})).toEqual([]);
  });

  it('renders a name, labels, and expressions together', () => {
    expect(
      selectorLines({
        name: 'glob:prod-*',
        matchLabels: { tier: 'critical' },
        matchExpressions: [{ key: 'env', operator: 'In', values: ['live', 'canary'] }]
      })
    ).toEqual(['glob:prod-*', 'tier=critical', 'env in (live, canary)']);
  });

  it('renders each label selector operator', () => {
    expect(
      selectorLines({
        matchExpressions: [
          { key: 'a', operator: 'In', values: ['1'] },
          { key: 'b', operator: 'NotIn', values: ['2', '3'] },
          { key: 'c', operator: 'Exists' },
          { key: 'd', operator: 'DoesNotExist' }
        ]
      })
    ).toEqual(['a in (1)', 'b notin (2, 3)', 'c exists', 'd does not exist']);
  });

  it('falls back to a readable form for an unrecognized operator', () => {
    expect(
      selectorLines({ matchExpressions: [{ key: 'a', operator: 'Gt', values: ['1'] }] })
    ).toEqual(['a Gt 1']);
  });
});
