import { Freight } from '@ui/gen/api/v2/models';

// A run of consecutive freight from one Warehouse.
export type WarehouseGroup<T extends Freight> = {
  warehouse: string;
  // the group's oldest freight, which stays put while new freight arrives at
  // the front of the timeline
  key: string;
  freights: T[];
};

const warehouseOf = (freight: Freight) => freight?.origin?.name || '';

const idOf = (freight: Freight) => freight?.metadata?.uid || freight?.metadata?.name || '';

// groupByWarehouse splits the freight timeline into runs of consecutive
// freight from the same Warehouse, keeping the order, so the timeline can name
// the Warehouse once above each run. A Warehouse whose freight is interleaved
// with another's gets a run each time it comes back. Group the whole timeline
// rather than the page of it on screen, so that showing more freight does not
// change a group's key.
export const groupByWarehouse = <T extends Freight>(freights: T[]): WarehouseGroup<T>[] => {
  const groups: WarehouseGroup<T>[] = [];

  for (const freight of freights) {
    const warehouse = warehouseOf(freight);
    const last = groups[groups.length - 1];

    if (last && last.warehouse === warehouse) {
      last.freights.push(freight);
    } else {
      groups.push({ warehouse, key: '', freights: [freight] });
    }
  }

  for (const group of groups) {
    group.key = `${group.warehouse}-${idOf(group.freights[group.freights.length - 1])}`;
  }

  return groups;
};

// collapseUnused replaces the unused freight before each piece of freight in
// use with entries counting it, each a copy of the first freight it counts. The
// unused freight is split where the Warehouse changes, so that each count is
// grouped under the Warehouse its freight came from. Unused freight after the
// last freight in use is left out.
export const collapseUnused = <T extends Freight>(
  freights: T[],
  inUse: (freight: T) => boolean
): (T & { count?: number })[] => {
  const collapsed: (T & { count?: number })[] = [];
  let unused: T[] = [];

  for (const freight of freights) {
    if (!inUse(freight)) {
      unused.push(freight);
      continue;
    }

    for (const run of groupByWarehouse(unused)) {
      collapsed.push({ ...run.freights[0], count: run.freights.length });
    }

    unused = [];
    collapsed.push(freight);
  }

  return collapsed;
};

// takeFreight keeps the first `count` freight of the grouped timeline, and the
// groups they fall in, keys and all.
export const takeFreight = <T extends Freight>(
  groups: WarehouseGroup<T>[],
  count: number
): WarehouseGroup<T>[] => {
  const taken: WarehouseGroup<T>[] = [];
  let left = count;

  for (const group of groups) {
    if (left <= 0) {
      break;
    }

    taken.push({ ...group, freights: group.freights.slice(0, left) });
    left -= group.freights.length;
  }

  return taken;
};

// getHeaderShift returns how far a group's header moves right to stay in view
// when the timeline is paged partway into the group: to the group's first card
// in view, but never past its last card. `left` and `width` place the group in
// the strip; `offset` is how far the strip is paged.
export const getHeaderShift = (
  offset: number,
  left: number,
  width: number,
  cardWidth: number
): number => Math.min(Math.max(0, offset - left), Math.max(0, width - cardWidth));

// getFocusedWarehouse returns the one Warehouse the freight timeline shows, or
// '' for every Warehouse. Promoting shows the freight the Stage can take,
// whichever Warehouse the timeline was narrowed to.
export const getFocusedWarehouse = (freightTimelineWarehouse: string, isPromotionMode: boolean) =>
  isPromotionMode ? '' : freightTimelineWarehouse;

// filterToWarehouse keeps the freight from `warehouse`, or all of it when no
// Warehouse is given.
export const filterToWarehouse = <T extends Freight>(freights: T[], warehouse: string): T[] =>
  warehouse ? freights.filter((freight) => freight?.origin?.name === warehouse) : freights;

// getFocusableWarehouses returns the Warehouses the freight timeline can be
// narrowed to: those the pipeline's Warehouse filter shows, or every one when
// that filter is empty. Any other would leave the timeline empty.
export const getFocusableWarehouses = (warehouses: string[], shownWarehouses: string[]) =>
  shownWarehouses.length > 0
    ? warehouses.filter((warehouse) => shownWarehouses.includes(warehouse))
    : warehouses;
