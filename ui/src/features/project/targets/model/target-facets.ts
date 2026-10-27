import { TargetRow, stateCounts } from './target-rows';
import { StateCounts, TargetState } from './target-state';

// Facets is what the reader has picked. A facet with nothing picked does not
// filter at all, so the panel starts with nothing ticked and every Target in;
// picking a value narrows the fleet to it, and picking more values of the same
// facet widens it again to any of them.
export type Facets = {
  // Label values picked, keyed by label key.
  labels: Record<string, string[]>;
  // States picked.
  states: TargetState[];
};

export const noFacets = (): Facets => ({ labels: {}, states: [] });

// facetSelections counts what is picked, so a panel can offer to clear itself
// only when there is something to clear.
export const facetSelections = (facets: Facets): number =>
  facets.states.length +
  Object.values(facets.labels).reduce((sum, values) => sum + values.length, 0);

// toggleFacet picks or unpicks one value of one facet. The key is a label key,
// or STATE_FACET for the status facet.
export const STATE_FACET = '';

export const toggleFacet = (facets: Facets, key: string, value: string): Facets => {
  const flip = (values: string[]) =>
    values.includes(value) ? values.filter((held) => held !== value) : [...values, value];
  if (key === STATE_FACET) {
    return { ...facets, states: flip(facets.states) as TargetState[] };
  }
  return { ...facets, labels: { ...facets.labels, [key]: flip(facets.labels[key] || []) } };
};

// matchesLabelFacets evaluates the picked label values against a row: for
// every key with something picked, the row must carry one of them. A Target
// that does not carry the key matches none. skipKey leaves one key out, for
// counting that key's own values.
const matchesLabelFacets = (
  row: TargetRow,
  labels: Record<string, string[]>,
  skipKey?: string
): boolean =>
  Object.entries(labels).every(([key, picked]) => {
    if (key === skipKey || !picked.length) {
      return true;
    }
    const value = row.target.metadata?.labels?.[key];
    return value !== undefined && picked.includes(value);
  });

const matchesStateFacet = (row: TargetRow, states: TargetState[]): boolean =>
  !states.length || states.includes(row.state);

// matchesFacets reports whether a row survives everything the panel has
// picked.
export const matchesFacets = (row: TargetRow, facets: Facets): boolean =>
  matchesLabelFacets(row, facets.labels) && matchesStateFacet(row, facets.states);

export type FacetValue = { value: string; count: number; selected: boolean };

export type LabelFacet = { key: string; values: FacetValue[] };

// labelFacets tallies each key's values over the rows the OTHER facets have
// kept, so a figure says what picking that value would bring in rather than
// collapsing to nothing for every value but the ones already picked.
export const labelFacets = (rows: TargetRow[], keys: string[], facets: Facets): LabelFacet[] =>
  keys
    .map((key) => {
      const picked = facets.labels[key] || [];
      const counts = new Map<string, number>();
      for (const row of rows) {
        if (
          !matchesLabelFacets(row, facets.labels, key) ||
          !matchesStateFacet(row, facets.states)
        ) {
          continue;
        }
        const value = row.target.metadata?.labels?.[key];
        if (value !== undefined) {
          counts.set(value, (counts.get(value) || 0) + 1);
        }
      }
      // A picked value keeps its entry even once another facet has left it
      // nothing, so there is always a box to unpick it with.
      for (const value of picked) {
        counts.set(value, counts.get(value) || 0);
      }
      return {
        key,
        values: [...counts.entries()]
          .map(([value, count]) => ({ value, count, selected: picked.includes(value) }))
          .sort((lhs, rhs) => rhs.count - lhs.count || lhs.value.localeCompare(rhs.value))
      };
    })
    .filter((facet) => facet.values.length);

// stateFacet tallies the states over the rows the label facets have kept,
// ignoring the picked states, on the same principle as labelFacets.
export const stateFacet = (rows: TargetRow[], facets: Facets): StateCounts =>
  stateCounts(rows.filter((row) => matchesLabelFacets(row, facets.labels)));
