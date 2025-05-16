import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { Watcher } from '@ui/features/project/pipelines/watcher';
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { useDocumentEvent } from '@ui/utils/document';

export const useEventsWatcher = (
  project: string,
  act: {
    onStage: (stage: Stage) => void;
    onWarehouse: (warehouse: Warehouse) => void;
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

    watcher.watchStages(act.onStage);
    watcher.watchWarehouses({
      onWarehouseEvent: act.onWarehouse
    });

    return () => {
      watcher.cancelWatch();
    };
  }, [isWindowVisible, project]);
};
