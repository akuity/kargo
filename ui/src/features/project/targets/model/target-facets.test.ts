import { describe, expect, test } from 'vitest';

import {
  Facets,
  STATE_FACET,
  facetSelections,
  labelFacets,
  matchesFacets,
  noFacets,
  stateFacet,
  toggleFacet
} from './target-facets';
import { targetRows } from './target-rows';
import { fleetStage, forStage, target } from './test-fixtures';

describe('facets', () => {
  const us = fleetStage('us-stage', [{}], 'abc');
  const fleet = () =>
    targetRows(
      [
        target(
          'a',
          { geo: 'us', tier: 'primary' },
          {
            status: { stages: { 'us-stage': forStage('abc', 'Healthy') } }
          }
        ),
        target(
          'b',
          { geo: 'us', tier: 'secondary' },
          {
            status: { stages: { 'us-stage': forStage('abc', 'Unhealthy') } }
          }
        ),
        target(
          'c',
          { geo: 'eu', tier: 'primary' },
          {
            status: { stages: { 'us-stage': forStage('abc', 'Healthy') } }
          }
        ),
        // Carries no tier at all.
        target(
          'd',
          { geo: 'eu' },
          {
            status: { stages: { 'us-stage': forStage('abc', 'Healthy') } }
          }
        )
      ],
      [us]
    );

  const names = (facets: Facets) =>
    fleet()
      .filter((row) => matchesFacets(row, facets))
      .map((row) => row.target.metadata?.name);

  test('nothing picked filters nothing', () => {
    expect(names(noFacets())).toEqual(['a', 'b', 'c', 'd']);
  });

  test('a picked label value narrows the fleet to it', () => {
    expect(names(toggleFacet(noFacets(), 'geo', 'eu'))).toEqual(['c', 'd']);
  });

  test('picking more values of one key widens it to any of them', () => {
    const facets = toggleFacet(toggleFacet(noFacets(), 'geo', 'eu'), 'geo', 'us');
    expect(names(facets)).toEqual(['a', 'b', 'c', 'd']);
  });

  test('picks on different keys must all hold', () => {
    const facets = toggleFacet(toggleFacet(noFacets(), 'geo', 'eu'), 'tier', 'primary');
    expect(names(facets)).toEqual(['c']);
  });

  test('a Target that does not carry a picked key is left out', () => {
    const facets = toggleFacet(toggleFacet(noFacets(), 'tier', 'primary'), 'tier', 'secondary');
    expect(names(facets)).toEqual(['a', 'b', 'c']);
  });

  test('a picked state narrows the fleet to it', () => {
    expect(names(toggleFacet(noFacets(), STATE_FACET, 'Unhealthy'))).toEqual(['b']);
  });

  test('toggling the same value twice unpicks it', () => {
    const once = toggleFacet(noFacets(), 'geo', 'eu');
    expect(names(toggleFacet(once, 'geo', 'eu'))).toHaveLength(4);
  });

  test('facetSelections counts every pick', () => {
    const facets = toggleFacet(toggleFacet(noFacets(), 'geo', 'eu'), STATE_FACET, 'Healthy');
    expect(facetSelections(noFacets())).toBe(0);
    expect(facetSelections(facets)).toBe(2);
  });

  test('values count commonest first, and keep their own key out of the tally', () => {
    const facets = toggleFacet(noFacets(), 'geo', 'eu');
    const [geo, tier] = labelFacets(fleet(), ['geo', 'tier'], facets);
    // geo ignores its own pick, so us still says what picking it would add.
    expect(geo).toEqual({
      key: 'geo',
      values: [
        { value: 'eu', count: 2, selected: true },
        { value: 'us', count: 2, selected: false }
      ]
    });
    // tier is counted over what geo has left, so the us Targets are gone.
    expect(tier.values).toEqual([{ value: 'primary', count: 1, selected: false }]);
  });

  test('a picked value keeps its entry once another facet empties it', () => {
    // b is the only secondary Target, and picking geo=eu leaves it out, so
    // tier=secondary has nothing left. It is picked, so without an entry there
    // would be no way to unpick it.
    const facets = toggleFacet(toggleFacet(noFacets(), 'geo', 'eu'), 'tier', 'secondary');
    const [, tier] = labelFacets(fleet(), ['geo', 'tier'], facets);
    expect(tier.values).toContainEqual({ value: 'secondary', count: 0, selected: true });
  });

  test('a key no surviving Target carries drops out of the panel', () => {
    expect(labelFacets(fleet(), ['geo', 'absent'], noFacets()).map((f) => f.key)).toEqual(['geo']);
  });

  test('the state tally ignores the picked states but honours the label picks', () => {
    const facets = toggleFacet(toggleFacet(noFacets(), 'geo', 'us'), STATE_FACET, 'Healthy');
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
