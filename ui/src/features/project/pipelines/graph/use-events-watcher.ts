import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { Watcher } from '@ui/features/project/pipelines/watcher';
import { queryCache } from '@ui/features/utils/cache';
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';

export const useEventsWatcher = (
  project: string,
  act?: {
    onStage: (stage: Stage) => void;
    onWarehouse: (warehouse: Warehouse) => void;
  },
  warehouses?: string[]
) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project) {
      return;
    }

    let watcher: Watcher | undefined;

    // (Re)establish the watch streams. Aborts any previous watcher first so we
    // never leak a connection.
    const connect = () => {
      watcher?.cancelWatch();
      watcher = new Watcher(project, client);
      watcher.watchStages(act?.onStage, warehouses);
      watcher.watchWarehouses({
        onWarehouseEvent: act?.onWarehouse,
        refreshHook: queryCache.freight.refetchQueryFreight
      });
    };

    connect();

    // Reconnect when the tab becomes visible again. The watch is left running
    // while the tab is hidden (we do NOT cancel on leave), but a hidden tab can
    // be throttled/frozen and have its stream silently dropped -- reconnecting
    // on return guarantees a live stream again.
    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        connect();
      }
    };
    document.addEventListener('visibilitychange', onVisibilityChange);

    return () => {
      document.removeEventListener('visibilitychange', onVisibilityChange);
      watcher?.cancelWatch();
    };
  }, [project, (warehouses || []).join(',')]);
};
