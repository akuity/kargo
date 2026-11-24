import { Freight } from '@ui/gen/api/v2/models';

// When the Freight's artifacts were discovered. The API orders Freight by this, falling back to
// creationTimestamp for Freight created before discoveredAt existed. It can differ from
// creationTimestamp when Freight is created after the fact, e.g. for an older artifact version.
export const getFreightDiscoveredAt = (freight?: Freight): Date | null => {
  const timestamp = freight?.discoveredAt || freight?.metadata?.creationTimestamp;

  return timestamp ? new Date(timestamp) : null;
};
