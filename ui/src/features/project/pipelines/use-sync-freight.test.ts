import { describe, expect, it } from 'vitest';

import { Freight, Stage } from '@ui/gen/api/v2/models';

import { missingFreightNames } from './use-sync-freight';

const freights = (...names: string[]): Record<string, Freight> =>
  Object.fromEntries(names.map((name) => [name, { metadata: { name } } as Freight]));

const freightInStages = (...names: string[]): Record<string, Stage[]> =>
  Object.fromEntries(names.map((name) => [name, [] as Stage[]]));

describe('missingFreightNames', () => {
  it('returns nothing when the list holds everything the Stages do', () => {
    expect(missingFreightNames(freights('a', 'b'), freightInStages('a', 'b'))).toEqual([]);
  });

  it('returns nothing when no Stage holds Freight', () => {
    expect(missingFreightNames(freights('a'), {})).toEqual([]);
  });

  it('returns Freight a Stage holds that the list does not contain', () => {
    expect(missingFreightNames(freights('a'), freightInStages('a', 'b'))).toEqual(['b']);
  });

  it('ignores list Freight that no Stage holds', () => {
    expect(missingFreightNames(freights('a', 'b', 'c'), freightInStages('b'))).toEqual([]);
  });

  it('reports Freight from a Warehouse the origin-filtered list excludes', () => {
    expect(missingFreightNames(freights('from-a'), freightInStages('from-a', 'from-b'))).toEqual([
      'from-b'
    ]);
  });

  it('sorts, keeping the effect dependency stable across equal inputs', () => {
    const a = missingFreightNames({}, freightInStages('c', 'a', 'b')).join(',');
    const b = missingFreightNames({}, freightInStages('b', 'c', 'a')).join(',');

    expect(Object.is(a, b)).toBe(true);
  });
});
