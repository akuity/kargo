import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { Watcher } from '@ui/features/project/pipelines/watcher';

export const useWatchFreight = (project: string) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project) {
      return;
    }

    const watcher = new Watcher(project, client);
    watcher.watchFreights();

    return () => {
      watcher.cancelWatch();
    };
  }, [project]);
};
