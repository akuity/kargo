import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { Watcher } from '@ui/features/project/pipelines/watcher';

export const useWatchPromotion = (project: string, promotion: string) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project || !promotion) {
      return;
    }

    const watcher = new Watcher(project, client);
    watcher.watchPromotion(promotion);

    return () => {
      watcher.cancelWatch();
    };
  }, [project, promotion]);
};
