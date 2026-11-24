import { useEffect, useRef } from 'react';

import { queryCache } from '@ui/features/utils/cache';
import { Freight, Stage } from '@ui/gen/api/v2/models';

// Freight that Stages hold but the Freight list lacks, sorted so the joined
// form is a stable effect dependency.
export const missingFreightNames = (
  freights: Record<string, Freight>,
  freightInStages: Record<string, Stage[]>
): string[] =>
  Object.keys(freightInStages)
    .filter((name) => !freights[name])
    .sort();

// Refetches the Freight list when a Stage holds Freight the list lacks, meaning
// the list is stale. At most once per name: the gap can be permanent (deleted
// Freight, or a Warehouse the origin filter excludes while the Stage still
// reports one piece of Freight per Warehouse), and retrying would loop -- every
// response is a new object, so every refetch re-renders.
export const useSyncFreight = (payload: {
  project: string;
  freights?: Record<string, Freight>;
  freightInStages?: Record<string, Stage[]>;
}) => {
  const { project, freights, freightInStages } = payload;

  const refetchedFor = useRef(new Set<string>());

  const missingKey =
    freights && freightInStages ? missingFreightNames(freights, freightInStages).join(',') : '';

  useEffect(() => {
    const unseen = missingKey.split(',').filter((name) => name && !refetchedFor.current.has(name));

    if (!unseen.length) {
      return;
    }

    for (const name of unseen) {
      refetchedFor.current.add(name);
    }

    queryCache.freight.refetchQueryFreight(project);
  }, [project, missingKey]);
};
