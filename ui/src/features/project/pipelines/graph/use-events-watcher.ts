import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import { Watcher } from '@ui/features/project/pipelines/watcher';
import { queryCache } from '@ui/features/utils/cache';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { useDocumentEvent } from '@ui/utils/document';

export const useEventsWatcher = (
  project: string,
  act?: {
    onStage: (stage: Stage) => void;
    onWarehouse: (warehouse: WarehouseExpanded) => void;
  }
) => {
  const client = useQueryClient();
  const isWindowVisible = useDocumentEvent(
    'visibilitychange',
    () => document.visibilityState === 'visible'
  );

  useEffect(() => {
    if (!isWindowVisible || !project) {
      return;
    }

    const watcher = new Watcher(project, client);

    watcher.watchStages(act?.onStage);
    watcher.watchWarehouses({
      onWarehouseEvent: act?.onWarehouse,
      refreshHook: queryCache.freight.refetchQueryFreight
    });

    return () => {
      watcher.cancelWatch();
    };
  }, [isWindowVisible, project]);
};
