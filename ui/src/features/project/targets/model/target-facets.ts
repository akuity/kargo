import { TargetRow, stateCounts } from './target-rows';
import { StateCounts, TargetState } from './target-state';

// Facets is what the reader has switched OFF, not on. Holding exclusions means
// a label value that appears later is included by default rather than silently
// filtered out by a choice made before it existed.
export type Facets = {
  // Label values switched off, keyed by label key.
  labels: Record<string, string[]>;
  // States switched off.
  states: TargetState[];
};

export const noFacets = (): Facets => ({ labels: {}, states: [] });

// facetExclusions counts what is switched off, so a panel can offer to clear
// itself only when there is something to clear.
export const facetExclusions = (facets: Facets): number =>
  facets.states.length +
  Object.values(facets.labels).reduce((sum, values) => sum + values.length, 0);

// toggleFacet switches one value of one facet on or off. The key is a label
// key, or STATE_FACET for the status facet.
export const STATE_FACET = '';

export const toggleFacet = (facets: Facets, key: string, value: string): Facets => {
  const drop = (values: string[]) =>
    values.includes(value) ? values.filter((held) => held !== value) : [...values, value];
  if (key === STATE_FACET) {
    return { ...facets, states: drop(facets.states) as TargetState[] };
  }
  return { ...facets, labels: { ...facets.labels, [key]: drop(facets.labels[key] || []) } };
};

// matchesLabelFacets evaluates the label exclusions against a row. A Target
// that does not carry a key is never excluded by it. skipKey leaves one key
// out, for counting that key's own values.
const matchesLabelFacets = (
  row: TargetRow,
  labels: Record<string, string[]>,
  skipKey?: string
): boolean =>
  Object.entries(labels).every(([key, excluded]) => {
    if (key === skipKey || !excluded.length) {
      return true;
    }
    const value = row.target.labels?.[key];
    return value === undefined || !excluded.includes(value);
  });

// matchesFacets reports whether a row survives everything the panel has
// switched off.
export const matchesFacets = (row: TargetRow, facets: Facets): boolean =>
  matchesLabelFacets(row, facets.labels) && !facets.states.includes(row.state);

export type FacetValue = { value: string; count: number; excluded: boolean };

export type LabelFacet = { key: string; values: FacetValue[] };

// labelFacets tallies each key's values over the rows the OTHER facets have
// kept, so a figure says what switching that value back on would bring rather
// than collapsing to nothing the moment it is switched off.
export const labelFacets = (rows: TargetRow[], keys: string[], facets: Facets): LabelFacet[] =>
  keys
    .map((key) => {
      const excluded = facets.labels[key] || [];
      const counts = new Map<string, number>();
      for (const row of rows) {
        if (!matchesLabelFacets(row, facets.labels, key) || facets.states.includes(row.state)) {
          continue;
        }
        const value = row.target.labels?.[key];
        if (value !== undefined) {
          counts.set(value, (counts.get(value) || 0) + 1);
        }
      }
      // A value switched off keeps its entry even once another facet has left
      // it nothing, so there is always a box to switch it back on with.
      for (const value of excluded) {
        counts.set(value, counts.get(value) || 0);
      }
      return {
        key,
        values: [...counts.entries()]
          .map(([value, count]) => ({ value, count, excluded: excluded.includes(value) }))
          .sort((lhs, rhs) => rhs.count - lhs.count || lhs.value.localeCompare(rhs.value))
      };
    })
    .filter((facet) => facet.values.length);

// stateFacet tallies the states over the rows the label facets have kept,
// ignoring the state exclusions, on the same principle as labelFacets.
export const stateFacet = (rows: TargetRow[], facets: Facets): StateCounts =>
  stateCounts(rows.filter((row) => matchesLabelFacets(row, facets.labels)));
