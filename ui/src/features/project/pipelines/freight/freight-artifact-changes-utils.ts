import { Freight, FreightReference } from '@ui/gen/api/v2/models';

import { FreightArtifactItem, getFreightArtifacts } from './freight-artifact-list-utils';
import { getFreightDiscoveredAt } from './get-freight-discovered-at';

// getPreviousFreight maps each piece of freight, by name, to the one its
// Warehouse discovered just before it. The oldest piece of each Warehouse has
// no entry. Pass every piece of freight, not a filtered list, so that
// filtering the timeline does not change what a card is compared with.
export const getPreviousFreight = (freights: Freight[]): Record<string, Freight> => {
  const oldestFirst = [...freights].sort(
    (a, b) =>
      (getFreightDiscoveredAt(a)?.getTime() || 0) - (getFreightDiscoveredAt(b)?.getTime() || 0)
  );

  const previous: Record<string, Freight> = {};
  const lastByWarehouse = new Map<string, Freight>();

  for (const freight of oldestFirst) {
    const warehouse = freight?.origin?.name || '';
    const before = lastByWarehouse.get(warehouse);

    if (before) {
      previous[freight?.metadata?.name || ''] = before;
    }

    lastByWarehouse.set(warehouse, freight);
  }

  return previous;
};

// Freight records each artifact as it was found, so the same artifact is the
// same in every piece of freight that carries it, down to its last field.
const identity = (artifact: FreightArtifactItem) => JSON.stringify(artifact);

// getChangedArtifacts picks out the artifacts of a piece of freight that
// differ from the previous one's: any artifact it did not carry exactly as it
// is. Different is all it takes -- an older version counts as much as a newer
// one, and so does another tag on the same image or commit, which Kargo makes
// new freight for too. With no previous freight to compare with, every
// artifact counts as changed.
export const getChangedArtifacts = (
  freight?: Freight | FreightReference,
  previous?: Freight | FreightReference
): Set<FreightArtifactItem> => {
  const before = new Set(getFreightArtifacts(previous).map(identity));

  return new Set(
    getFreightArtifacts(freight).filter((artifact) => !before.has(identity(artifact)))
  );
};
