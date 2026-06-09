import { queryCache } from '@ui/features/utils/cache';
import { Stage, Warehouse } from '@ui/gen/api/v2/models';
import { useDocumentEvent } from '@ui/utils/document';

import { useWatchStages } from '../use-watch-stages';
import { useWatchWarehouses } from '../use-watch-warehouses';

export const useEventsWatcher = (
  project: string,
  act?: {
    onStage: (stage: Stage) => void;
    onWarehouse: (warehouse: Warehouse) => void;
  },
  warehouses?: string[]
) => {
  const isWindowVisible = useDocumentEvent(
    'visibilitychange',
    () => document.visibilityState === 'visible'
  );

  // Pass empty string when not visible — each hook's guard handles it
  const activeProject = isWindowVisible ? project : '';

  useWatchStages(activeProject, act?.onStage, warehouses);
  useWatchWarehouses(activeProject, {
    refreshHook: () => queryCache.freight.refetchQueryFreight(project),
    onWarehouseEvent: act?.onWarehouse
  });
};
