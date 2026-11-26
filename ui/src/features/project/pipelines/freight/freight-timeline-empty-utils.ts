/** Height the timeline holds while empty, so the strip does not jump. */
export const MIN_FREIGHT_TIMELINE_HEIGHT = 96;

export const emptyMessages = {
  promotion: 'No Freight from the Warehouses this Stage requests is available.',
  filtered: 'No Freight matches the current filters.',
  'no-warehouses': 'No Freight yet. Create a Warehouse to start discovering artifacts.',
  'no-controller': 'No Freight yet. No controller is running for the Warehouses in this project.',
  'no-freight': 'No Freight yet. It shows up here as Warehouses discover new artifacts.'
} as const;

export type EmptyVariant = keyof typeof emptyMessages;

export const getVariant = (args: {
  promotionMode: boolean;
  filtered: boolean;
  hasWarehouses: boolean;
  hasWarehouseController: boolean;
}): EmptyVariant => {
  if (args.promotionMode) {
    return 'promotion';
  }

  if (args.filtered) {
    return 'filtered';
  }

  if (!args.hasWarehouses) {
    return 'no-warehouses';
  }

  // Freight will not show up as promised while no controller is discovering
  // artifacts for any of the Warehouses.
  return args.hasWarehouseController ? 'no-freight' : 'no-controller';
};
