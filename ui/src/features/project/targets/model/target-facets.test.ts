import { describe, expect, test } from 'vitest';

import { labelFacets, matchesFacets, noFacets, stateFacet, toggleFacet } from './target-facets';
import { targetRows } from './target-rows';
import { fleetStage, forStage, target } from './test-fixtures';

describe('facets', () => {
  const us = fleetStage('us-stage', [{}], 'abc');
  const fleet = () =>
    targetRows(
      [
        target('a', { geo: 'us', tier: 'primary' }, { 'us-stage': forStage('abc', 'Healthy') }),
        target('b', { geo: 'us', tier: 'secondary' }, { 'us-stage': forStage('abc', 'Unhealthy') }),
        target('c', { geo: 'eu', tier: 'primary' }, { 'us-stage': forStage('abc', 'Healthy') }),
        // Carries no tier at all.
        target('d', { geo: 'eu' }, { 'us-stage': forStage('abc', 'Healthy') })
      ],
      [us]
    );

  test('nothing is excluded to begin with', () => {
    expect(fleet().filter((row) => matchesFacets(row, noFacets()))).toHaveLength(4);
  });

  test('a switched-off label value takes its Targets out', () => {
    const facets = toggleFacet(noFacets(), 'geo', 'eu');
    expect(
      fleet()
        .filter((row) => matchesFacets(row, facets))
        .map((row) => row.target.name)
    ).toEqual(['a', 'b']);
  });

  test('a Target that does not carry the key is never excluded by it', () => {
    const facets = toggleFacet(toggleFacet(noFacets(), 'tier', 'primary'), 'tier', 'secondary');
    expect(
      fleet()
        .filter((row) => matchesFacets(row, facets))
        .map((row) => row.target.name)
    ).toEqual(['d']);
  });

  test('a switched-off state takes its Targets out', () => {
    const facets = toggleFacet(noFacets(), '', 'Unhealthy');
    expect(
      fleet()
        .filter((row) => matchesFacets(row, facets))
        .map((row) => row.target.name)
    ).toEqual(['a', 'c', 'd']);
  });

  test('toggling the same value twice puts it back', () => {
    const once = toggleFacet(noFacets(), 'geo', 'eu');
    expect(
      fleet().filter((row) => matchesFacets(row, toggleFacet(once, 'geo', 'eu')))
    ).toHaveLength(4);
  });

  test('values count commonest first, and keep their own key out of the tally', () => {
    const facets = toggleFacet(noFacets(), 'geo', 'eu');
    const [geo, tier] = labelFacets(fleet(), ['geo', 'tier'], facets);
    // geo ignores its own exclusion, so eu still says what it would bring back.
    expect(geo).toEqual({
      key: 'geo',
      values: [
        { value: 'eu', count: 2, excluded: true },
        { value: 'us', count: 2, excluded: false }
      ]
    });
    // tier is counted over what geo has left, so the eu Targets are gone.
    expect(tier.values).toEqual([
      { value: 'primary', count: 1, excluded: false },
      { value: 'secondary', count: 1, excluded: false }
    ]);
  });

  test('a switched-off value keeps its entry once another facet empties it', () => {
    // b is the only secondary Target and the only one geo=us removes, so
    // switching geo=us off leaves tier=secondary with nothing. It is switched
    // off too, so without an entry there would be no way to switch it back on.
    const facets = toggleFacet(toggleFacet(noFacets(), 'geo', 'us'), 'tier', 'secondary');
    const [, tier] = labelFacets(fleet(), ['geo', 'tier'], facets);
    expect(tier.values).toContainEqual({ value: 'secondary', count: 0, excluded: true });
  });

  test('a key no surviving Target carries drops out of the panel', () => {
    expect(labelFacets(fleet(), ['geo', 'absent'], noFacets()).map((f) => f.key)).toEqual(['geo']);
  });

  test('the state tally ignores the state exclusions but honours the label ones', () => {
    const facets = toggleFacet(toggleFacet(noFacets(), 'geo', 'eu'), '', 'Unhealthy');
    expect(stateFacet(fleet(), facets)).toEqual({
      Healthy: 1,
      Behind: 0,
      Progressing: 0,
      Unhealthy: 1,
      Unknown: 0,
      NotPromoted: 0
    });
  });
});
