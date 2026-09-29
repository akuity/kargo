/**
 * Height, in pixels, the freight timeline holds while it has no cards to show,
 * so the strip does not jump when the first piece of Freight arrives.
 */
export const MIN_FREIGHT_TIMELINE_HEIGHT = 96;

export const emptyMessages = {
  promotion: 'No Freight from the Warehouses this Stage requests is available.',
  filtered: 'No Freight matches the current filters.',
  'no-warehouses': 'No Freight yet. Create a Warehouse to start discovering artifacts.',
  'no-freight': 'No Freight yet. It shows up here as Warehouses discover new artifacts.'
} as const;

export type EmptyVariant = keyof typeof emptyMessages;

/**
 * Picks the reason the freight timeline is empty, most specific first: an
 * in-progress promotion narrows to a Stage's origins, explicit filters hide
 * what does exist, and otherwise the project simply has no Freight -- with or
 * without a Warehouse to produce it.
 */
export const getVariant = (args: {
  promotionMode: boolean;
  filtered: boolean;
  hasWarehouses: boolean;
}): EmptyVariant => {
  if (args.promotionMode) {
    return 'promotion';
  }

  if (args.filtered) {
    return 'filtered';
  }

  return args.hasWarehouses ? 'no-freight' : 'no-warehouses';
};
